# Restic Backup Dashboard

[![CI](https://github.com/fengcyborg/restic-backup-dashboard/actions/workflows/ci.yml/badge.svg)](https://github.com/fengcyborg/restic-backup-dashboard/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A secure, read-only dashboard for Restic backup pipelines. It turns success markers, systemd state, sanitized event logs, ZFS capacity, offsite sync progress, and restore-verification results into one responsive web view.

[中文文档](README.zh-CN.md)

## Why this project exists

`restic snapshots` proves that snapshots are listed. It does not tell an operator, at a glance, whether the newest local generation reached offsite storage, whether the scheduler is still active, or whether a recent restore test met policy. This dashboard fills that observability gap without giving the web process access to backup credentials.

The backend is written in Go because a small NAS service benefits from a single static binary, predictable memory use, fast startup, and an embedded frontend. Go 1.26 or newer is supported.

## Demo

```bash
go run ./cmd/restic-backup-dashboard serve --demo --listen 127.0.0.1:8080
```

Open <http://127.0.0.1:8080/?demo=1>. The interface follows the browser language and includes a Chinese/English switch.

## Security model

The collector and web server are deliberately separate trust zones:

```text
Restic / rclone / systemd / ZFS
              │ read-only observations
              ▼
       host-side collector
              │ atomic, sanitized status.json
              ▼
 unprivileged web process or container
              │ read-only API
              ▼
            browser
```

- The collector runs on the backup host and may read operational state.
- The generated JSON contains no repository URL, source path, command line, password, or provider credential.
- The web process only reads that JSON. It does not need a Docker socket, systemd socket, Restic password, or repository mount.
- The UI has no buttons for backup, restore, prune, forget, or delete.
- The server defaults to `127.0.0.1` and sends a restrictive CSP plus anti-framing and no-sniff headers.

See [Architecture and threat model](docs/ARCHITECTURE.md) for the full boundary.

## What it observes

- Incremental backup, offsite synchronization, and restore-audit freshness
- systemd service/timer state and safe exit status reporting
- Exact local/offsite generation comparison
- rclone or structured JSON transfer progress
- Restic JSON summary totals and short snapshot IDs
- Restore-audit policy, retention, sampled bytes, and dataset thresholds
- ZFS dataset capacity and pool health (optional)
- Explicit file or command dependency checks
- Prometheus metrics, liveness, and freshness-aware readiness endpoints

It does **not** execute or schedule Restic backups. Your existing automation remains the source of truth and emits small markers/events for the collector to observe.

## Production setup

Build and install the binary:

```bash
go build -trimpath -o restic-backup-dashboard ./cmd/restic-backup-dashboard
sudo install -m 0755 restic-backup-dashboard /usr/local/bin/restic-backup-dashboard
sudo install -d -m 0755 /etc/restic-backup-dashboard /var/lib/restic-backup-dashboard
sudo install -m 0644 configs/config.example.json /etc/restic-backup-dashboard/config.json
```

Adapt the example to your task names and marker paths, then validate and collect once:

```bash
restic-backup-dashboard validate-config --config /etc/restic-backup-dashboard/config.json
sudo restic-backup-dashboard collect --config /etc/restic-backup-dashboard/config.json
restic-backup-dashboard serve --listen 127.0.0.1:8080 \
  --status-file /var/lib/restic-backup-dashboard/status.json
```

The supplied [systemd units](deploy/systemd/) run the collector every minute and can run the web server as a hardened dynamic user. Review paths and hardening options before installing them on a host.

For the complete marker and event format, see [Configuration](docs/CONFIGURATION.md).

## Container deployment

Only the web half belongs in a container. Run `collect` on the host, make the resulting directory readable, and mount that directory read-only:

```bash
mkdir -p runtime
cp /var/lib/restic-backup-dashboard/status.json runtime/status.json
docker compose up -d
```

The example binds to loopback. Put your authenticated reverse proxy in front if remote access is required.

## HTTP endpoints

| Endpoint | Purpose |
|---|---|
| `/` | Embedded responsive dashboard |
| `/api/v1/status` | Sanitized schema-versioned JSON |
| `/healthz` | Process liveness |
| `/readyz` | Status exists, validates, and is fresh |
| `/metrics` | Prometheus text metrics |

## CLI

```text
restic-backup-dashboard serve
restic-backup-dashboard collect
restic-backup-dashboard validate-config
restic-backup-dashboard healthcheck
restic-backup-dashboard version
```

Run a command with `-h` for its flags.

## Project status

The project is an early public release. The status schema is versioned, configuration rejects unknown fields, and CI tests supported Go versions. Review configuration and restore-audit semantics against your own recovery policy before treating the green state as an operational guarantee.

Contributions are welcome; see [CONTRIBUTING.md](CONTRIBUTING.md). Security reports should follow [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE)
