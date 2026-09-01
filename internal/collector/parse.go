package collector

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const maxInputFileSize = 16 << 20

var (
	safeMarkerPattern = regexp.MustCompile(`^[A-Za-z0-9._:+-]{1,128}$`)
	rcloneProgress    = regexp.MustCompile(`(?i)Transferred:\s*([0-9.]+)\s*([KMGTPE]?i?B)\s*/\s*([0-9.]+)\s*([KMGTPE]?i?B),\s*([0-9]+)%.*?([0-9.]+)\s*([KMGTPE]?i?B)/s,\s*ETA\s*([^,\s]+)`)
)

type marker struct {
	value string
	time  time.Time
	found bool
}

type eventRecord struct {
	timestamp time.Time
	sourceID  string
	source    string
	token     string
	severity  string
	title     string
	values    map[string]string
}

type resticSummary struct {
	snapshotID     string
	totalBytes     *int64
	filesProcessed *int64
	filesNew       *int64
	filesChanged   *int64
}

func readMarker(path string) marker {
	if path == "" {
		return marker{}
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 4096 {
		return marker{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return marker{}
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return marker{}
	}
	result := marker{value: sanitizeMarker(value), time: info.ModTime(), found: true}
	if parsed, ok := parseTimestamp(value); ok {
		result.time = parsed
		result.value = parsed.Format(time.RFC3339)
	}
	return result
}

func sanitizeMarker(value string) string {
	value = strings.TrimSpace(value)
	digest := sha256.Sum256([]byte(value))
	return fmt.Sprintf("id-%x", digest[:6])
}

func parseTimestamp(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"Mon 2006-01-02 15:04:05 MST",
		"2006-01-02 15:04:05 MST",
		"2006-01-02 15:04:05 -0700",
	} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func readDurationSeconds(path string) *int64 {
	if path == "" {
		return nil
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 4096 {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	value := strings.TrimSpace(string(data))
	if key, parsed, ok := strings.Cut(value, "="); ok && strings.TrimSpace(key) == "duration_seconds" {
		value = strings.TrimSpace(parsed)
	}
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && seconds >= 0 {
		return &seconds
	}
	if duration, err := time.ParseDuration(value); err == nil && duration >= 0 {
		seconds := int64(duration.Seconds())
		return &seconds
	}
	return nil
}

func parseSystemdProperties(output string) map[string]string {
	properties := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		key, value, ok := strings.Cut(line, "=")
		if ok {
			properties[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return properties
}

func parseEventLine(line, sourceID, sourceName string, labels map[string]eventLabel) (eventRecord, bool) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) < 2 {
		return eventRecord{}, false
	}
	timestamp, ok := parseTimestamp(fields[0])
	if !ok {
		return eventRecord{}, false
	}
	label, ok := labels[fields[1]]
	if !ok {
		return eventRecord{}, false
	}
	values := make(map[string]string)
	for _, field := range fields[2:] {
		key, value, found := strings.Cut(field, "=")
		if !found || key == "" || len(key) > 64 || len(value) > 256 {
			continue
		}
		values[key] = strings.Trim(value, `"`)
	}
	return eventRecord{
		timestamp: timestamp,
		sourceID:  sourceID,
		source:    sourceName,
		token:     fields[1],
		severity:  normalizeSeverity(label.severity),
		title:     label.title,
		values:    values,
	}, true
}

type eventLabel struct {
	severity string
	title    string
}

func normalizeSeverity(value string) string {
	switch value {
	case "success", "warning", "error", "info":
		return value
	default:
		return "info"
	}
}

func eventDetail(values map[string]string) string {
	parts := make([]string, 0, 3)
	if value, ok := positiveInteger(values["duration_seconds"]); ok {
		parts = append(parts, fmt.Sprintf("duration %ds", value))
	}
	if value, ok := positiveInteger(values["snapshots"]); ok {
		parts = append(parts, fmt.Sprintf("snapshots %d", value))
	}
	if value, ok := integer(values["rc"]); ok {
		parts = append(parts, fmt.Sprintf("exit %d", value))
	}
	return strings.Join(parts, " · ")
}

func integer(value string) (int64, bool) {
	parsed, err := strconv.ParseInt(value, 10, 64)
	return parsed, err == nil
}

func positiveInteger(value string) (int64, bool) {
	parsed, ok := integer(value)
	return parsed, ok && parsed >= 0
}

func newestMatchingFile(pattern string) string {
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return ""
	}
	sort.Slice(matches, func(i, j int) bool {
		left, leftErr := os.Lstat(matches[i])
		right, rightErr := os.Lstat(matches[j])
		if leftErr != nil {
			return false
		}
		if rightErr != nil {
			return true
		}
		return left.ModTime().After(right.ModTime())
	})
	return matches[0]
}

func parseProgressFile(path, phase string) *progressData {
	if path == "" {
		return nil
	}
	var latest *progressData
	scanTailLines(path, maxInputFileSize, func(line string) {
		if parsed, ok := parseProgressLine(line, phase); ok {
			latest = &parsed
		}
	})
	return latest
}

type progressData struct {
	transferred *int64
	total       *int64
	percent     int
	speed       *int64
	eta         *int64
	phase       string
}

func parseProgressLine(line, phase string) (progressData, bool) {
	var object struct {
		Event       string  `json:"event"`
		Transferred int64   `json:"transferred_bytes"`
		Total       int64   `json:"total_bytes"`
		Percent     float64 `json:"percent"`
		Speed       int64   `json:"speed_bytes_per_second"`
		ETA         int64   `json:"eta_seconds"`
	}
	if json.Unmarshal([]byte(line), &object) == nil && object.Event == "dashboard_progress" {
		percent := int(math.Round(object.Percent))
		return progressData{
			transferred: pointerIfNonNegative(object.Transferred),
			total:       pointerIfNonNegative(object.Total),
			percent:     clampPercent(percent),
			speed:       pointerIfNonNegative(object.Speed),
			eta:         pointerIfNonNegative(object.ETA),
			phase:       firstNonEmpty(phase, "Transfer is running"),
		}, true
	}

	match := rcloneProgress.FindStringSubmatch(line)
	if len(match) != 9 {
		return progressData{}, false
	}
	transferred, err1 := parseByteQuantity(match[1], match[2])
	total, err2 := parseByteQuantity(match[3], match[4])
	percent, err3 := strconv.Atoi(match[5])
	speed, err4 := parseByteQuantity(match[6], match[7])
	eta, err5 := parseFlexibleDuration(match[8])
	if errors.Join(err1, err2, err3, err4, err5) != nil {
		return progressData{}, false
	}
	return progressData{
		transferred: &transferred,
		total:       &total,
		percent:     clampPercent(percent),
		speed:       &speed,
		eta:         &eta,
		phase:       phase,
	}, true
}

func parseByteQuantity(number, unit string) (int64, error) {
	value, err := strconv.ParseFloat(number, 64)
	if err != nil || value < 0 {
		return 0, errors.New("invalid byte quantity")
	}
	normalized := strings.ReplaceAll(strings.ToUpper(unit), "I", "")
	units := map[string]float64{
		"B": 1, "KB": 1 << 10, "MB": 1 << 20, "GB": 1 << 30,
		"TB": 1 << 40, "PB": 1 << 50, "EB": 1 << 60,
	}
	multiplier, ok := units[normalized]
	if !ok || value > float64(math.MaxInt64)/multiplier {
		return 0, errors.New("unsupported byte quantity")
	}
	return int64(math.Round(value * multiplier)), nil
}

func parseFlexibleDuration(value string) (int64, error) {
	if value == "-" {
		return 0, errors.New("unknown duration")
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration < 0 {
		return 0, errors.New("invalid duration")
	}
	return int64(duration.Seconds()), nil
}

func clampPercent(value int) int {
	return min(100, max(0, value))
}

func pointerIfNonNegative(value int64) *int64 {
	if value < 0 {
		return nil
	}
	return &value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func parseResticSummary(path string) resticSummary {
	if path == "" {
		return resticSummary{}
	}
	var latest resticSummary
	scanTailLines(path, maxInputFileSize, func(line string) {
		var object map[string]any
		decoder := json.NewDecoder(strings.NewReader(line))
		decoder.UseNumber()
		if decoder.Decode(&object) != nil {
			return
		}
		if messageType, _ := object["message_type"].(string); messageType != "summary" && object["snapshot_id"] == nil {
			return
		}
		latest.snapshotID = sanitizeSnapshot(stringValue(object["snapshot_id"]))
		latest.totalBytes = numberPointer(object["total_bytes_processed"])
		latest.filesProcessed = numberPointer(object["total_files_processed"])
		latest.filesNew = numberPointer(object["files_new"])
		latest.filesChanged = numberPointer(object["files_changed"])
	})
	return latest
}

func scanTailLines(path string, maxBytes int64, visit func(string)) bool {
	if path == "" || maxBytes <= 0 {
		return false
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	partialFirstLine := false
	if info.Size() > maxBytes {
		if _, err := file.Seek(-maxBytes, io.SeekEnd); err != nil {
			return false
		}
		partialFirstLine = true
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	if partialFirstLine {
		_ = scanner.Scan()
	}
	for scanner.Scan() {
		visit(scanner.Text())
	}
	return scanner.Err() == nil
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func numberPointer(value any) *int64 {
	number, ok := value.(json.Number)
	if !ok {
		return nil
	}
	parsed, err := number.Int64()
	if err != nil || parsed < 0 {
		return nil
	}
	return &parsed
}

func sanitizeSnapshot(value string) string {
	value = strings.TrimSpace(value)
	if !safeMarkerPattern.MatchString(value) {
		return ""
	}
	if len(value) > 12 {
		return value[:12]
	}
	return value
}
