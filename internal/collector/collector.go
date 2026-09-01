package collector

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fengcyborg/restic-backup-dashboard/internal/config"
	"github.com/fengcyborg/restic-backup-dashboard/internal/model"
)

type Collector struct {
	config config.Config
	runner Runner
	now    func() time.Time
}

type Option func(*Collector)

func WithRunner(runner Runner) Option {
	return func(collector *Collector) {
		collector.runner = runner
	}
}

func WithClock(now func() time.Time) Option {
	return func(collector *Collector) {
		collector.now = now
	}
}

func New(cfg config.Config, options ...Option) *Collector {
	collector := &Collector{config: cfg, runner: OSRunner{}, now: time.Now}
	for _, option := range options {
		option(collector)
	}
	return collector
}

func (c *Collector) Collect(ctx context.Context) model.Status {
	now := c.now()
	host := strings.TrimSpace(c.config.Hostname)
	if host == "" {
		host, _ = os.Hostname()
	}

	records, events := c.collectEvents()
	phases := c.collectPhases(ctx)
	tasks := make([]model.Task, 0, len(c.config.Tasks))
	for _, taskConfig := range c.config.Tasks {
		tasks = append(tasks, c.collectTask(ctx, now, taskConfig, phases[taskConfig.ID]))
	}

	syncStatus := c.collectSync(now, tasks)
	c.applySyncStatus(tasks, syncStatus)
	recovery := c.collectRecovery(now, tasks, records)
	storage := c.collectStorage(ctx)
	dependencies := c.collectDependencies(ctx)
	overall := c.collectOverall(tasks, recovery, storage, dependencies)

	return model.Status{
		SchemaVersion: 1,
		GeneratedAt:   now.Format(time.RFC3339),
		Host:          host,
		Display:       c.config.Display,
		Overall:       overall,
		Pipeline:      tasks,
		Sync:          syncStatus,
		Recovery:      recovery,
		Storage:       storage,
		Dependencies:  dependencies,
		Events:        events,
	}
}

func (c *Collector) collectTask(ctx context.Context, now time.Time, task config.Task, detectedPhase string) model.Task {
	service := c.systemdUnit(ctx, task.Service)
	timer := c.systemdUnit(ctx, task.Timer)
	success := readMarker(c.config.Resolve(task.SuccessMarker))
	active := task.Service != "" && service.Available && serviceIsRunning(service.ActiveState)
	status := taskStatus(now, task, service, timer, success, active)

	result := model.Task{
		ID:              task.ID,
		Name:            task.Name,
		Schedule:        task.Schedule,
		Status:          status,
		Active:          active,
		DurationSeconds: readDurationSeconds(c.config.Resolve(task.DurationMarker)),
		Service:         service,
		Timer:           timer,
	}
	if success.found {
		result.LastSuccess = success.time.Format(time.RFC3339)
		age := max(int64(0), int64(now.Sub(success.time).Seconds()))
		result.AgeSeconds = &age
	}
	if timer.NextRun != "" {
		result.NextRun = timer.NextRun
	}
	if active {
		result.Phase = firstNonEmpty(detectedPhase, task.RunningLabel, "Task is running")
	}
	return result
}

func serviceIsRunning(activeState string) bool {
	switch activeState {
	case "active", "activating", "reloading", "deactivating":
		return true
	default:
		return false
	}
}

func taskStatus(now time.Time, task config.Task, service, timer model.SystemdUnit, success marker, active bool) string {
	if active {
		return "running"
	}
	if task.Service != "" && service.Available {
		failedExit := service.ExitStatus != nil && *service.ExitStatus != 0 && *service.ExitStatus != 75
		if service.Result == "failed" || failedExit {
			return "error"
		}
	}
	if task.Timer != "" && (!timer.Available || timer.ActiveState != "active") {
		return "error"
	}
	if !success.found {
		return "unknown"
	}
	age := now.Sub(success.time)
	if age <= task.HealthyFor.Duration {
		return "healthy"
	}
	if age <= task.ErrorAfter.Duration {
		return "warning"
	}
	return "error"
}

func (c *Collector) systemdUnit(ctx context.Context, name string) model.SystemdUnit {
	if name == "" {
		return model.SystemdUnit{}
	}
	argv := []string{
		"systemctl", "show", name, "--no-pager",
		"--property=LoadState", "--property=ActiveState", "--property=SubState",
		"--property=Result", "--property=ExecMainStatus",
		"--property=NextElapseUSecRealtime", "--property=LastTriggerUSec",
	}
	output, err := c.runner.Run(ctx, argv)
	if err != nil {
		return model.SystemdUnit{}
	}
	properties := parseSystemdProperties(output)
	unit := model.SystemdUnit{
		LoadState:   properties["LoadState"],
		ActiveState: properties["ActiveState"],
		SubState:    properties["SubState"],
		Result:      properties["Result"],
		Available:   properties["LoadState"] == "loaded",
	}
	if exitStatus, err := strconv.Atoi(properties["ExecMainStatus"]); err == nil {
		unit.ExitStatus = &exitStatus
	}
	if parsed, ok := parseTimestamp(properties["NextElapseUSecRealtime"]); ok {
		unit.NextRun = parsed.Format(time.RFC3339)
	}
	if parsed, ok := parseTimestamp(properties["LastTriggerUSec"]); ok {
		unit.LastTrigger = parsed.Format(time.RFC3339)
	}
	return unit
}

func (c *Collector) collectPhases(ctx context.Context) map[string]string {
	result := make(map[string]string)
	if len(c.config.PhaseRules) == 0 {
		return result
	}
	output, err := c.runner.Run(ctx, []string{"ps", "-eo", "args="})
	if err != nil {
		return result
	}
	lines := strings.Split(output, "\n")
	for _, rule := range c.config.PhaseRules {
		for _, line := range lines {
			matched := len(rule.ContainsAll) > 0
			for _, term := range rule.ContainsAll {
				if term == "" || !strings.Contains(line, term) {
					matched = false
					break
				}
			}
			if matched {
				result[rule.TaskID] = rule.Label
				break
			}
		}
	}
	return result
}

func (c *Collector) collectSync(now time.Time, tasks []model.Task) model.Sync {
	result := model.Sync{TaskID: c.config.Sync.TaskID}
	if c.config.Sync.TaskID == "" {
		return result
	}
	local := readMarker(c.config.Resolve(c.config.Sync.SourceGenerationMarker))
	uploaded := readMarker(c.config.Resolve(c.config.Sync.UploadedGenerationMarker))
	if local.found {
		result.LocalGeneration = local.value
	}
	if uploaded.found {
		result.UploadedGeneration = uploaded.value
	}
	if local.found && uploaded.found {
		caughtUp := local.value == uploaded.value
		result.CaughtUp = &caughtUp
		lag := int64(0)
		if !caughtUp {
			lag = max(0, int64(now.Sub(uploaded.time).Seconds()))
		}
		result.LagSeconds = &lag
	} else if local.found {
		caughtUp := false
		lag := max(0, int64(now.Sub(local.time).Seconds()))
		result.CaughtUp = &caughtUp
		result.LagSeconds = &lag
	}
	if task, ok := findTask(tasks, c.config.Sync.TaskID); ok && task.Active {
		path := newestMatchingFile(c.config.Resolve(c.config.Sync.ProgressLogGlob))
		if progress := parseProgressFile(path, task.Phase); progress != nil {
			result.Progress = &model.Progress{
				TransferredBytes:    progress.transferred,
				TotalBytes:          progress.total,
				Percent:             progress.percent,
				SpeedBytesPerSecond: progress.speed,
				ETASeconds:          progress.eta,
				Phase:               progress.phase,
			}
		}
	}
	return result
}

func (c *Collector) applySyncStatus(tasks []model.Task, syncStatus model.Sync) {
	if c.config.Sync.TaskID == "" || syncStatus.CaughtUp == nil || *syncStatus.CaughtUp {
		return
	}
	for index := range tasks {
		if tasks[index].ID != c.config.Sync.TaskID || tasks[index].Active {
			continue
		}
		if syncStatus.LagSeconds != nil && time.Duration(*syncStatus.LagSeconds)*time.Second > c.config.Sync.ErrorAfter.Duration {
			tasks[index].Status = "error"
		} else {
			tasks[index].Status = "warning"
		}
		return
	}
}

func (c *Collector) collectRecovery(now time.Time, tasks []model.Task, records []eventRecord) model.Recovery {
	result := model.Recovery{
		Status:        "unknown",
		Retention:     c.config.Recovery.Retention,
		SamplePercent: c.config.Recovery.SamplePercent,
		Checks:        make([]model.RecoveryCheck, 0, len(c.config.Recovery.Checks)),
		Datasets:      make([]model.Dataset, 0, len(c.config.Recovery.Datasets)),
	}
	if c.config.Recovery.AuditTaskID == "" {
		return result
	}
	auditRecord, hasAudit := findLatestEvent(records, c.config.Recovery.AuditSourceID, c.config.Recovery.SuccessEvent)
	auditFresh := false
	if hasAudit {
		result.LastAudit = auditRecord.timestamp.Format(time.RFC3339)
		age := max(int64(0), int64(now.Sub(auditRecord.timestamp).Seconds()))
		result.AuditAgeSeconds = &age
		if task, ok := findTaskConfig(c.config.Tasks, c.config.Recovery.AuditTaskID); ok {
			auditFresh = now.Sub(auditRecord.timestamp) <= task.ErrorAfter.Duration
		}
		if count, ok := positiveInteger(auditRecord.values[c.config.Sync.SnapshotCountKey]); ok {
			result.RemoteSnapshotCount = &count
		}
	}
	if result.RemoteSnapshotCount == nil && c.config.Sync.SuccessEvent != "" {
		if syncRecord, ok := findLatestEvent(records, "", c.config.Sync.SuccessEvent); ok {
			if count, valid := positiveInteger(syncRecord.values[c.config.Sync.SnapshotCountKey]); valid {
				result.RemoteSnapshotCount = &count
			}
		}
	}
	if _, ok := findLatestEvent(records, c.config.Recovery.SemanticSourceID, c.config.Recovery.SemanticEvent); ok {
		result.LocalSemanticCheckAvailable = true
	}

	allDatasetsReady := true
	var total int64
	hasTotal := false
	for _, datasetConfig := range c.config.Recovery.Datasets {
		summary := parseResticSummary(c.config.Resolve(datasetConfig.SummaryFile))
		dataset := model.Dataset{
			ID:             datasetConfig.ID,
			Name:           datasetConfig.Name,
			LocalSnapshot:  summary.snapshotID,
			LocalBytes:     summary.totalBytes,
			FilesProcessed: summary.filesProcessed,
			FilesNew:       summary.filesNew,
			FilesChanged:   summary.filesChanged,
		}
		if hasAudit {
			if size, ok := positiveInteger(auditRecord.values[datasetConfig.AuditSizeKey]); ok {
				dataset.VerifiedRestoreBytes = &size
				total += size
				hasTotal = true
				if size < datasetConfig.MinimumBytes {
					allDatasetsReady = false
				}
			} else {
				allDatasetsReady = false
			}
			dataset.RemoteSnapshot = sanitizeSnapshot(auditRecord.values[datasetConfig.AuditSnapshotKey])
			if dataset.RemoteSnapshot == "" {
				allDatasetsReady = false
			}
		} else {
			allDatasetsReady = false
		}
		result.Datasets = append(result.Datasets, dataset)
	}
	if hasTotal {
		result.TotalRestoreBytes = &total
	}
	for _, check := range c.config.Recovery.Checks {
		result.Checks = append(result.Checks, model.RecoveryCheck{
			ID:        check.ID,
			Name:      check.Name,
			Passed:    auditFresh && allDatasetsReady,
			CheckedAt: result.LastAudit,
		})
	}
	ready := auditFresh && allDatasetsReady
	if ready {
		result.Status = "healthy"
	} else if !hasAudit {
		result.Status = "unknown"
	} else {
		result.Status = "error"
	}
	return result
}

func (c *Collector) collectStorage(ctx context.Context) model.Storage {
	result := model.Storage{Dataset: c.config.Storage.ZFSDataset, Status: "unknown"}
	if c.config.Storage.ZFSDataset != "" {
		output, err := c.runner.Run(ctx, []string{"zfs", "list", "-Hp", "-o", "used,avail,quota", c.config.Storage.ZFSDataset})
		if err == nil {
			fields := strings.Fields(output)
			if len(fields) >= 3 {
				used, usedErr := strconv.ParseInt(fields[0], 10, 64)
				available, availableErr := strconv.ParseInt(fields[1], 10, 64)
				quota, quotaErr := strconv.ParseInt(fields[2], 10, 64)
				if usedErr == nil && availableErr == nil {
					result.UsedBytes = &used
					result.AvailableBytes = &available
					if quotaErr != nil || quota <= 0 {
						quota = used + available
					}
					result.QuotaBytes = &quota
					if quota > 0 {
						percent := float64(used) / float64(quota) * 100
						result.UsedPercent = &percent
						result.Status = thresholdStatus(percent, c.config.Storage.WarnPercent, c.config.Storage.ErrorPercent)
					}
				}
			}
		}
		if result.UsedBytes == nil {
			result.Status = "error"
		}
	}

	result.Pool = model.PoolHealth{Status: "unknown", Summary: "Pool health is not configured"}
	if len(c.config.Storage.PoolCommand) > 0 {
		output, err := c.runner.Run(ctx, c.config.Storage.PoolCommand)
		healthy := err == nil && strings.Contains(strings.ToLower(output), strings.ToLower(c.config.Storage.PoolHealthyContains))
		result.Pool.Healthy = healthy
		if healthy {
			result.Pool.Status = "healthy"
			result.Pool.Summary = "Storage pool reports healthy"
		} else {
			result.Pool.Status = "error"
			result.Pool.Summary = "Storage pool needs attention"
			result.Status = "error"
		}
	}
	return result
}

func thresholdStatus(value, warning, critical float64) string {
	if value >= critical {
		return "error"
	}
	if value >= warning {
		return "warning"
	}
	return "healthy"
}

func (c *Collector) collectDependencies(ctx context.Context) []model.Dependency {
	result := make([]model.Dependency, 0, len(c.config.Dependencies))
	for _, dependency := range c.config.Dependencies {
		healthy := false
		switch dependency.Kind {
		case "file":
			info, err := os.Stat(c.config.Resolve(dependency.Path))
			healthy = err == nil && info.Mode().IsRegular() && info.Size() > 0
		case "command":
			output, err := c.runner.Run(ctx, dependency.Command)
			output = strings.TrimSpace(output)
			healthy = err == nil
			if healthy && dependency.ExpectedExact != "" {
				healthy = output == dependency.ExpectedExact
			}
			if healthy && dependency.ExpectedContains != "" {
				healthy = strings.Contains(output, dependency.ExpectedContains)
			}
		}
		status := "error"
		summary := firstNonEmpty(dependency.ErrorSummary, "Dependency check failed")
		if healthy {
			status = "healthy"
			summary = firstNonEmpty(dependency.HealthySummary, "Available")
		}
		result = append(result, model.Dependency{ID: dependency.ID, Name: dependency.Name, Status: status, Summary: summary})
	}
	return result
}

func (c *Collector) collectEvents() ([]eventRecord, []model.Event) {
	records := make([]eventRecord, 0)
	for _, source := range c.config.Events.Sources {
		labels := make(map[string]eventLabel, len(source.Events))
		for token, label := range source.Events {
			labels[token] = eventLabel{severity: label.Severity, title: label.Title}
		}
		matches, _ := filepath.Glob(c.config.Resolve(source.Glob))
		for _, path := range matches {
			scanTailLines(path, maxInputFileSize, func(line string) {
				if record, ok := parseEventLine(line, source.ID, source.Name, labels); ok {
					records = append(records, record)
				}
			})
		}
	}
	sort.SliceStable(records, func(i, j int) bool { return records[i].timestamp.After(records[j].timestamp) })
	if len(records) > 5000 {
		records = records[:5000]
	}
	limit := min(c.config.Events.MaxItems, len(records))
	events := make([]model.Event, 0, limit)
	for _, record := range records[:limit] {
		events = append(events, model.Event{
			Timestamp: record.timestamp.Format(time.RFC3339),
			Severity:  record.severity,
			Source:    record.source,
			Title:     record.title,
			Detail:    eventDetail(record.values),
		})
	}
	return records, events
}

func (c *Collector) collectOverall(tasks []model.Task, recovery model.Recovery, storage model.Storage, dependencies []model.Dependency) model.Overall {
	statuses := make([]string, 0, len(tasks)+len(dependencies)+2)
	for _, task := range tasks {
		statuses = append(statuses, task.Status)
	}
	for _, dependency := range dependencies {
		statuses = append(statuses, dependency.Status)
	}
	if c.config.Recovery.AuditTaskID != "" {
		statuses = append(statuses, recovery.Status)
	}
	if c.config.Storage.ZFSDataset != "" || len(c.config.Storage.PoolCommand) > 0 {
		statuses = append(statuses, storage.Status)
	}
	status := aggregateStatus(statuses)
	return model.Overall{
		Status:          status,
		Summary:         overallSummary(status),
		RecoveryReady:   recovery.Status == "healthy",
		RecoverySummary: recoverySummary(recovery.Status),
	}
}

func aggregateStatus(statuses []string) string {
	hasWarning := false
	hasUnknown := false
	hasRunning := false
	for _, status := range statuses {
		switch status {
		case "error":
			return "error"
		case "warning":
			hasWarning = true
		case "unknown":
			hasUnknown = true
		case "running":
			hasRunning = true
		}
	}
	if hasWarning || hasUnknown {
		return "warning"
	}
	if hasRunning {
		return "running"
	}
	return "healthy"
}

func overallSummary(status string) string {
	switch status {
	case "healthy":
		return "Backup, offsite sync, and recovery verification are within policy"
	case "running":
		return "A backup workflow is currently running"
	case "warning":
		return "The backup workflow is delayed or has incomplete status data"
	default:
		return "At least one backup control needs attention"
	}
}

func recoverySummary(status string) string {
	switch status {
	case "healthy":
		return "The latest restore verification passed"
	case "error":
		return "The restore verification is stale or failed policy checks"
	default:
		return "No current restore verification is available"
	}
}

func findTask(tasks []model.Task, id string) (model.Task, bool) {
	for _, task := range tasks {
		if task.ID == id {
			return task, true
		}
	}
	return model.Task{}, false
}

func findTaskConfig(tasks []config.Task, id string) (config.Task, bool) {
	for _, task := range tasks {
		if task.ID == id {
			return task, true
		}
	}
	return config.Task{}, false
}

func findLatestEvent(records []eventRecord, sourceID, token string) (eventRecord, bool) {
	if token == "" {
		return eventRecord{}, false
	}
	for _, record := range records {
		if record.token == token && (sourceID == "" || record.sourceID == sourceID) {
			return record, true
		}
	}
	return eventRecord{}, false
}

func (c *Collector) String() string {
	return fmt.Sprintf("collector(base_dir=%q, tasks=%d)", c.config.BaseDir, len(c.config.Tasks))
}
