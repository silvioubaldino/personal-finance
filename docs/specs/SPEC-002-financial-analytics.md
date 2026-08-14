---
id: SPEC-002
type: spec
title: Endpoint de análises financeiras (dashboard summary)
status: draft
created: 2026-08-14
updated: 2026-08-14
owner: Silvio Ubaldino
parents: [AYD-003@context]
children: []
related: [GLO]
tags: [analytics, dashboard]
superseded_by: null
---

# Spec: Análises financeiras (parte da api)

> Detalha O QUÊ a api faz para cumprir o `AYD-003@context`. Congela ao virar `approved`.

## Objetivo

Expor **um único endpoint agregador** que devolve, por período, tudo que a tela de Análises
desenha: série mensal de renda×despesa, orçado×realizado do mês selecionado, total de
`Invoice` por mês empilhado por `CreditCard`, distribuição da quantidade de despesas por dia
da semana e os KPIs de receita/despesa totais.

A api **não cria tabela nem migração**: agrega `Movement`, `Estimate` e `Invoice` que já
existem. Nenhum cliente reagrega nada.

## Critérios de aceite

```gherkin
Cenário: série mensal cobre todo o span, inclusive meses sem movimentação
  Dado um período de janeiro a março com movimentações só em janeiro e março
  Quando o cliente chama GET /v2/dashboard/summary
  Então monthly_series traz 3 entradas, com fevereiro zerado

Cenário: orçado vs realizado usa o mês de "to"
  Dado um período multi-mês e Estimates cadastrados para o mês de "to"
  Quando o cliente chama GET /v2/dashboard/summary
  Então current_month traz o mês/ano de "to"
  E realized considera só os Movements pagos daquele mês

Cenário: faturas empilhadas por cartão preservam a ordem entre meses
  Dado dois CreditCards com Invoices em meses diferentes do período
  Quando o cliente chama GET /v2/dashboard/summary
  Então credit_card_invoices.cards vem ordenado por nome
  E cada série mensal traz by_card com TODOS os cartões, usando 0 onde não houve fatura
  E total é a soma de by_card[].amount

Cenário: legenda carrega a cor do próprio cartão
  Dado um CreditCard com color cadastrada
  Quando o cliente chama GET /v2/dashboard/summary
  Então cards[].color traz a cor do cartão

Cenário: cartão sem cor cadastrada
  Dado um CreditCard sem color
  Quando o cliente chama GET /v2/dashboard/summary
  Então cards[].color vem vazio, cabendo ao cliente aplicar seu fallback

Cenário: fatura entra no mês do vencimento
  Dado uma Invoice com due_date em fevereiro
  Quando o cliente chama GET /v2/dashboard/summary para janeiro-fevereiro
  Então a fatura aparece na entrada de fevereiro da série

Cenário: distribuição por dia da semana conta despesa pendente
  Dado uma compra no cartão ainda não paga
  Quando o cliente chama GET /v2/dashboard/summary
  Então ela é contada em expense_weekday_distribution

Cenário: distribuição por dia da semana ignora transferência interna
  Dado um Movement de saída com type_payment internal_transfer
  Quando o cliente chama GET /v2/dashboard/summary
  Então ele NÃO é contado em expense_weekday_distribution

Cenário: distribuição sempre traz os sete dias
  Dado um período sem nenhuma despesa
  Quando o cliente chama GET /v2/dashboard/summary
  Então expense_weekday_distribution traz 7 entradas com count 0 e percentage 0

Cenário: período inválido é rejeitado
  Dado from posterior a to
  Quando o cliente chama GET /v2/dashboard/summary
  Então a api responde 400
```

## Contratos consumidos/expostos

Expõe `GET /v2/dashboard/summary?from&to` exatamente como definido em `AYD-003@context`
(§ Contrato). Esta SPEC **não redefine** campos nem semântica — em caso de divergência, o
AYD vence e a correção é PR no repo de contexto.

Autenticação: Firebase (`user_token`), como todas as rotas clean-arch. Todo dado é isolado
por `user_id` via `BuildBaseQuery`.

## Modelo de dados / componentes afetados

| Camada | Arquivo | Mudança |
|---|---|---|
| Domain | `internal/domain/dashboard.go` | `DashboardSummary` ganha `CreditCardInvoices` e `ExpenseWeekdayDistribution`; `DashboardKPIs` reduzido a `TotalIncome`/`TotalExpense`; novos tipos `CreditCardInvoiceSummary`, `CreditCardRef` (id + nome + cor), `CreditCardInvoicePoint`, `CreditCardInvoiceSlice`, `ExpenseWeekdayPoint` |
| Usecase | `internal/usecase/dashboard_usecase.go` | Nova dependência `DashboardInvoiceRepository`; `buildCreditCardInvoices`, `buildCardLegend` e `buildExpenseWeekdayDistribution`; `buildKPIs` deixa de calcular médias, saldo e taxa de poupança |
| Repository | `internal/infrastructure/repository/invoice_repository.go` | `FindByPeriod(ctx, period)`, filtrando por `due_date` (mesma convenção de `FindByMonth`) |
| Bootstrap | `internal/bootstrap/dashboard/setup.go` | Injeta `GetInvoiceRepository()` no usecase |
| API | `internal/infrastructure/api/dashboard_api.go` | Sem mudança — o handler só serializa o que o usecase devolve |

Sem migração: nenhuma tabela nova, nenhuma coluna nova.

## Casos de borda & fora de escopo

- **Borda:** período sem faturas → `cards: []` e uma entrada zerada por mês.
- **Borda:** `Invoice` sem `credit_card_id` é ignorada (não há como empilhar).
- **Borda:** dois cartões de mesmo nome → desempate estável pelo `credit_card_id`.
- **Borda:** `color` é opcional no `CreditCard`; quando vazia, sobe vazia (a api não inventa
  cor — o fallback é decisão de apresentação, do cliente). Sem custo extra de query: o
  repositório de `Invoice` já faz `Preload("CreditCard")`.
- **Borda:** período sem despesa → `percentage` 0 nos sete dias (nunca divide por zero).
- **Fora:** top categorias no tempo, fixo×variável e projeção de fluxo de caixa.
- **Fora:** `GetExpenseMovements` filtra só por `amount < 0` e portanto inclui transferências
  internas em `monthly_series` e nos KPIs, contrariando o GLO. A distribuição por dia da
  semana já as exclui; alinhar o resto é correção à parte (registrada no `AYD-003@context`).
