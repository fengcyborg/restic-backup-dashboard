# Contributing

[简体中文](CONTRIBUTING.zh-CN.md)

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
- Update English and Chinese user-facing strings together.

Please use an issue for larger schema or architecture changes before investing in an implementation.

## Documentation

User-facing Markdown is maintained as an English file and a paired `.zh-CN.md` file. Update both in the same pull request and keep their headings, examples, and links structurally aligned. Command names, JSON fields, status values, file paths, and metric names remain unchanged in translations.

The code and tests are the source of truth for runtime behavior. When behavior changes, update the relevant README, configuration, architecture, security, and example content in both languages.

## Security issues

Do not open a public issue for a vulnerability or accidental secret exposure. Follow [SECURITY.md](SECURITY.md) ([简体中文](SECURITY.zh-CN.md)).
