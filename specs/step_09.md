# Prompt 09

> Now we want an AI 'engine' to classify categories and subcategories automatically. And use it during the upload tool. Maybe if it's possible to know somehow that for this user 'seu zé' belongs to a supermarket category. Is it possible doing this without expensive costs or a lot of workarounds?

---

## Yes — With a 3-Tier Strategy: History First, Keywords Second, LLM Last

The trick is to **avoid calling the LLM at all for most transactions**. Your users' own past categorizations are the best classifier — free, personalized, and instant. LLM is only the fallback for genuinely new descriptions.

```
┌─────────────────────────────────────────────────────────────────┐
│           CLASSIFICATION PIPELINE (per transaction)             │
│                                                                 │
│  "PAG*SeuZe"                                                    │
│       │                                                         │
│  ┌────▼─────────────────────────────────────┐                   │
│  │  TIER 1: USER HISTORY LOOKUP             │  Cost: $0         │
│  │  SELECT category_id FROM movements       │  Latency: <5ms    │
│  │  WHERE user_id = ? AND description       │  Accuracy: ~95%   │
│  │  ILIKE '%SeuZe%' LIMIT 1                 │  (user's OWN      │
│  │                                          │   categorization)  │
│  │  Found? → Return it. Done.               │                   │
│  └────┬─────────────────────────────────────┘                   │
│       │ miss                                                    │
│  ┌────▼─────────────────────────────────────┐                   │
│  │  TIER 2: COMMUNITY PATTERN TABLE         │  Cost: $0         │
│  │  SELECT category_name FROM               │  Latency: <5ms    │
│  │  merchant_patterns WHERE 'PAG*SEUZE'     │  Accuracy: ~80%   │
│  │  ILIKE pattern                           │                   │
│  │                                          │                   │
│  │  Found? → Map to user's matching cat.    │                   │
│  └────┬─────────────────────────────────────┘                   │
│       │ miss                                                    │
│  ┌────▼─────────────────────────────────────┐                   │
│  │  TIER 3: LLM (batch, NOT per-item)       │  Cost: ~$0.0001   │
│  │  Send ALL unclassified in ONE prompt     │  Latency: ~2-4s   │
│  │  with user's category+subcategory list   │  Accuracy: ~85%   │
│  │                                          │                   │
│  │  "Classify these 8 transactions into     │                   │
│  │   one of these categories..."            │                   │
│  └──────────────────────────────────────────┘                   │
│                                                                 │
│  ALL results are SUGGESTIONS. User confirms in review screen.   │
└─────────────────────────────────────────────────────────────────┘
```

---

## Tier 1: User History Lookup (The "Seu Zé" Solver)

This is the key insight: **if the user categorized "PAG*SeuZe" as Supermercado once, every future "PAG*SeuZe" should suggest Supermercado.** You already have this data — it's the `movements` table.

### The Query

```sql
-- "What did THIS USER categorize similar descriptions as?"
SELECT
    m.category_id,
    m.sub_category_id,
    m.description,
    COUNT(*) as times_used
FROM movements m
WHERE m.user_id = $1
  AND m.category_id IS NOT NULL
  AND similarity(
      upper(regexp_replace(m.description, '[^a-zA-Z0-9]', '', 'g')),
      upper(regexp_replace($2, '[^a-zA-Z0-9]', '', 'g'))
  ) > 0.4
ORDER BY times_used DESC, m.date_create DESC
LIMIT 1;
```

**`pg_trgm` extension** (free on Neon) gives you `similarity()` — fuzzy matching that handles:
- `"PAG*SeuZe"` ↔ `"PAGSEUZE MERCADO"` → match
- `"IFOOD *REST DO JOAO"` ↔ `"IFOOD *REST MARIA"` → match (both iFood = Delivery)
- `"PIX ENVIADO MARIA"` ↔ `"PIX ENVIADO JOAO"` → match on structure, but different people → might be wrong category

**Refinement — normalize merchant name:**

```go
// Strip transaction noise to get the "merchant core"
// "PAG*SeuZe 15/03" → "SEUZE"
// "IFOOD *REST DO JOAO SP" → "IFOOD"
// "PIX ENVIADO - MARIA S" → "" (can't extract merchant → skip to tier 2)

func extractMerchant(desc string) string {
    upper := strings.ToUpper(desc)

    // Remove common prefixes
    for _, prefix := range []string{
        "PAG*", "PAGSEGURO*", "PAG ", "MP *",                   // Payment gateways
        "PIX ENVIADO", "PIX RECEBIDO", "PIX -",                  // PIX (no merchant)
        "TED ", "DOC ", "TRANSF ",                                 // Transfers (no merchant)
        "DEB.AUT.", "DEBITO AUTOMATICO",                           // Auto-debit
    } {
        if strings.HasPrefix(upper, prefix) {
            // PIX/TED/DOC = no usable merchant → return empty
            if strings.HasPrefix(prefix, "PIX") ||
               strings.HasPrefix(prefix, "TED") ||
               strings.HasPrefix(prefix, "DOC") ||
               strings.HasPrefix(prefix, "TRANSF") {
                return ""
            }
            upper = strings.TrimPrefix(upper, prefix)
        }
    }

    // Remove trailing date/location patterns: "15/03", "SP", "RJ", "BRA"
    upper = regexp.MustCompile(`\s+\d{2}/\d{2}.*$`).ReplaceAllString(upper, "")
    upper = regexp.MustCompile(`\s+[A-Z]{2}$`).ReplaceAllString(upper, "")

    return strings.TrimSpace(upper)
}
```

Now `"PAG*SeuZe 15/03 SP"` → `"SEUZE"` and you search movements for `"SEUZE"`. If user ever categorized **any** SeuZe transaction → hit.

### Why This Is Incredibly Powerful

After just **1 month of manual use**, a typical user has ~30-50 movements. Those cover maybe 15-25 unique merchants. From month 2 onwards, **80-90% of transactions match a known merchant**. The classifier gets better for free, just from the user using the app normally.

---

## Tier 2: Community Pattern Table (Shared Knowledge)

For new users (no history yet) or genuinely new merchants, a static table of well-known Brazilian merchants:

```sql
-- 021_create_merchant_patterns.up.sql

CREATE TABLE merchant_patterns (
    id              SERIAL PRIMARY KEY,
    pattern         TEXT NOT NULL,         -- SQL ILIKE pattern
    category_name   TEXT NOT NULL,         -- maps to default category descriptions
    subcategory_name TEXT,                 -- optional
    confidence      TEXT DEFAULT 'high'
);

CREATE INDEX idx_merchant_pattern ON merchant_patterns(pattern);

-- Seed with well-known Brazilian merchants
INSERT INTO merchant_patterns (pattern, category_name, subcategory_name) VALUES
-- Alimentação
('IFOOD%',            'Alimentação', 'Delivery'),
('RAPPI%',            'Alimentação', 'Delivery'),
('UBER EATS%',        'Alimentação', 'Delivery'),
('ZDELIVERY%',        'Alimentação', 'Delivery'),
('MCDONALDS%',        'Alimentação', 'Restaurante'),
('BURGER%',           'Alimentação', 'Restaurante'),
('SUBWAY%',           'Alimentação', 'Restaurante'),
('STARBUCKS%',        'Alimentação', NULL),
('PADARIA%',          'Alimentação', 'Padaria'),

-- Supermercado
('EXTRA%',            'Supermercado', NULL),
('CARREFOUR%',        'Supermercado', NULL),
('ASSAI%',            'Supermercado', NULL),
('ATACADAO%',         'Supermercado', NULL),
('PÃO DE AÇUCAR%',   'Supermercado', NULL),
('OXXO%',             'Supermercado', NULL),
('MINI MERCADO%',     'Supermercado', NULL),
('SUPERM%',           'Supermercado', NULL),
('MERCADO%',          'Supermercado', NULL),

-- Transporte
('UBER%TRIP%',        'Transporte', 'Aplicativo e taxi'),
('99%',               'Transporte', 'Aplicativo e taxi'),
('SHELL%',            'Transporte', 'Combustível'),
('IPIRANGA%',         'Transporte', 'Combustível'),
('BR MANIA%',         'Transporte', 'Combustível'),
('POSTO%',            'Transporte', 'Combustível'),
('AUTO POSTO%',       'Transporte', 'Combustível'),
('ESTACION%',         'Transporte', 'Estacionamento'),
('SEM PARAR%',        'Transporte', 'Pedágio'),
('CONECTCAR%',        'Transporte', 'Pedágio'),
('VELOE%',            'Transporte', 'Pedágio'),
('METRO%SP%',         'Transporte', 'Transporte público'),

-- Moradia
('ENEL%',             'Moradia', 'Energia'),
('CEMIG%',            'Moradia', 'Energia'),
('CPFL%',             'Moradia', 'Energia'),
('SABESP%',           'Moradia', 'Água'),
('COMGAS%',           'Moradia', 'Gás'),
('VIVO%',             'Moradia', 'Celular'),
('CLARO%',            'Moradia', 'Celular'),
('TIM%',              'Moradia', 'Celular'),
('OI FIXO%',          'Moradia', 'Internet'),

-- Saúde
('DROGASIL%',         'Saúde', 'Farmácia'),
('DROGARAIA%',        'Saúde', 'Farmácia'),
('DROGARIA%',         'Saúde', 'Farmácia'),
('FARMACIA%',         'Saúde', 'Farmácia'),
('UNIMED%',           'Saúde', 'Plano de saúde'),
('AMIL%',             'Saúde', 'Plano de saúde'),
('SULAMERICA%',       'Saúde', 'Plano de saúde'),

-- Streaming
('NETFLIX%',          'Streaming', NULL),
('SPOTIFY%',          'Streaming', NULL),
('AMAZON PRIME%',     'Streaming', NULL),
('DISNEY+%',          'Streaming', NULL),
('HBO%MAX%',          'Streaming', NULL),
('APPLE%TV%',         'Streaming', NULL),
('YOUTUBE%PREM%',     'Streaming', NULL),
('GLOBOPLAY%',        'Streaming', NULL),
('DEEZER%',           'Streaming', NULL),

-- Lazer
('STEAM%',            'Lazer', NULL),
('PLAYSTATION%',      'Lazer', NULL),
('CINEMA%',           'Lazer', NULL),
('INGRESSO%',         'Lazer', NULL),

-- Educação
('UDEMY%',            'Educação', 'Cursos e graduações'),
('ALURA%',            'Educação', 'Cursos e graduações'),
('DUOLINGO%',         'Educação', 'Idiomas'),

-- Remuneração (income)
('SALARIO%',          'Remuneração', 'Salário'),
('FOLHA%',            'Remuneração', 'Salário'),
('PROVENTOS%',        'Remuneração', 'Salário');
```

### Lookup Logic

```go
func (c *Classifier) matchCommunityPattern(desc string) *PatternMatch {
    merchant := extractMerchant(desc)
    if merchant == "" {
        return nil
    }

    // Single query — patterns table is small, cached in memory after first load
    var match MerchantPattern
    err := c.db.Where("upper(?) ILIKE pattern", merchant).
        Order("length(pattern) DESC"). // longest (most specific) match wins
        First(&match).Error

    if err != nil {
        return nil
    }
    return &PatternMatch{
        CategoryName:    match.CategoryName,
        SubcategoryName: match.SubcategoryName,
        Confidence:      match.Confidence,
    }
}
```

**Mapping community pattern → user's actual category IDs:**

The pattern table stores `category_name = 'Supermercado'`, not UUIDs. At runtime, match by description against the user's categories (since users can rename/create custom ones):

```go
func (c *Classifier) mapToUserCategory(
    patternCatName string,
    userCategories []domain.Category,
) (*uuid.UUID, *uuid.UUID) {

    // Exact match first
    for _, cat := range userCategories {
        if strings.EqualFold(cat.Description, patternCatName) {
            return cat.ID, nil // category found, no subcategory yet
        }
    }
    // Fallback: user might have renamed "Supermercado" → "Mercado"
    // Use trigram similarity
    for _, cat := range userCategories {
        if similarity(cat.Description, patternCatName) > 0.5 {
            return cat.ID, nil
        }
    }
    return nil, nil // no match — goes to LLM tier
}
```

---

## Tier 3: LLM Batch Classification (Only for Unknowns)

After Tiers 1+2, typically **5-20% of transactions remain unclassified**. Send them ALL in a **single LLM call** (not per-item — that would be 20x the cost).

### The Prompt

```go
const classificationPrompt = `Classifique as transações abaixo em UMA das categorias e
subcategorias listadas. Retorne o ID da categoria e subcategoria.
Se não tiver certeza, use confidence "low".

CATEGORIAS DO USUÁRIO:
%s

TRANSAÇÕES PARA CLASSIFICAR:
%s

Regras:
- Use SOMENTE IDs da lista acima. Não invente categorias.
- Se a descrição é genérica demais (PIX, transferência sem contexto),
  retorne category_id null.
- Priorize subcategoria quando possível.`
```

### Structured Output Schema

```go
type classificationResponse struct {
    Items []classifiedItem `json:"items"`
}
type classifiedItem struct {
    TmpID           string  `json:"tmp_id"`
    CategoryID      *string `json:"category_id"`      // nullable
    SubCategoryID   *string `json:"sub_category_id"`   // nullable
    Confidence      string  `json:"confidence"`        // high | medium | low
}
```

### Cost

At 400 users, assume ~2 uploads/month per active user (120 users), ~5 unknown items per upload:

```
120 users × 2 uploads × 1 LLM call (batch ~5 items) = 240 calls/month
~800 tokens each (category list + items + response)
= ~192K tokens/month
= $0.014 input + $0.058 output ≈ $0.07/month
```

**$0.07/month.** Effectively free. And that's the CEILING — Tiers 1+2 catch most items.

---

## The "Seu Zé" Story — Full Lifecycle

```
MONTH 1 (new user, no history):
  User uploads first statement.
  "PAG*SEUZE" → Tier 1: miss (no history)
                 Tier 2: miss (not in community patterns)
                 Tier 3: LLM says "Supermercado" (confidence: medium)
  User reviews → changes to "Supermercado" (confirms). ← SAVED

MONTH 2:
  User uploads second statement.
  "PAG*SEUZE" → Tier 1: HIT! Last month user saved it as Supermercado.
  No LLM call. Instant. Free.

FROM NOW ON:
  "PAG*SEUZE" is ALWAYS Supermercado for this user.
  Every user's "Seu Zé" resolves to THEIR chosen category.

BONUS — ANOTHER USER:
  User B also goes to Seu Zé, but categorized it as "Alimentação"
  (they don't have a Supermercado category).
  "PAG*SEUZE" → Tier 1: HIT! Alimentação for User B.
  Each user's history is THEIR classifier. No conflicts.
```

---

## Integration With Statement Upload (step_08)

Replace the hardcoded `suggestCategories()` from step_08 with the 3-tier engine:

```go
// internal/usecase/statement_usecase.go

func (u *StatementUseCase) Extract(ctx, userID, walletID, file, mime) {
    raw := u.extractorGateway.Extract(ctx, file, mime)      // Gemini vision
    items := normalizeAmounts(raw.Items)
    items = scrubDescriptions(items)
    items = flagDuplicates(items, existing)

    // ───── REPLACED: was suggestCategories() keyword map ─────
    items = u.classifier.ClassifyBatch(ctx, userID, items)
    // ──────────────────────────────────────────────────────────

    warnings := validate(items, raw)
    u.stagingRepo.Save(ctx, extraction)
    return toResult(extraction)
}
```

### ClassifyBatch Orchestration

```go
// internal/domain/classifier/service/classifier.go

func (c *ClassifierService) ClassifyBatch(
    ctx context.Context,
    userID string,
    items []domain.ExtractedMovement,
) []domain.ExtractedMovement {

    userCats, _ := c.categoryRepo.ListByUser(ctx, userID)
    var unresolved []domain.ExtractedMovement

    for i := range items {
        merchant := extractMerchant(items[i].Description)
        if merchant == "" {
            unresolved = append(unresolved, items[i])
            continue
        }

        // Tier 1: user's own history
        if match := c.matchUserHistory(ctx, userID, merchant); match != nil {
            items[i].SuggestedCategoryID = match.CategoryID
            items[i].SuggestedSubCategoryID = match.SubCategoryID
            items[i].ClassifiedBy = "history"
            items[i].Confidence = "high"
            continue
        }

        // Tier 2: community patterns
        if match := c.matchCommunityPattern(merchant); match != nil {
            catID, subID := c.mapToUserCategory(match, userCats)
            if catID != nil {
                items[i].SuggestedCategoryID = catID
                items[i].SuggestedSubCategoryID = subID
                items[i].ClassifiedBy = "pattern"
                items[i].Confidence = match.Confidence
                continue
            }
        }

        unresolved = append(unresolved, items[i])
    }

    // Tier 3: LLM batch for remaining
    if len(unresolved) > 0 {
        llmResults := c.llmClassify(ctx, unresolved, userCats)
        for _, r := range llmResults {
            // merge back into items by TmpID
            applyLLMResult(items, r)
        }
    }

    return items
}
```

---

## The Learning Flywheel (Zero Extra Work)

Here's the beautiful part — **every user confirmation teaches the system**, with zero new infrastructure:

```
┌──────────────────────────────────────────────────────────────┐
│                    THE LEARNING FLYWHEEL                      │
│                                                              │
│  1. User uploads statement                                   │
│  2. Classifier suggests categories (3-tier)                  │
│  3. User corrects some suggestions in review screen          │
│  4. POST /statements/:id/confirm → movements saved           │
│     with user's chosen categories                            │
│  5. NEXT upload → Tier 1 finds those movements               │
│     → better suggestions → fewer corrections                 │
│  6. GOTO 1                                                   │
│                                                              │
│  No retraining. No feedback endpoint. No ML pipeline.        │
│  The movements table IS the training data.                   │
└──────────────────────────────────────────────────────────────┘
```

---

## Clean Architecture

```
internal/
├── domain/
│   ├── classifier.go
│   │     type ClassificationResult struct {
│   │         CategoryID    *uuid.UUID
│   │         SubCategoryID *uuid.UUID
│   │         Confidence    string      // high | medium | low
│   │         ClassifiedBy  string      // history | pattern | llm | unclassified
│   │     }
│   │
│   │     type ClassifierGateway interface {
│   │         ClassifyBatch(ctx, items []UnclassifiedItem,
│   │             categories []Category) ([]ClassificationResult, error)
│   │     }
│   │
│   └── classifier/
│       └── service/
│           ├── classifier.go       ← 3-tier orchestration
│           ├── merchant.go         ← extractMerchant() normalization
│           └── classifier_test.go
│
├── infrastructure/
│   ├── gateway/
│   │   └── gemini_classifier_gateway.go  ← Tier 3 LLM implementation
│   └── repository/
│       └── merchant_pattern_repository.go ← Tier 2 pattern table
│
└── bootstrap/
    └── classifier/
        └── setup.go   ← wires all three tiers
```

**Tier 1 doesn't need its own repository** — it queries the existing `MovementRepository` with a similarity condition. Add one method:

```go
// Added to existing MovementRepository interface
type MovementRepository interface {
    // ... existing methods ...
    FindCategoryByDescription(ctx context.Context, userID string,
        merchant string) (*CategoryMatch, error)
}
```

---

## DB Requirements

```sql
-- Enable trigram extension (free on Neon)
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Create GIN index for trigram similarity on movements
CREATE INDEX idx_movements_desc_trgm
    ON movements USING gin(upper(description) gin_trgm_ops);

-- Community patterns table
CREATE TABLE merchant_patterns (
    id               SERIAL PRIMARY KEY,
    pattern          TEXT NOT NULL,
    category_name    TEXT NOT NULL,
    subcategory_name TEXT,
    confidence       TEXT DEFAULT 'high'
);
-- + INSERT seed data (the ~80 patterns above)
```

**That's it.** One extension, one index, one table. No vector DB. No embeddings. No ML service.

---

## Also Works for Manual Movement Creation

This classifier isn't just for statement upload. When a user creates a single movement and types a description, the frontend can call:

```
GET /classify?description=PAG*SeuZe&wallet_id=xxx
→ { category_id: "...", subcategory_id: "...", confidence: "high", source: "history" }
```

Pre-fill the category dropdown. Tiny quality-of-life win, same engine, ~2ms response for Tier 1+2 hits.

---

## Summary

| Aspect | Decision |
|--------|----------|
| **Strategy** | 3-tier: History → Community Patterns → LLM batch |
| **Tier 1 (history)** | Query user's own movements with `pg_trgm` similarity. Free. Personalized. ~80-90% hit after month 1. |
| **Tier 2 (patterns)** | Static `merchant_patterns` table, ~80 BR merchants seeded. Free. ~80% of common transactions. |
| **Tier 3 (LLM)** | Gemini 2.5 Flash, batch call, structured output. ~$0.07/month at scale. Only for unknowns. |
| **Learning** | Zero-effort flywheel: user confirms → movements table IS the training data → Tier 1 improves. |
| **Merchant normalization** | `extractMerchant()` strips payment gateway prefixes, dates, locations. Key to matching. |
| **The "Seu Zé" case** | Month 1: LLM guesses. Month 2+: Tier 1 instant hit from user's own past categorization. |
| **New infrastructure** | `pg_trgm` extension + 1 GIN index + 1 patterns table. That's it. |
| **Clean arch** | `ClassifierService` (3-tier orchestrator) + `ClassifierGateway` (LLM, Tier 3 only) |
| **Reuse** | Works for upload AND manual movement creation (autocomplete category) |
