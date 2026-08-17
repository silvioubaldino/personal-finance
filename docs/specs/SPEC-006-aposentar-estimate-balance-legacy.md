---
id: SPEC-006
type: spec
title: Aposentar o legacy de Estimate e Balance
status: draft
created: 2026-08-17
updated: 2026-08-17
owner: Silvio Ubaldino
parents: [AYD-005@context]
children: []
related: [GLO]
tags: [estimate, planning, balance, cleanup]
superseded_by: null
---

# Spec: Aposentar o legacy de Estimate e Balance (parte deste repo)

> **Doc único deste repo para a feature:** o *o quê* (objetivo, critérios, contratos) e o
> *como* (plano de implementação). A parte "o quê" **congela ao virar `approved`**; a
> checklist do plano segue sendo marcada durante a execução, sem mudar o `status`.

## Objetivo

Executar a Fase 4 de AYD-005@context: remover as rotas e o código legacy de `Estimate` e
`Balance` da api, agora que `GET /v2/estimate/summary` (SPEC-002@api) é a fonte única de
orçado × realizado e os clientes migraram para consumi-lo. É trabalho de limpeza — nenhum
contrato novo, só remoção do que ficou órfão.

## Gate de pré-requisitos (bloqueante — não iniciar sem checar)

Esta SPEC **só começa** depois que, em produção:

1. `SPEC-003@web` (consumir o summary) estiver em produção;
2. `SPEC-004@mobile` (consumir o summary) estiver em produção;
3. `SPEC-005@web` — a Fase 3.5 do AYD, migração do CRUD de `Estimate` no web de `/estimate`
   legacy para `/v2/estimate/*` — estiver em produção.

Enquanto o web ainda escrever em `/estimate` legacy (leitura **e** escrita), o legacy não
pode sair (AYD-005@context, decisão 3). Antes de tocar em qualquer remoção deste plano:

- `grep -rn "'/estimate'\|\"/estimate\"\|/estimate/\|estimate\." ` nos repos `web`
  (`personal-finance-frontend-v2`) e `mobile` (`personal-finance-mobile`) por chamadas às
  rotas legacy (`/estimate`, `/sub-estimate`, `/balance/estimate/period`) — não deve sobrar
  nenhuma chamada de produção (ignorar apenas testes/mocks apontando para strings iguais).
- Confirmar no changelog de cada front que as três SPECs acima estão com status de release
  publicado, não só código mergeado em `develop`.

Se qualquer uma das três não estiver em produção, **parar aqui** e devolver ao orquestrador —
não é decisão desta SPEC forçar a remoção antes do gate.

## Critérios de aceite

```gherkin
Cenário: Gate de pré-requisitos verificado antes de remover
  Dado o grep por chamadas às rotas legacy nos repos web e mobile
  Quando SPEC-003@web, SPEC-004@mobile e SPEC-005@web estão em produção
  Então nenhuma chamada de produção às rotas legacy é encontrada
  E a remoção pode prosseguir

Cenário: Rotas legacy de Estimate removidas
  Dado o código de internal/domain/estimate/* removido e a fiação em cmd/api/main.go retirada
  Quando o servidor sobe
  Então GET /estimate, POST /estimate, PUT /estimate/:id, POST /sub-estimate e PUT /sub-estimate/:id
    deixam de existir (404 de rota não registrada, não erro 500)

Cenário: Rota legacy de Balance removida
  Dado o código de internal/domain/balance/* removido e a fiação em cmd/api/main.go retirada
  Quando o servidor sobe
  Então GET /balance/estimate/period deixa de existir

Cenário: Rota v2 de balance por estimate removida
  Dado internal/bootstrap/balance/setup.go removido (ou sua rota de estimate/period retirada)
  Quando o servidor sobe
  Então GET /v2/balance/estimate/period deixa de existir

Cenário: Rotas v2 em uso não regridem
  Dado a remoção do legacy aplicada
  Quando o servidor sobe
  Então GET /v2/estimate/, POST /v2/estimate/, PUT /v2/estimate/:id, DELETE /v2/estimate/:id,
    POST /v2/sub-estimate/, PUT /v2/sub-estimate/:id, DELETE /v2/sub-estimate/:id e
    GET /v2/estimate/summary continuam respondendo sem alteração de contrato

Cenário: Suíte e lint passam após a remoção
  Dado o código legacy removido
  Quando `make all` é executado
  Então format, lint e test passam sem erro, sem referências soltas ao pacote removido
```

## Contratos consumidos/expostos

Nenhum contrato novo. Esta SPEC **remove exposição** de contratos já superados pelo v2
(`GET /v2/estimate/summary`, SPEC-002@api): `GET /estimate`, `POST /estimate`,
`PUT /estimate/:id`, `POST /sub-estimate`, `PUT /sub-estimate/:id`,
`GET /balance/estimate/period` e `GET /v2/balance/estimate/period`. Nada disso é redefinido —
apenas deixa de ser servido, conforme a Fase 4 de AYD-005@context.

## Modelo de dados / componentes afetados

- `internal/domain/estimate/repository/estimate_categories.go` (+ teste)
- `internal/domain/estimate/api/handler.go`
- `internal/domain/estimate/errors.go`
- `internal/domain/estimate/service/estimate_category.go`
- `internal/domain/balance/api/handler.go`
- `internal/domain/balance/service/balance.go`
- `internal/bootstrap/balance/setup.go` (rota v2 de `/v2/balance/estimate/period` — confirmar
  no código, no momento da execução, se o pacote inteiro sai ou só essa rota, caso
  `balance_usecase.go`/`Balance` interface passe a não ter mais consumidor nenhum depois do
  SPEC-002@api já ter corrigido e reaproveitado a lógica de `consolidated` — ver nota abaixo)
- `cmd/api/main.go` — fiação manual do legacy (linhas hoje em torno de 12-19 e 158-163):
  imports `balanceApi`, `balanceService`, `estimateApi`, `estimateRepository`,
  `estimateService`, e as chamadas `estimateApi.NewBalanceHandlers`,
  `balanceApi.NewBalanceHandlers`
- `docs/swagger.yaml` — remover os paths correspondentes

Nenhuma entidade de domínio nova ou removida (`EstimateCategories`/`EstimateSubCategories`
continuam existindo, usados pelo v2 e pelo summary).

## Casos de borda & fora de escopo

- Borda: se `internal/usecase/balance_usecase.go` (`Balance`/`CalculateBalance`, hoje
  consumido só por `GET /v2/balance/estimate/period`) ficar sem nenhum consumidor após esta
  remoção, avaliar no PR se ele também sai junto — checar primeiro se o usecase de summary
  (SPEC-002@api) reaproveitou a lógica de `getBalanceSum` diretamente ou apenas se inspirou
  nela; se reaproveitou por import, este arquivo não pode ser removido.
- Fora: os dois hooks órfãos `useEstimateBalance` (`hooks/use-estimates.ts:157`@web,
  `src/hooks/use-estimates.ts:29`@mobile) — mencionados no AYD como parte da Fase 4, mas sua
  remoção é código de front e pertence às SPECs de web/mobile, não a esta. Migração de dados
  ou de schema — não há, nenhuma tabela muda.

---

## Plano de implementação

> Parte de trabalho deste doc. Decisão técnica não trivial vira TDR, não fica só aqui.

### Abordagem técnica

Remoção pura: apagar os pacotes legacy inteiros, retirar a fiação manual em `cmd/api/main.go`
(este repo não usa `bootstrap/{feature}/setup.go` para o legacy — é fiação inline, conforme
`CLAUDE.md` → Dual Architecture), atualizar `docs/swagger.yaml`, rodar `make all`. Antes de
qualquer edição, executar o gate de pré-requisitos descrito acima; sem ele verificado, não
prosseguir.

### Passos

1. Verificar o gate de pré-requisitos (grep nos repos `web`/`mobile` por chamadas às rotas
   legacy + status de release das três SPECs listadas). Registrar o resultado no PR.
2. Remover `internal/domain/estimate/` (repository, api, service, errors) e seus testes.
3. Remover `internal/domain/balance/` (api, service) e seus testes.
4. Em `cmd/api/main.go`: remover os imports `balanceApi`, `balanceService`, `estimateApi`,
   `estimateRepository`, `estimateService` e as linhas de wiring (`estimateRepo := ...`,
   `estimateService := ...`, `estimateApi.NewBalanceHandlers(...)`,
   `balanceService := ...`, `balanceApi.NewBalanceHandlers(...)`).
5. Avaliar `internal/bootstrap/balance/setup.go` e `internal/usecase/balance_usecase.go`: se
   `GET /v2/balance/estimate/period` era o único consumidor e o usecase de summary não os
   reaproveita por import direto, remover `balance.Setup` de
   `internal/bootstrap/setup.go:71` e o pacote inteiro; caso contrário, remover só a rota,
   mantendo o usecase se ainda for dependência de código vivo.
6. Atualizar `docs/swagger.yaml`, removendo os paths legados.
7. Rodar `make all`; corrigir qualquer referência solta (imports órfãos, testes quebrados).
8. Testar manualmente (ou via teste de integração leve) que as rotas v2 em uso continuam
   respondendo: `GET /v2/estimate/`, `POST /v2/estimate/`, `GET /v2/estimate/summary`.

### Arquivos / módulos afetados

Ver "Modelo de dados / componentes afetados" acima — lista completa dos arquivos a remover ou
editar.

### Testes (ver `docs/conventions/testing.md`)

- **Aceite (mapeia os critérios acima):** teste de integração/smoke que sobe o servidor e
  confirma 404 nas seis rotas legacy removidas e 200/sucesso nas rotas v2 equivalentes
  listadas nos critérios.
- **Unit/integração:** nenhum teste novo de unidade é esperado (é remoção); garantir que
  `go build ./...` e `make test` não quebram por referência órfã ao pacote removido.

### Checklist

- [ ] Gate de pré-requisitos verificado (grep web/mobile + status de release das 3 SPECs) e
      registrado no PR
- [ ] Remover `internal/domain/estimate/*`
- [ ] Remover `internal/domain/balance/*`
- [ ] Remover fiação legacy em `cmd/api/main.go` (imports + wiring)
- [ ] Avaliar e, se aplicável, remover `internal/bootstrap/balance/setup.go` +
      `internal/usecase/balance_usecase.go`
- [ ] Atualizar `docs/swagger.yaml` (remover paths legados)
- [ ] Confirmar que nenhuma rota v2 em uso regride
- [ ] `make all` passa (format + lint + test)

## Divergências a levar ao AYD

- Nenhuma. Único ponto de atenção registrado como borda (não como divergência de contrato): a
  decisão sobre remover ou não `balance_usecase.go` inteiro depende de uma escolha de
  implementação feita em SPEC-002@api (reaproveitar `getBalanceSum` por import direto vs.
  reimplementar a lógica de teto/piso no usecase de summary) — não é ambiguidade do AYD, é
  detalhe a resolver no PR desta SPEC, lendo o código então existente.
