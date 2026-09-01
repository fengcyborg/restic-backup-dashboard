package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadExample(t *testing.T) {
	t.Parallel()
	cfg, err := Load(filepath.Join("..", "..", "configs", "config.example.json"))
	if err != nil {
		t.Fatalf("Load example: %v", err)
	}
	if len(cfg.Tasks) != 3 {
		t.Fatalf("got %d tasks, want 3", len(cfg.Tasks))
	}
	if cfg.Sync.TaskID != "offsite" {
		t.Fatalf("sync task = %q, want offsite", cfg.Sync.TaskID)
	}
	if got := cfg.Resolve("markers/local.success"); got != "/var/lib/restic-automation/markers/local.success" {
		t.Fatalf("resolved path = %q", got)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	path := filepath.Join(directory, "config.json")
	content := `{
  "base_dir": "/tmp/backup",
  "unknown_field": true,
  "tasks": [{
    "id": "backup", "kind": "backup", "name": "Backup", "schedule": "hourly",
    "healthy_for": "1h", "error_after": "2h"
  }]
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("Load error = %v, want unknown field", err)
	}
}

func TestOutputPathOverride(t *testing.T) {
	t.Parallel()
	cfg := Config{BaseDir: "/var/lib/automation", OutputFile: "dashboard/status.json"}
	if got := cfg.OutputPath(""); got != "/var/lib/automation/dashboard/status.json" {
		t.Fatalf("default output = %q", got)
	}
	if got := cfg.OutputPath("custom/status.json"); got != filepath.Clean("custom/status.json") {
		t.Fatalf("override output = %q", got)
	}
}
