# Configuration reference

The collector accepts strict JSON. Unknown fields are rejected so typos fail during `validate-config` instead of silently disabling a check. Start with [`configs/config.example.json`](../configs/config.example.json).

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

## Tasks

Each task has a stable `id`, one of the kinds `backup`, `sync`, or `audit`, display labels, optional systemd service/timer units, marker paths, and two freshness thresholds:

- up to `healthy_for`: healthy;
- between `healthy_for` and `error_after`: warning;
- older than `error_after`: error.

An active service is `running`. A failed result or non-zero exit status is an error; exit status `75` is reserved for temporary lock contention and falls back to marker freshness.

## Dependencies and command safety

A `file` dependency only reports whether a configured regular file exists and is non-empty. A `command` dependency executes an explicit argv array, then compares trimmed stdout with `expected_exact` and/or `expected_contains`. Raw stdout is never returned to the browser.

The configuration is trusted input. Store it as root-owned `0644` or stricter and never let the web user edit it.

## Optional ZFS status

Set `storage.zfs_dataset` to collect `used`, `available`, and quota values with `zfs list`. If quota is unset, the collector uses `used + available` as the displayed total. `pool_command` and `pool_healthy_contains` provide an explicit read-only health probe. Leave both empty on a non-ZFS host.

## Collection cadence

The browser refreshes every 15 seconds, but it only displays the most recent JSON. The supplied timer regenerates JSON about once per minute. This keeps host-side command load tiny while making state changes visible in roughly 0–75 seconds. Backup/sync scheduling remains independent.
