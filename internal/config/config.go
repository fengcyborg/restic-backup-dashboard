package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/fengcyborg/restic-backup-dashboard/internal/model"
)

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("duration must be a string such as 30m or 8h: %w", err)
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", value, err)
	}
	d.Duration = parsed
	return nil
}

type Config struct {
	Display      model.Display `json:"display"`
	BaseDir      string        `json:"base_dir"`
	OutputFile   string        `json:"output_file"`
	Hostname     string        `json:"hostname"`
	Tasks        []Task        `json:"tasks"`
	Sync         Sync          `json:"sync"`
	Recovery     Recovery      `json:"recovery"`
	Storage      Storage       `json:"storage"`
	Dependencies []Dependency  `json:"dependencies"`
	Events       Events        `json:"events"`
	PhaseRules   []PhaseRule   `json:"phase_rules"`
}

type Task struct {
	ID             string   `json:"id"`
	Kind           string   `json:"kind"`
	Name           string   `json:"name"`
	Schedule       string   `json:"schedule"`
	Service        string   `json:"service"`
	Timer          string   `json:"timer"`
	SuccessMarker  string   `json:"success_marker"`
	DurationMarker string   `json:"duration_marker"`
	HealthyFor     Duration `json:"healthy_for"`
	ErrorAfter     Duration `json:"error_after"`
	RunningLabel   string   `json:"running_label"`
}

type Sync struct {
	TaskID                   string   `json:"task_id"`
	SourceGenerationMarker   string   `json:"source_generation_marker"`
	UploadedGenerationMarker string   `json:"uploaded_generation_marker"`
	ProgressLogGlob          string   `json:"progress_log_glob"`
	SuccessEvent             string   `json:"success_event"`
	ErrorAfter               Duration `json:"error_after"`
	SnapshotCountKey         string   `json:"snapshot_count_key"`
}

type Recovery struct {
	AuditTaskID      string          `json:"audit_task_id"`
	AuditSourceID    string          `json:"audit_source_id"`
	SuccessEvent     string          `json:"success_event"`
	SamplePercent    float64         `json:"sample_percent"`
	Retention        model.Retention `json:"retention"`
	Datasets         []Dataset       `json:"datasets"`
	Checks           []Check         `json:"checks"`
	SemanticEvent    string          `json:"local_semantic_event"`
	SemanticSourceID string          `json:"local_semantic_source_id"`
}

type Dataset struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	SummaryFile      string `json:"summary_file"`
	AuditSizeKey     string `json:"audit_size_key"`
	AuditSnapshotKey string `json:"audit_snapshot_key"`
	MinimumBytes     int64  `json:"minimum_bytes"`
}

type Check struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Storage struct {
	ZFSDataset          string   `json:"zfs_dataset"`
	WarnPercent         float64  `json:"warn_percent"`
	ErrorPercent        float64  `json:"error_percent"`
	PoolCommand         []string `json:"pool_command"`
	PoolHealthyContains string   `json:"pool_healthy_contains"`
}

type Dependency struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Kind             string   `json:"kind"`
	Path             string   `json:"path"`
	Command          []string `json:"command"`
	ExpectedExact    string   `json:"expected_exact"`
	ExpectedContains string   `json:"expected_contains"`
	HealthySummary   string   `json:"healthy_summary"`
	ErrorSummary     string   `json:"error_summary"`
}

type Events struct {
	Sources  []EventSource `json:"sources"`
	MaxItems int           `json:"max_items"`
}

type EventSource struct {
	ID     string                `json:"id"`
	Name   string                `json:"name"`
	Glob   string                `json:"glob"`
	Events map[string]EventLabel `json:"event_labels"`
}

type EventLabel struct {
	Severity string `json:"severity"`
	Title    string `json:"title"`
}

type PhaseRule struct {
	TaskID      string   `json:"task_id"`
	ContainsAll []string `json:"contains_all"`
	Label       string   `json:"label"`
}

func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if err := cfg.applyDefaultsAndValidate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c *Config) applyDefaultsAndValidate() error {
	if strings.TrimSpace(c.BaseDir) == "" {
		return errors.New("base_dir is required")
	}
	if c.Display.Title == "" {
		c.Display.Title = "Restic Backup Dashboard"
	}
	if c.Display.Subtitle == "" {
		c.Display.Subtitle = "Restic backup observability"
	}
	if c.Display.ProviderName == "" {
		c.Display.ProviderName = "Offsite storage"
	}
	if c.Display.RepositoryName == "" {
		c.Display.RepositoryName = "Restic repository"
	}
	if c.OutputFile == "" {
		c.OutputFile = "dashboard/status.json"
	}
	if len(c.Tasks) == 0 {
		return errors.New("at least one task is required")
	}
	ids := make(map[string]struct{}, len(c.Tasks))
	for i := range c.Tasks {
		task := &c.Tasks[i]
		if task.ID == "" || task.Name == "" {
			return fmt.Errorf("tasks[%d]: id and name are required", i)
		}
		if _, exists := ids[task.ID]; exists {
			return fmt.Errorf("duplicate task id %q", task.ID)
		}
		ids[task.ID] = struct{}{}
		if !slices.Contains([]string{"backup", "sync", "audit"}, task.Kind) {
			return fmt.Errorf("task %q: kind must be backup, sync, or audit", task.ID)
		}
		if task.HealthyFor.Duration <= 0 || task.ErrorAfter.Duration <= 0 {
			return fmt.Errorf("task %q: healthy_for and error_after must be positive", task.ID)
		}
		if task.ErrorAfter.Duration < task.HealthyFor.Duration {
			return fmt.Errorf("task %q: error_after must not be shorter than healthy_for", task.ID)
		}
	}
	if c.Sync.TaskID != "" {
		if _, ok := ids[c.Sync.TaskID]; !ok {
			return fmt.Errorf("sync.task_id %q does not reference a task", c.Sync.TaskID)
		}
		if c.Sync.ErrorAfter.Duration <= 0 {
			return errors.New("sync.error_after must be positive")
		}
		if c.Sync.SourceGenerationMarker == "" || c.Sync.UploadedGenerationMarker == "" {
			return errors.New("sync generation markers are required when sync.task_id is set")
		}
	}
	if c.Recovery.AuditTaskID != "" {
		if _, ok := ids[c.Recovery.AuditTaskID]; !ok {
			return fmt.Errorf("recovery.audit_task_id %q does not reference a task", c.Recovery.AuditTaskID)
		}
		if c.Recovery.SuccessEvent == "" {
			return errors.New("recovery.success_event is required when recovery.audit_task_id is set")
		}
	}
	if c.Recovery.SamplePercent < 0 || c.Recovery.SamplePercent > 100 {
		return errors.New("recovery.sample_percent must be between 0 and 100")
	}
	datasetIDs := make(map[string]struct{}, len(c.Recovery.Datasets))
	for i, dataset := range c.Recovery.Datasets {
		if dataset.ID == "" || dataset.Name == "" {
			return fmt.Errorf("recovery.datasets[%d]: id and name are required", i)
		}
		if _, exists := datasetIDs[dataset.ID]; exists {
			return fmt.Errorf("duplicate recovery dataset id %q", dataset.ID)
		}
		datasetIDs[dataset.ID] = struct{}{}
		if dataset.MinimumBytes < 0 {
			return fmt.Errorf("recovery dataset %q: minimum_bytes cannot be negative", dataset.ID)
		}
		if c.Recovery.AuditTaskID != "" && (dataset.AuditSizeKey == "" || dataset.AuditSnapshotKey == "") {
			return fmt.Errorf("recovery dataset %q: audit_size_key and audit_snapshot_key are required", dataset.ID)
		}
	}
	if c.Recovery.Retention.KeepLast < 0 || c.Recovery.Retention.KeepDaily < 0 {
		return errors.New("recovery retention values cannot be negative")
	}
	checkIDs := make(map[string]struct{}, len(c.Recovery.Checks))
	for i, check := range c.Recovery.Checks {
		if check.ID == "" || check.Name == "" {
			return fmt.Errorf("recovery.checks[%d]: id and name are required", i)
		}
		if _, exists := checkIDs[check.ID]; exists {
			return fmt.Errorf("duplicate recovery check id %q", check.ID)
		}
		checkIDs[check.ID] = struct{}{}
	}
	if c.Events.MaxItems <= 0 {
		c.Events.MaxItems = 50
	}
	if c.Events.MaxItems > 500 {
		return errors.New("events.max_items must not exceed 500")
	}
	if c.Storage.WarnPercent == 0 {
		c.Storage.WarnPercent = 85
	}
	if c.Storage.ErrorPercent == 0 {
		c.Storage.ErrorPercent = 95
	}
	if c.Storage.WarnPercent < 0 || c.Storage.ErrorPercent > 100 || c.Storage.WarnPercent >= c.Storage.ErrorPercent {
		return errors.New("storage percentages must satisfy 0 <= warn_percent < error_percent <= 100")
	}
	if c.Storage.ZFSDataset != "" && len(c.Storage.PoolCommand) == 0 {
		c.Storage.PoolCommand = []string{"zpool", "status", "-x"}
	}
	if c.Storage.PoolHealthyContains == "" {
		c.Storage.PoolHealthyContains = "all pools are healthy"
	}
	for i, dependency := range c.Dependencies {
		if dependency.ID == "" || dependency.Name == "" {
			return fmt.Errorf("dependencies[%d]: id and name are required", i)
		}
		switch dependency.Kind {
		case "file":
			if dependency.Path == "" {
				return fmt.Errorf("dependency %q: path is required", dependency.ID)
			}
		case "command":
			if len(dependency.Command) == 0 {
				return fmt.Errorf("dependency %q: command is required", dependency.ID)
			}
		default:
			return fmt.Errorf("dependency %q: kind must be file or command", dependency.ID)
		}
	}
	sourceIDs := make(map[string]struct{}, len(c.Events.Sources))
	knownEvents := make(map[string]struct{})
	for i, source := range c.Events.Sources {
		if source.ID == "" || source.Name == "" || source.Glob == "" {
			return fmt.Errorf("events.sources[%d]: id, name, and glob are required", i)
		}
		if _, exists := sourceIDs[source.ID]; exists {
			return fmt.Errorf("duplicate event source id %q", source.ID)
		}
		sourceIDs[source.ID] = struct{}{}
		for token, label := range source.Events {
			if token == "" || label.Title == "" {
				return fmt.Errorf("event source %q: tokens and titles cannot be empty", source.ID)
			}
			if !slices.Contains([]string{"info", "success", "warning", "error"}, label.Severity) {
				return fmt.Errorf("event source %q token %q: invalid severity", source.ID, token)
			}
			knownEvents[token] = struct{}{}
		}
	}
	if c.Recovery.AuditSourceID != "" {
		if _, exists := sourceIDs[c.Recovery.AuditSourceID]; !exists {
			return fmt.Errorf("recovery.audit_source_id %q does not reference an event source", c.Recovery.AuditSourceID)
		}
	}
	if c.Recovery.SemanticSourceID != "" {
		if _, exists := sourceIDs[c.Recovery.SemanticSourceID]; !exists {
			return fmt.Errorf("recovery.local_semantic_source_id %q does not reference an event source", c.Recovery.SemanticSourceID)
		}
	}
	for _, token := range []string{c.Sync.SuccessEvent, c.Recovery.SuccessEvent, c.Recovery.SemanticEvent} {
		if token == "" {
			continue
		}
		if _, exists := knownEvents[token]; !exists {
			return fmt.Errorf("configured event %q is missing from events.sources", token)
		}
	}
	for i, rule := range c.PhaseRules {
		if _, exists := ids[rule.TaskID]; !exists {
			return fmt.Errorf("phase_rules[%d].task_id %q does not reference a task", i, rule.TaskID)
		}
		if len(rule.ContainsAll) == 0 || rule.Label == "" {
			return fmt.Errorf("phase_rules[%d]: contains_all and label are required", i)
		}
	}
	return nil
}

func (c Config) Resolve(path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(c.BaseDir, filepath.Clean(path))
}

func (c Config) OutputPath(override string) string {
	if override != "" {
		if filepath.IsAbs(override) {
			return override
		}
		return filepath.Clean(override)
	}
	return c.Resolve(c.OutputFile)
}
