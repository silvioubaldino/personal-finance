# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make all           # Format, lint, and test
make test          # Run all tests with race detection and coverage
make linter        # Run golangci-lint
make gofumpt       # Format with gofumpt
make imports       # Fix imports with goimports
```

Run a single test file:
```bash
go test ./internal/path/to/package/...
```

Run the server:
```bash
go run ./cmd/api/main.go   # Requires .env file and PostgreSQL running
docker-compose up          # Start PostgreSQL dependency
```

## Architecture

This is a **Go + Gin** REST API for a personal finance app. It uses PostgreSQL via GORM and Firebase Auth.

### Dual Architecture (Legacy + Clean)

The codebase has two parallel patterns that coexist during gradual migration:

**Legacy** (older features: movement, wallet, category, balance, estimate):
```
cmd/api/main.go  →  internal/domain/{feature}/service/  →  internal/domain/{feature}/repository/
                 →  internal/domain/{feature}/api/
```

**Clean Architecture** (newer features: creditcard, invoice, transfer, subscription, agent, etc.):
```
internal/bootstrap/{feature}/setup.go  →  internal/usecase/{feature}_usecase.go
                                        →  internal/infrastructure/api/{feature}_api.go
                                        →  internal/infrastructure/repository/{feature}_repository.go
```

New features should use the clean architecture path. `cmd/api/main.go` calls `bootstrap.SetupCleanArchComponents()` for new features and manually wires legacy features inline.

### Bootstrap & Dependency Injection

`internal/bootstrap/registry/registry.go` is a lazy-initialized singleton registry. It initializes repositories on first use (nil check pattern). Feature setup modules in `internal/bootstrap/{feature}/setup.go` pull dependencies from the registry and wire them together.

Three setup entry points in `internal/bootstrap/setup.go`:
- `SetupInternalJobs` — internal cron jobs, protected by API key auth
- `SetupPublicComponents` — unauthenticated routes (e.g. subscription webhooks)
- `SetupCleanArchComponents` — all authenticated clean-arch features

### Authentication

Firebase Auth middleware (`internal/plataform/authentication/firebase.go`) verifies ID tokens and injects `user_id` into the request context. All clean-arch repositories call `ctx.Value(authentication.UserID).(string)` to isolate data per user.

Two auth strategies:
- **Firebase token** (header: `user_token`) — for all user-facing API routes
- **API key** (header: `x-api-key`) — for internal job routes under `/jobs`

### Data Access Patterns

- `BuildBaseQuery(ctx, db, tableName)` in `internal/infrastructure/repository/query_helper.go` enforces user-scoped queries uniformly — always use this for multi-tenant data.
- Repository methods accept `*gorm.DB` as `tx` parameter. If `nil`, the method creates its own local transaction. If provided, it participates in the caller's transaction.
- DB models live in `internal/infrastructure/repository/model.go` and have `FromDomain()` / `ToDomain()` conversion methods.

### Error Handling

Three error layers:
1. **Repository errors** (`internal/infrastructure/repository/errors.go`) — e.g. `ErrMovementNotFound`, `ErrDatabaseError`
2. **Domain errors** (`internal/domain/errors.go`) — semantic errors like `ErrNotFound`, `ErrInvalidInput`, `ErrWalletInsufficient`; wrapped with `domain.WrapXxx(err, context)` helpers
3. **HTTP mapping** (`internal/infrastructure/api/errors_handler.go`) — `HandleErr()` uses `errors.Is` to map to HTTP status codes

Always wrap errors upward using the domain wrap helpers; the API layer handles final HTTP mapping.

### Environment Variables

| Variable | Purpose |
|---|---|
| `GOOGLE_PROJECT_ID` | Firebase project ID |
| `DATABASE_URL` | PostgreSQL connection string |
| `LOG_LEVEL` | `info` / `debug` / `error` (default: `info`) |
| `LOG_FORMAT` | `text` / `json` (default: `text`) |
| `ENVIRONMENT` | Set to `production` to enable Gin release mode |

### Logging

Custom Zap wrapper in `pkg/log/`. Use `log.InfoContext(ctx, msg, fields...)` / `log.ErrorContext(ctx, msg, fields...)` for structured, context-aware logging. `log.Err(err)` is the field helper for errors.
