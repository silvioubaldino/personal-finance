---
id: TDR-001
type: tdr
title: Suíte de testes de aceite em Gherkin (godog) com a app in-process e Postgres efêmero
status: proposed            # proposed → accepted → superseded
created: 2026-10-02
updated: 2026-10-02
owner: Silvio Ubaldino
parents: []
related: [SPEC-008, GLO]
tags: [testing, acceptance, godog, testcontainers, bootstrap, clock]
superseded_by: null
---

# TDR-001: Suíte de testes de aceite em Gherkin com a app in-process e Postgres efêmero

> Append-only: nunca reescreva. Decisão nova = novo TDR que substitui este.
> Decisão interna da api: não muda contrato, não afeta web/mobile — por isso TDR + SPEC,
> sem AYD.

## Contexto

As regras mais arriscadas do backend são as **sequências**: update e delete de `Movement`
nas duas modalidades (**one** e **all-next**) cruzadas com `RecurrentMovement` (ocorrências
físicas e virtuais, quebra de cadeia), `CreditCard`/`Invoice` (fatura aberta, paga,
parcialmente paga, remanescente), parcelamento (`installment_group_id`) e
`InternalTransfer` (par `pair_id`). O resultado de cada operação depende do estado deixado
pelas anteriores — saldo de `Wallet`, valor de `Invoice`, limite do `CreditCard`, cadeia de
recorrência.

Hoje isso é coberto só por:

- **unit tests de usecase com mocks** (`testify/mock`) — validam o ramo da função, não o
  estado acumulado no banco depois de uma sequência de operações;
- **testes de repositório em SQLite em memória** — não rodam as migrations reais.

Forças técnicas locais que moldam a decisão:

1. **SQL específico de Postgres** nas migrations (`owner to silvioubaldino`,
   `generated always as identity`, `AT TIME ZONE 'UTC'::text`, `information_schema`,
   `jsonb`) e nos repositórios (`category_id::text` em `movement_repository.go`, `ILIKE`,
   `ON CONFLICT`). Um banco em memória testaria outro sistema.
2. **Bootstrap monolítico** em `cmd/api/main.go`: `godotenv`, OTLP, `InitializeDatabase`,
   `NewFirebaseAuth` (`log.Fatal` sem credencial), rotas legacy montadas inline e
   `r.Run()` bloqueante — não dá para instanciar a app dentro de um teste.
3. **Autenticação Firebase** — sem token real não se passa do middleware.
4. **Regras que leem `time.Now()`** (data padrão de pagamento de `Invoice`, contagem mensal
   de `Limits`, `initial_date` padrão de `Wallet`, recálculo de saldo, `Period.Validate`)
   — cenário dependeria do dia em que roda.
5. **Sem CI** (`.github/workflows` não existe) — a suíte tem de nascer pronta para rodar no
   GitHub Actions em todo push.

## Decisão

**D1 — Gherkin em inglês, executado pelo godog dentro do `go test`.** Arquivos `.feature`
em `test/acceptance/features/`, step definitions em Go, `godog.TestSuite` dentro de um
`TestFeatures(t)`. Build tag `acceptance`: `make test` (unit) continua sem Docker;
`make test-acceptance` roda a suíte.

**D2 — Caixa-preta via HTTP sobre as rotas `/v2`.** A suíte sobe a app **in-process** pelo
**mesmo composition root de produção** (`internal/app.New`) e a serve com
`httptest.Server`. Os passos falam só HTTP e decodificam as respostas em structs próprias
do harness (não importam `internal/domain/output`): se o contrato JSON quebrar, a suíte
quebra — este repo é o dono do contrato. Leitura direta do banco só para **invariantes**
que a API não expõe.

**D3 — Postgres real e efêmero via testcontainers-go.** Um container por processo de teste
(`TestMain`), mesma imagem do `docker-compose.yaml`, `POSTGRES_USER=silvioubaldino`
(exigido pelas migrations), migrations reais aplicadas por `database.RunMigrations`.
Override `ACCEPTANCE_DATABASE_URL` reaproveita um Postgres já de pé (iteração local).

**D4 — Isolamento por usuário, não por banco.** Todo dado é escopado por `user_id`; cada
cenário roda como um `user_id` novo (UUID). Sem truncate entre cenários, cenários em
paralelo, um banco para a suíte toda. Bônus: uma query que esquecer o escopo de usuário
vaza entre cenários e aparece na suíte.

**D5 — Autenticação fake vive no harness.** Implementa `authentication.Authenticator`, lê o
`user_token` (`<user_id>` ou `<user_id>;plan=free`) e monta o `AuthContext` + a chave
`authentication.UserID` no contexto. Fica em `test/acceptance/harness`, **fora do binário
de produção**.

**D6 — Relógio no contexto.** Pacote `pkg/clock` com `clock.Now(ctx)` (override no contexto,
senão `time.Now()`) e `clock.WithNow(ctx, t)`. O harness embrulha o `http.Handler` da app e
converte o header `X-Test-Now` em `clock.WithNow` — cada cenário tem o seu "hoje" e roda em
paralelo com os outros. Migram para `clock.Now(ctx)` **só as leituras de regra de negócio**;
timestamps de auditoria (`date_create`, `date_update`) seguem em `time.Now()`.

**D7 — Bootstrap extraído, sem ramo "se teste" no código de produção.** `cmd/api/main.go`
vira só leitura de ambiente + `app.New(cfg)` + `Run`. Produção e suíte montam a app pelo
mesmo `app.New`, mudando só as dependências injetadas (DB, `Authenticator`). Nenhum
usecase/repositório sabe que está em teste. A única "consciência de teste" é
`ENVIRONMENT=test` (`environment.Test`), usado para defaults de infraestrutura (formato de
log, modo do gin) — e o harness se recusa a subir se `ENVIRONMENT=production`.

**D8 — Cenários organizados por capacidade de negócio**, um arquivo por comportamento, tags
para recortes transversais. Reuso pela **biblioteca de passos** (vocabulário estável) e por
**aliases** (entidades referenciadas por nome no Gherkin, resolvidas para IDs num `World`
por cenário) — nunca por cenário que depende de outro. **Invariantes globais** (fatura =
soma dos itens, limite do cartão, saldo de carteira, par de transferência) checadas no
`After` de **todo** cenário.

**D9 — Cenário descreve o comportamento desejado.** Onde o atual diverge, o cenário leva
`@known-bug` e sai do gate de CI (roda num alvo próprio, que acusa quando passar a passar).
A suíte não grava bug como regra.

## Alternativas & trade-offs

- **SQLite em memória** — rejeitado: as migrations não rodam e queries de Postgres dos
  fluxos de movimento quebrariam; testaria outro sistema.
- **embedded-postgres** (sem Docker) — viável, mas baixa binário por plataforma e se afasta
  da imagem oficial; testcontainers roda nativo no GitHub Actions (Docker já disponível).
- **Postgres compartilhado do `docker-compose`** — mantido só como override local
  (`ACCEPTANCE_DATABASE_URL`); no CI, container efêmero.
- **Binário externo (exec/docker) e testes contra a porta** — mais fiel ao deploy, mas
  obriga flags de teste dentro do binário para trocar auth/relógio, é mais lento e pior de
  depurar. In-process com o mesmo `app.New` dá a mesma montagem de rotas/middlewares.
- **Chamar os usecases direto (sem HTTP)** — perde binding, `HandleErr` e DTO de saída,
  que são justamente o contrato.
- **Relógio global congelado** — mais simples, mas força cenários de data em série.
- **Truncate/rollback entre cenários** — mais lento e impede paralelismo; o escopo por
  `user_id` já isola.
- **Organizar por entidade ou por usecase** — por entidade quebra nos cruzamentos (um
  update all-next de parcela em fatura paga não é de uma entidade só); por usecase gera
  arquivos minúsculos acoplados ao código.
- **Go puro table-driven, sem Gherkin** — menos legível como especificação executável das
  regras; o harness fica reaproveitável por testes Go assim mesmo.

**Custos aceitos:** `godog` e `testcontainers-go` entram no `go.mod` (só em pacote de
teste, fora do binário); Docker obrigatório para `make test-acceptance`; refactor de
`cmd/api/main.go`; meia dúzia de call sites trocam `time.Now()` por `clock.Now(ctx)`.
