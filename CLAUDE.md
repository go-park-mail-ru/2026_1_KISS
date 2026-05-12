# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
make ci                # Full CI: lint + test (same as GitHub Actions)
make test              # Tests with -race, coverage must be >= 70%
make lint              # golangci-lint (errcheck, gosimple, govet, staticcheck, gosec, gofmt, goimports)
make proto             # Regenerate gRPC code from api/proto/*.proto
make generate          # Regenerate gomock mocks
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

Note: the Dockerfile expects the build context to be `build/` (it `COPY`s from `py-runner/` and `runner/` subdirectories). The image is created on-demand by the runner service when a notebook session starts, so it is **not** held by any long-running container — `docker image prune -a` will remove it. The deploy workflow and `make docker-up` rebuild it explicitly.

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

**Runner Service** (`cmd/runner`) — code execution in Docker containers. No database (sessions in-memory). Uses `NotebookAdapter`/`BlockAdapter` (`internal/runner/grpc/notebook_adapter.go`) that implement repository interfaces via gRPC calls to Notebook Service. Manages container lifecycle, idle session reaping via background goroutine.

**Storage Service** (`cmd/storage`) — centralized file storage with metadata. Owns `files` table in PostgreSQL. Stores files on local filesystem (`/app/uploads/`) organized by categories: `avatars`, `feedback`, `datasets`, `files`. Exposes gRPC streaming upload, file CRUD, and admin analytics endpoints.

**Issue Service** (`cmd/issue`) — feedback and support tickets. Owns `issues` and `issue_messages` tables. CRUD for issues with status workflow (New → In Progress → Resolved → Closed) and admin responses.

**Notification Service** (`cmd/notification`) — stateless email sending via SMTP. Receives `SendEmail` gRPC calls with recipient, subject, and body. Uses custom SMTP dial with `InsecureSkipVerify` for internal relay. No database.

## Key Patterns

**Layered architecture per service:**
```
internal/{service}/
├── app/app.go           # Wiring (DI, server setup)
├── grpc/server.go       # gRPC delivery layer
├── usecase/             # Business logic (interfaces + implementations)
└── repository/          # Data access (interfaces in interfaces.go, postgres/ implementations)
```

**Error flow:** Domain errors (`internal/domain/errors.go`: ErrNotFound, ErrUnauthorized, ErrConflict, ErrInvalidInput, ErrForbidden, ErrSessionExpired) → `grpcutil.DomainToGRPCError()` maps to gRPC status codes → Gateway uses `grpcutil.GRPCToDomainError()` → `httputil.MapDomainError()` maps to HTTP status codes.

**Proto definitions** in `api/proto/{auth,notebook,runner,storage,issue,notification}/`. Generated Go code in `pkg/api/`. Regenerate with `make proto` (requires `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`).

**Mocks** in `internal/mocks/`, generated via `go.uber.org/mock`. `go:generate` directives are in interface files. gRPC client mocks (AuthServiceClient, NotebookServiceClient, RunnerServiceClient, StorageServiceClient, IssueServiceClient, NotificationServiceClient) used for gateway handler tests.

**Testing:** gomock for interfaces, `sqlmock` for PostgreSQL repos, `miniredis` for Redis, `httptest` for HTTP handlers, `bufconn` for gRPC server tests. Table-driven tests throughout.

## Configuration

All config via environment variables, loaded in `internal/pkg/config/config.go`. Key variables:
- `AUTH_GRPC_ADDR`, `NOTEBOOK_GRPC_ADDR`, `RUNNER_GRPC_ADDR`, `STORAGE_GRPC_ADDR`, `ISSUE_GRPC_ADDR`, `NOTIFICATION_GRPC_ADDR` — service discovery
- `DATABASE_URL` — PostgreSQL connection string
- `REDIS_HOST`, `REDIS_PORT` — Redis connection
- `GRPC_PORT` — per-service gRPC listen port
- `MAIL_FROM`, `MAIL_SMTP_HOST`, `MAIL_SMTP_PORT`, `APP_URL` — email/notification config

## Monitoring

Prometheus (`deploy/prometheus/`) scrapes all 7 services + node-exporter. Grafana (`deploy/grafana/`) dashboards: `services.json` (HTTP/gRPC metrics, system metrics), `business.json` (business KPIs). Each gRPC service exposes metrics on port 9090 via `metrics.StartMetricsServer()`. Gateway exposes metrics on port 8080 at `/metrics`.

## CI & Commits

- Conventional Commits in English: `type(scope): description`
- Scopes: `auth`, `notebook`, `runner`, `gateway`, `storage`, `issue`, `notification`, `proto`, `grpc`, `infra`, `ci`
- Pre-commit hooks via Lefthook (`lefthook install` after clone)
- CI: lint then test, coverage >= 70%, excludes generated/wiring/infra code
- Local CI check: `act pull_request`
