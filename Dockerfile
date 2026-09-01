# syntax=docker/dockerfile:1

FROM golang:1.27-alpine AS build
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
COPY web ./web
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${BUILD_DATE}" \
    -o /out/restic-backup-dashboard ./cmd/restic-backup-dashboard

FROM scratch
COPY --from=build /out/restic-backup-dashboard /restic-backup-dashboard
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/restic-backup-dashboard"]
CMD ["serve", "--listen", "0.0.0.0:8080", "--status-file", "/data/status.json"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD ["/restic-backup-dashboard", "healthcheck", "--url", "http://127.0.0.1:8080/healthz"]
