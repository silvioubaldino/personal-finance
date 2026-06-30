---
id: SPEC-001
type: spec
title: "Invoice Import — api (Phases 1–3)"
status: draft
created: 2026-06-30
updated: 2026-06-30
owner: Silvio Ubaldino
parents: [AYD-004@context]
children: []
related: [GLO]
tags: [invoice, statement, import, ai]
superseded_by: null
---

# Spec: Invoice Import — api (Phases 1–3)

> Detalha O QUÊ este repo (api) faz para cumprir o AYD-004 nas Fases 1, 2 e 3.
> Congela ao virar `approved`.

## Objetivo

Generalizar o pipeline de import de documentos financeiros (`/v2/statements`) para suportar
tanto extrato bancário (`Statement`) quanto fatura de cartão de crédito (`Invoice`), sem
quebrar os clientes existentes. Inclui:

- **Fase 1:** `/extract` aceita `source_type`; resposta ganha `document_type`, `confidence`,
  `warnings`; substitui hard-fail `ErrStatementNotAStatement` por `document_type: unknown` +
  warning `low_confidence`.
- **Fase 2:** Prompt dedicado para fatura (`invoice`), parsing de parcelas
  (`installment_number`/`total_installments`), `invoice_meta` (fechamento, vencimento, total).
  Feature label `invoice_extract` para tracking de tokens separado.
- **Fase 3:** Novo endpoint `POST /v2/statements/confirm-invoice` com
  `StatementUseCase.ConfirmInvoice` — cria `Movement`s com `TypePayment=credit_card`,
  `IsPaid=false`, reutiliza `InvoiceUseCase` para resolver/criar fatura e atualizar limite.
  Hash de idempotência escopado por `credit_card_id`.

## Critérios de aceite

```gherkin
Cenário: Extração com source_type=invoice retorna campos de fatura
  Dado um arquivo PDF ou imagem de fatura de cartão
  E source_type="invoice" no form-data
  Quando POST /v2/statements/extract é chamado
  Então a resposta 200 contém document_type="invoice"
  E confidence entre 0.0 e 1.0
  E movements com type_payment="credit_card"
  E invoice_meta quando detectável no documento

Cenário: Extração com source_type=invoice mas IA detecta statement gera warning
  Dado um PDF de extrato bancário
  E source_type="invoice" no form-data
  Quando POST /v2/statements/extract é chamado
  Então a resposta 200 contém document_type="statement"
  E warnings inclui {type: "document_type_mismatch", expected: "invoice", detected: "statement"}
  E nenhum erro HTTP 4xx é retornado

Cenário: Extração sem source_type usa detecção automática
  Dado um arquivo PDF ou imagem
  E source_type ausente no form-data
  Quando POST /v2/statements/extract é chamado
  Então a resposta 200 contém document_type detectado pela IA
  E a resposta é retrocompatível com clientes que não enviam source_type

Cenário: Documento ambíguo retorna document_type=unknown e warning, nunca 4xx
  Dado um documento que a IA não consegue classificar com confiança
  Quando POST /v2/statements/extract é chamado
  Então a resposta é 200 (não 4xx)
  E document_type="unknown"
  E warnings inclui {type: "low_confidence"}
  E movements pode ser vazio

Cenário: confirm-invoice cria movimentos com TypePayment=credit_card e IsPaid=false
  Dado credit_card_id válido
  E movements com date, description e amount
  Quando POST /v2/statements/confirm-invoice é chamado
  Então cada movement é criado com TypePayment=credit_card e IsPaid=false
  E a resposta contém created > 0

Cenário: confirm-invoice deduplica por credit_card_id
  Dado que um movimento já foi importado anteriormente para o mesmo cartão
  (mesmo hash: userID + creditCardID + date + amount + description normalizada)
  Quando POST /v2/statements/confirm-invoice é chamado com o mesmo movimento
  Então o movimento é ignorado (skipped)
  E created=0, skipped=1

Cenário: confirm-invoice com parcelas chama geração de série de installments
  Dado um movimento com installment_number=3 e total_installments=12
  Quando POST /v2/statements/confirm-invoice é chamado
  Então são gerados 10 movimentos (restantes da parcela 3 até 12)
  E cada parcela vai para a fatura do mês correspondente

Cenário: confirm-invoice em fatura já paga retorna erro
  Dado credit_card_id cujo invoice alvo já está pago (IsPaid=true)
  Quando POST /v2/statements/confirm-invoice é chamado
  Então a resposta é 422
  E o body contém o erro ErrInvoiceAlreadyPaid

Cenário: confirm-invoice com cartão inexistente retorna erro
  Dado credit_card_id que não pertence ao usuário autenticado
  Quando POST /v2/statements/confirm-invoice é chamado
  Então a resposta é 404 ou 400

Cenário: Extract com ErrStatementNotAStatement legado vira soft-fail
  Dado que o gateway retorna ErrStatementNotAStatement
  Quando StatementUseCase.Extract é chamado
  Então o resultado é StatementExtractResult com document_type=unknown
  E warnings inclui low_confidence
  E nenhum erro é propagado ao handler
```

## Contratos consumidos/expostos

Contratos definidos em AYD-004@context. Este repo NÃO os redefine.

### Endpoint: `POST /v2/statements/extract` (estendido)
- Novo campo de form-data: `source_type` (opcional, `"statement"` | `"invoice"`)
- Resposta 200 ganha campos aditivos: `document_type`, `confidence`, `warnings[]`, `invoice_meta`
- `ExtractedMovement` ganha `installment_number` e `total_installments` (opcionais)
- Retrocompatível: clientes sem `source_type` continuam recebendo o mesmo comportamento

### Endpoint: `POST /v2/statements/confirm-invoice` (novo)
- Request: `{ credit_card_id, invoice_id?, movements[] }`
- Response: `{ created, skipped, errors[] }` (mesmo shape do `/confirm`)
- Erros: `ErrInvoiceAlreadyPaid` (422), cartão não encontrado (404)

### `POST /v2/statements/classify` — inalterado
### `POST /v2/statements/confirm` — inalterado (caminho statement)

## Modelo de dados / componentes afetados

- `internal/domain/statement.go`:
  - `DocumentType` enum (`statement` | `invoice` | `unknown`)
  - `ExtractWarning` struct
  - `InvoiceMeta` struct
  - `InvoiceConfirmInput` struct
  - `ExtractedMovement` ganha `InstallmentNumber`, `TotalInstallments`
  - `StatementExtractResult` ganha `DocumentType`, `Confidence`, `Warnings`, `InvoiceMeta`
  - `ComputeIdempotencyHash` generalizado: aceita `scopeKey string` (walletID.String() ou creditCardID.String())

- `internal/usecase/statement_usecase.go`:
  - `StatementVisionGateway.ExtractMovements` agora recebe `sourceType string`
  - `StatementInvoiceUseCase` interface (estreita, declarada aqui)
  - `StatementCreditCardRepository` interface (estreita, declarada aqui)
  - `StatementUseCase` ganha `invoiceUseCase` e `creditCardRepo`
  - `Extract` recebe `sourceType string`; soft-fail para `ErrStatementNotAStatement`
  - `ConfirmInvoice` método novo

- `internal/infrastructure/gateway/gemini_vision_gateway.go`:
  - Prompts: `statementExtractionPrompt` (atualizado), `invoiceExtractionPrompt` (novo), `autoDetectionPrompt` (novo)
  - Seleção de prompt por `sourceType`
  - Feature label `invoice_extract` para tokens de fatura
  - Novo shape de resposta JSON: objeto com `document_type`, `confidence`, `invoice_meta`, `movements`

- `internal/infrastructure/api/statement_api.go`:
  - `StatementUsecase` interface ganha `ConfirmInvoice`
  - Handler `ConfirmInvoice` registrado em `POST /v2/statements/confirm-invoice`
  - Handler `Extract` lê campo `source_type` do form-data

- `internal/bootstrap/statement/setup.go`:
  - Injeta `InvoiceUseCase` e `CreditCardRepository` no `StatementUseCase`

## Casos de borda & fora de escopo

- **Borda:** documento ambíguo nunca retorna 4xx — sempre 200 com `document_type=unknown` + warning
- **Borda:** `source_type` ausente = modo auto-detecção (retrocompatível)
- **Borda:** importar em fatura já paga → 422 ao encontrar o primeiro item com invoice paga
- **Borda:** movimento parcelado gera série a partir da parcela `installment_number` até `total_installments`
- **Fora de escopo:** Fase 4 (web/mobile — bifurcação do confirm, UI de parcelas, entrada "Importar fatura")
- **Fora de escopo:** Fase 5 (validação de total, métricas `biz_invoice_imports_total` avançadas)
- **Fora de escopo:** heurísticas estruturais de texto (mencionar vs. depender da IA apenas)
- **Fora de escopo:** `POST /v2/statements/classify` — inalterado, sem mudanças nesta SPEC
