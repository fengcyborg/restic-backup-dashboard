BINARY := restic-backup-dashboard

.PHONY: build test check demo

build:
	go build -trimpath -o bin/$(BINARY) ./cmd/restic-backup-dashboard

test:
	go test -race ./...

check:
	gofmt -w cmd internal web
	go vet ./...
	go test ./...

demo:
	go run ./cmd/restic-backup-dashboard serve --demo --listen 127.0.0.1:8080
