---
id: SPEC-002
type: spec
title: Endpoint de summary de planejamentos (orçado × realizado)
status: draft
created: 2026-08-17
updated: 2026-08-17
owner: Silvio Ubaldino
parents: [AYD-005@context]
children: []
related: [GLO]
tags: [estimate, planning, balance, consistency]
superseded_by: null
---

# Spec: Endpoint de summary de planejamentos (parte deste repo)

> **Doc único deste repo para a feature:** o *o quê* (objetivo, critérios, contratos) e o
> *como* (plano de implementação). A parte "o quê" **congela ao virar `approved`**; a
> checklist do plano segue sendo marcada durante a execução, sem mudar o `status`.

## Objetivo

Implementar `GET /v2/estimate/summary?month&year`, dono único do cálculo de orçado × realizado
para a página de Planejamentos (Fase 2 de AYD-005@context). Hoje o cálculo está reimplementado
quatro vezes (api legacy, api v2, web, mobile) e diverge em cinco pontos — a api passa a ser
a única fonte, devolvendo números prontos; web e mobile só exibem.

Esta SPEC também corrige o **débito técnico bloqueante** que impede reusar o cálculo hoje
existente e move para a api a **normalização de sinal na escrita** de `Estimate`, hoje feita
em duplicata em cada front.

## Critérios de aceite

```gherkin
Cenário: Movement pendente entra em realized mas não em realized_paid
  Dado um Movement de "-300" em Alimentação com is_paid=false no mês consultado
  Quando GET /v2/estimate/summary é chamado para esse mês
  Então a linha de Alimentação soma "-300" em realized
  E realized_paid não inclui esse valor

Cenário: InternalTransfer não entra no realizado
  Dado duas Movements com type_payment=internal_transfer (pair_id) no mês consultado
  Quando GET /v2/estimate/summary é chamado para esse mês
  Então nenhuma das duas pernas aparece somada em nenhuma categoria
  E nenhuma linha "fantasma" de transferência aparece em categories[]

Cenário: invoice_payment não entra no realizado
  Dado um Movement de type_payment=invoice_payment referente ao pagamento de uma Invoice
  Quando GET /v2/estimate/summary é chamado para o mês desse pagamento
  Então esse Movement não é somado em realized de nenhuma categoria

Cenário: Compra no cartão entra via itens da Invoice
  Dado uma compra de "-400" no CreditCard, categorizada em Restaurante, dentro de uma Invoice do mês consultado
  Quando GET /v2/estimate/summary é chamado para esse mês
  Então a linha de Restaurante soma "-400" em realized, obtida a partir de invoices[].movements

Cenário: invoice_remainder entra no mês da Invoice que o recebe
  Dado um Movement type_payment=invoice_remainder, categoria fixa de cartão, dentro da Invoice seguinte
  Quando GET /v2/estimate/summary é chamado para o mês dessa Invoice seguinte
  Então o valor do remainder soma em realized da categoria fixa de cartão nesse mês
  E não aparece no mês da Invoice original

Cenário: progress é null quando budgeted é zero
  Dado uma categoria sem Estimate no mês (budgeted = 0) mas com Movements realizados
  Quando GET /v2/estimate/summary é chamado
  Então a linha dessa categoria tem progress = null

Cenário: Linha virtual para categoria com gasto e sem planejamento
  Dado uma categoria com Movements no mês mas sem Estimate cadastrado
  Quando GET /v2/estimate/summary é chamado
  Então a categoria aparece em categories[] com is_planned=false, estimate_id=null e budgeted=0

Cenário: is_income vem da flag da Category, não do sinal do valor
  Dado um Movement de estorno em categoria de despesa (is_income=false) com valor positivo
  Quando GET /v2/estimate/summary é chamado
  Então a linha da categoria mantém is_income=false
  E o valor do estorno reduz o total de despesa, não aparece como receita

Cenário: Ordenação do array categories
  Dado categorias de receita e despesa, planejadas e não planejadas, no mesmo mês
  Quando GET /v2/estimate/summary é chamado
  Então o array vem com receitas antes de despesas, planejadas antes de não planejadas, e alfabética por nome dentro de cada bloco

Cenário: month inválido retorna 400
  Dado month=13 ou month não numérico
  Quando GET /v2/estimate/summary é chamado
  Então a resposta é 400 com erro de invalid input

Cenário: year inválido retorna 400
  Dado year não numérico
  Quando GET /v2/estimate/summary é chamado
  Então a resposta é 400 com erro de invalid input

Cenário: Isolamento por user_id
  Dado dois usuários com Estimates e Movements na mesma categoria e mês
  Quando o usuário A chama GET /v2/estimate/summary
  Então o payload só contém dados do usuário A

Cenário: Golden file — cenário de "Impacto medido" do AYD-005
  Dado o cenário completo descrito em "Golden file" abaixo
  Quando GET /v2/estimate/summary é chamado para o mês desse cenário
  Então totals.income.consolidated = 5000
  E totals.expense.consolidated = -1300
  E totals.period_balance = 3700
  E a linha de Alimentação tem realized = -1300 e realized_paid = -1000
  E totals.expense.realized_paid = -1000
```

## Contratos consumidos/expostos

Contrato definido em AYD-005@context, seção "Contrato (fonte da verdade)" — este repo **não
o redefine**. Resumo do que é exposto (ver o AYD para a semântica completa de cada campo):

```
GET /v2/estimate/summary?month=8&year=2026
Auth: Firebase (header user_token)
```

Response: `{ month, year, totals: { income, expense, period_balance }, categories[] }`, com
`categories[].subcategories[]` (dois níveis apenas). Campos: `estimate_id`, `category_id`,
`category_name`, `is_income`, `is_planned`, `budgeted`, `realized`, `realized_paid`, `result`,
`progress`, e em `totals.{income,expense}` também `consolidated`.

**Recorte canônico de `realized`** (AYD-005@context): dentro — `Movement` normal do período
(pago ou pendente), compra no `CreditCard` via itens da `Invoice`, `invoice_remainder` via
itens da `Invoice` do mês que o recebe, `Movement` recorrente projetado. Fora —
`internal_transfer` (as duas pernas), `invoice_payment`.

Endpoints legados (`GET /estimate`, `POST/PUT /estimate`, `/sub-estimate`,
`GET /v2/estimate/`, `GET /v2/balance/estimate/period`) **continuam intactos** — mudança
puramente aditiva.

### Golden file — "Impacto medido" (transcrito do AYD)

Mesmo mês. Planejado: Alimentação `-1.000`, Salário `+5.000`. Movimentações do mês:

| Movement | Amount | is_paid | type_payment | Category |
|---|---:|---|---|---|
| Salário | `+5.000` | true | (normal) | Salário (income) |
| Mercado | `-600` | true | (normal, débito) | Alimentação |
| Mercado (pendente) | `-300` | **false** | (normal, débito) | Alimentação |
| Restaurante | `-400` | true | `credit_card` (item de Invoice do mês) | Alimentação |
| Pagamento da fatura | `-400` | true | `invoice_payment` | (categoria fixa de cartão) |
| Transferência interna | `1.000` / `-1.000` | true | `internal_transfer` (par) | (categorias de transferência) |

Resultado esperado sob o contrato, batendo com o payload de exemplo do AYD:

| Campo | Valor |
|---|---:|
| `totals.income` | `budgeted 5000` · `realized 5000` · `realized_paid 5000` · `consolidated 5000` |
| `totals.expense` | `budgeted -1000` · `realized -1300` · `realized_paid -1000` · `consolidated -1300` |
| `totals.period_balance` | `3700` |
| Alimentação | `realized -1300` · `realized_paid -1000` · `result -300` · `progress 1.3` |

`realized_paid` de Alimentação é `-1000` = mercado pago `-600` + restaurante no cartão `-400`:
a compra no cartão é `is_paid = true` (o que fica pendente é a `Invoice`, não o item), então
entra nos dois campos. O único valor que separa `realized` de `realized_paid` neste cenário é o
mercado pendente de `-300`. Este é o conjunto de números que `SPEC-003@web` e `SPEC-004@mobile`
devem bater.

## Modelo de dados / componentes afetados

Nenhuma entidade nova, nenhuma migração (conforme o AYD). Usa `Estimate`
(`EstimateCategories`/`EstimateSubCategories`), `Movement`, `Category`/`Subcategory`,
`Invoice` — todos já existentes.

## Casos de borda & fora de escopo

- Borda: mês/ano sem nenhum `Estimate` cadastrado (só linhas virtuais); mês sem nenhum
  `Movement` (categorias planejadas aparecem com `realized=0`, `progress=0`); categoria com
  `Estimate` de categoria mas sem nenhum de subcategoria (bloco `subcategories` vazio).
- Fora: modal de detalhes (lista de `Movement`s) — continua em `GET /v2/movements/`; migração
  do CRUD de `Estimate` para v2 (Fase 3.5, SPEC futura no web); tipos gerados do contrato;
  endpoint por categoria (`.../summary/{category_id}/movements`) — questão em aberto no AYD,
  não bloqueia esta SPEC.

---

## Plano de implementação

> Parte de trabalho deste doc. Decisão técnica não trivial vira TDR, não fica só aqui.

### Abordagem técnica

Feature nova em clean architecture, seguindo o padrão de `internal/bootstrap/balance/setup.go`
e `internal/bootstrap/estimate/setup.go`: um usecase novo (`internal/usecase/estimate_summary_usecase.go`
ou método adicional em `estimate_usecase.go` — decidir no PR, mantendo a struct nomeada pela
entidade, sem sufixo `Usecase`), consumindo `MovementRepository`, `EstimateRepository` e
`InvoiceRepository`/usecase de `Invoice` já existentes. Handler novo em
`internal/infrastructure/api/estimate_api.go` (rota adicional no mesmo grupo `/v2/estimate`)
ou arquivo próprio `estimate_summary_api.go`, registrado em `internal/bootstrap/estimate/setup.go`.

Passos centrais do cálculo:
1. Buscar `Movement`s do período com o recorte canônico: reaproveitar
   `MovementRepository.FindByPeriod` (`internal/infrastructure/repository/movement_repository.go:98-122`),
   que já exclui `credit_card` e `invoice_remainder` do resultado (linha 107-110) — mas
   **inclui `internal_transfer` e `invoice_payment`**, que precisam ser filtrados no usecase
   (ou em uma nova query), pois o recorte canônico do AYD exclui os dois.
2. Buscar itens de `Invoice` do mês via o mesmo caminho de
   `Invoice.FindDetailedInvoicesByPeriod` (`internal/usecase/invoice_usecase.go:118-138`), que
   usa `MovementRepository.FindByInvoiceID` (`movement_repository.go:261-281`) — este já
   exclui `invoice_payment` por padrão (linha 269), então os itens retornados já são as
   compras no cartão + eventuais `invoice_remainder` daquela `Invoice`. Somar esses itens em
   `realized`, no mês da `Invoice` (não no mês da compra original, quando divergir — caso do
   `invoice_remainder`).
3. Somar por `Category`/`Subcategory`, separando `realized` (tudo do passo 1+2) de
   `realized_paid` (mesmo conjunto, restrito a `is_paid=true`).
4. Buscar `Estimate`s do mês via `EstimateRepository.FindCategoriesByMonth` /
   `FindSubcategoriesByMonth` (usados em `estimate_usecase.go:43-69`).
5. Montar as linhas: cruzar `Estimate` × soma realizada por categoria/subcategoria; categoria
   com soma e sem `Estimate` vira linha virtual (`is_planned=false`, `estimate_id=null`,
   `budgeted=0`).
6. Calcular `result = realized - budgeted`; `progress = abs(realized)/abs(budgeted)` ou `null`
   se `budgeted=0`; `consolidated` reaproveitando a lógica de teto/piso hoje em
   `getBalanceSum`/`balance_usecase.go:70-97` (min para despesa, max para receita) — adaptar
   para trabalhar com `uuid.UUID` normal, não ponteiro (ver correção do débito técnico abaixo).
7. `totals.period_balance = totals.income.consolidated + totals.expense.consolidated`.
8. Ordenar: receita antes de despesa, planejado antes de não planejado, alfabético por nome.

**Correção do débito técnico bloqueante** (pré-requisito, antes do passo 6 acima funcionar
corretamente):
- `internal/domain/movement.go:83-93` (`GetSumByCategory`): troca `map[*uuid.UUID]float64`
  por `map[uuid.UUID]float64`, lendo e escrevendo consistentemente a chave (hoje lê
  `movement.CategoryID` e escreve `movement.Category.ID` — dois ponteiros distintos que nunca
  batem `ok`).
- `internal/domain/estimatecategory.go:23-33` (`GetEstimateByCategory`): mesmo padrão, mesma
  correção.
- `internal/usecase/balance_usecase.go:70-97` (`getBalanceSum`): hoje desreferencia ponteiros
  (`*id`) para colapsar em `map[uuid.UUID]float64` — atualizar as assinaturas para já receber
  `map[uuid.UUID]float64` dos dois pontos acima e remover a desreferenciação.
- Adicionar teste table-driven cobrindo duas `Movement`s (ou dois `Estimate`s) da mesma
  categoria, hoje quebrado por essa correção.

**Normalização de sinal na escrita** (`POST`/`PUT` de `Estimate`): hoje
`EstimateHandler.AddEstimateCategory`/`UpdateEstimateCategoryAmount`
(`internal/infrastructure/api/estimate_api.go:71-88, 109-134`) aceitam `amount` como veio no
body, sem normalizar sinal. Adicionar no usecase (`estimate_usecase.go`,
`AddEstimateCategory`/`UpdateEstimateCategoryAmount`/`AddEstimateSubCategory`/
`UpdateEstimateSubCategoryAmount`) uma normalização: buscar a `Category` (via
`CategoryRepository`, já disponível no registry — `Registry.GetCategoryRepository()`), checar
`IsIncome` (`internal/domain/category.go:14`) e forçar o sinal do `amount` (negativo para
despesa, positivo para receita) antes de persistir. Isso descarta a necessidade dos
equivalentes hoje duplicados em `normalizeEstimateAmount`@web e no `EstimateModal`@mobile —
mas essa remoção é trabalho das SPECs de web/mobile, não desta.

### Passos

1. Corrigir `GetSumByCategory` (`internal/domain/movement.go`) e `GetEstimateByCategory`
   (`internal/domain/estimatecategory.go`) para `map[uuid.UUID]float64`; ajustar
   `getBalanceSum` (`internal/usecase/balance_usecase.go`) e seus testes existentes.
2. Adicionar normalização de sinal na escrita em `estimate_usecase.go` (as quatro operações de
   escrita), buscando `IsIncome` via `CategoryRepository`.
3. Criar o usecase de summary (arquivo novo ou método em `estimate_usecase.go`), implementando
   o recorte canônico e a semântica de campos descritos acima; interface de dependências
   estreita (`EstimateSummaryMovementRepository`, `EstimateSummaryInvoiceReader`, etc.,
   conforme a skill `go-usecases`).
4. Criar/estender o handler em `internal/infrastructure/api/estimate_api.go` com
   `GET /v2/estimate/summary`, validando `month` (1-12) e `year` (inteiro) com
   `domain.WrapInvalidInput` em caso de erro.
5. Atualizar `internal/bootstrap/estimate/setup.go` para injetar as novas dependências
   (repositório de movement/invoice) no usecase de summary.
6. Atualizar `docs/swagger.yaml` com o novo path, parâmetros e schema de resposta.
7. Escrever os testes table-driven (um por regra da tabela de semântica) e o teste de golden
   file com o cenário de "Impacto medido".
8. Rodar `make all` (format + lint + test).

### Arquivos / módulos afetados

- `internal/domain/movement.go` (correção `GetSumByCategory`)
- `internal/domain/estimatecategory.go` (correção `GetEstimateByCategory`)
- `internal/usecase/balance_usecase.go` (ajuste `getBalanceSum` para o novo tipo de mapa)
- `internal/usecase/estimate_usecase.go` (normalização de sinal na escrita + usecase de summary)
- `internal/infrastructure/api/estimate_api.go` (novo handler `GET /v2/estimate/summary`)
- `internal/bootstrap/estimate/setup.go` (DI das novas dependências)
- `docs/swagger.yaml` (novo path documentado)

### Testes (ver `docs/conventions/testing.md`)

- **Aceite (mapeia os critérios acima):** um teste table-driven por cenário Gherkin listado
  (pendente em `realized`/fora de `realized_paid`; `internal_transfer` fora; `invoice_payment`
  fora; compra no cartão via itens de `Invoice`; `invoice_remainder` no mês certo; `progress`
  null; linha virtual; `is_income` pela flag; ordenação; `month`/`year` inválidos → 400;
  isolamento por `user_id`); mais o teste de golden file do cenário "Impacto medido".
- **Unit/integração:** teste da correção de `GetSumByCategory`/`GetEstimateByCategory` com
  duas movimentações/estimates da mesma categoria; teste da normalização de sinal na escrita
  (despesa forçada negativa, receita forçada positiva); teste de `consolidated` (teto/piso)
  reaproveitando os casos hoje cobertos em `balance_usecase_test.go`, adaptados ao novo tipo
  de mapa.

### Checklist

- [x] Corrigir `GetSumByCategory` (movement.go) para `map[uuid.UUID]float64`
- [x] Corrigir `GetEstimateByCategory` (estimatecategory.go) para `map[uuid.UUID]float64`
- [x] Ajustar `getBalanceSum` (balance_usecase.go) e seus testes existentes
- [x] Implementar normalização de sinal na escrita de `Estimate` (POST/PUT, category e sub-category)
- [x] Implementar usecase de summary com o recorte canônico de `realized`
- [x] Implementar handler `GET /v2/estimate/summary` com validação de `month`/`year`
- [x] Registrar DI em `internal/bootstrap/estimate/setup.go`
- [x] Atualizar `docs/swagger.yaml`
- [x] Testes table-driven por regra da tabela de semântica
- [x] Teste de golden file do cenário "Impacto medido" (receita 5000, despesa -1300, saldo 3700)
- [x] Confirmar que os endpoints legados (`/estimate`, `/sub-estimate`, `/v2/estimate/`,
      `/v2/balance/estimate/period`) continuam respondendo sem alteração
- [ ] `make all` passa (format + lint + test) — `go build`/`gofmt`/`go test` passam no escopo
      tocado; `golangci-lint` não roda neste ambiente (binário local v2.11.4 incompatível com
      `.golangci.yml` v1, pré-existente, não relacionado a esta mudança)

## Divergências a levar ao AYD

- Nenhuma divergência de contrato. Uma dúvida sobre a composição de `realized_paid` no golden
  file (a compra no cartão entra ou não?) foi levantada ao escrever esta SPEC e **resolvida na
  revisão de 17/ago/2026 pelo próprio AYD**, sem adenda: o payload de exemplo já traz
  `realized_paid: -1000` para despesa, ou seja `-600` (mercado pago) + `-400` (restaurante no
  cartão). O `is_paid` do item de cartão é `true` — o que fica pendente é a `Invoice`, não o
  item —, então a compra no cartão entra nos dois campos e só o mercado pendente de `-300`
  separa `realized` de `realized_paid`. O golden file acima já reflete isso.
