# Security policy

## Reporting a vulnerability

Please use GitHub's private vulnerability reporting feature for this repository. Do not include real repository URLs, provider tokens, Restic passwords, hostnames, or private paths in a public issue.

Include the affected commit/version, impact, reproduction using synthetic data, and any suggested mitigation. Maintainers will acknowledge a complete report as soon as practical and coordinate disclosure after a fix is available.

## Supported versions

Until the first stable release, security fixes target the latest version on the default branch.

## Deployment boundary

The built-in HTTP server is deliberately read-only and defaults to loopback. It does not provide user authentication or TLS. Use an authenticated reverse proxy for network access, keep the collector configuration root-owned, and mount only sanitized `status.json` into a web container.
