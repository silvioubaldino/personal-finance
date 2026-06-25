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
    `internal/bootstrap/{feature}/setup.go`. Features novas devem seguir este caminho.
- **Convenções por camada (clean architecture):** regras completas e templates canônicos
  vivem nas skills abaixo — fonte única da verdade, não duplicadas aqui; resumo do essencial:
  - **Usecase** (`internal/usecase/{feature}_usecase.go`) — skill `go-usecases`: struct
    nomeada pela entidade, sem sufixo `Usecase` (`Movement`, não `MovementUsecase`);
    interfaces de dependência (`{Feature}Repository`/`Gateway`/`UseCase`) declaradas no
    próprio arquivo, nunca importa o struct concreto; erro do dependente sobe com
    `fmt.Errorf("...: %w", err)`, regra de negócio rejeitada sobe com `domain.WrapXxx`.
  - **Handler** (`internal/infrastructure/api/{feature}_api.go`) — skill `go-api-handlers`:
    mesma regra de interface estreita (`{Feature}Usecase`) declarada no próprio arquivo;
    handler sem lógica de negócio (só parse → chama usecase → formata resposta); todo erro
    passa por `HandleErr(c, ctx, err)`, nunca um JSON de erro feito à mão.
  - **Bootstrap/DI** (`internal/bootstrap/{feature}/setup.go`) — skill `go-bootstrap`: um
    `Setup(r, registry)` por feature, registrado em `SetupCleanArchComponents`; repositório
    exposto via `Registry.Get{X}Repository()` (lazy nil-check); gateways externos (Stripe,
    Firebase...) nunca ficam memoizados no registry.
- **Tratamento de erros / logging:** três camadas de erro — repository → domain → HTTP (ver
  `CLAUDE.md` → Error Handling); sempre subir o erro com `domain.WrapXxx(err, contexto)`.
  Logging estruturado via `pkg/log` (`log.InfoContext`/`log.ErrorContext`, `log.Err(err)`).
