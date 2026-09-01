# Contributing

Thank you for helping improve Restic Backup Dashboard.

## Development

Requirements: Go 1.26 or newer. The project intentionally has no third-party Go or frontend runtime dependencies.

```bash
make check
make test
make demo
```

Open <http://127.0.0.1:8080/?demo=1> after starting the demo.

## Pull requests

- Keep the collector/web privilege separation intact.
- Do not add repository URLs, source paths, command output, credentials, or raw log text to the public status schema.
- Add focused tests for parsing, status decisions, and HTTP behavior.
- Run `gofmt`, `go vet`, `go test -race ./...`, and `node --check web/static/app.js` before submitting.
- Update English and Chinese user-facing strings together when practical.

Please use an issue for larger schema or architecture changes before investing in an implementation.

## Security issues

Do not open a public issue for a vulnerability or accidental secret exposure. Follow [SECURITY.md](SECURITY.md).
