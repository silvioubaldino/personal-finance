# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run all checks (format, lint, test)
make all

# Individual steps
make ensure-deps     # go mod tidy
make gofumpt         # format with gofumpt
make imports         # format imports with goimports
make linter          # run golangci-lint
make test            # run tests with race detector and coverage

# Run a single test
go test ./internal/usecase/... -run TestMyTest -v

# Run tests for a specific package
go test ./internal/usecase/... -count=1 -race
```

## Architecture

This is a personal finance backend written in Go, following **Clean/Hexagonal Architecture**:

```
cmd/api/main.go           → HTTP server entry (Gin), routes wired here
internal/bootstrap/       → Dependency injection; each subdomain has its own setup
internal/domain/          → Entities, interfaces, and domain-specific API handlers/services
internal/usecase/         → Business logic orchestration (one file per feature)
internal/infrastructure/  → Adapters: GORM repositories, Firebase/Gemini/MercadoPago gateways
internal/plataform/       → Cross-cutting: auth middleware, DB init, session
db/migrations/            → SQL migrations (golang-migrate, ~25 versions)
```

**Dependency flow:** `domain` ← `usecase` ← `bootstrap` → `infrastructure`

Domain packages (e.g. `movement`, `wallet`, `category`) each contain their own API handler structs and service interfaces. The `usecase` layer implements those interfaces and calls repository/gateway interfaces defined in `domain`.

**Bootstrap pattern:** `internal/bootstrap/setup.go` wires everything. Each feature module (movement, invoice, agent, etc.) has a `bootstrap/<module>/` package that creates and returns the assembled components.

## Key Tech

- **Web:** Gin Gonic
- **ORM:** GORM (PostgreSQL in prod, SQLite in tests)
- **Auth:** Firebase Auth (JWT middleware in `internal/plataform/authentication/`)
- **AI:** Google Gemini Vision + Vertex AI (`internal/infrastructure/gateway/`)
- **Payments:** Mercado Pago webhooks with HMAC validation
- **Observability:** Zap logging + OpenTelemetry

## Testing Notes

- Tests use `go-sqlmock` for DB and testify for assertions
- Mocks live in `internal/usecase/mocks.go`
- SQLite is used as the in-memory DB for integration tests
- The `make test` command runs with `-race` — keep tests race-free
