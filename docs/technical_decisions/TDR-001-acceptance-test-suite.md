---
id: TDR-001
type: tdr
title: Suíte de testes de aceite em Gherkin (godog) com a app in-process e Postgres efêmero
status: proposed            # proposed → accepted → superseded
created: 2026-10-02
updated: 2026-11-11
owner: Silvio Ubaldino
parents: []
related: [SPEC-008, GLO]
tags: [testing, acceptance, godog, testcontainers, bootstrap, clock]
superseded_by: null
---

# TDR-001: Suíte de testes de aceite em Gherkin com a app in-process e Postgres efêmero

> Append-only: decisão nova = novo TDR que substitui este. Decisão interna da api, sem
> mudança de contrato — por isso TDR + SPEC, sem AYD.

## Contexto

As regras mais arriscadas são as **sequências**: update e delete de `Movement` (*one* e
*all-next*) cruzados com `RecurrentMovement`, `CreditCard`/`Invoice`, parcelamento e
`InternalTransfer`. O resultado depende do estado deixado pelas operações anteriores (saldo,
fatura, limite, cadeia de recorrência). Hoje só há unit tests com mocks e testes de repositório
em SQLite, que não rodam as migrations reais.

Forças locais:

1. **SQL de Postgres** nas migrations e repositórios (`owner to`, `::text`, `ILIKE`,
   `ON CONFLICT`, `jsonb`) — um banco em memória testaria outro sistema.
2. **Bootstrap monolítico** em `cmd/api/main.go` (`godotenv`, `NewFirebaseAuth` com `log.Fatal`,
   `r.Run()` bloqueante) — a app não sobe dentro de um teste.
3. **Firebase** — sem token real não se passa do middleware.
4. **`time.Now()`** em regras de negócio — o resultado dependeria do dia da execução.
5. **Sem CI** — a suíte precisa nascer pronta para rodar em todo push.

## Decisão

- **D1 — Gherkin em inglês no `go test`, via godog.** `.feature` em `test/acceptance/features/`,
  build tag `acceptance`: `make test` segue sem Docker; `make test-acceptance` roda a suíte.
- **D2 — Caixa-preta HTTP nas rotas `/v2`, app in-process** pelo mesmo `internal/app.New` de
  produção, servida por `httptest.Server`. Os passos usam structs próprias, sem importar
  `internal/domain/output`: se o contrato JSON quebrar, a suíte quebra. O banco só é lido nas
  invariantes.
- **D3 — Postgres real e efêmero via testcontainers**, mesma imagem do `docker-compose`,
  usuário `silvioubaldino` (exigido pelas migrations), migrations reais.
  `ACCEPTANCE_DATABASE_URL` reaproveita um Postgres existente.
- **D4 — Isolamento por usuário**: cada cenário roda com um `user_id` novo; sem truncate,
  cenários em paralelo. Query sem escopo de usuário vaza entre cenários e aparece na suíte.
- **D5 — Auth fake no harness** (`Authenticator` lendo `user_token`), fora do binário de produção.
- **D6 — Relógio no contexto**: `pkg/clock` (`Now(ctx)`, `WithNow`). O harness converte o header
  `X-Test-Now` em `WithNow`, então cada cenário tem o seu "hoje". Migram só leituras de regra
  de negócio; auditoria (`date_create`/`date_update`) segue em `time.Now()`.
- **D7 — Bootstrap extraído, sem ramo "se teste" em produção.** `main.go` lê ambiente e chama
  `app.New(cfg)` com DB e `Authenticator` injetados. A única consciência de teste é
  `ENVIRONMENT=test`; o harness recusa subir em `production`.
- **D8 — Cenários por capacidade de negócio**, um arquivo por comportamento, tags para cortes
  transversais. Reuso pelo vocabulário de passos e por aliases, nunca por cenário dependente de
  outro. Invariantes globais (fatura, limite, saldo, par de transferência) no `After` de todo cenário.
- **D9 — O cenário descreve o comportamento desejado.** Onde a api diverge, leva `@known-bug`
  e sai do gate de CI; a suíte não grava bug como regra.

## Alternativas

| Alternativa | Por que não |
|---|---|
| SQLite em memória | migrations e queries Postgres não rodam; testaria outro sistema |
| embedded-postgres | baixa binário por plataforma e foge da imagem oficial; Docker já existe no GitHub Actions |
| Postgres do `docker-compose` | mantido só como override local |
| Binário externo contra a porta | exige flags de teste no binário para auth/relógio; mais lento e difícil de depurar |
| Chamar usecases direto | perde binding, `HandleErr` e DTO, que são o contrato |
| Relógio global congelado | força cenários de data em série |
| Truncate/rollback entre cenários | mais lento e impede paralelismo |
| Organizar por entidade ou por usecase | quebra nos cruzamentos; gera arquivos pequenos acoplados ao código |
| Go table-driven sem Gherkin | menos legível como especificação executável |

**Custos aceitos:** `godog` e `testcontainers-go` no `go.mod` (fora do binário), com bump
transitivo de dependências (ex.: `pgx`); Docker obrigatório para `make test-acceptance`;
refactor de `cmd/api/main.go`; alguns `time.Now()` viram `clock.Now(ctx)`.
