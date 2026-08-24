---
id: SPEC-002
type: spec
title: Endpoint de análises financeiras (dashboard summary)
status: review
created: 2026-08-14
updated: 2026-08-22
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
da semana, total de despesa por `Category` no período e os KPIs de receita/despesa totais.

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
  E realized considera só os Movements pagos daquele mês, sob o recorte canônico

Cenário: realized é a soma pura, sem o teto/piso do Balance legacy
  Dado uma Category de receita com 5000 orçado e 4800 realizado no mês
  Quando o cliente chama GET /v2/dashboard/summary
  Então budget.income.realized traz 4800 — o mesmo realized_paid de GET /v2/estimate/summary,
    e não os 5000 que o teto/piso mostrava

Cenário: a compra no cartão entra itemizada e o pagamento da fatura não
  Dado uma Invoice paga de -350, com uma compra de -300 em Alimentação e um
    invoice_payment de -350
  Quando o cliente chama GET /v2/dashboard/summary
  Então kpis.total_expense traz -300, não -650
  E expense_by_category traz -300 em Alimentação

Cenário: item de fatura conta no mês do vencimento dela
  Dado uma compra em 30 de janeiro numa Invoice que vence em 10 de fevereiro
  Quando o cliente chama GET /v2/dashboard/summary para janeiro-fevereiro
  Então ela soma na entrada de fevereiro de monthly_series
  E aparece no dia da semana de 30 de janeiro, que é quando a compra aconteceu

Cenário: receita e despesa saem da flag da Category, não do sinal
  Dado um estorno de +200 numa Category de despesa
  Quando o cliente chama GET /v2/dashboard/summary
  Então ele reduz o expense do mês, em vez de virar income

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

Cenário: distribuição por dia da semana ignora o remanescente de fatura
  Dado um Movement com type_payment invoice_remainder
  Quando o cliente chama GET /v2/dashboard/summary
  Então ele NÃO é contado em expense_weekday_distribution — é saldo empurrado, não compra

Cenário: distribuição sempre traz os sete dias
  Dado um período sem nenhuma despesa
  Quando o cliente chama GET /v2/dashboard/summary
  Então expense_weekday_distribution traz 7 entradas com count 0 e percentage 0

Cenário: despesa por categoria soma só o que está pago
  Dado duas despesas na mesma Category, uma paga e uma pendente
  Quando o cliente chama GET /v2/dashboard/summary
  Então expense_by_category traz um total que soma só a paga

Cenário: despesa por categoria ignora transferência interna
  Dado um Movement de saída com type_payment internal_transfer numa Category
  Quando o cliente chama GET /v2/dashboard/summary
  Então essa Category não conta esse valor em expense_by_category
  E o mesmo vale para os category_id fixos de transferência interna

Cenário: despesa por categoria vem ordenada da maior para a menor
  Dado três Categories com totais de despesa diferentes no período
  Quando o cliente chama GET /v2/dashboard/summary
  Então expense_by_category vem ordenado do maior gasto (em módulo) para o menor

Cenário: categoria sem despesa no período não aparece
  Dado uma Category sem nenhum Movement pago no período
  Quando o cliente chama GET /v2/dashboard/summary
  Então ela não tem entrada em expense_by_category (sem zero-fill, ao contrário de by_card)

Cenário: os totais do payload fecham entre si
  Dado um período com movimentações avulsas, compras no cartão e uma Category de despesa
    que fechou o período positiva por causa de um estorno
  Quando o cliente chama GET /v2/dashboard/summary
  Então a soma de expense_by_category[].total é igual a kpis.total_expense
  E kpis.total_expense é igual à soma de monthly_series[].expense
  E kpis.total_income é igual à soma de monthly_series[].income
  E current_month.budget.{income,expense}.realized é a fatia do mês de "to" desses números

Cenário: categoria que fechou o período positiva continua no array
  Dado uma Category de despesa com -800 de gasto e +1500 de estorno no período
  Quando o cliente chama GET /v2/dashboard/summary
  Então ela aparece em expense_by_category com total +700
  E é ela que faz a soma fechar com kpis.total_expense

Cenário: Movement sem Category fica fora de todo agregado de dinheiro
  Dado um Movement pago sem category_id
  Quando o cliente chama GET /v2/dashboard/summary
  Então ele não entra em monthly_series, kpis, expense_by_category nem budget.realized

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
| Domain | `internal/domain/dashboard.go` | `DashboardSummary` ganha `CreditCardInvoices`, `ExpenseWeekdayDistribution` e `ExpenseByCategory`; `DashboardKPIs` reduzido a `TotalIncome`/`TotalExpense`; novos tipos `CreditCardInvoiceSummary`, `CreditCardRef` (id + nome + cor), `CreditCardInvoicePoint`, `CreditCardInvoiceSlice`, `ExpenseWeekdayPoint`, `CategoryExpensePoint` (id + nome + cor + total) |
| Usecase | `internal/usecase/dashboard_usecase.go` | Depende de `DashboardInvoiceUseCase` (`FindDetailedInvoicesByPeriod`) no lugar do repositório de `Invoice`: os agregados de dinheiro passam a somar `realizedEntry` — o Movement do recorte canônico junto com o mês em que ele conta. `buildCurrentMonth` reusa `aggregateRealized` + `buildTotals` do summary de planejamentos e devolve `RealizedPaid`, sem o teto/piso de `getBalanceSum`. `buildExpenseWeekdayDistribution` exclui `invoice_remainder`; `buildExpenseByCategory` classifica por `Category.IsIncome` |
| Usecase | `internal/usecase/estimate_usecase.go` | `isCanonicalRealized` extraído como **única** definição do recorte no servidor, e `internalTransferCategoryIDs` movido para junto dela |
| Usecase | `internal/usecase/invoice_usecase.go` | `FindDetailedInvoicesByPeriod` honra o período inteiro (`repo.FindByPeriod` no lugar de `repo.FindByMonth(period.From)`) e busca os itens de todas as faturas numa query só |
| Repository | `internal/infrastructure/repository/movement_repository.go` | `FindByInvoiceIDs(ctx, ids)` — versão em lote de `FindByInvoiceID`, para o span multi-mês não virar uma query por fatura |
| Bootstrap | `internal/bootstrap/dashboard/setup.go` | Monta o usecase de `Invoice` e injeta no dashboard |
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
- **Borda:** `expense_by_category` usa só pagos (regra geral de "realizado", decisão #2 — ao
  contrário da distribuição por dia da semana, que é a única exceção). Categoria com uma
  única despesa ainda aparece; sem despesa no período, não aparece (nenhum zero-fill).
- **Borda:** `Category` de despesa que fecha o período positiva (estorno maior que o gasto)
  **fica** em `expense_by_category`, com o total positivo — é ela que faz
  `sum(expense_by_category) == kpis.total_expense`. Só o total exatamente zero sai. Cabe ao
  cliente filtrar por `total < 0` antes do módulo, na hora de desenhar as barras.
- **Borda:** `Movement` sem `category_id` fica fora de todos os agregados de dinheiro — não
  há como classificar receita×despesa nem agrupar. Mesmo corte de `aggregateRealized`.
- **Borda:** compra no cartão de uma `Invoice` que vence fora do span não entra em nenhum
  agregado do período — a seleção é por `due_date`, e é isso que mantém
  `sum(monthly_series) == kpis`.
- **Borda:** a compra no cartão nasce `is_paid = false` e só vira paga quando a `Invoice` é
  paga (`PayByInvoiceID`); o `invoice_remainder` nunca é marcado pago. Por isso o cartão
  entra no realizado no mês em que a fatura é quitada, e o remanescente fica só nos
  agregados que aceitam pendente.
- **Fora:** agrupar categorias pequenas em "Outros" quando há muitas — fica para quando isso
  se mostrar necessário na tela real.
- **Fora:** top categorias no tempo, fixo×variável e projeção de fluxo de caixa.
- **Fora:** `GetExpenseMovements` filtra só por `amount < 0` e portanto incluía
  transferências internas em `monthly_series` e nos KPIs, contrariando o GLO. Corrigido em
  `#224`/`#226` com `GetOperationalMovements` e o filtro pelos `category_id` de transferência
  interna.
- **Resolvido:** os três recortes de `type_payment` que conviviam na mesma resposta viraram
  um só, o canônico de `AYD-005@context` (decidido pelo owner em 22/ago/2026 e registrado em
  `AYD-003@context` § "Recorte de 'realizado'"). A garantia de não contar a fatura duas vezes
  passou a ser do recorte — `isCanonicalRealized` recusa qualquer `Movement` que pertença a
  uma `Invoice`, porque ele entra pelos itens dela — e não mais do filtro de SQL herdado do
  `Agent`, que era o que segurava a duplicação sem estar documentado em lugar nenhum.
