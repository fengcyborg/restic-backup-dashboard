# Architecture and threat model

## Components

The executable has two independent runtime modes:

1. `collect` runs briefly on the backup host. It reads configured markers, bounded log files, systemd properties, process matches, optional ZFS state, and explicit dependency probes. It then atomically replaces one sanitized JSON document.
2. `serve` is a long-running unprivileged HTTP process. It reads that document on each API request and serves the frontend embedded at build time.

The separation is intentional. A compromise of the web process should not yield a Restic password, an offsite provider token, repository write access, host process arguments, or a Docker/systemd control socket.

## Data minimization

The public status contract permits display labels and operational values only:

- timestamps, durations, byte totals, percentages, short snapshot IDs;
- configured task/dependency names;
- allow-listed event titles and three numeric detail keys;
- normalized service states and exit status;
- aggregate health decisions.

Raw command output and arbitrary log messages are never copied into status JSON. Non-timestamp generation IDs become one-way 12-hex-character fingerprints, snapshot IDs are limited to 12 safe characters, and event detail only accepts `duration_seconds`, `snapshots`, and `rc` integers. Marker and log readers reject symbolic links.

## Trust assumptions

- Collector configuration is trusted code-equivalent input because command dependencies and process matching are explicit argv/substring rules. Keep it root-owned and not writable by backup jobs or the web user.
- Backup jobs may write markers, JSON summaries, and event files. The collector applies file-size limits, strict timestamp parsing, numeric parsing, and safe-field projection.
- The sanitized status directory must not contain secrets. Mount only that directory into the web container.
- Authentication and TLS are intentionally left to a mature reverse proxy. The built-in server defaults to loopback and is not an identity provider.

## Failure behavior

- Missing/failed task timers, failed service results, unhealthy dependencies, and failed storage checks become visible health states.
- Missing successful restore audits never produce a recovery-ready result.
- `/readyz` returns `503` when the document is missing, invalid, too large, too old, or implausibly far in the future.
- `/healthz` only reports process liveness so an orchestrator can distinguish a live process from stale collection data.
- Collection writes use a same-directory temporary file, `fsync`, mode `0644`, and atomic rename; the web server never sees a partially written document.

## Network surface

The HTTP server accepts `GET` and `HEAD` only. It sets CSP, anti-framing, no-sniff, referrer, permissions, and cross-origin-opener headers. API responses are `no-store`; static assets may be cached for one hour.

The supplied container drops all capabilities, runs as UID/GID 65532, uses a read-only root filesystem, and mounts status data read-only.
