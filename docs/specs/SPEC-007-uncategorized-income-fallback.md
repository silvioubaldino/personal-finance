---
id: SPEC-007
type: spec
title: Categoria de fallback de receita no import de extrato
status: draft
created: 2026-08-25
updated: 2026-08-25
owner: Silvio Ubaldino
parents: [AYD-006@context]
children: []
related: [AYD-003@context, AYD-005@context, GLO]
tags: [import, category, analytics]
superseded_by: null
---

# Spec: Categoria de fallback do import — flavor de receita (parte da api)

## Objetivo

Implementar o papel da api em `AYD-006@context`: criar a `Category` default
`Sem categoria (receita)` (`is_income = true`), rotear o fallback do confirm de `Statement`
pelo sinal do `amount` no momento da escrita e fazer o backfill das linhas já gravadas.

O defeito corrigido: hoje todo `Movement` importado sem categoria cai em `Sem categoria`
(`is_income = false`), então uma entrada (`amount > 0`) nasce classificada como despesa e
contamina os agregados de Análises, Planejamentos e Dashboard — causa raiz da divergência
de 24/ago/2026 (`AYD-003@context` § Divergência de 24/ago/2026).

Correção é **na escrita**, não na leitura: nenhum agregado passa a conhecer a categoria de
fallback como exceção, e a regra "classifica pelo `is_income`, nunca pelo sinal"
(`AYD-005@context` § Recorte canônico) fica intacta.

## Critérios de aceite

```gherkin
Cenário: entrada sem categoria no confirm do import
  Dado um movimento extraído de extrato com amount > 0 e sem category_id
  Quando o usuário confirma o import
  Então o Movement é gravado na Category "Sem categoria (receita)" (is_income = true)

Cenário: saída sem categoria no confirm do import
  Dado um movimento extraído de extrato com amount < 0 e sem category_id
  Quando o usuário confirma o import
  Então o Movement é gravado na Category "Sem categoria" (is_income = false)

Cenário: categoria escolhida pelo usuário prevalece sobre o fallback
  Dado um movimento extraído de extrato com amount > 0 e com category_id preenchido
  Quando o usuário confirma o import
  Então o Movement é gravado na Category escolhida, sem fallback

Cenário: movimento vinculado a recorrência também roteia pelo sinal
  Dado um movimento extraído com recurrence_id, amount > 0 e sem category_id
  E não existe Movement da recorrência no mês
  Quando o usuário confirma o import
  Então o Movement criado usa "Sem categoria (receita)"

Cenário: sugestão por histórico ignora as duas flavors de fallback
  Dado que os Movements anteriores com a mesma descrição estão em "Sem categoria (receita)"
  Quando o import classifica a descrição por histórico
  Então nenhuma sugestão de categoria é devolvida por essa via

Cenário: backfill das entradas já gravadas
  Dado Movements com amount > 0 apontando para "Sem categoria"
  Quando a migração de backfill roda
  Então esses Movements passam a apontar para "Sem categoria (receita)"
  E os Movements com amount <= 0 permanecem em "Sem categoria"
```

## Contratos consumidos/expostos

Definidos em `AYD-006@context` § Decisão de contrato. Esta SPEC não redefine nada:

- `POST /v2/statements/confirm` — **sem mudança de payload**. Muda só qual categoria de
  fallback é gravada quando `category_id` vem nulo.
- `GET /v2/categories` — passa a devolver `Sem categoria (receita)` como qualquer outra
  categoria default (`user_id = default_category_id`). Sem mudança de schema.

## Modelo de dados / componentes afetados

- Uma linha nova em `categories`: `Sem categoria (receita)`, `is_income = true`,
  `user_id = 'default_category_id'`, id `3fad33b7-48da-467f-be49-2e50b1226b82`.
  O id **não** segue o padrão hex de `UncategorizedCategoryID` /
  `InternalTransferOutCategoryID` (que diferem por um nibble), como pede o AYD.
- Nenhuma entidade nova, nenhuma coluna nova.

## Casos de borda & fora de escopo

- **Borda — `amount == 0`:** cai no fallback de despesa (`Sem categoria`), mantendo o
  comportamento atual. Só `amount > 0` vai para a flavor de receita.
- **Borda — sugestão por histórico:** `FindRecentCategorizedByNormalizedDescription` já
  exclui `Sem categoria`; passa a excluir também a flavor de receita, senão o fallback
  vira sugestão e se propaga.
- **Borda — backfill idempotente:** roda por par (categoria, sinal); rodar duas vezes não
  muda nada a mais, e o `down` reverte pelo mesmo par.
- **Fora:** esconder as flavors do seletor de categoria da revisão do import — é web/mobile.
- **Fora:** sugerir categoria por histórico na revisão (`AYD-004@context`).
- **Fora:** tornar `is_income` derivável em vez de flag por categoria.

---

## Plano de implementação

### Abordagem técnica

Uma constante nova no domínio, uma migração (seed + backfill) e uma decisão de roteamento
de uma linha no `StatementUseCase.Confirm` — o `resolveCategoryID` passa a receber o
`amount` e escolher a flavor. Como o `categoryID` é resolvido uma vez por movimento, antes
de bifurcar entre o caminho de recorrência e o caminho normal, a correção cobre os dois.

### Passos

1. `domain.UncategorizedIncomeCategoryID` em `internal/domain/statement.go`.
2. Migração `026_add_uncategorized_income_category`: insere a categoria default e faz o
   backfill (`up`); reverte backfill e remove a categoria (`down`).
3. `resolveCategoryID` passa a rotear pelo sinal do `amount` no `statement_usecase.go`.
4. `FindRecentCategorizedByNormalizedDescription` exclui as duas flavors.
5. Testes de unidade dos cenários de aceite.
6. Linha no `CHANGELOG.md` (o backfill muda números que o usuário já vê: no caso medido a
   despesa do ano vai de −79.947,06 para −85.477,06 — vale nota de release).

### Arquivos / módulos afetados

- `internal/domain/statement.go`
- `internal/usecase/statement_usecase.go`
- `internal/infrastructure/repository/movement_repository.go`
- `db/migrations/026_add_uncategorized_income_category.{up,down}.sql`
- `internal/usecase/statement_usecase_test.go`
- `CHANGELOG.md`

### Testes (ver `docs/conventions/testing.md`)

- **Aceite:** table-driven em `statement_usecase_test.go` cobrindo sinal positivo, negativo,
  zero, categoria escolhida pelo usuário e caminho de recorrência.
- **Unit/integração:** teste do `resolveCategoryID`; o backfill é verificado pela migração
  em ambiente local (não há suíte de migração neste repo).

### Checklist

- [x] Constante de domínio da flavor de receita
- [x] Migração 026 (seed + backfill + down)
- [x] Roteamento por sinal no confirm
- [x] Exclusão das duas flavors na sugestão por histórico
- [x] Testes dos critérios de aceite
- [x] `go build ./...` + testes das packages afetadas verdes (`make all` não roda local:
      gofumpt/goimports/golangci-lint não instalados; `TestPushNotifications_SendDailyUnpaidPush`
      já falhava em `develop`)
- [x] Linha no `CHANGELOG.md`
