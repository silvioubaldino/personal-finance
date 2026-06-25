# Estilo de código (deste repo)

- **Linguagem / versão:** Go (ver `go.mod` para a versão mínima).
- **Linter / formatter:** `make gofumpt` (gofumpt), `make imports` (goimports), `make linter`
  (golangci-lint, config em `.code_quality/.golangci.yml`: gofmt, goimports, govet). `make all`
  roda format + lint + test de uma vez.
- **Nomenclatura:** use os termos do glossário (`docs/shared/_meta/glossary.md`) — sempre em
  **inglês** (structs, campos, rotas, entidades canônicas: `Movement`, `Wallet`, `Invoice`...).
  Documentação e comentários podem ser em português.
- **Estrutura de pastas:** duas arquiteturas convivem durante a migração gradual (ver
  `CLAUDE.md` → Architecture):
  - **Legacy** (movement, wallet, category, balance, estimate):
    `internal/domain/{feature}/{service,repository,api}`.
  - **Clean Architecture** (features novas: creditcard, invoice, transfer, subscription,
    agent...): `internal/usecase/`, `internal/infrastructure/{api,repository}/`,
    `internal/bootstrap/{feature}/setup.go`.
  - Features novas devem seguir o caminho clean architecture (skills `go-usecases`,
    `go-api-handlers`, `go-bootstrap` detalham as convenções de cada camada).
- **Tratamento de erros / logging:** três camadas de erro — repository → domain → HTTP (ver
  `CLAUDE.md` → Error Handling); sempre subir o erro com `domain.WrapXxx(err, contexto)`.
  Logging estruturado via `pkg/log` (`log.InfoContext`/`log.ErrorContext`, `log.Err(err)`).
