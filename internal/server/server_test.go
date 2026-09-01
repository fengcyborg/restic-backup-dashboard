package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fengcyborg/restic-backup-dashboard/internal/model"
)

func TestStatusAPIAndSecurityHeaders(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	statusPath := writeStatusFixture(t, now)
	handler, err := New(Options{StatusFile: statusPath, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Security-Policy"); !strings.Contains(got, "frame-ancestors 'none'") {
		t.Fatalf("CSP = %q", got)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
}

func TestReadinessRejectsStaleStatus(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	statusPath := writeStatusFixture(t, now.Add(-10*time.Minute))
	handler, err := New(Options{StatusFile: statusPath, MaxStatusAge: 3 * time.Minute, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status code = %d, want 503", recorder.Code)
	}
}

func TestStaticAndMethodRestriction(t *testing.T) {
	t.Parallel()
	handler, err := New(Options{Demo: true})
	if err != nil {
		t.Fatal(err)
	}
	staticRecorder := httptest.NewRecorder()
	handler.ServeHTTP(staticRecorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if staticRecorder.Code != http.StatusOK || !strings.Contains(staticRecorder.Body.String(), "Restic Backup Dashboard") {
		t.Fatalf("static response = %d, %q", staticRecorder.Code, staticRecorder.Body.String())
	}
	readyRecorder := httptest.NewRecorder()
	handler.ServeHTTP(readyRecorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if readyRecorder.Code != http.StatusOK {
		t.Fatalf("demo readiness = %d, body=%s", readyRecorder.Code, readyRecorder.Body.String())
	}
	methodRecorder := httptest.NewRecorder()
	handler.ServeHTTP(methodRecorder, httptest.NewRequest(http.MethodPost, "/", nil))
	if methodRecorder.Code != http.StatusMethodNotAllowed || methodRecorder.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("method response = %d, Allow=%q", methodRecorder.Code, methodRecorder.Header().Get("Allow"))
	}
}

func TestMetricsContainOnlyNormalizedState(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	statusPath := writeStatusFixture(t, now)
	handler, err := New(Options{StatusFile: statusPath, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `dashboard_task_status{task="local",status="healthy"} 1`) {
		t.Fatalf("metrics response = %d, %s", recorder.Code, recorder.Body.String())
	}
}

func writeStatusFixture(t *testing.T, generated time.Time) string {
	t.Helper()
	status := model.Status{
		SchemaVersion: 1,
		GeneratedAt:   generated.Format(time.RFC3339),
		Overall:       model.Overall{Status: "healthy", RecoveryReady: true},
		Pipeline:      []model.Task{{ID: "local", Name: "Backup", Status: "healthy"}},
		Storage:       model.Storage{Status: "unknown", Pool: model.PoolHealth{Status: "unknown"}},
		Recovery:      model.Recovery{Status: "healthy"},
	}
	content, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "status.json")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
