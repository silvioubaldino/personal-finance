# Prompt 07

> We want to implement now a system documentation that explains how the app works like a google docs or markdown. And this must be available to the agent to read when needed to help the user. How should be this flow?

---

## What This Actually Is (And Isn't)

This is a **third knowledge source**, distinct from the other two:

| Source | Answers | Nature | Already Designed |
|--------|---------|--------|------------------|
| **AgentContext + tools** | "How much did I spend?" | User's DATA | step_02, step_03 |
| **agent_memories** | "What are my goals?" | User's PROFILE | step_05 |
| **App docs** ← this | "How do invoices work here?" | PRODUCT knowledge | Now |

App docs answer things the model can't know because it's **your proprietary business logic**:

- "Why is my invoice showing R$0 when I have card purchases?" → Because your invoice closes on day X, purchases after that go to next month
- "What's the difference between a recurrent movement and an estimate?" → Your app's specific semantics
- "Why can't I add a 3rd wallet?" → Free plan limit is 2
- "What happens if I delete a category with movements?" → Your cascade rules
- "How do installments appear across invoices?" → Your specific implementation

Without this, the agent **hallucinates plausible-sounding app behavior** that may be wrong.

---

## Design Decision: Skip RAG, Use "Table of Contents + Fetch" Pattern

This is the recurring theme: your data is small and structured, so RAG is overkill.

```
┌─────────────────────────────────────────────────────────────┐
│  RAG (vector search)               TOC + Fetch (recommended)│
├─────────────────────────────────────────────────────────────┤
│  Needs: embedding model,           Needs: a Postgres table  │
│         vector DB (pgvector),                               │
│         chunk strategy,            Cost: $0                 │
│         similarity threshold                                │
│                                    Precision: 100% (agent   │
│  Cost: ~$0.02/1K embeddings        picks exact doc by slug) │
│        + pgvector ops                                       │
│                                    Complexity: one table,   │
│  Precision: ~80-90% (semantic      one tool, ~50 lines Go   │
│  search has false positives)                                │
│                                    Breaks at: ~100+ docs    │
│  Complexity: chunking, reranking,  (TOC too big for prompt) │
│  embedding refresh on doc update                            │
│                                                             │
│  Good for: 1000s of unstructured   Good for: ~15-30 curated │
│  docs, unknown topics              docs, known topics       │
└─────────────────────────────────────────────────────────────┘
```

A personal finance app has maybe **15-25 documentable concepts**. The TOC fits in ~200 tokens. The agent reads the TOC, decides if any doc is relevant, fetches ONE, and answers. That's it.

---

## The Flow

```
┌──────────────────────────────────────────────────────────────┐
│  SYSTEM PROMPT (always present)                              │
│                                                              │
│  ## Documentação do App                                      │
│  Se o usuário perguntar como o APP funciona (não sobre os   │
│  DADOS dele), use read_app_doc(slug). Docs disponíveis:     │
│                                                              │
│  - wallets: O que são carteiras, como o saldo é calculado   │
│  - credit-cards: Fechamento, vencimento, limite             │
│  - invoices: Como faturas são geradas, quando fecham, ...   │
│  - movements: Tipos, categorias, como afetam saldo          │
│  - recurrence: Movimentos recorrentes vs lançamento único   │
│  - installments: Como parcelas aparecem nas faturas         │
│  - estimates: Orçamento/planejamento vs gasto real          │
│  - transfers: Transferência interna entre carteiras         │
│  - plan-limits: Limites do plano free vs plus               │
│  - categories: Categorias padrão, subcategorias, cores      │
│  - delete-rules: O que acontece ao deletar carteira/cat/... │
│                                                              │
│  Só busque se a pergunta é sobre o FUNCIONAMENTO do app.    │
│  Perguntas sobre os DADOS do usuário → use as outras tools. │
└──────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────────┐
│  User: "Por que minha fatura mostra R$0 se comprei ontem?"  │
└──────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────────┐
│  Agent reasoning:                                            │
│  "Isso é sobre COMO faturas funcionam no app, não sobre      │
│   os dados em si. Vou ler o doc de invoices."                │
│                                                              │
│  → calls read_app_doc("invoices")                            │
└──────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────────┐
│  Tool returns full markdown (~300-600 tokens):               │
│                                                              │
│  # Faturas de Cartão                                         │
│  Cada cartão tem um dia de FECHAMENTO e um dia de            │
│  VENCIMENTO. Compras feitas APÓS o fechamento entram na      │
│  fatura do mês seguinte...                                   │
│  [full explanation]                                          │
└──────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────────┐
│  Agent (combines doc + user's actual data via tools):        │
│                                                              │
│  "Seu cartão fecha no dia 10. Como você comprou ontem        │
│   (dia 12), essa compra vai pra fatura que fecha dia 10      │
│   do MÊS QUE VEM. A fatura atual mostra R$0 porque já        │
│   fechou. Quer ver o que já está na próxima fatura?"         │
└──────────────────────────────────────────────────────────────┘
```

**The magic:** the agent combines generic doc (how invoices work) + specific user data (your closing day is 10) → personalized explanation.

---

## Storage: Markdown Files → DB Table

### Source of Truth: Markdown in Repo

```
docs/app/
├── wallets.md
├── credit-cards.md
├── invoices.md
├── movements.md
├── recurrence.md
├── installments.md
├── estimates.md
├── transfers.md
├── plan-limits.md
├── categories.md
└── delete-rules.md
```

**Why markdown in repo:**
- Versioned with code → doc changes when feature changes, in the same PR
- Devs write it (they know the logic)
- Easy to review in PR diffs
- No separate CMS needed

### Runtime: Synced to DB Table

```sql
-- 019_create_app_docs.up.sql

CREATE TABLE app_docs (
    slug         TEXT PRIMARY KEY,       -- 'invoices', 'wallets'
    title        TEXT NOT NULL,          -- 'Faturas de Cartão'
    summary      TEXT NOT NULL,          -- 1-line for TOC (~80 chars)
    content      TEXT NOT NULL,          -- Full markdown body
    language     TEXT DEFAULT 'pt-BR',   -- future: multi-lang
    updated_at   TIMESTAMP DEFAULT now()
);
```

**Why DB (not read from disk at runtime):**
- Cloud Run containers are ephemeral, filesystem reads are fine but DB is more consistent with your architecture
- Enables hot-fix: update a doc via admin endpoint without redeploy
- Enables future: product team edits docs via admin UI
- `summary` column gives you the TOC without parsing markdown

### Sync Mechanism

```go
// Option A: Migration-style seed (simple, deploy-coupled)
// db/migrations/019_create_app_docs.up.sql includes INSERTs

// Option B: Startup sync (preferred)
// internal/bootstrap/agent/setup.go

//go:embed docs/app/*.md
var appDocsFS embed.FS

func syncAppDocs(repo *repository.AppDocRepository) error {
    entries, _ := appDocsFS.ReadDir("docs/app")
    for _, e := range entries {
        raw, _ := appDocsFS.ReadFile("docs/app/" + e.Name())
        doc := parseMarkdownFrontmatter(raw) // extracts slug/title/summary
        repo.Upsert(doc)
    }
    return nil
}
```

`go:embed` compiles the markdown into the binary → no filesystem dependency on Cloud Run, sync runs once on startup. Docs are always in sync with the deployed code version.

---

## Markdown Format (With Frontmatter)

```markdown
---
slug: invoices
title: Faturas de Cartão
summary: Como faturas são geradas, fechamento, vencimento e pagamento
---

# Faturas de Cartão

## Ciclo da Fatura

Cada cartão de crédito tem duas datas importantes:

- **Dia de fechamento**: Quando a fatura "fecha". Compras feitas
  DEPOIS deste dia entram na fatura do mês seguinte.
- **Dia de vencimento**: Prazo para pagar a fatura sem juros.

## Como o App Gera Faturas

O app cria uma fatura automaticamente quando você registra a
primeira compra no cartão dentro de um ciclo. A fatura fica com
status "aberta" até o dia de fechamento.

## Por Que Minha Fatura Está R$0?

Se você fez uma compra mas a fatura atual mostra R$0, provavelmente:
1. A compra foi feita APÓS o fechamento → está na próxima fatura
2. A compra foi registrada em outra carteira por engano

## Pagamento da Fatura

Quando você paga a fatura, o app registra um movimento de saída
na carteira escolhida e marca a fatura como "paga". Se pagar
parcial, o restante aparece como "rotativo" na próxima fatura.

## Parcelas

Compras parceladas aparecem distribuídas: 1/3 na fatura atual,
2/3 na próxima, etc. O app mostra o total comprometido em
faturas futuras.
```

**Writing guidelines:**
- Write for the AGENT to read, not end-users directly (the agent will rephrase)
- Be precise about business rules (the agent can't guess your cascade logic)
- Include common "why is X happening" scenarios — those are the questions users ask
- Keep each doc < 800 words (~600 tokens). If longer, split.

---

## Clean Architecture Layout

```
docs/app/                                  ← Source (go:embed'd)
├── *.md

internal/
├── domain/
│   └── appdoc.go
│       │
│       │  type AppDoc struct {
│       │      Slug     string
│       │      Title    string
│       │      Summary  string   // for TOC
│       │      Content  string   // full markdown
│       │      Language string
│       │  }
│       │
│       │  type AppDocRepository interface {
│       │      GetBySlug(ctx, slug string) (*AppDoc, error)
│       │      ListSummaries(ctx, lang string) ([]AppDoc, error)
│       │      Upsert(ctx, doc AppDoc) error
│       │  }
│
├── infrastructure/
│   └── repository/
│       └── app_doc_repository.go          ← GORM implementation
│
└── bootstrap/agent/
    ├── docs/app/*.md                      ← go:embed source lives here
    └── setup.go
        │
        │  //go:embed docs/app/*.md
        │  var appDocsFS embed.FS
        │
        │  func Setup(r *gin.Engine, reg *registry.Registry) {
        │      docRepo := repository.NewAppDocRepository(reg.GetDB())
        │      syncAppDocs(docRepo, appDocsFS)  // upsert on startup
        │      ...
        │  }
```

---

## ADK Tool Integration

```go
// internal/infrastructure/gateway/adk_agent_gateway.go

type readAppDocArgs struct {
    Slug string `json:"slug" jsonschema:"The doc slug from the available list"`
}

type readAppDocResult struct {
    Title   string `json:"title"`
    Content string `json:"content"`
}

func (g *ADKAgentGateway) buildReadAppDocTool() tool.Tool {
    t, _ := functiontool.New(
        functiontool.Config{
            Name: "read_app_doc",
            Description: "Read documentation about how a specific app " +
                "feature works. Only use when the user asks about app " +
                "BEHAVIOR, not about their DATA.",
        },
        func(ctx tool.Context, args readAppDocArgs) (readAppDocResult, error) {
            doc, err := g.appDocRepo.GetBySlug(ctx.Context(), args.Slug)
            if err != nil {
                return readAppDocResult{}, fmt.Errorf(
                    "doc '%s' not found. Available: %s",
                    args.Slug, g.availableSlugs)
            }
            return readAppDocResult{
                Title:   doc.Title,
                Content: doc.Content,
            }, nil
        },
    )
    return t
}
```

### TOC Injection in System Prompt

```go
func (g *ADKAgentGateway) buildDocsTOC(ctx context.Context) string {
    docs, _ := g.appDocRepo.ListSummaries(ctx, "pt-BR")

    var b strings.Builder
    b.WriteString("\n## Documentação do App (use read_app_doc)\n")
    b.WriteString("Se a pergunta é sobre COMO o app funciona, leia o doc relevante:\n\n")

    for _, d := range docs {
        fmt.Fprintf(&b, "- %s: %s\n", d.Slug, d.Summary)
    }

    b.WriteString("\nNão leia docs para perguntas sobre DADOS do usuário.\n")
    return b.String()
}
```

~200 tokens for 15 docs. Loaded once at gateway init, cached in memory.

---

## Which Docs to Write (Starting Set)

Based on your domain model, these are the concepts with non-obvious business logic:

| Slug | Why It Needs a Doc |
|------|---------------------|
| `wallets` | Balance calculation, what affects it, initial balance |
| `credit-cards` | Closing day vs due day — universally confusing |
| `invoices` | Auto-generation, status transitions, partial payment → rotativo |
| `movements` | Income vs expense sign convention, what `is_paid` means |
| `recurrence` | Recurrent movement ≠ recurring real transaction, how they generate |
| `installments` | How they spread across invoices, total vs per-month view |
| `estimates` | Budget vs actual, how to interpret deltas |
| `transfers` | Internal transfer is 2 movements, doesn't affect net worth |
| `plan-limits` | Free: 2 wallets / 1 card / 50 movs. Why limits exist. |
| `categories` | Default categories, subcategory hierarchy, color meaning |
| `delete-rules` | What cascades, what orphans, what's blocked |

**~11 docs × ~500 words each = ~5,500 words total.** A half-day of writing for someone who knows the system. Massive ROI: every "how does X work" question answered correctly forever.

---

## Decision: When Does the Agent Read Docs?

This is a precision question. Over-reading docs wastes tokens; under-reading causes hallucination.

**System prompt guidance (already shown above):**
> Só busque docs se a pergunta é sobre o FUNCIONAMENTO do app.
> Perguntas sobre os DADOS do usuário → use as outras tools.

**Examples the agent should learn to discriminate:**

| User Question | Docs Needed? | Why |
|---------------|--------------|-----|
| "Quanto gastei em delivery?" | ❌ No | Pure data query |
| "Por que minha fatura está R$0?" | ✅ Yes (`invoices`) | App mechanics |
| "Posso comprar um carro?" | ❌ No | Financial advice + data |
| "Como adiciono um gasto recorrente?" | ✅ Yes (`recurrence`) | How-to |
| "Por que não consigo criar outra carteira?" | ✅ Yes (`plan-limits`) | App rules |
| "O que é melhor, pagar à vista ou parcelar?" | ❌ No | Financial advice |
| "Se eu deletar essa categoria, perco os lançamentos?" | ✅ Yes (`delete-rules`) | App behavior |

If Gemini 2.5 Flash is over-eager reading docs in testing, add **2-3 few-shot examples** of correct discrimination to the system prompt (~150 tokens).

---

## Scaling Path (If You Ever Need It)

The TOC pattern breaks around ~50-80 docs (TOC gets too big for the prompt). If you ever get there:

```
Step 1 (now):     TOC in prompt + fetch by slug         → 15-30 docs
Step 2 (if >50):  Group TOC by section, 2-step fetch    → 50-150 docs
                  list_doc_sections() → read_app_doc(slug)
Step 3 (if >200): pgvector on summary column,           → 200+ docs
                  semantic search, top-3 retrieval
```

You'll likely never need Step 2 for a personal finance app. Step 3 is for when you're documenting a platform.

---

## Summary

| Aspect | Decision |
|--------|----------|
| **Pattern** | TOC in system prompt + `read_app_doc(slug)` tool. No RAG. |
| **Source of truth** | Markdown files in repo (`docs/app/*.md`), versioned with code |
| **Runtime storage** | `app_docs` Postgres table, synced from `go:embed` on startup |
| **TOC cost** | ~200 tokens, always in prompt |
| **Doc fetch cost** | ~400-600 tokens, only when agent decides it's needed |
| **Starting set** | ~11 docs covering non-obvious business logic |
| **Writing style** | For the agent to read and rephrase, not end-user prose |
| **Clean arch** | `AppDoc` domain entity, `AppDocRepository` interface, GORM impl, tool in ADK adapter |
| **Hot-fix path** | Admin can `UPDATE app_docs` without redeploy; next deploy re-syncs from markdown |
