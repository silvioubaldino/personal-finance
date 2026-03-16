# Prompt 08

> Now we want a new feature that provides the user the option to upload a file in PDF or image of the bank statement from the month or the week. The system must have the capability to read, understand and parse this data to our model and save it.

---

## Core Decision: Gemini Multimodal with Structured Output (Not OCR + Parser)

You're already on Vertex AI. Gemini 2.5 Flash is **natively multimodal** — it reads PDFs and images directly, no pre-processing OCR step. Combined with **Structured Output** (`response_schema`), the model is forced to emit valid JSON matching your schema.

| Approach | Why Not |
|----------|---------|
| Tesseract OCR + regex parser | Breaks on every bank's layout. Nubank ≠ Itaú ≠ Bradesco ≠ C6 ≠ Inter. Maintenance nightmare. |
| Google Document AI Form Parser | Purpose-built for forms, but $1.50/1000 pages + custom extractor training. Overkill for MVP. |
| OCR → raw text → LLM | Two steps where one works. Gemini reads the PDF directly with better layout understanding. |
| **Gemini Vision + response_schema** | One API call. Handles any layout. Forced JSON output. Same Vertex AI + ZDR you already have. ✅ |

**Cost:** ~258 tokens per image/page (fixed) + ~500-1500 output tokens per page. A 3-page statement ≈ $0.001. Negligible.

---

## The Critical Principle: Extract ≠ Import

**Never auto-save extracted data.** LLMs confuse `1.234,56` with `1,234.56`, miss negative signs, hallucinate a row that isn't there. One wrong amount destroys trust in a finance app.

```
┌────────────────────────────────────────────────────────────────────┐
│                    THREE-STAGE FLOW (mandatory)                    │
└────────────────────────────────────────────────────────────────────┘

  STAGE 1: EXTRACT                STAGE 2: REVIEW              STAGE 3: IMPORT
  (Backend + Gemini)              (Frontend + User)            (Backend only)
  ────────────────────            ────────────────────          ────────────────────
  PDF/image → LLM                 User sees parsed list        Validated movements
  → raw ExtractedMovement[]       Edits amounts/dates           → MovementUseCase
  → dedup check                   Picks category per row        (existing code)
  → confidence scoring            Unchecks duplicates           → batch insert
  → NO DB WRITE                   Confirms wallet               → update wallet balance
                                  → POST /confirm
```

The extraction result is **ephemeral** — stored in a staging table with a TTL, not in `movements`. Only user confirmation promotes it to real data.

---

## Two Endpoints

```
POST /statements/extract
  Content-Type: multipart/form-data
  Body: file (PDF or image, max 10MB), wallet_id

  Response: {
    "extraction_id": "uuid",
    "wallet_id": "uuid",
    "period_detected": { "from": "2026-03-01", "to": "2026-03-31" },
    "items": [
      {
        "tmp_id": "uuid",
        "date": "2026-03-05",
        "description": "PIX RECEBIDO MARIA S",   ← scrubbed (see LGPD section)
        "amount": 1500.00,
        "direction": "in",
        "suggested_category_id": "uuid",
        "suggested_category_name": "Salário",
        "confidence": "high",
        "duplicate_of": null                      ← or existing movement_id
      },
      {
        "tmp_id": "uuid",
        "date": "2026-03-07",
        "description": "IFOOD",
        "amount": -45.90,
        "direction": "out",
        "suggested_category_id": "uuid",
        "suggested_category_name": "Alimentação",
        "confidence": "high",
        "duplicate_of": "existing-movement-uuid"  ← flagged, unchecked by default
      }
    ],
    "warnings": [
      "3 items com valor abaixo de R$5,00 — confira se não são taxas",
      "Período se sobrepõe com 8 movimentos já cadastrados"
    ]
  }

─────────────────────────────────────────────────────────────────

POST /statements/{extraction_id}/confirm
  Body: {
    "items": [
      { "tmp_id": "uuid", "date": "...", "amount": ..., "category_id": "...", "include": true },
      { "tmp_id": "uuid", "include": false },   ← user unchecked duplicate
      ...
    ]
  }

  Response: {
    "imported": 23,
    "skipped": 2,
    "movement_ids": ["uuid", "uuid", ...]
  }
```

**Why `extraction_id` + staging table, not stateless:** Prevents the frontend from forging movements that were never extracted. Confirm only accepts `tmp_id`s that exist in the staged extraction. Also lets the user close the browser and come back.

---

## Gemini Structured Output Schema

This is the contract with the LLM. The model **cannot** return anything that doesn't match.

```go
// internal/infrastructure/gateway/statement_extractor_gateway.go

// What Gemini MUST return (enforced by response_schema)
type extractionResponse struct {
    PeriodFrom  string              `json:"period_from"`  // YYYY-MM-DD or ""
    PeriodTo    string              `json:"period_to"`
    BankName    string              `json:"bank_name"`    // best guess, may be ""
    Items       []extractedItem     `json:"items"`
}

type extractedItem struct {
    Date        string  `json:"date"`         // YYYY-MM-DD
    Description string  `json:"description"`  // raw merchant/counterparty text
    Amount      float64 `json:"amount"`       // ALWAYS positive
    Direction   string  `json:"direction"`    // "in" | "out"  ← see note below
    Confidence  string  `json:"confidence"`   // "high" | "medium" | "low"
}
```

**Design note on `amount` + `direction`:** Gemini often flips signs when the statement uses Brazilian formatting (comma decimal, `D`/`C` markers, red text). Asking for **positive amount + explicit direction enum** is far more reliable than asking for signed floats. You convert to your `Movement.Amount` convention (negative = expense) in the use case layer.

### The Extraction Prompt

```go
const extractionPrompt = `Você é um parser de extratos bancários brasileiros.

Extraia TODAS as transações do documento. Para cada uma retorne:
- date: formato YYYY-MM-DD. Se o extrato mostra só DD/MM, use o ano do
  período do extrato. Se ambíguo, confidence = "low".
- description: texto do lançamento COMO ESTÁ, sem interpretar.
  Máximo 60 caracteres. NÃO inclua CPF, números de conta, ou nomes
  completos de pessoas físicas — se aparecer nome, use só o primeiro nome.
- amount: valor SEMPRE POSITIVO (sem sinal). Formato brasileiro usa
  vírgula decimal: "1.234,56" = 1234.56.
- direction: "in" se é crédito/entrada (PIX recebido, depósito,
  transferência recebida, salário). "out" se é débito/saída (compra,
  PIX enviado, pagamento, saque, tarifa).
- confidence: "high" se data+valor+direção estão claros. "medium" se
  teve que inferir algo. "low" se o texto está cortado ou ilegível.

IGNORE:
- Linhas de saldo ("SALDO", "SALDO ANTERIOR", "SALDO FINAL")
- Linhas de total/subtotal
- Cabeçalhos e rodapés

Se não conseguir ler o documento, retorne items vazio.`
```

### The API Call

```go
func (g *GeminiStatementExtractor) Extract(
    ctx context.Context,
    fileBytes []byte,
    mimeType string, // "application/pdf" or "image/jpeg" etc.
) (*domain.StatementExtraction, error) {

    resp, err := g.client.Models.GenerateContent(ctx,
        "gemini-2.5-flash",
        []*genai.Content{{
            Role: "user",
            Parts: []*genai.Part{
                {Text: extractionPrompt},
                {InlineData: &genai.Blob{MIMEType: mimeType, Data: fileBytes}},
            },
        }},
        &genai.GenerateContentConfig{
            ResponseMIMEType: "application/json",
            ResponseSchema:   extractionResponseSchema, // generated from struct
            Temperature:      ptr(float32(0.1)),        // near-deterministic
        },
    )
    // ... unmarshal, map to domain
}
```

`Temperature: 0.1` — extraction is not creative. Lower temp = more consistent parsing.

---

## Post-Extraction Pipeline (Use Case Layer)

The LLM gives you raw extractions. The use case enriches them before showing the user:

```go
// internal/usecase/statement_usecase.go

func (u *StatementUseCase) Extract(
    ctx context.Context,
    userID string,
    walletID uuid.UUID,
    file []byte,
    mimeType string,
) (*domain.StatementExtractionResult, error) {

    // 0. Validate wallet belongs to user
    wallet, err := u.walletRepo.GetByID(ctx, walletID, userID)
    if err != nil { return nil, err }

    // 1. LLM extraction (gateway)
    raw, err := u.extractorGateway.Extract(ctx, file, mimeType)
    if err != nil { return nil, err }

    // 2. Sign normalization: (amount, direction) → signed amount
    items := normalizeAmounts(raw.Items)

    // 3. PII scrub (LGPD — strip what LLM might have missed)
    items = scrubDescriptions(items)

    // 4. Dedup check against existing movements
    existing, _ := u.movementRepo.ListByWalletAndPeriod(
        ctx, walletID, raw.PeriodFrom, raw.PeriodTo)
    items = flagDuplicates(items, existing)

    // 5. Category suggestion (NOT LLM — cheap heuristic)
    userCategories, _ := u.categoryRepo.ListByUser(ctx, userID)
    items = suggestCategories(items, userCategories)

    // 6. Sanity warnings
    warnings := validate(items, raw)

    // 7. Stage (NOT in movements table)
    extraction := &domain.StatementExtraction{
        ID:        uuid.New(),
        UserID:    userID,
        WalletID:  walletID,
        Items:     items,
        Warnings:  warnings,
        ExpiresAt: time.Now().Add(24 * time.Hour),
    }
    u.stagingRepo.Save(ctx, extraction)

    return toResult(extraction), nil
}
```

### Dedup Heuristic

```go
// Same date + amount within R$0.01 + description similarity > 0.7 → duplicate
func flagDuplicates(extracted []Item, existing []domain.Movement) []Item {
    for i := range extracted {
        for _, ex := range existing {
            sameDate   := extracted[i].Date.Equal(*ex.Date)
            sameAmount := math.Abs(extracted[i].Amount - ex.Amount) < 0.01
            similar    := trigramSimilarity(extracted[i].Description, ex.Description) > 0.7

            if sameDate && sameAmount && similar {
                extracted[i].DuplicateOf = ex.ID
                extracted[i].IncludeByDefault = false  // unchecked in UI
                break
            }
        }
    }
    return extracted
}
```

### Category Suggestion (Zero-Cost, No LLM)

Don't burn another LLM call for this. A keyword map handles 80% of Brazilian transactions:

```go
var categoryKeywords = map[string][]string{
    "Alimentação":  {"IFOOD", "RAPPI", "UBER EATS", "REST", "LANCH", "PADARIA", "MERCADO", "SUPERM"},
    "Transporte":   {"UBER", "99", "POSTO", "SHELL", "IPIRANGA", "ESTACION", "METRO", "ONIBUS"},
    "Moradia":      {"ALUGUEL", "CONDOMINIO", "ENEL", "CEMIG", "SABESP", "COMGAS", "VIVO", "CLARO", "TIM"},
    "Saúde":        {"FARMACIA", "DROGA", "DROGASIL", "HOSPITAL", "CLINICA", "UNIMED", "AMIL"},
    "Lazer":        {"NETFLIX", "SPOTIFY", "CINEMA", "INGRESSO", "STEAM"},
    "Salário":      {"SALARIO", "PAGAMENTO", "FOLHA", "PROVENTOS"},
}

func suggestCategory(desc string, userCats []domain.Category) *uuid.UUID {
    upper := strings.ToUpper(desc)
    for catName, keywords := range categoryKeywords {
        for _, kw := range keywords {
            if strings.Contains(upper, kw) {
                if cat := findByName(userCats, catName); cat != nil {
                    return cat.ID
                }
            }
        }
    }
    return nil // user picks manually
}
```

**Later optimization:** Learn from user's past movements — "last time description contained 'PAG*JOAO' user categorized as Moradia". That's a simple `SELECT ... WHERE description ILIKE ... ORDER BY date DESC LIMIT 1`. Still no LLM.

### Sanity Warnings

```go
func validate(items []Item, raw *Extraction) []string {
    var warnings []string

    if len(items) == 0 {
        warnings = append(warnings, "Nenhuma transação encontrada. Documento ilegível?")
    }
    if len(items) > 200 {
        warnings = append(warnings, "Mais de 200 transações — confira o período, pode ter páginas duplicadas")
    }

    // Date out of range = probable parse error
    for _, it := range items {
        if it.Date.Before(raw.PeriodFrom.AddDate(0, 0, -5)) ||
           it.Date.After(raw.PeriodTo.AddDate(0, 0, 5)) {
            warnings = append(warnings, fmt.Sprintf(
                "Transação em %s está fora do período do extrato — confira a data",
                it.Date.Format("02/01")))
        }
    }

    // Suspiciously round numbers cluster = possible hallucination
    roundCount := 0
    for _, it := range items {
        if math.Mod(it.Amount, 10) == 0 { roundCount++ }
    }
    if float64(roundCount)/float64(len(items)) > 0.8 && len(items) > 5 {
        warnings = append(warnings, "Muitos valores redondos — confira se os centavos foram lidos corretamente")
    }

    return warnings
}
```

---

## LGPD: This Is More Sensitive Than Chat

Bank statements contain **direct identifiers** the agent chat never sees:

| Data in Statement | Risk | Mitigation |
|-------------------|------|------------|
| Account holder full name | Direct PII | Prompt instructs LLM to ignore; backend regex-strips anyway |
| Account number / agência | Financial identifier | Never extracted (not in response schema) |
| CPF (some banks show it) | Highest sensitivity | Regex-strip in `scrubDescriptions()` — `\d{3}\.\d{3}\.\d{3}-\d{2}` |
| Counterparty full names (PIX) | Third-party PII | Prompt: "primeiro nome apenas". Backend: truncate at first space if >1 word looks like name |
| Raw file bytes | Everything above | **Never persisted.** Streamed to Gemini, discarded. See below. |

### File Lifecycle (Critical)

```go
// The file NEVER touches your disk or DB.

func (h *StatementAPI) Extract(c *gin.Context) {
    file, _, _ := c.Request.FormFile("file")
    defer file.Close()

    // Read into memory ONLY (max 10MB enforced by gin)
    bytes, _ := io.ReadAll(file)

    // Send to Gemini (Vertex AI + ZDR = not retained there either)
    result, _ := h.useCase.Extract(ctx, userID, walletID, bytes, mimeType)

    // bytes goes out of scope → GC'd
    // Gemini (with ZDR) doesn't keep it
    // Only EXTRACTED DATA (scrubbed) is staged
    c.JSON(200, result)
}
```

**What IS stored:** The scrubbed extraction in `statement_extractions` staging table (24h TTL). **What is NOT stored:** The raw PDF/image. Ever.

### Scrub Layer (Defense in Depth)

Even though the prompt says "no CPF, primeiro nome only", don't trust it:

```go
var (
    cpfPattern     = regexp.MustCompile(`\d{3}\.?\d{3}\.?\d{3}-?\d{2}`)
    accountPattern = regexp.MustCompile(`\b\d{4,6}-?\d{1,2}\b`)  // conta corrente
    // Full name heuristic: 2+ capitalized words, not known merchants
)

func scrubDescriptions(items []Item) []Item {
    for i := range items {
        d := items[i].Description
        d = cpfPattern.ReplaceAllString(d, "***")
        d = accountPattern.ReplaceAllString(d, "***")
        d = truncateFullNames(d)
        if len(d) > 60 { d = d[:60] }
        items[i].Description = d
    }
    return items
}
```

---

## Clean Architecture Layout

```
internal/
├── domain/
│   └── statement.go
│       │
│       │  type StatementExtraction struct {
│       │      ID        uuid.UUID
│       │      UserID    string
│       │      WalletID  uuid.UUID
│       │      Items     []ExtractedMovement
│       │      Warnings  []string
│       │      CreatedAt time.Time
│       │      ExpiresAt time.Time         // 24h TTL
│       │  }
│       │
│       │  type ExtractedMovement struct {
│       │      TmpID             uuid.UUID
│       │      Date              time.Time
│       │      Description       string     // scrubbed
│       │      Amount            float64    // signed (your convention)
│       │      SuggestedCategory *uuid.UUID
│       │      Confidence        string
│       │      DuplicateOf       *uuid.UUID
│       │      IncludeByDefault  bool
│       │  }
│       │
│       │  type StatementExtractorGateway interface {
│       │      Extract(ctx, file []byte, mime string) (*RawExtraction, error)
│       │  }
│       │
│       │  type StatementStagingRepository interface {
│       │      Save(ctx, ext *StatementExtraction) error
│       │      GetByID(ctx, id uuid.UUID, userID string) (*StatementExtraction, error)
│       │      Delete(ctx, id uuid.UUID) error
│       │      PurgeExpired(ctx) (int, error)
│       │  }
│
├── usecase/
│   ├── statement_usecase.go
│   │     Extract(ctx, userID, walletID, file, mime) → StatementExtractionResult
│   │     Confirm(ctx, userID, extractionID, selections) → ImportResult
│   │       └─ delegates to existing MovementUseCase.Create() per item
│   │
│   └── statement_usecase_test.go
│         Mock the gateway, test: dedup, scrub, category suggestion, sign normalization
│
├── infrastructure/
│   ├── api/
│   │   └── statement_api.go
│   │         POST /statements/extract      (multipart, max 10MB)
│   │         POST /statements/:id/confirm
│   │
│   ├── gateway/
│   │   └── gemini_statement_extractor.go
│   │         Implements StatementExtractorGateway
│   │         Same Vertex AI client config as agent (southamerica-east1, ZDR)
│   │         response_schema + temperature 0.1
│   │
│   └── repository/
│       └── statement_staging_repository.go
│             GORM, JSONB for items column
│
└── bootstrap/
    └── statement/
        └── setup.go
              Wire: gateway + staging repo + movement usecase + category repo
              Register routes with auth middleware
              Register /jobs/statements/purge-expired
```

---

## DB Migration

```sql
-- 020_create_statement_extractions.up.sql

CREATE TABLE statement_extractions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     TEXT NOT NULL,
    wallet_id   UUID NOT NULL REFERENCES wallets(id),
    items       JSONB NOT NULL DEFAULT '[]',   -- []ExtractedMovement
    warnings    TEXT[] DEFAULT '{}',
    status      TEXT DEFAULT 'pending',         -- pending | confirmed | expired
    created_at  TIMESTAMP DEFAULT now(),
    expires_at  TIMESTAMP DEFAULT now() + interval '24 hours'
);

CREATE INDEX idx_stmt_ext_user    ON statement_extractions(user_id);
CREATE INDEX idx_stmt_ext_expires ON statement_extractions(expires_at)
    WHERE status = 'pending';

-- Purge job: DELETE FROM statement_extractions WHERE expires_at < now();
```

**Why JSONB for items:** This is ephemeral staging data, not queryable business data. No need to normalize into a child table. Gets deleted in 24h anyway.

---

## Plan Limits Integration

This feature can bypass the `50 movements/month` free limit in one upload. Two options:

```go
// Option A: Block at extract time (early, saves LLM cost)
func (u *StatementUseCase) Extract(ctx, userID, ...) {
    if err := u.limitsValidator.CanImportBatch(ctx, userID, estimatedCount); err != nil {
        return nil, err  // "Plano free permite 50/mês, você já tem 45"
    }
    ...
}

// Option B: Block at confirm time (user sees what they WOULD get)
func (u *StatementUseCase) Confirm(ctx, userID, extractionID, selections) {
    selectedCount := countIncluded(selections)
    if err := u.limitsValidator.CanImportBatch(ctx, userID, selectedCount); err != nil {
        return nil, err  // User can uncheck some, or upgrade
    }
    ...
}
```

**Recommend Option B** — better UX. User sees the value (look, 30 transactions parsed!) before hitting the paywall. Natural upsell moment.

---

## Sync vs Async

| Statement Size | Gemini Latency | Decision |
|----------------|---------------|----------|
| 1 page (screenshot, weekly) | ~3-5s | Sync ✅ |
| 3-5 pages (monthly PDF) | ~8-15s | Sync with long timeout (30s) ✅ |
| 10+ pages | ~30s+ | Risk of Cloud Run timeout |

**MVP: Sync with 30s timeout.** Gin handler sets `c.Request.Context()` deadline. If users hit timeouts with huge PDFs:

**Later: Async** — `POST /extract` returns `202 Accepted` + `extraction_id` immediately, background goroutine (or Cloud Tasks) does the work, frontend polls `GET /statements/:id` until `status != "processing"`. Same staging table, just add a `processing` status.

---

## Failure Modes & Handling

| Failure | Detection | Response |
|---------|-----------|----------|
| File isn't a statement (random PDF) | `items` empty + no period detected | `422` "Não parece um extrato bancário" |
| Blurry photo | Most items `confidence: "low"` | Return with warning "Imagem pouco nítida, confira valores com atenção" |
| Gemini timeout/error | HTTP error from Vertex | `503` "Tente novamente" — don't charge user's limit |
| Wrong wallet selected | Can't detect automatically | Warning if period's existing movements in that wallet = 0 but other wallet has many: "Tem certeza? A carteira X tem mais movimentos nesse período" |
| Credit card statement (not bank) | `bank_name` contains card keywords | Warning "Parece fatura de cartão, não extrato de conta. Use a carteira do cartão?" — or future: separate flow |

---

## Summary

| Aspect | Decision |
|--------|----------|
| **Extraction engine** | Gemini 2.5 Flash multimodal + `response_schema` + `temperature: 0.1` |
| **Infrastructure** | Same Vertex AI client (`southamerica-east1`, ZDR) — zero new infra |
| **Flow** | 3-stage: Extract → Review (human) → Confirm. Never auto-save. |
| **Endpoints** | `POST /statements/extract` + `POST /statements/:id/confirm` |
| **Amount handling** | LLM returns positive + direction enum. You apply sign. Avoids flip errors. |
| **File storage** | None. Memory → Gemini → discard. Only scrubbed extraction staged. |
| **Staging** | `statement_extractions` table, JSONB items, 24h TTL, purge job |
| **Dedup** | date + amount + trigram(description) > 0.7 → flag, uncheck by default |
| **Category suggest** | Keyword map (no LLM). Later: learn from user history. |
| **PII** | Prompt instructs + backend regex-scrubs (CPF, account #, full names) |
| **Plan limits** | Check on Confirm (not Extract) — show value before paywall |
| **Sync/Async** | Sync 30s timeout for MVP. Async via Cloud Tasks if needed later. |
| **Cost** | ~$0.001/statement. Irrelevant. |
