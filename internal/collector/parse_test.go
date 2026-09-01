package collector

import "testing"

func TestParseRcloneProgress(t *testing.T) {
	t.Parallel()
	progress, ok := parseProgressLine("Transferred: 1.500 GiB / 3.000 GiB, 50%, 12.0 MiB/s, ETA 2m3s", "Uploading")
	if !ok {
		t.Fatal("progress line was not parsed")
	}
	if progress.percent != 50 || progress.total == nil || *progress.total != 3*(1<<30) {
		t.Fatalf("unexpected progress: %#v", progress)
	}
	if progress.eta == nil || *progress.eta != 123 {
		t.Fatalf("ETA = %v, want 123", progress.eta)
	}
}

func TestParseStructuredProgress(t *testing.T) {
	t.Parallel()
	line := `{"event":"dashboard_progress","transferred_bytes":100,"total_bytes":200,"percent":50,"speed_bytes_per_second":10,"eta_seconds":10,"phase":"Reconciling"}`
	progress, ok := parseProgressLine(line, "fallback")
	if !ok || progress.phase != "fallback" || progress.percent != 50 {
		t.Fatalf("unexpected progress: %#v, parsed=%t", progress, ok)
	}
}

func TestOpaqueMarkerIsFingerprinted(t *testing.T) {
	t.Parallel()
	marker := sanitizeMarker("this-value-must-not-reach-the-browser")
	if marker == "this-value-must-not-reach-the-browser" || len(marker) != 15 || marker[:3] != "id-" {
		t.Fatalf("marker fingerprint = %q", marker)
	}
	if marker != sanitizeMarker("this-value-must-not-reach-the-browser") {
		t.Fatal("marker fingerprint is not deterministic")
	}
}

func TestEventProjectionDropsArbitraryValues(t *testing.T) {
	t.Parallel()
	labels := map[string]eventLabel{"backup_success": {severity: "success", title: "Backup completed"}}
	record, ok := parseEventLine("2026-09-01T10:00:00Z backup_success duration_seconds=42 path=/private/source token=secret", "local", "Local", labels)
	if !ok {
		t.Fatal("event was not parsed")
	}
	if got := eventDetail(record.values); got != "duration 42s" {
		t.Fatalf("detail = %q", got)
	}
}
