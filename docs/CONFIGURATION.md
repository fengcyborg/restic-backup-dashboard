# Configuration reference

[简体中文](CONFIGURATION.zh-CN.md)

The collector accepts strict JSON. Unknown fields are rejected so typos fail during `validate-config` instead of silently disabling a check. Start with [`configs/config.example.json`](../configs/config.example.json).

Validate every edited configuration before collecting status:

```bash
restic-backup-dashboard validate-config --config /etc/restic-backup-dashboard/config.json
```

## General rules

- `base_dir` is required. Relative marker, summary, log, dependency-file, and default output paths are resolved beneath it. Absolute paths remain absolute.
- Durations are JSON strings accepted by Go's `time.ParseDuration`, such as `30m`, `2h`, or `168h`. There is no `d` unit; use hours for multi-day values.
- Display names, schedules, summaries, and phase labels are trusted operator-authored text and may be localized. IDs, event tokens, JSON field names, status values, and metric names should remain stable ASCII identifiers.
- Empty optional sections disable their observation. The dashboard never starts backup, sync, restore, retention, or scheduling jobs.

## Top-level fields

| Field | Required | Purpose |
|---|---:|---|
| `display` | No | Dashboard title, subtitle, provider name, and repository display name. Built-in English defaults fill empty values. |
| `base_dir` | Yes | Base directory used to resolve relative input and output paths. |
| `output_file` | No | Sanitized status destination. Defaults to `dashboard/status.json` below `base_dir`; `collect --output` overrides it. |
| `hostname` | No | Public host label. The collector uses the operating-system hostname when empty. |
| `tasks` | Yes | One or more backup, sync, or audit task definitions. |
| `sync` | No | Local/offsite generation comparison and active transfer progress. |
| `recovery` | No | Restore-audit policy signals and Restic summary projection. |
| `storage` | No | Optional ZFS capacity and pool-health observation. |
| `dependencies` | No | Explicit file and command probes. |
| `events` | No | Allow-listed, sanitized event sources. |
| `phase_rules` | No | Trusted labels selected by process-argument substring matches. |

`display` accepts `title`, `subtitle`, `provider_name`, and `repository_name`. These values are copied to public status and must not contain sensitive infrastructure details.

## Task fields

Each entry in `tasks` accepts:

| Field | Required | Purpose |
|---|---:|---|
| `id` | Yes | Unique stable identifier. Referenced by other sections and exported in metrics. |
| `kind` | Yes | One of `backup`, `sync`, or `audit`. |
| `name` | Yes | Browser-visible task name. |
| `schedule` | No | Browser-visible description; it does not create a schedule. |
| `service` | No | systemd service unit inspected with `systemctl show`. |
| `timer` | No | systemd timer unit inspected for availability, activity, and next run. |
| `success_marker` | No | File containing the latest successful generation or timestamp. |
| `duration_marker` | No | File containing the latest successful duration. |
| `healthy_for` | Yes | Positive freshness window for a healthy result. |
| `error_after` | Yes | Positive error threshold, not shorter than `healthy_for`. |
| `running_label` | No | Trusted fallback phase shown while the service is active. |

Status evaluation is ordered: an active service is `running`; a failed service result or non-zero exit status other than `75` is `error`; a configured timer that is unavailable or inactive is `error`; a missing success marker is `unknown`; otherwise marker age produces `healthy`, `warning`, or `error`. Exit status `75` is reserved for temporary lock contention and falls back to marker freshness.

## Sync fields

Set `sync.task_id` to enable generation comparison. It must reference a task; `source_generation_marker`, `uploaded_generation_marker`, and a positive `error_after` are then required.

- `progress_log_glob` locates the newest transfer-progress file while the sync task is active.
- `success_event` names an event token used to find the latest remote snapshot count and must exist in an event source when set.
- `snapshot_count_key` names the numeric event key that carries that count.

If both generation markers exist, their sanitized values are compared exactly. When only the source marker exists, the sync is behind. A non-running sync task is `warning` while behind and becomes `error` after `sync.error_after`.

## Recovery fields

Set `recovery.audit_task_id` to enable recovery-readiness evaluation. It must reference a task, and `success_event` is required. `audit_source_id` can restrict that event to one configured source.

- `sample_percent` is display metadata between 0 and 100; the audit job remains responsible for actually sampling data.
- `retention.keep_last` and `retention.keep_daily` are non-negative display metadata. The dashboard does not run `forget` or `prune`.
- Each `datasets` entry requires a unique `id` and a `name`. `summary_file` points to Restic JSON Lines output. When recovery is enabled, `audit_size_key` and `audit_snapshot_key` are required; `minimum_bytes` is a non-negative readiness threshold.
- Each `checks` entry requires a unique `id` and a browser-visible `name`. Checks are declarative labels: the dashboard marks them passed together only when the audit event is fresh and every dataset has a valid snapshot plus enough verified bytes.
- `local_semantic_event` and `local_semantic_source_id` optionally indicate that a separate local semantic check has been observed.

The audit event is fresh through the referenced audit task's `error_after` threshold. Recovery is `healthy` only when a fresh success event exists and all configured dataset policies pass. It is `unknown` when no success event exists, otherwise `error`.

## Storage fields

- `zfs_dataset` enables `zfs list -Hp -o used,avail,quota`. Leave it empty on non-ZFS hosts.
- `warn_percent` and `error_percent` default to 85 and 95. They must satisfy `0 <= warn_percent < error_percent <= 100`.
- `pool_command` is an explicit argv array for a read-only pool-health probe. When `zfs_dataset` is set and the command is empty, it defaults to `zpool status -x`.
- `pool_healthy_contains` is matched case-insensitively against stdout and defaults to `all pools are healthy`.

If ZFS quota is unset, the collector uses `used + available` as the displayed total.

## Dependency fields

Each dependency requires a stable operational `id`, a browser-visible `name`, and a `kind`:

- A `file` dependency requires `path` and is healthy only when it resolves to a non-empty regular file.
- A `command` dependency requires an explicit `command` argv array. Exit status must be zero; non-empty `expected_exact` and `expected_contains` checks must also pass. No shell is inserted.

`healthy_summary` and `error_summary` are trusted browser-visible messages. Raw file contents and command output are never exported. The entire configuration is trusted input: store it as root-owned `0644` or stricter and never let the web user or backup jobs edit it.

## Event-source fields

`events.max_items` defaults to 50 and cannot exceed 500. Each source requires a unique `id`, a display `name`, a `glob`, and an `event_labels` map. Each event token maps to a trusted `title` and one of the severities `info`, `success`, `warning`, or `error`.

Tokens referenced by sync or recovery configuration must appear in at least one event source. Source IDs referenced by recovery configuration must also exist.

## Phase-rule fields

Each phase rule requires a task `task_id`, a non-empty `contains_all` list, and a trusted `label`. The collector reads `ps -eo args=` and selects the label when one process line contains every configured substring. Process arguments themselves are never exported. Use specific terms to avoid accidental matches.

## Marker contract

Success and generation marker files should contain one RFC 3339 timestamp:

```text
2026-09-01T09:59:42+08:00
```

For exact generation comparison, the local backup writes `sync.source_generation_marker`, and a successful offsite reconciliation copies that exact value to `sync.uploaded_generation_marker`. A marker can instead contain an opaque identifier; the collector compares one-way fingerprints and exposes only the short fingerprint. Symbolic-link marker files are rejected.

Duration marker files contain either integer seconds, a Go duration, or `duration_seconds=N`:

```text
83
```

Write markers only after the corresponding operation and its required validation have succeeded. Use an atomic rename in the backup script so the collector cannot read a partial marker.

## Restic summary files

`recovery.datasets[].summary_file` accepts Restic JSON Lines output. The collector only projects these summary fields:

- `snapshot_id` (truncated to 12 safe characters)
- `total_bytes_processed`
- `total_files_processed`
- `files_new`
- `files_changed`

Source paths and error messages are never projected.

## Sanitized event format

Each configured event log uses one event per line:

```text
2026-09-01T10:03:18+08:00 offsite_sync_success duration_seconds=124 snapshots=81 rc=0
```

The timestamp must be RFC 3339. The second field must exist in that source's `event_labels` map. Event titles and severity come from trusted configuration, not the log line. Browser-visible details are restricted to the numeric keys `duration_seconds`, `snapshots`, and `rc`.

A successful audit event can additionally carry per-dataset values used internally by the policy engine:

```text
2026-09-01T05:22:01+08:00 restore_audit_success snapshots=81 documents_bytes=202513437719 documents_snapshot=103db44fd829 photos_bytes=278158956721 photos_snapshot=135cc9f5a77a
```

Emit the success event only after all configured recovery checks have passed. The dashboard treats a fresh success event plus every dataset's `minimum_bytes` and snapshot ID as the recovery-ready signal. A dashboard cannot independently prove semantics that the audit script did not test.

## Progress input

While the configured sync task is active, the newest `progress_log_glob` file is inspected. Standard rclone transfer-stat lines are supported, as is a path-free JSON line:

```json
{"event":"dashboard_progress","transferred_bytes":1073741824,"total_bytes":2147483648,"percent":50,"speed_bytes_per_second":12582912,"eta_seconds":85}
```

The visible phase label comes from trusted task configuration or a configured `phase_rule`; arbitrary text from a transfer log is not exposed.

## Collection cadence

The browser refreshes every 15 seconds, but it only displays the most recent JSON. The supplied timer regenerates JSON about once per minute. This keeps host-side command load tiny while making state changes visible in roughly 0–75 seconds. Backup/sync scheduling remains independent.

## Input and execution limits

- Marker and duration files must be regular, non-symbolic-link files no larger than 4 KiB.
- Event, progress, and Restic summary readers accept regular files, inspect at most the last 16 MiB, and use a 1 MiB scanner line limit.
- At most 5,000 parsed event records participate in policy evaluation; at most `events.max_items` are exported.
- Every configured or built-in command has a 15-second timeout. Stdout larger than 4 MiB is rejected and stderr is discarded.
- The HTTP server accepts status documents up to 4 MiB, requires `schema_version` 1, rejects unknown fields, and treats data older than `serve --max-status-age` as not ready.
