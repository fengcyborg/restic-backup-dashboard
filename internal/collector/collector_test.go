package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fengcyborg/restic-backup-dashboard/internal/config"
	"github.com/fengcyborg/restic-backup-dashboard/internal/model"
)

type fakeRunner struct{}

func (fakeRunner) Run(_ context.Context, argv []string) (string, error) {
	switch argv[0] {
	case "systemctl":
		unit := argv[2]
		if strings.HasSuffix(unit, ".timer") {
			return "LoadState=loaded\nActiveState=active\nSubState=waiting\nResult=success\nExecMainStatus=0\nNextElapseUSecRealtime=2026-09-01T10:15:00Z\n", nil
		}
		return "LoadState=loaded\nActiveState=inactive\nSubState=dead\nResult=success\nExecMainStatus=0\n", nil
	case "zfs":
		return "1073741824\t3221225472\t4294967296\n", nil
	case "zpool":
		return "all pools are healthy\n", nil
	case "ps":
		return "init\n", nil
	default:
		return "", fmt.Errorf("unexpected command: %v", argv)
	}
}

func TestCollectHealthySanitizedStatus(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	now := time.Date(2026, 9, 1, 10, 5, 0, 0, time.UTC)
	writeTestFile(t, directory, "markers/local.success", now.Add(-5*time.Minute).Format(time.RFC3339))
	writeTestFile(t, directory, "markers/offsite.success", now.Add(-2*time.Minute).Format(time.RFC3339))
	writeTestFile(t, directory, "markers/offsite.generation", now.Add(-5*time.Minute).Format(time.RFC3339))
	writeTestFile(t, directory, "markers/audit.success", now.Add(-time.Hour).Format(time.RFC3339))
	writeTestFile(t, directory, "markers/local.duration", "83")
	writeTestFile(t, directory, "markers/offsite.duration", "124")
	writeTestFile(t, directory, "markers/audit.duration", "240")
	writeTestFile(t, directory, "summaries/documents.jsonl", `{"message_type":"summary","snapshot_id":"103db44fd829abcdef","total_bytes_processed":2048,"total_files_processed":10,"files_new":1,"files_changed":2}`)
	writeTestFile(t, directory, "secret/password", "synthetic-password")
	auditTime := now.Add(-time.Hour).Format(time.RFC3339)
	writeTestFile(t, directory, "logs/audit.log", auditTime+" restore_audit_success duration_seconds=240 snapshots=11 documents_bytes=2048 documents_snapshot=103db44fd829 path=/private/source token=do-not-publish\n")

	duration := func(value time.Duration) config.Duration { return config.Duration{Duration: value} }
	cfg := config.Config{
		Display:  model.Display{Title: "Test dashboard"},
		BaseDir:  directory,
		Hostname: "test-host",
		Tasks: []config.Task{
			{ID: "local", Kind: "backup", Name: "Backup", Service: "backup.service", Timer: "backup.timer", SuccessMarker: "markers/local.success", DurationMarker: "markers/local.duration", HealthyFor: duration(time.Hour), ErrorAfter: duration(2 * time.Hour)},
			{ID: "offsite", Kind: "sync", Name: "Offsite", Service: "offsite.service", Timer: "offsite.timer", SuccessMarker: "markers/offsite.success", DurationMarker: "markers/offsite.duration", HealthyFor: duration(time.Hour), ErrorAfter: duration(4 * time.Hour)},
			{ID: "audit", Kind: "audit", Name: "Audit", Service: "audit.service", Timer: "audit.timer", SuccessMarker: "markers/audit.success", DurationMarker: "markers/audit.duration", HealthyFor: duration(7 * 24 * time.Hour), ErrorAfter: duration(9 * 24 * time.Hour)},
		},
		Sync: config.Sync{TaskID: "offsite", SourceGenerationMarker: "markers/local.success", UploadedGenerationMarker: "markers/offsite.generation", ErrorAfter: duration(4 * time.Hour), SnapshotCountKey: "snapshots"},
		Recovery: config.Recovery{
			AuditTaskID: "audit", AuditSourceID: "audit", SuccessEvent: "restore_audit_success", SamplePercent: 1,
			Retention: model.Retention{KeepLast: 10, KeepDaily: 7},
			Datasets:  []config.Dataset{{ID: "documents", Name: "Documents", SummaryFile: "summaries/documents.jsonl", AuditSizeKey: "documents_bytes", AuditSnapshotKey: "documents_snapshot", MinimumBytes: 1024}},
			Checks:    []config.Check{{ID: "restore", Name: "Restore passed"}},
		},
		Storage:      config.Storage{ZFSDataset: "tank/backups", WarnPercent: 85, ErrorPercent: 95, PoolCommand: []string{"zpool", "status", "-x"}, PoolHealthyContains: "all pools are healthy"},
		Dependencies: []config.Dependency{{ID: "password", Name: "Password", Kind: "file", Path: "secret/password", HealthySummary: "Configured"}},
		Events:       config.Events{MaxItems: 10, Sources: []config.EventSource{{ID: "audit", Name: "Audit", Glob: "logs/audit.log", Events: map[string]config.EventLabel{"restore_audit_success": {Severity: "success", Title: "Restore passed"}}}}},
	}

	status := New(cfg, WithRunner(fakeRunner{}), WithClock(func() time.Time { return now })).Collect(context.Background())
	if status.Overall.Status != "healthy" || !status.Overall.RecoveryReady {
		t.Fatalf("overall = %#v", status.Overall)
	}
	if status.Sync.CaughtUp == nil || !*status.Sync.CaughtUp {
		t.Fatalf("sync = %#v", status.Sync)
	}
	if status.Recovery.TotalRestoreBytes == nil || *status.Recovery.TotalRestoreBytes != 2048 {
		t.Fatalf("recovery = %#v", status.Recovery)
	}
	content, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{directory, "/private/source", "do-not-publish", "synthetic-password"} {
		if strings.Contains(string(content), forbidden) {
			t.Fatalf("status leaked %q: %s", forbidden, content)
		}
	}
}

func TestApplySyncStatusMarksLag(t *testing.T) {
	t.Parallel()
	collector := New(config.Config{Sync: config.Sync{TaskID: "offsite", ErrorAfter: config.Duration{Duration: time.Hour}}})
	caughtUp := false
	shortLag := int64((30 * time.Minute).Seconds())
	tasks := []model.Task{{ID: "offsite", Status: "healthy"}}
	collector.applySyncStatus(tasks, model.Sync{CaughtUp: &caughtUp, LagSeconds: &shortLag})
	if tasks[0].Status != "warning" {
		t.Fatalf("short lag status = %q, want warning", tasks[0].Status)
	}
	longLag := int64((2 * time.Hour).Seconds())
	collector.applySyncStatus(tasks, model.Sync{CaughtUp: &caughtUp, LagSeconds: &longLag})
	if tasks[0].Status != "error" {
		t.Fatalf("long lag status = %q, want error", tasks[0].Status)
	}
}

func writeTestFile(t *testing.T, base, name, content string) {
	t.Helper()
	path := filepath.Join(base, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
