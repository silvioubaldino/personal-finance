# AyD — Análise e Desenho de Import de Fatura de Cartão (Invoice Import)

> **Status:** Proposta para discussão
> **Escopo:** `personal-finance` (backend Go/Cloud Run) · `personal-finance-frontend-v2` (web) · `personal-finance-mobile` (mobile). Documento de referência **cross-camada**: backend implementa, frontends consomem.
> **Objetivo:** Estender o fluxo de import de documentos (hoje só **extrato bancário / statement**) para também importar **fatura de cartão de crédito / invoice**, recebendo formatos heterogêneos de diferentes bancos, com **diferenciação confiável** entre os dois tipos de documento.
> **Contrato:** A §6 (API) e §7 (modelo de dados) são o **contrato canônico** entre backend e frontends. Mudanças nessas seções exigem revisão conjunta.

---

## 1. Sumário executivo

| Tema | Decisão proposta |
|---|---|
| **Como diferenciar statement × invoice** | **Defesa em camadas**, não confiar só na IA nem só no usuário: (1) o cliente envia a **intenção** (`source_type`), porque na UI ele já escolheu "importar fatura do cartão X" vs "importar extrato da conta Y"; (2) a IA **também detecta** e devolve `document_type` + `confidence`; (3) divergência vira **warning não-fatal** para a UI confirmar, em vez do hard-fail atual. |
| **Forma do "tipo"** | **Enum** `document_type` (`statement` / `invoice` / `unknown`), **não** um boolean `is_invoice` — permite crescer (ex.: `receipt`) sem quebrar contrato. |
| **Superfície de API** | **Reusar** `/extract` e `/classify` (parametrizados por tipo); **bifurcar só o confirm**, porque a persistência de fatura diverge estruturalmente da de extrato. Novo endpoint **`POST /v2/statements/confirm-invoice`**. |
| **Persistência da fatura** | **Reusar a `InvoiceUseCase` existente** (`FindOrCreateInvoiceForMovement` + atualização de amount/limite), não reimplementar regra de fatura. Garante consistência com o lançamento manual de cartão. |
| **Prompt da IA** | Um prompt por tipo (statement / invoice), selecionado por `source_type`; e um modo **detecção** quando `source_type` vier ausente. O prompt de invoice extrai **parcelas** e **metadados da fatura** (fechamento, vencimento, total). |
| **Idempotência** | Variante da hash de idempotência escopada por **`credit_card_id`** (em vez de `wallet_id`), pois itens de fatura não pertencem a uma carteira. |
| **Compatibilidade** | 100% retrocompatível: clientes atuais que chamam `/extract` + `/confirm` sem `source_type` continuam funcionando como "statement". |

**TL;DR do fluxo proposto:**

```
                       ┌──────────────────────────────────────────────┐
   upload (PDF/img)    │  POST /v2/statements/extract                 │
   + source_type ─────▶│  - decripta PDF                              │
   (statement|invoice  │  - escolhe prompt por source_type           │
    |ausente=auto)     │  - Vision extrai itens + detecta tipo        │
                       └───────────────┬──────────────────────────────┘
                                       │ resposta inclui:
                                       │  document_type (detectado)
                                       │  confidence
                                       │  warnings[]  (ex.: mismatch)
                                       │  invoice_meta (se invoice)
                                       │  movements[]
                                       ▼
                       ┌──────────────────────────────────────────────┐
                       │  POST /v2/statements/classify (inalterado)   │
                       │  histórico + IA → sugestões de categoria     │
                       └───────────────┬──────────────────────────────┘
                                       │
              document_type=statement  │  document_type=invoice
              ┌────────────────────────┴───────────────────────────┐
              ▼                                                      ▼
   ┌──────────────────────────┐                  ┌──────────────────────────────────┐
   │ POST .../confirm         │                  │ POST .../confirm-invoice          │
   │ wallet_id                │                  │ credit_card_id                    │
   │ → Movement (IsPaid=true) │                  │ → reusa InvoiceUseCase            │
   │ → débito/pix/ted         │                  │ → credit_card movements           │
   │ hash: userID+walletID    │                  │ → invoice amount + limite cartão  │
   └──────────────────────────┘                  │ → parcelas                        │
                                                  │ hash: userID+creditCardID         │
                                                  └──────────────────────────────────┘
```

---

## 2. Estado atual (mapeamento)

O import vive em **`/v2/statements`** (`internal/infrastructure/api/statement_api.go`) e é um pipeline de 3 etapas, **inteiramente voltado para extrato bancário**.

| Etapa | Endpoint | Camada | O que faz hoje |
|---|---|---|---|
| **Extract** | `POST /v2/statements/extract` | `StatementUseCase.Extract` → `GeminiVisionGateway.ExtractMovements` | Recebe PDF/imagem (multipart `file` + opcional `password`), valida tamanho/mime, decripta PDF protegido em memória, envia ao Gemini Vision com prompt **hardcoded de extrato**. Retorna `[]ExtractedMovement`. |
| **Classify** | `POST /v2/statements/classify` | `StatementUseCase.Classify` | Fase 1: lookup de histórico por descrição normalizada (grátis). Fase 2: batch para IA categorizar o que sobrou. Retorna `[]CategorySuggestion`. |
| **Confirm** | `POST /v2/statements/confirm` | `StatementUseCase.Confirm` | Calcula hash de idempotência, deduplica contra hashes existentes, insere `Movement` com `WalletID`, `IsPaid=true`. |

**Contratos atuais (domínio):** `internal/domain/statement.go`

```go
type ExtractedMovement struct {
    Date          string      `json:"date"`           // "YYYY-MM-DD"
    Description   string      `json:"description"`
    Amount        float64     `json:"amount"`         // + entrada / - saída
    TypePayment   TypePayment `json:"type_payment,omitempty"`
    RecurrenceID  *uuid.UUID  `json:"recurrence_id,omitempty"`
    CategoryID    *uuid.UUID  `json:"category_id,omitempty"`
    SubCategoryID *uuid.UUID  `json:"sub_category_id,omitempty"`
}
```

**Prompt atual** (`gemini_vision_gateway.go`): instruções estritas anti-prompt-injection, extrai `date/description/amount/type_payment` ∈ {pix, ted, doc, debit_card}. **Se o documento não for extrato, retorna `{"error":"not_a_statement"}`** → mapeado para `domain.ErrStatementNotAStatement`. **Uma fatura cairia aqui e seria rejeitada.**

**Persistência de fatura já existente (caminho manual):** `internal/usecase/invoice_usecase.go` + `movement_usecase.go:135-208`. Um lançamento de cartão hoje passa por `handleCreditCardMovement` → `getInvoice` → `FindOrCreateInvoiceForMovement`, atualizando `invoice.Amount` e o limite do cartão (`UpdateLimitDelta`), com suporte a parcelas (`GenerateInstallmentMovements`). **É essa lógica que o import de fatura deve reusar.**

---

## 3. Análise do gap — por que invoice não encaixa no fluxo atual

A diferença **não é só o prompt**. O `Confirm` do statement insere movimentos de carteira; uma fatura precisa de um caminho de persistência distinto.

| Aspecto | Statement (`Confirm` hoje) | Invoice (necessário) |
|---|---|---|
| Vínculo principal | `WalletID` | `CreditCardInfo{CreditCardID, InvoiceID}` via `FindOrCreateInvoiceForMovement` |
| `IsPaid` | `true` (já caiu na conta) | `false` (paga-se a **fatura**, não o item) |
| `type_payment` | `debit_card` / `pix` / `ted` / `doc` | `credit_card` |
| Efeitos colaterais | nenhum além do insert | `invoiceRepo.UpdateAmount` + `creditCardRepo.UpdateLimitDelta` |
| Parcelas | inexistente | `PARCELA 03/12` → `InstallmentNumber` / `TotalInstallments` |
| Hash de idempotência | `userID + walletID + date + amount + desc` | escopo natural é `credit_card_id`, não wallet |
| Metadados do documento | nenhum | fechamento, vencimento, total da fatura (para casar a invoice certa e validar o total) |
| Sinal do amount | `+` crédito / `-` débito | quase tudo é despesa; estornos/pagamentos invertem |

**Conclusão:** extract dá para generalizar via prompt; **o confirm precisa de um caminho próprio** (`confirm-invoice`) que reaproveita a `InvoiceUseCase`.

---

## 4. Princípios de desenho

1. **Diferenciação em camadas, não aposta única.** Intenção do usuário **e** detecção da IA, com reconciliação explícita. Nenhuma das duas sozinha é confiável com formatos heterogêneos de bancos.
2. **Falhar suave, não duro.** Documento ambíguo vira `unknown` + `warning` para a UI decidir — nunca um 4xx silencioso que trava o usuário.
3. **Reuso > reimplementação.** A regra de fatura já existe e é testada; o import a **orquestra**, não a duplica.
4. **Contrato estável e agnóstico.** Enum versionável, campos opcionais aditivos, erros com `type` legível por máquina. Frontends programam contra a §6, não contra detalhes internos do Go.
5. **Retrocompatibilidade.** Tudo que existe hoje continua funcionando sem `source_type`.
6. **Incremental.** Entregar em fases; a Fase 1 não muda nada de quebra.

---

## 5. Estratégia de diferenciação statement × invoice (núcleo do desenho)

A pergunta central do projeto: *"e se a IA não conseguir diferenciar?"*. Resposta: **não depender de uma fonte só.** Três sinais combinados:

### 5.1 As três fontes de verdade

| Fonte | Confiança | Papel |
|---|---|---|
| **Intenção do cliente** (`source_type` no request) | Alta na maioria dos casos | Na UI o usuário já escolheu o contexto (tela "Importar fatura do cartão X"). É de graça e quase sempre correta. |
| **Detecção da IA** (`document_type` + `confidence` na resposta) | Média/alta, varia por banco | Rede de segurança: pega o caso em que o usuário sobe o arquivo errado. |
| **Heurísticas estruturais** (opcional, barato) | Média | Sinais fortes no texto: presença de "fatura/vencimento/limite/parcela" vs "saldo/extrato/agência". Reforça a detecção sem custo de LLM. |

### 5.2 Matriz de reconciliação

| `source_type` (cliente) | `document_type` (IA) | Resultado | Ação |
|---|---|---|---|
| `invoice` | `invoice` | ✅ acordo | segue para `confirm-invoice` |
| `statement` | `statement` | ✅ acordo | segue para `confirm` |
| `invoice` | `statement` (ou vice-versa) | ⚠️ **mismatch** | retorna `200` com `warnings: [{type: "document_type_mismatch", detected, expected}]`; UI confirma com o usuário antes do confirm |
| qualquer | `unknown` / `confidence` baixa | ⚠️ incerto | retorna `200` com warning `low_confidence`; UI pede confirmação |
| **ausente** (auto) | `invoice`/`statement` | ℹ️ IA decide | usa o detectado; UI mostra "detectamos uma fatura, confirma?" |
| **ausente** (auto) | `unknown` | ⚠️ incerto | UI pergunta o tipo explicitamente |

> **Regra de ouro:** a extração **nunca falha** por ambiguidade de tipo. Ela sempre retorna `200` com o que conseguiu extrair + os warnings. Quem decide o caminho de `confirm` é o cliente, com base nos sinais. Isso substitui o atual `ErrStatementNotAStatement` (hard-fail) por um `document_type: "unknown"` informativo.

### 5.3 Limiar de confiança

Reusar a constante já existente `ClassificationConfidenceThreshold = 0.6` (`statement_usecase.go:17`) como referência; `confidence < 0.6` na detecção de tipo dispara o warning `low_confidence`.

---

## 6. Contrato de API (referência canônica para todos os clientes)

> Esta seção é **a fonte da verdade** para web e mobile. Campos marcados *(novo)* ainda não existem; o restante é o comportamento atual mantido.

### 6.1 Enum `document_type`

```
"statement"  — extrato bancário (conta corrente/poupança)
"invoice"    — fatura de cartão de crédito
"unknown"    — IA não conseguiu determinar com confiança
```

### 6.2 `POST /v2/statements/extract`

**Request** — `multipart/form-data`:

| Campo | Tipo | Obrig. | Descrição |
|---|---|---|---|
| `file` | binário | sim | PDF, JPEG ou PNG. Máx **10 MB**. |
| `password` | string | não | Senha de abertura para PDF protegido. |
| `source_type` *(novo)* | string | não | `statement` \| `invoice`. **Ausente = modo auto** (IA decide). Retrocompatível: ausência ⇒ comportamento de hoje. |

**Response `200`** *(campos novos são aditivos)*:

```jsonc
{
  "document_type": "invoice",          // (novo) tipo detectado pela IA
  "confidence": 0.94,                  // (novo) 0.0–1.0 da detecção de tipo
  "warnings": [                         // (novo) não-fatais; [] quando tudo ok
    { "type": "document_type_mismatch", "expected": "statement", "detected": "invoice" }
  ],
  "invoice_meta": {                     // (novo) presente só quando invoice
    "closing_date": "2026-06-03",      // fechamento, se legível
    "due_date": "2026-06-10",          // vencimento, se legível
    "total_amount": -3450.27           // total da fatura, se legível
  },
  "movements": [
    {
      "date": "2026-05-12",
      "description": "MERCADO LIVRE PARCELA 03/12",
      "amount": -120.00,
      "type_payment": "credit_card",   // invoice: sempre credit_card
      "installment_number": 3,         // (novo) só invoice, se houver parcela
      "total_installments": 12         // (novo) só invoice, se houver parcela
    }
  ],
  "errors": ["movement #7: missing date"]   // erros de extração por item (já existe)
}
```

**Erros (typed):**

| Situação | HTTP | `error.type` |
|---|---|---|
| Arquivo > 10MB | 413/422 | — (`ErrStatementFileTooLarge`) |
| Mime inválido | 400 | — |
| PDF protegido sem senha | 422 | `statement_password_required` |
| Senha incorreta | 422 | `statement_wrong_password` |
| ~~Documento não é extrato~~ | ~~422~~ | **removido** → vira `document_type: "unknown"` + warning na resposta `200` |

### 6.3 `POST /v2/statements/classify` — **inalterado**

Recebe `{ "movements": [...] }`, retorna `{ "suggestions": [{description, category_id, subcategory_id, confidence, source}] }`. Funciona igual para itens de extrato e de fatura (categoriza por descrição). `source` ∈ `"history"` | `"ai"`.

### 6.4 `POST /v2/statements/confirm` — **inalterado** (caminho statement)

```jsonc
// request
{ "wallet_id": "uuid", "movements": [ ExtractedMovement, ... ] }
// response
{ "created": 12, "skipped": 3, "errors": ["Could not save 'X': duplicate entry"] }
```

### 6.5 `POST /v2/statements/confirm-invoice` *(novo)* — caminho invoice

**Request:**

```jsonc
{
  "credit_card_id": "uuid",            // obrigatório — substitui wallet_id
  "invoice_id": "uuid|null",           // opcional: força a fatura alvo; senão é resolvida pela data
  "movements": [
    {
      "date": "2026-05-12",
      "description": "MERCADO LIVRE PARCELA 03/12",
      "amount": -120.00,
      "category_id": "uuid|null",
      "sub_category_id": "uuid|null",
      "installment_number": 3,         // opcional
      "total_installments": 12         // opcional
    }
  ]
}
```

**Response** (mesmo shape do confirm de statement):

```jsonc
{ "created": 18, "skipped": 2, "errors": ["..."] }
```

**Semântica (backend):** para cada item, monta um `Movement` com `TypePayment = credit_card`, `IsPaid = false` e `CreditCardInfo{CreditCardID, InvoiceID}`, então delega à `InvoiceUseCase` para resolver/criar a fatura por data (`FindOrCreateInvoiceForMovement`), somar em `invoice.Amount` e no limite do cartão (`UpdateLimitDelta`). Itens parcelados geram a série via `GenerateInstallmentMovements`. Deduplicação por hash de idempotência escopada por `credit_card_id`.

**Erros adicionais:**

| Situação | HTTP | Observação |
|---|---|---|
| `credit_card_id` ausente/inexistente | 400/404 | reusa mapeamento de cartão |
| Cartão sem carteira default e item sem wallet | 400 | `ErrCreditCardNoDefaultWallet` (já existe) |
| Estouro de limite do cartão | 403 | `ErrCreditCardLimitReached` (já existe) |
| Fatura alvo já paga | 422 | não permite import em fatura fechada/paga |

### 6.6 Contrato de erro (formato global — já existente)

```jsonc
{ "error": { "code": 422, "message": "texto legível", "type": "machine_readable_opcional" } }
```
Clientes devem ramificar por **`error.type`** quando presente (ex.: `statement_password_required`), e cair no `code`/`message` quando ausente. Ver `errors_handler.go`.

---

## 7. Modelo de dados e domínio (mudanças no backend)

### 7.1 `ExtractedMovement` (aditivo)

```go
type ExtractedMovement struct {
    // ... campos atuais ...
    InstallmentNumber *int `json:"installment_number,omitempty"` // (novo)
    TotalInstallments *int `json:"total_installments,omitempty"` // (novo)
}
```

### 7.2 Novos tipos de domínio

```go
type DocumentType string
const (
    DocStatement DocumentType = "statement"
    DocInvoice   DocumentType = "invoice"
    DocUnknown   DocumentType = "unknown"
)

type ExtractWarning struct {
    Type     string `json:"type"`               // "document_type_mismatch" | "low_confidence"
    Expected string `json:"expected,omitempty"`
    Detected string `json:"detected,omitempty"`
}

type InvoiceMeta struct {
    ClosingDate *string  `json:"closing_date,omitempty"` // "YYYY-MM-DD"
    DueDate     *string  `json:"due_date,omitempty"`
    TotalAmount *float64 `json:"total_amount,omitempty"`
}

// StatementExtractResult ganha (aditivo):
//   DocumentType DocumentType
//   Confidence   float64
//   Warnings     []ExtractWarning
//   InvoiceMeta  *InvoiceMeta

type InvoiceConfirmInput struct {
    CreditCardID uuid.UUID           `json:"credit_card_id"`
    InvoiceID    *uuid.UUID          `json:"invoice_id,omitempty"`
    Movements    []ExtractedMovement `json:"movements"`
}
```

### 7.3 Idempotência por cartão

Generalizar `ComputeIdempotencyHash` (`statement.go:75`) para aceitar o escopo. Hoje: `userID|walletID|date|amount|desc`. Para fatura: `userID|creditCardID|date|amount|desc`. Mesma normalização de descrição.

### 7.4 Reuso, não reescrita

`confirm-invoice` orquestra a `InvoiceUseCase` já existente:
- `FindOrCreateInvoiceForMovement(ctx, invoiceID, creditCardID, date)` — resolve/cria a fatura pela data (`invoice_usecase.go:64`).
- `UpdateAmount` + `creditCardRepo.UpdateLimitDelta` — efeitos colaterais (mesma lógica de `movement_usecase.go:getInvoice`).
- `GenerateInstallmentMovements` (`movement.go:132`) — explode parcelas.

> **Decisão de design:** o `StatementUseCase` ganha um método `ConfirmInvoice` que injeta a `InvoiceUseCase` (e `creditCardRepo`). Wiring em `internal/bootstrap/statement/setup.go`, pegando dependências do registry como os demais features clean-arch.

---

## 8. Prompts da IA (gateway de visão)

Em vez de um prompt genérico tentando adivinhar tudo, **seleção por `source_type`**:

| `source_type` | Prompt usado | Saída |
|---|---|---|
| `statement` | prompt de extrato (atual) | itens + `document_type` confirmado |
| `invoice` | **prompt de fatura** *(novo)* | itens (sempre `credit_card`) + parcelas + `invoice_meta` |
| ausente (auto) | **prompt de detecção** *(novo)* | primeiro classifica o tipo, depois extrai conforme o tipo |

**Prompt de fatura — pontos obrigatórios:**
- Manter o bloco anti-prompt-injection atual (documento é **dado passivo**).
- Extrair `date / description / amount` de cada lançamento; `amount` negativo para compras, positivo para estornos/pagamentos.
- Detectar **parcelas** no texto (`"03/12"`, `"PARC 3/12"`, `"PARCELA 03 DE 12"`) → `installment_number` / `total_installments`.
- Extrair **metadados** da fatura: fechamento, vencimento, total (`invoice_meta`), quando legíveis.
- Sempre retornar `type_payment: "credit_card"`.
- Em vez de `{"error":"not_a_statement"}`, retornar `document_type: "unknown"` quando não parecer fatura nem extrato.

**Prompt de detecção (auto):** retorna `{document_type, confidence}` + os movimentos no formato do tipo detectado, num único call (economiza uma ida ao modelo).

> **Token usage:** manter `recordTokenUsage(ctx, "statement_extract", ...)`; adicionar feature label `"invoice_extract"` para separar custo nas métricas de negócio (`biz_ai_tokens_total`).

---

## 9. Guia de integração para frontends

> Web (`lib/api/fetcher.ts`) e mobile (`src/lib/api/fetcher.ts`) — passo a passo de UI.

1. **Tela de import** já sabe o contexto. Enviar `source_type`:
   - Fluxo "Importar fatura" (a partir de um cartão) → `source_type=invoice`.
   - Fluxo "Importar extrato" (a partir de uma carteira) → `source_type=statement`.
   - Fluxo genérico "Importar documento" → omitir `source_type` (modo auto).
2. **Chamar `/extract`.** Tratar:
   - `error.type === "statement_password_required"` → pedir senha e reenviar com `password`.
   - `error.type === "statement_wrong_password"` → avisar senha incorreta.
3. **Ler `warnings` da resposta `200`:**
   - `document_type_mismatch` → modal: *"Você selecionou X, mas isto parece ser Y. Importar como Y?"*. A escolha do usuário decide qual `confirm` chamar.
   - `low_confidence` / `document_type === "unknown"` → pedir ao usuário que escolha o tipo.
4. **Chamar `/classify`** com os `movements` (igual para os dois tipos).
5. **Bifurcar o confirm pelo tipo efetivo:**
   - `statement` → `POST /confirm` com `wallet_id`.
   - `invoice` → `POST /confirm-invoice` com `credit_card_id` (e `invoice_id` se a UI já tiver a fatura aberta em mãos).
6. **Render do resultado:** `{created, skipped, errors}` é idêntico nos dois caminhos.
7. **Parcelas (invoice):** exibir `installment_number/total_installments` quando presentes; o backend cria a série completa, então a UI deve avisar "compra parcelada gera N lançamentos futuros".

> **Correlação/observabilidade:** alinhado ao AyD de monitoramento, enviar `X-Request-ID` pelo `fetcher.ts` também nessas chamadas para rastrear o import ponta-a-ponta.

---

## 10. Plano de implementação faseado

| Fase | Entregas | Quebra contrato? | Esforço |
|---|---|---|---|
| **1 — Detecção sem quebra** | `/extract` aceita `source_type`; resposta ganha `document_type`+`confidence`+`warnings`; trocar `not_a_statement` (hard-fail) por `unknown`+warning. Prompt de detecção. | Não (aditivo) | S |
| **2 — Extração de fatura** | Prompt de fatura dedicado; parsing de parcelas (`installment_number/total_installments`); `invoice_meta`. Feature label de tokens. | Não (aditivo) | M |
| **3 — Persistência de fatura** | `InvoiceConfirmInput` + `StatementUseCase.ConfirmInvoice` reusando `InvoiceUseCase`; endpoint `POST /v2/statements/confirm-invoice`; hash de idempotência por `credit_card_id`; wiring no bootstrap; testes (usecase + handler). | Não (novo endpoint) | M |
| **4 — Frontends** | Web e mobile: `source_type` no fluxo, tratamento de `warnings`, bifurcação do confirm, UI de parcelas. | Consome contrato | M |
| **5 — Endurecimento** | Validação do total (`invoice_meta.total_amount` × soma dos itens) como warning; métricas de negócio (`biz_invoice_imports_total`, itens importados); telemetria de mismatch. | Não | S |

---

## 11. Riscos e mitigações

| Risco | Mitigação |
|---|---|
| **IA classifica tipo errado** (bancos heterogêneos) | Defesa em camadas (§5): intenção do cliente + detecção + warning de mismatch; nunca decide sozinha o `confirm`. |
| **Parcelas mal interpretadas** (formatos variados de "x/y") | Prompt com exemplos múltiplos; `installment_*` são opcionais — na dúvida, importa como lançamento simples (não inventa série). UI permite revisar antes do confirm. |
| **Import duplicado** ao reenviar a mesma fatura | Hash de idempotência por `credit_card_id` + dedup contra hashes existentes (mesma estratégia já provada no statement). |
| **Total da fatura não bate** com soma dos itens (OCR perdeu linha) | Fase 5: comparar `invoice_meta.total_amount` com a soma; divergência vira **warning**, não bloqueio — usuário decide. |
| **Importar em fatura já paga/fechada** | `ConfirmInvoice` valida status da invoice (reusa `ErrInvoiceCannotModify`/`ErrInvoiceAlreadyPaid`). |
| **Estouro de limite** ao importar fatura grande | Reusa `validateCreditLimit` da `InvoiceUseCase`; retorna `ErrCreditCardLimitReached` (403). |
| **Custo de LLM** sobe com dois prompts | Modo auto faz detecção+extração num único call; tokens medidos por feature (`invoice_extract`) para acompanhar custo. |
| **Frontends antigos** quebrarem | Tudo aditivo; ausência de `source_type` = comportamento atual (statement). |

---

## 12. Decisões em aberto (precisam de confirmação)

1. **Endpoint do confirm de fatura:** `POST /v2/statements/confirm-invoice` (proposto, mantém coesão no mesmo grupo) **vs** `POST /v2/invoices/import` (mais "RESTful" sob o recurso fatura). Recomendo o primeiro pela proximidade com o pipeline `extract/classify`.
2. **Resolução da fatura alvo:** confiar na data de cada item (`FindOrCreateInvoiceForMovement` resolve a fatura por período) **vs** exigir `invoice_id` único no request. Recomendo **por data** (cobre faturas que cruzam o fechamento), com `invoice_id` opcional como override.
3. **Validação de total** (`invoice_meta.total_amount`): warning informativo (recomendado) **vs** bloqueio do confirm. Recomendo warning na Fase 5.
4. **Modo auto como default?** Se `source_type` ausente deve assumir `statement` (retrocompatível) **vs** rodar detecção. Recomendo: ausente ⇒ detecção, mas tratando `unknown` como statement no `confirm` legado para não quebrar clientes atuais.
5. **Sinal do `amount` na fatura:** padronizar despesa como **negativo** (consistente com o resto do app) — confirmar com frontends que já assumem essa convenção.

---

### Apêndice A — Glossário

| Termo | Significado |
|---|---|
| **Statement / Extrato** | Movimentações de uma conta (corrente/poupança), com saldo corrente. Vincula-se a uma **carteira** (`wallet`). |
| **Invoice / Fatura** | Conjunto de compras de um **cartão de crédito** num período (fechamento→vencimento). Vincula-se a um **cartão** e gera lançamentos `credit_card` não pagos até o pagamento da fatura. |
| **`document_type`** | Enum que diferencia o documento importado (`statement`/`invoice`/`unknown`). |
| **`source_type`** | Intenção declarada pelo cliente no `/extract` (qual tipo ele *acha* que está enviando). |
| **Parcela** | Compra dividida (`x/y`); o backend materializa a série de `y` lançamentos via `GenerateInstallmentMovements`. |
| **Idempotência** | Hash determinístico por documento+escopo que impede import duplicado; escopo = wallet (statement) ou cartão (invoice). |

### Apêndice B — Arquivos impactados (referência de implementação)

| Camada | Arquivo | Mudança |
|---|---|---|
| Domínio | `internal/domain/statement.go` | `DocumentType`, `ExtractWarning`, `InvoiceMeta`, `InvoiceConfirmInput`, campos de parcela, hash por escopo |
| Usecase | `internal/usecase/statement_usecase.go` | método `ConfirmInvoice`, injeção de `InvoiceUseCase`/`creditCardRepo` |
| Gateway | `internal/infrastructure/gateway/gemini_vision_gateway.go` | prompts de fatura/detecção, seleção por `source_type`, retorno de tipo/meta |
| API | `internal/infrastructure/api/statement_api.go` | ler `source_type` no `/extract`; handler `ConfirmInvoice` |
| Bootstrap | `internal/bootstrap/statement/setup.go` | wiring da `InvoiceUseCase` + `creditCardRepo` no `StatementUseCase` |
| Reuso | `internal/usecase/invoice_usecase.go`, `internal/usecase/movement.go` | **sem alteração** — apenas consumidos |
