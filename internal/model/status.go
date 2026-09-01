package model

// Status is the sanitized contract shared by collectors and the web server.
// It intentionally contains no repository URLs, command lines, or credentials.
type Status struct {
	SchemaVersion int          `json:"schema_version"`
	GeneratedAt   string       `json:"generated_at"`
	Host          string       `json:"host"`
	Display       Display      `json:"display"`
	Overall       Overall      `json:"overall"`
	Pipeline      []Task       `json:"pipeline"`
	Sync          Sync         `json:"sync"`
	Recovery      Recovery     `json:"recovery"`
	Storage       Storage      `json:"storage"`
	Dependencies  []Dependency `json:"dependencies"`
	Events        []Event      `json:"events"`
}

type Display struct {
	Title          string `json:"title"`
	Subtitle       string `json:"subtitle"`
	ProviderName   string `json:"provider_name"`
	RepositoryName string `json:"repository_name"`
}

type Overall struct {
	Status          string `json:"status"`
	Summary         string `json:"summary"`
	RecoveryReady   bool   `json:"recovery_ready"`
	RecoverySummary string `json:"recovery_summary"`
}

type Task struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	Schedule        string      `json:"schedule"`
	Status          string      `json:"status"`
	Active          bool        `json:"active"`
	Phase           string      `json:"phase,omitempty"`
	LastSuccess     string      `json:"last_success,omitempty"`
	AgeSeconds      *int64      `json:"age_seconds,omitempty"`
	DurationSeconds *int64      `json:"duration_seconds,omitempty"`
	NextRun         string      `json:"next_run,omitempty"`
	Service         SystemdUnit `json:"service"`
	Timer           SystemdUnit `json:"timer"`
}

type SystemdUnit struct {
	LoadState   string `json:"load_state"`
	ActiveState string `json:"active_state"`
	SubState    string `json:"sub_state"`
	Result      string `json:"result,omitempty"`
	ExitStatus  *int   `json:"exit_status,omitempty"`
	NextRun     string `json:"next_run,omitempty"`
	LastTrigger string `json:"last_trigger,omitempty"`
	Available   bool   `json:"available"`
}

type Sync struct {
	TaskID             string    `json:"task_id,omitempty"`
	CaughtUp           *bool     `json:"caught_up"`
	LagSeconds         *int64    `json:"lag_seconds,omitempty"`
	LocalGeneration    string    `json:"local_generation,omitempty"`
	UploadedGeneration string    `json:"uploaded_generation,omitempty"`
	Progress           *Progress `json:"progress,omitempty"`
}

type Progress struct {
	TransferredBytes    *int64 `json:"transferred_bytes,omitempty"`
	TotalBytes          *int64 `json:"total_bytes,omitempty"`
	Percent             int    `json:"percent"`
	SpeedBytesPerSecond *int64 `json:"speed_bytes_per_second,omitempty"`
	ETASeconds          *int64 `json:"eta_seconds,omitempty"`
	Phase               string `json:"phase,omitempty"`
}

type Recovery struct {
	Status                      string          `json:"status"`
	LastAudit                   string          `json:"last_audit,omitempty"`
	AuditAgeSeconds             *int64          `json:"audit_age_seconds,omitempty"`
	TotalRestoreBytes           *int64          `json:"total_restore_bytes,omitempty"`
	RemoteSnapshotCount         *int64          `json:"remote_snapshot_count,omitempty"`
	Retention                   Retention       `json:"retention"`
	SamplePercent               float64         `json:"sample_percent"`
	Checks                      []RecoveryCheck `json:"checks"`
	Datasets                    []Dataset       `json:"datasets"`
	LocalSemanticCheckAvailable bool            `json:"local_semantic_check_available"`
}

type Retention struct {
	KeepLast  int `json:"keep_last"`
	KeepDaily int `json:"keep_daily"`
}

type RecoveryCheck struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Passed    bool   `json:"passed"`
	CheckedAt string `json:"checked_at,omitempty"`
}

type Dataset struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	LocalSnapshot        string `json:"local_snapshot,omitempty"`
	RemoteSnapshot       string `json:"remote_snapshot,omitempty"`
	LocalBytes           *int64 `json:"local_bytes,omitempty"`
	VerifiedRestoreBytes *int64 `json:"verified_restore_bytes,omitempty"`
	FilesProcessed       *int64 `json:"files_processed,omitempty"`
	FilesNew             *int64 `json:"files_new,omitempty"`
	FilesChanged         *int64 `json:"files_changed,omitempty"`
}

type Storage struct {
	Dataset        string     `json:"dataset,omitempty"`
	UsedBytes      *int64     `json:"used_bytes,omitempty"`
	AvailableBytes *int64     `json:"available_bytes,omitempty"`
	QuotaBytes     *int64     `json:"quota_bytes,omitempty"`
	UsedPercent    *float64   `json:"used_percent,omitempty"`
	Status         string     `json:"status"`
	Pool           PoolHealth `json:"pool"`
}

type PoolHealth struct {
	Status  string `json:"status"`
	Healthy bool   `json:"healthy"`
	Summary string `json:"summary"`
}

type Dependency struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Summary string `json:"summary"`
}

type Event struct {
	Timestamp string `json:"timestamp"`
	Severity  string `json:"severity"`
	Source    string `json:"source"`
	Title     string `json:"title"`
	Detail    string `json:"detail,omitempty"`
}
