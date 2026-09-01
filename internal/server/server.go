package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/fengcyborg/restic-backup-dashboard/internal/model"
	webassets "github.com/fengcyborg/restic-backup-dashboard/web"
)

const defaultMaxStatusBytes int64 = 4 << 20

type Options struct {
	StatusFile     string
	Demo           bool
	MaxStatusAge   time.Duration
	MaxStatusBytes int64
	Now            func() time.Time
}

type Handler struct {
	options Options
	assets  fs.FS
}

func New(options Options) (http.Handler, error) {
	assets, err := webassets.Assets()
	if err != nil {
		return nil, fmt.Errorf("open embedded assets: %w", err)
	}
	if !options.Demo && strings.TrimSpace(options.StatusFile) == "" {
		return nil, errors.New("status file is required unless demo mode is enabled")
	}
	if options.MaxStatusAge <= 0 {
		options.MaxStatusAge = 3 * time.Minute
	}
	if options.MaxStatusBytes <= 0 {
		options.MaxStatusBytes = defaultMaxStatusBytes
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	handler := &Handler{options: options, assets: assets}
	return securityHeaders(http.HandlerFunc(handler.route)), nil
}

func (h *Handler) route(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		writer.Header().Set("Allow", "GET, HEAD")
		writeJSONError(writer, request, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	switch request.URL.Path {
	case "/api/v1/status":
		h.status(writer, request)
	case "/healthz":
		writeJSON(writer, request, http.StatusOK, map[string]string{"status": "ok"})
	case "/readyz":
		h.ready(writer, request)
	case "/metrics":
		h.metrics(writer, request)
	default:
		h.static(writer, request)
	}
}

func (h *Handler) status(writer http.ResponseWriter, request *http.Request) {
	status, err := h.loadStatus()
	if err != nil {
		writeJSONError(writer, request, http.StatusServiceUnavailable, "status is unavailable")
		return
	}
	writer.Header().Set("Cache-Control", "no-store")
	writeJSON(writer, request, http.StatusOK, status)
}

func (h *Handler) ready(writer http.ResponseWriter, request *http.Request) {
	status, err := h.loadStatus()
	if err != nil || h.isStale(status) {
		writeJSONError(writer, request, http.StatusServiceUnavailable, "status is unavailable or stale")
		return
	}
	writeJSON(writer, request, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *Handler) metrics(writer http.ResponseWriter, request *http.Request) {
	status, err := h.loadStatus()
	if err != nil {
		http.Error(writer, "dashboard_status_available 0\n", http.StatusServiceUnavailable)
		return
	}
	generated, _ := time.Parse(time.RFC3339, status.GeneratedAt)
	age := max(0, h.options.Now().Sub(generated).Seconds())
	ready := 0
	if status.Overall.RecoveryReady {
		ready = 1
	}
	var output strings.Builder
	output.WriteString("# HELP dashboard_status_available Whether the sanitized status document is readable.\n")
	output.WriteString("# TYPE dashboard_status_available gauge\n")
	output.WriteString("dashboard_status_available 1\n")
	output.WriteString("# HELP dashboard_status_age_seconds Age of the latest collected status.\n")
	output.WriteString("# TYPE dashboard_status_age_seconds gauge\n")
	fmt.Fprintf(&output, "dashboard_status_age_seconds %.0f\n", age)
	output.WriteString("# HELP dashboard_recovery_ready Whether the latest recovery verification satisfies policy.\n")
	output.WriteString("# TYPE dashboard_recovery_ready gauge\n")
	fmt.Fprintf(&output, "dashboard_recovery_ready %d\n", ready)
	output.WriteString("# HELP dashboard_overall_status Current aggregate status.\n")
	output.WriteString("# TYPE dashboard_overall_status gauge\n")
	fmt.Fprintf(&output, "dashboard_overall_status{status=%q} 1\n", escapeMetricLabel(status.Overall.Status))
	output.WriteString("# HELP dashboard_task_status Current status for each configured task.\n")
	output.WriteString("# TYPE dashboard_task_status gauge\n")
	for _, task := range status.Pipeline {
		fmt.Fprintf(&output, "dashboard_task_status{task=%q,status=%q} 1\n", escapeMetricLabel(task.ID), escapeMetricLabel(task.Status))
	}
	writer.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(http.StatusOK)
	if request.Method != http.MethodHead {
		_, _ = io.WriteString(writer, output.String())
	}
}

func (h *Handler) static(writer http.ResponseWriter, request *http.Request) {
	name := strings.TrimPrefix(path.Clean(request.URL.Path), "/")
	if name == "." || name == "" {
		name = "index.html"
	}
	if strings.Contains(name, "..") || strings.HasPrefix(name, ".") {
		http.NotFound(writer, request)
		return
	}
	content, err := fs.ReadFile(h.assets, name)
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	contentType := mime.TypeByExtension(path.Ext(name))
	if contentType == "" {
		contentType = http.DetectContentType(content)
	}
	writer.Header().Set("Content-Type", contentType)
	if name == "index.html" || name == "status.example.json" {
		writer.Header().Set("Cache-Control", "no-cache")
	} else {
		writer.Header().Set("Cache-Control", "public, max-age=3600")
	}
	writer.Header().Set("Content-Length", strconv.Itoa(len(content)))
	writer.WriteHeader(http.StatusOK)
	if request.Method != http.MethodHead {
		_, _ = writer.Write(content)
	}
}

func (h *Handler) loadStatus() (model.Status, error) {
	var reader io.Reader
	if h.options.Demo {
		content, err := fs.ReadFile(h.assets, "status.example.json")
		if err != nil {
			return model.Status{}, err
		}
		reader = bytes.NewReader(content)
	} else {
		file, err := os.Open(h.options.StatusFile)
		if err != nil {
			return model.Status{}, err
		}
		defer file.Close()
		reader = file
	}
	limited := io.LimitReader(reader, h.options.MaxStatusBytes+1)
	content, err := io.ReadAll(limited)
	if err != nil {
		return model.Status{}, err
	}
	if int64(len(content)) > h.options.MaxStatusBytes {
		return model.Status{}, errors.New("status document exceeds size limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var status model.Status
	if err := decoder.Decode(&status); err != nil {
		return model.Status{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return model.Status{}, errors.New("status document contains trailing data")
	}
	if status.SchemaVersion != 1 {
		return model.Status{}, fmt.Errorf("unsupported schema version %d", status.SchemaVersion)
	}
	if _, err := time.Parse(time.RFC3339, status.GeneratedAt); err != nil {
		return model.Status{}, errors.New("invalid generated_at timestamp")
	}
	if !validStatus(status.Overall.Status) {
		return model.Status{}, errors.New("invalid overall status")
	}
	return status, nil
}

func (h *Handler) isStale(status model.Status) bool {
	if h.options.Demo {
		return false
	}
	generated, err := time.Parse(time.RFC3339, status.GeneratedAt)
	if err != nil {
		return true
	}
	age := h.options.Now().Sub(generated)
	return age > h.options.MaxStatusAge || age < -5*time.Minute
}

func validStatus(value string) bool {
	switch value {
	case "healthy", "warning", "error", "running", "unknown":
		return true
	default:
		return false
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'")
		writer.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		writer.Header().Set("Permissions-Policy", "camera=(), geolocation=(), microphone=()")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(writer, request)
	})
}

func writeJSON(writer http.ResponseWriter, request *http.Request, statusCode int, value any) {
	content, err := json.Marshal(value)
	if err != nil {
		writeJSONError(writer, request, http.StatusInternalServerError, "response encoding failed")
		return
	}
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.Header().Set("Content-Length", strconv.Itoa(len(content)))
	writer.WriteHeader(statusCode)
	if request.Method != http.MethodHead {
		_, _ = writer.Write(content)
	}
}

func writeJSONError(writer http.ResponseWriter, request *http.Request, statusCode int, message string) {
	content, _ := json.Marshal(map[string]string{"error": message})
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(statusCode)
	if request.Method != http.MethodHead {
		_, _ = writer.Write(content)
	}
}

func escapeMetricLabel(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\n", "\\n")
	return strings.ReplaceAll(value, "\"", "\\\"")
}
