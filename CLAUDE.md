# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
make ci                # Full CI: lint + test (same as GitHub Actions)
make test              # Tests with -race, coverage must be >= 70%
make lint              # golangci-lint (errcheck, gosimple, govet, staticcheck, gosec, gofmt, goimports)
make proto             # Regenerate gRPC code from api/proto/*.proto
make generate          # Regenerate gomock mocks
make easyjson          # Regenerate easyjson marshaling code (run after adding JSON-tagged structs)
make docker-up         # Start all services via docker-compose
make docker-down       # Stop services
make migrate           # Run database migrations
make run-gateway       # Run gateway locally
make run-auth          # Run auth service locally
make run-notebook      # Run notebook service locally
make run-runner        # Run runner service locally
make run-storage       # Run storage service locally
make run-notification  # Run notification service locally
```

Run a single test:
```bash
go test -v -run TestFunctionName ./internal/path/to/package/...
```

The Python runner image must be built before first `docker compose up` (or use `make docker-up`, which builds it automatically):
```bash
make build-runner-image
# or, directly:
docker build -t kiss-python-runner -f build/py-runner/Dockerfile build/
```

Note: the Dockerfile expects the build context to be `build/` (it `COPY`s from `py-runner/` and `runner/` subdirectories). The image is used by the worker pool to pre-warm containers at startup — `docker image prune -a` will remove it. The deploy workflow and `make docker-up` rebuild it explicitly.

## Architecture

Microservices-based collaborative notebook platform (Jupyter-like). Single `go.mod` monorepo, services communicate via gRPC.

```
Client (HTTP:8080) --> Gateway --> Auth Service          (gRPC:9001, PostgreSQL + Redis)
                              --> Notebook Service       (gRPC:9002, PostgreSQL)
                              --> Runner Service         (gRPC:9003, Docker containers)
                              --> Storage Service        (gRPC:9004, PostgreSQL + local FS)
                              --> Issue Service          (gRPC:9005, PostgreSQL)
                              --> Notification Service   (gRPC:9006, SMTP)
                                       |
                     Auth -----------> Notification (email verification)
                     Runner ---------> Notebook (fetches blocks for execution)
```

**Gateway** (`cmd/gateway`) — sole HTTP entry point. Translates REST to gRPC. Connects to all 6 backend services. Owns middleware chain (CORS, CSRF, rate limiting, auth via gRPC ValidateSession). No database access.

**Auth Service** (`cmd/auth`) — authentication + profile management. Owns `users` table + Redis sessions with TTL. Email verification flow: register → verification token (24h TTL) → confirm endpoint. Unverified users cannot login. Background goroutine auto-deletes unverified users after 24 hours. Has `AuthProvider` Strategy pattern (`internal/auth/provider/`) for future OAuth — only `LocalProvider` (email+password) implemented.

**Notebook Service** (`cmd/notebook`) — CRUD for notebooks and code blocks. Owns `notebooks`, `blocks`, `block_outputs`, `file_permissions` tables. Exposes `GetBlocksByNotebookID` for Runner.

**Runner Service** (`cmd/runner`) — code execution in Docker containers. No database (sessions in-memory). Uses `NotebookAdapter`/`BlockAdapter` (`internal/runner/grpc/notebook_adapter.go`) that implement repository interfaces via gRPC calls to Notebook Service.

Architecture: fixed-size **worker pool** (`internal/runner/pool/`) of pre-warmed containers. Execution lifecycle per request: `StartSession` → acquire worker from pool → restore dill snapshot (if exists) → execute → save dill snapshot to Storage Service → release worker back to pool. Workers are released immediately after execution completes (async after snapshot save for single-block execution, sync for run-all). `StopSession` is a no-op if worker was already released. Idle reaper (`StartIdleReaper`) runs as a safety net for sessions that were acquired but never executed (e.g. client disconnect).

Key env vars: `RUNNER_POOL_SIZE` (default 5), `RUNNER_QUEUE_MAX` (default 50), `RUNNER_SNAPSHOT_MAX_BYTES` (default 512 MiB), `RUNNER_IDLE_TIMEOUT` (default 15m). When pool is exhausted, requests queue up to `RUNNER_QUEUE_MAX`; beyond that, `ErrServiceUnavailable` → HTTP 503.

Python agent endpoints (inside container): `POST /execute`, `POST /snapshot` (returns base64 dill dump of kernel globals), `POST /restore` (restores globals from base64 dill), `POST /restart` (kernel restart, called by pool on worker release).

**Storage Service** (`cmd/storage`) — centralized file storage with metadata. Owns `files` table in PostgreSQL. Stores files on local filesystem (`/app/uploads/`) organized by categories: `avatars`, `feedback`, `datasets`, `files`, `sessions` (runner kernel snapshots). Exposes gRPC streaming upload/download (`UploadFile`, `DownloadFile`), file CRUD, and admin analytics endpoints. `ListFiles` accepts optional `notebook_id` filter for snapshot lookup.

**Issue Service** (`cmd/issue`) — feedback and support tickets. Owns `issues` and `issue_messages` tables. CRUD for issues with status workflow (New → In Progress → Resolved → Closed) and admin responses.

**Notification Service** (`cmd/notification`) — stateless email sending via SMTP. Receives `SendEmail` gRPC calls with recipient, subject, and body. Uses custom SMTP dial with `InsecureSkipVerify` for internal relay. No database.

## JSON Serialization (easyjson)

The project uses [easyjson](https://github.com/mailru/easyjson) for zero-allocation JSON marshaling. Generated files are named `<package>_easyjson.go` (one per package, not one per source file).

**Covered packages** (in generation order — order matters due to dependencies):
```
internal/domain/
internal/pkg/dto/
internal/pkg/httputil/
internal/auth/provider/
internal/gateway/handler/
internal/payment/yookassa/
internal/runner/notebook_session/
```

**When to regenerate:** After adding or modifying any struct with `json:` tags in the above packages, run `make easyjson`.

**How it integrates:**
- `httputil.JSON(w, status, v)` — automatically calls `easyjson.Marshal` if `v` implements `easyjson.Marshaler`
- `httputil.DecodeJSON(r, &dst)` — automatically calls `easyjson.Unmarshal` if `dst` implements `easyjson.Unmarshaler`
- Redis repositories — call `easyjson.Marshal` / `easyjson.Unmarshal` directly (see `session.go`, `oauth_state.go`)

Because easyjson generates `MarshalJSON`/`UnmarshalJSON`, the standard `encoding/json` will also use the fast path automatically (they satisfy `json.Marshaler`/`json.Unmarshaler`). However, prefer calling `easyjson.Marshal` / `easyjson.Unmarshal` directly — it skips the standard library's reflection overhead and `json.NewDecoder` streaming buffering entirely.

**Important:** `make easyjson` creates a temporary bootstrap stub (e.g., `handler_easyjson.go`) during generation and then replaces it with the real output. Do **not** commit stub files that start with `// TEMPORARY AUTOGENERATED FILE` — they cause duplicate-method errors on the next run. If generation fails with "already declared" errors, delete all `*_easyjson.go` in that package and re-run.

**Adding a new package:** Add it to the `easyjson:` target in Makefile *before* any package that imports it.

## Key Patterns

**Layered architecture per service:**
```
internal/{service}/
├── app/app.go           # Wiring (DI, server setup)
├── grpc/server.go       # gRPC delivery layer
├── usecase/             # Business logic (interfaces + implementations)
└── repository/          # Data access (interfaces in interfaces.go, postgres/ implementations)
```

**Error flow:** Domain errors (`internal/domain/errors.go`: ErrNotFound, ErrUnauthorized, ErrConflict, ErrInvalidInput, ErrForbidden, ErrSessionExpired, ErrServiceUnavailable) → `grpcutil.DomainToGRPCError()` maps to gRPC status codes → Gateway uses `grpcutil.GRPCToDomainError()` → `httputil.MapDomainError()` maps to HTTP status codes. `ErrServiceUnavailable` → `ResourceExhausted` (gRPC) → 503 (HTTP), used when runner pool queue is full.

**Proto definitions** in `api/proto/{auth,notebook,runner,storage,issue,notification}/`. Generated Go code in `pkg/api/`. Regenerate with `make proto` (requires `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`).

**Mocks** in `internal/mocks/`, generated via `go.uber.org/mock`. `go:generate` directives are in interface files. gRPC client mocks (AuthServiceClient, NotebookServiceClient, RunnerServiceClient, StorageServiceClient, IssueServiceClient, NotificationServiceClient) used for gateway handler tests. Runner-specific mocks: `MockWorkerPool` (from `runner_service.WorkerPool` interface), `MockRepository` (from `snapshot.Repository`), `MockManager` (from `container.Manager`).

**Testing:** gomock for interfaces, `sqlmock` for PostgreSQL repos, `miniredis` for Redis, `httptest` for HTTP handlers, `bufconn` for gRPC server tests. Table-driven tests throughout.

## Configuration

All config via environment variables, loaded in `internal/pkg/config/config.go`. Key variables:
- `AUTH_GRPC_ADDR`, `NOTEBOOK_GRPC_ADDR`, `RUNNER_GRPC_ADDR`, `STORAGE_GRPC_ADDR`, `ISSUE_GRPC_ADDR`, `NOTIFICATION_GRPC_ADDR` — service discovery
- `DATABASE_URL` — PostgreSQL connection string
- `REDIS_HOST`, `REDIS_PORT` — Redis connection
- `GRPC_PORT` — per-service gRPC listen port
- `MAIL_FROM`, `MAIL_SMTP_HOST`, `MAIL_SMTP_PORT`, `APP_URL` — email/notification config
- `RUNNER_POOL_SIZE` — number of pre-warmed worker containers (default 5)
- `RUNNER_QUEUE_MAX` — max pending acquire requests before 503 (default 50)
- `RUNNER_SNAPSHOT_MAX_BYTES` — max dill snapshot size in bytes (default 512 MiB)
- `RUNNER_IDLE_TIMEOUT` — idle eviction timeout; sessions are released this long after last activity (default 1m)

## Monitoring

Prometheus (`deploy/prometheus/`) scrapes all 7 services + node-exporter. Grafana (`deploy/grafana/`) dashboards: `services.json` (HTTP/gRPC metrics, system metrics), `business.json` (business KPIs). Each gRPC service exposes metrics on port 9090 via `metrics.StartMetricsServer()`. Gateway exposes metrics on port 8080 at `/metrics`.

## CI & Commits

- Conventional Commits in English: `type(scope): description`
- Scopes: `auth`, `notebook`, `runner`, `gateway`, `storage`, `issue`, `notification`, `proto`, `grpc`, `infra`, `ci`
- Pre-commit hooks via Lefthook (`lefthook install` after clone)
- CI: lint then test, coverage >= 70%, excludes generated/wiring/infra code
- Local CI check: `act pull_request`
