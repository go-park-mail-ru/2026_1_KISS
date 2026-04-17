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
```

Run a single test:
```bash
go test -v -run TestFunctionName ./internal/path/to/package/...
```

The Python runner image must be built before first `docker compose up`:
```bash
docker build -t kiss-python-runner -f build/py-runner/Dockerfile build/runner/
```

## Architecture

Microservices-based collaborative notebook platform (Jupyter-like). Single `go.mod` monorepo, services communicate via gRPC.

```
Client (HTTP:8080) --> Gateway --> Auth Service     (gRPC:9001, PostgreSQL + Redis)
                              --> Notebook Service  (gRPC:9002, PostgreSQL)
                              --> Runner Service    (gRPC:9003, Docker containers)
                                       |
                                       └--> Notebook Service (gRPC, fetches blocks)
```

**Gateway** (`cmd/gateway`) — sole HTTP entry point. Translates REST to gRPC. Owns middleware chain (CORS, CSRF, rate limiting, auth via gRPC ValidateSession). No database access.

**Auth Service** (`cmd/auth`) — authentication + profile management. Owns `users` table + Redis sessions with TTL. Profile (avatar, password, email) is merged here because it shares the same table. Has `AuthProvider` Strategy pattern (`internal/auth/provider/`) for future OAuth (Yandex ID, VK ID, Google ID) — only `LocalProvider` (email+password) implemented. Migration `010_create_oauth_accounts.sql` is ready.

**Notebook Service** (`cmd/notebook`) — CRUD for notebooks and code blocks. Owns `notebooks`, `blocks`, `block_outputs`, `file_permissions` tables. Exposes `GetBlocksByNotebookID` for Runner.

**Runner Service** (`cmd/runner`) — code execution in Docker containers. No database. Uses `NotebookAdapter`/`BlockAdapter` (`internal/runner/grpc/notebook_adapter.go`) that implement repository interfaces via gRPC calls to Notebook Service — `runner_service.go` is unchanged from the monolith, adapters are injected via DI. Manages container lifecycle, idle session reaping.

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

**Proto definitions** in `api/proto/{auth,notebook,runner}/`. Generated Go code in `pkg/api/`. Regenerate with `make proto` (requires `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`).

**Mocks** in `internal/mocks/`, generated via `go.uber.org/mock`. `go:generate` directives are in interface files. gRPC client mocks (AuthServiceClient, NotebookServiceClient, RunnerServiceClient) used for gateway handler tests.

**Testing:** gomock for interfaces, `sqlmock` for PostgreSQL repos, `miniredis` for Redis, `httptest` for HTTP handlers, `bufconn` for gRPC server tests. Table-driven tests throughout.

## Configuration

All config via environment variables, loaded in `internal/pkg/config/config.go`. Key variables:
- `AUTH_GRPC_ADDR`, `NOTEBOOK_GRPC_ADDR`, `RUNNER_GRPC_ADDR` — service discovery
- `DATABASE_URL` — PostgreSQL connection string
- `REDIS_HOST`, `REDIS_PORT` — Redis connection
- `GRPC_PORT` — per-service gRPC listen port

## CI & Commits

- Conventional Commits in English: `type(scope): description`
- Scopes: `auth`, `notebook`, `runner`, `gateway`, `proto`, `grpc`, `infra`, `ci`
- Pre-commit hooks via Lefthook (`lefthook install` after clone)
- CI: lint then test, coverage >= 70%, excludes generated/wiring/infra code
- Local CI check: `act pull_request`
