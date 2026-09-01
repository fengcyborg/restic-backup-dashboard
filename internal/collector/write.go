package collector

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fengcyborg/restic-backup-dashboard/internal/model"
)

func WriteStatus(path string, status model.Status) error {
	if path == "" {
		return fmt.Errorf("status output path is empty")
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create status directory: %w", err)
	}
	file, err := os.CreateTemp(directory, ".status-*.json")
	if err != nil {
		return fmt.Errorf("create temporary status: %w", err)
	}
	temporaryPath := file.Name()
	defer os.Remove(temporaryPath)

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(status); err != nil {
		_ = file.Close()
		return fmt.Errorf("encode status: %w", err)
	}
	if err := file.Chmod(0o644); err != nil {
		_ = file.Close()
		return fmt.Errorf("set status permissions: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync status: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close status: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace status: %w", err)
	}
	return nil
}
