---
id: SPEC-010
type: spec
title: Adiantamento de fatura de cartão de crédito (parte da api)
status: draft
created: 2026-11-26
updated: 2026-11-26
owner: Silvio Ubaldino
parents: [AYD-007@context]
children: []
related: [AYD-004@context, AYD-003@context, SPEC-001, SPEC-008, GLO]
tags: [invoice, credit-card, payment, import, statement]
superseded_by: null
---

# Spec: Adiantamento de fatura (parte da api)

## Objetivo

Implementar o papel da api em `AYD-007@context`: o mecanismo de `InvoiceAdvancePayment`
(pagar parte de uma `Invoice` antes de quitá-la), os campos derivados da `Invoice`, o `pay`/
`revert-pay` sobre o restante, o ajuste do import da fatura do cartão (filtra e classifica
todas as linhas de pagamento, confere o total com os adiantamentos) e o vínculo da linha de
pagamento do cartão no import do extrato bancário com a `Invoice`.

Contratos **não** são redefinidos aqui: vivem no AYD-007.

## Critérios de aceite

```gherkin
Cenário: adiantamento debita a carteira, libera limite e mantém a fatura aberta
  Dado uma Invoice aberta de -1503,90 e uma Wallet com saldo suficiente
  Quando o usuário adianta 150,00 em 11/01
  Então existe um Movement invoice_advance_payment de -150,00 ligado à Invoice, is_paid = true
  E o saldo da Wallet cai 150,00 e o limite do CreditCard sobe 150,00
  E a Invoice segue aberta com amount -1503,90, advance_paid_amount -150,00 e remaining_amount -1353,90

Cenário: adiantar o restante inteiro ou mais é proibido
  Dado uma Invoice aberta com restante -1503,90
  Quando o usuário adianta 1503,90 (ou mais)
  Então a api responde 422 advance_payment_settles_invoice e nada é gravado

Cenário: fatura paga não aceita adiantamento nem desfazimento
  Dado uma Invoice paga
  Quando o usuário adianta ou desfaz um adiantamento
  Então a api responde 409 invoice_already_paid

Cenário: data do adiantamento depois do vencimento
  Quando o usuário informa payment_date posterior a due_date
  Então a api responde 422 advance_payment_after_due_date

Cenário: pay opera sobre o restante (caso Mercado Pago)
  Dado uma Invoice de -1503,90 com adiantamentos de -199,00
  Quando o usuário paga sem informar amount
  Então a quitação é de -1304,90 e nenhuma sobra vai para a fatura seguinte

Cenário: revert-pay desfaz só a quitação
  Dado uma Invoice quitada que tem adiantamentos
  Quando o usuário reverte o pagamento
  Então a quitação some e os adiantamentos permanecem

Cenário: desfazer adiantamento
  Quando o usuário apaga um adiantamento da Invoice aberta
  Então o valor volta à Wallet, o limite volta a ser consumido e o Movement é apagado
  E um movement_id que não é adiantamento desta Invoice responde 404 advance_payment_not_found

Cenário: adiantamento nunca é despesa nem realizado
  Dado um Movement invoice_advance_payment no período
  Então ele fica fora do realizado (planejamentos, análises, dashboard) e dos SQLs do agente

Cenário: import da fatura separa pagamento da anterior e adiantamentos
  Dado uma fatura (cartão fecha dia 29, vence dia 4) com linhas de pagamento em 30/12, 11/01 e 16/01
  Quando o usuário extrai a fatura
  Então 30/12 sai excluded com exclusion_reason invoice_payment
  E 11/01 e 16/01 saem excluded com exclusion_reason advance_payment, com amount positivo
  E o total confere (Σ itens + Σ advance_payment ≈ invoice_meta.total_amount) sem total_amount_mismatch
  E o warning advance_payment_detected traz detected "199.00"

Cenário: fatura anterior paga com atraso (desempate pelo total)
  Dado que a classificação por data não reconcilia com o total e "tudo invoice_payment" reconcilia
  Então todas as linhas de pagamento saem invoice_payment

Cenário: confirm-invoice nunca registra pagamento
  Dado uma linha de pagamento (marcada ou não) enviada ao confirm-invoice
  Então ela é pulada

Cenário: linha de extrato que quita uma fatura
  Dado um extrato com saída igual ao restante de uma Invoice aberta e termo de fatura/cartão na descrição
  Quando o usuário extrai o extrato
  Então a linha traz invoice_payment_match (kind settlement, confiança 0.95) e invoice_payment pré-decidido
  E a resposta traz o warning invoice_payment_match_found

Cenário: linha de extrato que adianta uma fatura
  Dado uma saída marcada como pagamento de cartão, menor que o restante e até o vencimento
  Então a linha traz invoice_payment_match (kind advance, confiança 0.7) sem decisão pré-preenchida

Cenário: confirm de linha vinculada não cria despesa
  Dado uma linha com invoice_payment
  Quando o usuário confirma o extrato
  Então a api executa a regra do pay (settlement) ou do adiantamento (advance) com a Wallet, data e valor da linha
  E nenhuma despesa é criada, e invoice_payments_created conta o pagamento
  E violar a regra (fatura já paga, valor ≥ restante num advance) manda a linha para errors e skipped, sem despesa

Cenário: reimportar o mesmo extrato
  Dado que o hash da linha foi gravado no Movement de pagamento
  Quando o mesmo extrato é confirmado de novo, ou o adiantamento já existe com mesmo valor e data a até 3 dias
  Então a linha é pulada
```

## Contratos consumidos/expostos

Expostos (definidos em `AYD-007@context`, não redefinidos aqui):
- `POST /v2/invoices/{id}/advance-payments`, `DELETE /v2/invoices/{id}/advance-payments/{movement_id}`.
- `Invoice` + `advance_paid_amount`, `remaining_amount` (todas as respostas) e `advance_payments[]`
  (`GET /v2/invoices/{id}`, `/detailed` e os dois endpoints acima).
- `pay` (restante) e `revert-pay` (só a quitação).
- `Movement.type_payment = invoice_advance_payment`.
- Import da fatura: `exclusion_reason: advance_payment`, warning `advance_payment_detected`, checksum reformulado.
- Import do extrato: `invoice_payment_match`, `invoice_payment`, warning `invoice_payment_match_found`,
  `invoice_payments_created` no `/confirm`.

## Modelo de dados / componentes afetados

- Nenhuma tabela/coluna nova: o adiantamento é um `Movement` com `type_payment` próprio; os derivados da
  `Invoice` são calculados na leitura.
- `internal/domain/typepayment.go` — `TypePaymentInvoiceAdvancePayment`, `IsInvoicePayment()`,
  `InvoicePaymentTypes()` e `NonRealizedTypePayments()` (conjunto único usado em Go e nos SQLs).
- `internal/domain/invoice.go` — derivados e `InvoiceAdvancePayment`.
- `internal/domain/statement.go` — janela de datas do import da fatura, matcher linha × fatura.
- `internal/usecase/invoice_advance_usecase.go`, `invoice_usecase.go`, `statement_usecase.go`.
- `internal/infrastructure/repository` — `FindAdvancePaymentsByInvoiceIDs`, `FindUnpaid`; filtros por conjunto.
- `internal/infrastructure/gateway/gemini_vision_gateway.go` — prompts (fatura, extrato, auto).

## Casos de borda & fora de escopo

- Borda: soma de adiantamentos arredondada a centavos; payment_date comparado só pelo dia; pagamento
  datado depois do fechamento da fatura importada cai em `invoice_payment`.
- Fora (questões em aberto do AYD-007): fatura anterior parcialmente paga (sobra rolada pelo banco);
  aviso prévio, na revisão, de extrato com pagamento de fatura já quitada pelo botão.
- Fora: clientes (web/mobile) e agregados do agente que leem `invoice.amount` como "a pagar"
  (`GetCreditCardsSummary`), que seguem no valor bruto.

---

## Plano de implementação

### Abordagem técnica

Conjunto de tipos de pagamento de fatura definido num lugar só (`TypePayment.IsInvoicePayment`) e
reaproveitado pelos filtros. Adiantamento = `Movement` `invoice_advance_payment` (id próprio),
quitação continua `invoice_payment` (um por fatura), então `FindInvoicePaymentByInvoiceID` e
`DeleteByInvoiceID` seguem certos sem mudança. Derivados da `Invoice` montados no usecase por uma
consulta em lote (`FindAdvancePaymentsByInvoiceIDs`). Import da fatura: toda linha de pagamento é
filtrada e classificada por data (janela vencimento da anterior → fechamento da alvo) com desempate
pelo total. Import do extrato: matcher puro no domínio; o confirm executa pela mesma regra do
`pay`/adiantamento via `Invoice.PayInvoice`/`AdvancePayment`.

### Passos
1. Tipo + helper centralizado; trocar os filtros soltos (repo de movimentos, agente, estimate).
2. Derivados da `Invoice` e `advance_payments[]` nas respostas.
3. Endpoints de adiantamento (+ erros tipados) e `pay`/`revert-pay` sobre o restante.
4. Import da fatura: prompt, classificação, checksum, warning.
5. Import do extrato: prompt (`is_card_payment`), `invoice_payment_match`, execução no `/confirm`.
6. Swagger, testes, changelog.

### Arquivos / módulos afetados
- Ver "Modelo de dados / componentes afetados"; `internal/infrastructure/api/{invoice_api,errors_handler}.go`,
  `internal/domain/output/invoice.go`, `docs/swagger.yaml`, `test/acceptance/steps/{invariants,import_steps}.go`.

### Testes (ver `docs/conventions/testing.md`)
- **Aceite (mapeia os critérios acima):** unit tests table-driven por critério (usecase, domínio, handler).
  Cenários Gherkin (`test/acceptance/features`) e a invariante "adiantamento nunca é despesa" no
  `SPEC-008` **ainda não escritos** — a suíte exige Docker.
- **Unit/integração:** `invoice_advance_usecase_test.go`, `statement_advance_payment_test.go`,
  `canonical_realized_test.go`, `domain/{typepayment,invoice_advance,statement_payment_window,statement_invoice_match}_test.go`,
  `api/invoice_api_test.go`.

### Checklist
- [x] `invoice_advance_payment` + helper centralizado, aplicado nos SQLs (movimentos, agente) e no realizado (estimate/dashboard)
- [x] `advance_paid_amount`, `remaining_amount`, `advance_payments[]` nas respostas de `Invoice`
- [x] `POST/DELETE /v2/invoices/{id}/advance-payments` com os erros do AYD
- [x] `pay` e `revert-pay` sobre o restante; revert-pay desfaz só a quitação
- [x] Import da fatura: pagamentos filtrados, `advance_payment`, desempate pelo total, checksum, warning
- [x] Import do extrato: `invoice_payment_match` e execução no `/confirm` (dedup por hash e por adiantamento)
- [x] Swagger atualizado
- [ ] Cenários Gherkin de aceite e invariante de adiantamento no SPEC-008 (requer Docker)
- [ ] Validar os cenários de import de fatura existentes na suíte de aceite (a classificação por data pode mudar o motivo de linhas de pagamento)
