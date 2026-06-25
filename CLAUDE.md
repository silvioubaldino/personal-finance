# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Service repo (api) of the Personal Finance product. Shares product context (vision,
requirements, cross-repo design, domain glossary) with the web and mobile repos through
`personal-finance-context`.

## This repo's role

Owner of the API contracts. This repo implements what the AYD (cross-repo design) docs in the
context repo define for the API; web and mobile consume those contracts. A contract change is
a PR in `personal-finance-context` (AYD/ADR) — never redefined locally here.

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
| `STRIPE_SECRET_KEY` | Stripe API secret key (web subscriptions) |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook signing secret for `/webhooks/stripe` |
| `STRIPE_SUCCESS_URL` | Default Checkout success redirect URL |
| `STRIPE_CANCEL_URL` | Default Checkout cancel redirect URL |
| `REVENUECAT_WEBHOOK_AUTH_KEY` | Bearer key validating `/webhooks/revenuecat` (mobile IAP) |
| `MERCADOPAGO_*` | Legacy MercadoPago vars; webhook/cancel kept for existing subscribers |

### Logging

Custom Zap wrapper in `pkg/log/`. Use `log.InfoContext(ctx, msg, fields...)` / `log.ErrorContext(ctx, msg, fields...)` for structured, context-aware logging. `log.Err(err)` is the field helper for errors.

## Engineering conventions (local)

@docs/conventions/code-style.md
@docs/conventions/testing.md
@docs/conventions/git.md

## Docs framework (summary)

This repo follows the specs-driven docs framework shared across the product's repos. The
framework's full rules live in `personal-finance-context` (`_meta/conventions.md`,
`_meta/glossary.md`).

- **Read-only context:** run `docs/scripts/sync-context.sh` to populate `docs/shared/` (a
  **gitignored** mirror of the context repo — never edit it here). Once synced:
  `docs/shared/manifest.md` (doc map) · `docs/shared/_meta/glossary.md` (ALWAYS use these
  terms) · `docs/shared/_meta/conventions.md` (IDs, frontmatter, `ID@repo` refs).
- **What lives in this repo:** `docs/specs/` (SPEC), `docs/plans/` (PLAN),
  `docs/technical_decisions/` (local TDR), `docs/conventions/` (CONV), `docs/swagger.yaml`
  (API contract, still pending migration into the framework). The changelog stays at the
  existing root `CHANGELOG.md` (Keep a Changelog format, predates this framework and plays
  the same role as the framework's `docs/changelog.md`).
- **Contracts only change in the context repo** (AYD/ADR). If this API diverges from an AYD,
  flag it — do not adapt silently (see `conventions.md` §5).
- **Feature flow:** read the relevant AYD in `docs/shared/design/` → create/update the SPEC
  here (`parents: [AYD-NNN@context]`) → write the PLAN and implement → contract changed? go
  back to the AYD in `personal-finance-context` before proceeding.
