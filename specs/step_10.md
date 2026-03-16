# Prompt 10

> Should you summarize and clarify the whole architecture that we changed here (what is new). Share important decisions and strategies, do's and dont's.

---

## Executive Summary

We designed **five new capabilities** sharing **one LLM backend** on Vertex AI, with **zero new infrastructure services** beyond what you already run (Cloud Run + Neon + Firebase).

```
┌─────────────────────────────────────────────────────────────────┐
│                      FIVE NEW CAPABILITIES                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. AI AGENT          Chat assistant for financial health        │
│  2. AGENT MEMORY      Persistent user profile across sessions    │
│  3. APP DOCS          "How does X work?" knowledge base          │
│  4. STATEMENT UPLOAD  PDF/image → parsed movements               │
│  5. AUTO CLASSIFIER   Suggest category from description          │
│                                                                  │
│  All five share: Vertex AI southamerica-east1 + ZDR + Gemini    │
│  2.5 Flash. One gateway config. One compliance story.            │
└─────────────────────────────────────────────────────────────────┘
```

Total estimated monthly LLM cost at 400 users: **~$5-10**.

---

## 1. New Components (Complete File Map)

### Domain Layer

```
internal/domain/
├── agent.go                    AgentConversation, AgentMessage, AgentMemory,
│                               AgentContext, AgentGateway, AgentMemoryRepository,
│                               AgentSessionRepository, AgentAuditRepository,
│                               FinancialReferenceCard
├── appdoc.go                   AppDoc, AppDocRepository
├── statement.go                StatementExtraction, ExtractedMovement,
│                               StatementExtractorGateway, StatementStagingRepository
├── classifier.go               ClassificationResult, ClassifierGateway
│
├── agent/service/
│   ├── context_builder.go      Aggregates wallets/balance/spending → AgentContext
│   └── pseudonymizer.go        Wallet names → tokens, PII stripping
│
└── classifier/service/
    ├── classifier.go           3-tier orchestration (history/pattern/LLM)
    └── merchant.go             extractMerchant() normalization
```

### Use Case Layer

```
internal/usecase/
├── agent_usecase.go            Chat() — consent gate → context → LLM → audit
├── statement_usecase.go        Extract() + Confirm() — 3-stage flow
└── (classifier is a domain service, called from statement_usecase)
```

### Infrastructure Layer

```
internal/infrastructure/
├── api/
│   ├── agent_api.go            POST /agent/chat (SSE streaming later)
│   └── statement_api.go        POST /statements/extract, /:id/confirm
│
├── gateway/
│   ├── adk_agent_gateway.go    ADK wrapper: tools, agentic loop, system prompt
│   │                           Also: buildDocsTOC(), read_app_doc tool
│   ├── gemini_statement_extractor.go   Multimodal + response_schema
│   └── gemini_classifier_gateway.go    Tier 3 batch classification
│
└── repository/
    ├── agent_memory_repository.go
    ├── agent_session_repository.go
    ├── agent_audit_repository.go
    ├── app_doc_repository.go
    ├── statement_staging_repository.go
    ├── merchant_pattern_repository.go
    └── (movement_repository.go extended: FindCategoryByDescription)
```

### Bootstrap Layer

```
internal/bootstrap/
├── agent/
│   ├── setup.go                Wire agent + inject ref card + sync docs
│   └── docs/app/*.md           go:embed source for app documentation
│
├── statement/setup.go
└── classifier/setup.go
```

### Database Migrations

```
db/migrations/
├── 018_create_agent_tables.up.sql
│     agent_conversations, agent_messages, agent_memories, agent_audit_log
├── 019_create_app_docs.up.sql
│     app_docs (synced from embedded markdown on startup)
├── 020_create_statement_extractions.up.sql
│     statement_extractions (JSONB items, 24h TTL)
├── 021_create_merchant_patterns.up.sql
│     merchant_patterns + ~80 seed rows (BR merchants)
├── 022_enable_pgtrgm.up.sql
│     CREATE EXTENSION pg_trgm + GIN index on movements.description
└── 023_create_financial_reference.up.sql
│     financial_reference (single row: Selic, IPCA, typical rates)
```

### New Dependencies

```
go.mod:
  + google.golang.org/adk          ADK Go (agent framework)
  + google.golang.org/genai        Gemini client (Vertex AI)
  (pg_trgm is a Postgres extension, not a Go dep)
```

---

## 2. Foundational Decisions (Apply Everywhere)

### ✅ DO: Vertex AI, Not AI Studio

| | AI Studio Free | Vertex AI |
|---|---|---|
| Uses prompts for training | **YES** ❌ | No ✅ |
| Data residency | No | `southamerica-east1` ✅ |
| ZDR available | No | Yes ✅ |
| LGPD compliant | **NO** | Yes (with DPA) |
| Cost at 400 users | $0 | ~$5-10/mo |

**The free tier costs user data. That's the price.** For a finance app under LGPD, Vertex AI is non-negotiable.

```go
// Hardcoded in every gateway — no accidental misconfig
&genai.ClientConfig{
    Project:  env.GCPProject,
    Location: "southamerica-east1",   // NEVER "global"
    Backend:  genai.BackendVertexAI,  // NEVER AI Studio
}
```

### ✅ DO: ADK as Infrastructure Detail

ADK lives **only** in `internal/infrastructure/gateway/`. Domain and use cases never import `google.golang.org/adk`. Same rule as GORM.

```
Domain        ──▶  defines AgentGateway interface
UseCase       ──▶  depends on AgentGateway interface
Infrastructure ──▶ ADKAgentGateway implements it (ADK types here only)
```

ADK Go is 4 months old. If it breaks, you replace one package.

### ❌ DON'T: RAG Anywhere

Four times we evaluated RAG. Four times it lost to something simpler:

| Use Case | Why Not RAG | What Instead |
|----------|------------|--------------|
| User financial data | Structured SQL, not documents | Function calling over use cases |
| Agent memory | 15-40 rows per user, typed | Postgres + ILIKE, TOC in prompt |
| App docs | ~15 known topics | TOC in prompt + `read_app_doc(slug)` |
| Category classification | User history IS the knowledge | `pg_trgm` on movements table |

**Add pgvector only when you have 200+ unstructured documents.** You're nowhere near that.

### ❌ DON'T: TOON

Saves ~$0.60/month at 400 users. Can't touch tool schemas (API contract). Adds complexity. Revisit at $100+/month LLM spend.

### ❌ DON'T: Fine-Tuning

Prompt engineering + reference card is 95% as good. You don't have 10k golden dialogues. Flash is already good enough.

---

## 3. Per-Feature Decision Summary

### Feature 1: AI Agent (step_02, step_03, step_06)

**Architecture**
- ADK `llmagent` + `runner` handles the agentic loop (LLM → tool → LLM)
- Tools are `functiontool` closures wrapping your **existing** use cases — zero logic duplication
- ADK `InMemoryService` for single-request session; **your** Postgres for conversation persistence

**Knowledge stack (4 layers in system prompt)**
1. Localization primer — Brazilian context (~300 tokens, static)
2. Reference card — Selic/IPCA/rates (~100 tokens, from DB/config, updated monthly)
3. Reasoning frameworks — priority hierarchy, buy-decision checklist (~400 tokens)
4. Safety rails — never recommend investments (CVM), never invent rates

**LGPD flow**
```
consent gate → build context → pseudonymize → LLM → depseudonymize → audit log
```

**Tools exposed**: `get_balance`, `list_movements`, `get_estimates`, `list_wallets`, `get_invoice`, `save_memory`, `search_memories`, `read_app_doc`

### Feature 2: Agent Memory (step_05)

**7 memory types** — each with distinct lifecycle:

| Type | What | Expires |
|------|------|---------|
| `goal` | "Save R$15k by Dec" | target_date + 30d |
| `fact` | "Salary on 5th" / "R$20k savings outside app" | Never (revalidate 90d) |
| `constraint` | "Coffee untouchable" / "Wallet 2 = emergency" | Never |
| `insight` | "Overspends X by R$Y for N months" — **patterns not snapshots** | 90d (force recheck) |
| `commitment` | "Reduce delivery to R$250" | check_date + 60d |
| `risk_profile` | Debt tolerance, advice tone (singleton) | Never, upsert |
| `life_event` | "Baby in Nov" | event_date + 180d |

**Golden rule:** If `SELECT SUM()` can answer it → AgentContext (fresh). If user **told** you or you **reasoned** it → memory.

**Creation:** Explicit (user volunteers) + Elicited (onboarding questions) + Derived (agent notices pattern).

**Prompt budget:** Inject ~10-15 memories per request, selected by type + relevance. Not all 50.

### Feature 3: App Docs (step_07)

**Pattern:** TOC + Fetch (not RAG)
- ~11 markdown files in `docs/app/`, `go:embed`'d into binary
- Synced to `app_docs` table on startup
- TOC (~200 tokens) always in system prompt
- `read_app_doc(slug)` tool fetches full content on demand

**Write for the agent to read and rephrase.** Include "why is X happening" scenarios — those are the real questions.

**Starting set:** wallets, credit-cards, invoices, movements, recurrence, installments, estimates, transfers, plan-limits, categories, delete-rules.

### Feature 4: Statement Upload (step_08)

**Extraction:** Gemini 2.5 Flash multimodal + `response_schema` + `temperature: 0.1`. Reads PDF/image directly, no OCR preprocessing.

**3-stage flow — mandatory:**
```
Extract (LLM, no DB write) → Review (user, frontend) → Confirm (batch to MovementUseCase)
```

**Key trick:** LLM returns `amount` (always positive) + `direction` enum (`in`/`out`). You apply sign. Avoids the common sign-flip bug on BR formatting.

**LGPD:** Raw file → memory → Gemini (ZDR) → discarded. Never touches disk/DB. Only **scrubbed** extraction staged (24h TTL). Backend regex-strips CPF/account even if prompt caught it.

**Plan limits:** Check on **Confirm**, not Extract — user sees value before paywall.

### Feature 5: Auto Classifier (step_09)

**3-tier pipeline — cheapest first:**

```
Tier 1: USER HISTORY       pg_trgm on movements  $0     80-90% hit after month 1
Tier 2: COMMUNITY PATTERNS merchant_patterns tbl  $0     ~80 BR merchants seeded
Tier 3: LLM BATCH          single call for rest   $0.07  only 5-20% reach here
                                                   /mo
```

**Merchant normalization is the key:** `"PAG*SeuZe 15/03 SP"` → `"SEUZE"`. Strip gateway prefixes, dates, locations. Then fuzzy-match.

**Learning flywheel:** Every user confirmation writes to `movements` → next upload Tier 1 finds it. No retraining. No feedback endpoint. The movements table IS the model.

**Reuse:** Same engine powers manual movement creation (autocomplete category dropdown).

---

## 4. Do's and Don'ts

### ✅ DO

| Rule | Applies To | Why |
|------|-----------|-----|
| **Consent gate before every LLM call** | Agent, Statement | LGPD Art. 7. Reuse `user_consents` table with `ai_assistant` type. |
| **Pseudonymize before LLM, reverse after** | Agent | "Wallet 1" not "Nubank". Even provider breach leaks nothing useful. |
| **Aggregate in prompt, details on demand** | Agent | "Top 3 categories" in system prompt. Full movement list only if user asks, via tool. |
| **Human review before any DB write** | Statement, Classifier | LLMs hallucinate. One wrong amount = trust destroyed. |
| **History before LLM** | Classifier | User's past choices beat any model. Free. Personalized. |
| **Batch LLM calls** | Classifier, Statement | 20 items in 1 prompt, not 20 prompts. |
| **Reasoning frameworks in prompt** | Agent | Force structured thinking. Same question → same methodology. |
| **`go:embed` for docs/prompts** | Docs, Agent | Version with code. Doc changes in same PR as feature changes. |
| **Expire everything ephemeral** | Memory, Staging, Audit | 24h extractions. 90d insights. 30d conversations. Purge via `/jobs`. |
| **Golden tests before prompt changes** | Agent | 20 cases: mustSay / mustNotSay. Manual, not CI. Cheap insurance. |
| **`temperature: 0.1` for extraction** | Statement, Classifier | Parsing is not creative. Low temp = consistent. |
| **Check plan limits at Confirm** | Statement | Show value before paywall. Upsell moment. |

### ❌ DON'T

| Rule | Why |
|------|-----|
| **No Google Search/Maps grounding** | 30-day retention you CANNOT disable. Breaks ZDR. |
| **No global Vertex endpoint** | Breaks data residency. Use `southamerica-east1` regional. |
| **No auto-save extracted data** | One hallucinated R$ amount destroys trust in a finance app. |
| **No snapshot insights in memory** | "Spent R$890 on food" is stale in 30 days. Only save **patterns** ("overspends by R$150 for 4 months"). |
| **No PII in agent_memories content** | CPF/email/account regex-block in use case layer. |
| **No investment product recommendations** | CVM regulates that. Agent is a coach, not an advisor. |
| **No inventing rates** | Use reference card or say "I don't know the current rate". |
| **No ADK imports in domain/usecase** | Clean arch. ADK is an infrastructure detail. Swappable. |
| **No raw file persistence** | Statement bytes: memory → Gemini → GC. Never disk/DB. |
| **No per-item LLM calls** | Batch classification. 20x cost difference. |
| **No all-memories-in-every-prompt** | Select ~15 by type + relevance. Cap ~500 tokens. |
| **No RAG for structured/small data** | 4 features, 4 simpler alternatives won. |

---

## 5. LGPD Compliance Map

```
┌────────────────────────────────────────────────────────────────┐
│  L1: LEGAL BASIS (Art. 7)                                      │
│      user_consents.consent_type = 'ai_assistant'               │
│      Checked in AgentUseCase + StatementUseCase before LLM     │
├────────────────────────────────────────────────────────────────┤
│  L2: CONTRACTUAL (Art. 39)                                     │
│      Google Cloud DPA — Google = Operador, you = Controlador   │
├────────────────────────────────────────────────────────────────┤
│  L3: TECHNICAL (Art. 46)                                       │
│      Vertex AI southamerica-east1 regional endpoint            │
│      ZDR: disable caching + opt-out abuse monitoring           │
│      Pseudonymization layer (agent)                            │
│      PII regex scrub (statement)                               │
│      Raw files never persisted (statement)                     │
├────────────────────────────────────────────────────────────────┤
│  L4: TRANSPARENCY (Art. 9)                                     │
│      Privacy policy disclosure: Google Operador, SP region,    │
│      ZDR, no-training guarantee                                │
├────────────────────────────────────────────────────────────────┤
│  L5: AUDIT (Art. 37)                                           │
│      agent_audit_log: tools_called, tokens, region, provider   │
│      On YOUR Postgres, not Google's. 90d retention.            │
└────────────────────────────────────────────────────────────────┘
```

---

## 6. Cost Roll-Up (400 Users)

| Component | Monthly | Notes |
|-----------|---------|-------|
| Agent chat | ~$5-8 | 18k calls × ~2.5k tokens avg |
| Statement extraction | ~$0.25 | 240 uploads × ~$0.001 |
| Classification Tier 3 | ~$0.07 | 240 batch calls × ~800 tokens |
| App docs reads | included in agent | Tool call during chat |
| Memory operations | $0 | Pure Postgres |
| pg_trgm | $0 | Neon extension |
| **Total LLM** | **~$5-10/mo** | |
| Infrastructure delta | **$0** | No new services |

---

## 7. Implementation Order (Suggested)

```
PHASE 1: Foundation
  □ Vertex AI project setup (southamerica-east1, ZDR config, DPA)
  □ go.mod: add genai + adk
  □ Migration 018-023 (all tables)
  □ user_consents: add 'ai_assistant' type
  □ Shared genai client factory in infrastructure/gateway/

PHASE 2: Classifier (smallest, standalone, immediate value)
  □ extractMerchant() + tests
  □ merchant_patterns seed
  □ ClassifierService (3-tier)
  □ GET /classify endpoint (manual movement autocomplete)

PHASE 3: Statement upload (depends on classifier)
  □ GeminiStatementExtractor gateway
  □ StatementUseCase (Extract + Confirm)
  □ Dedup + scrub + validate
  □ 2 endpoints + purge job

PHASE 4: Agent (biggest, depends on nothing above)
  □ FinancialReferenceCard + monthly update job
  □ docs/app/*.md (write the 11 docs) + sync
  □ AgentContextBuilder + Pseudonymizer
  □ agent_memories repo + lifecycle rules
  □ ADK gateway: system prompt assembly + tools
  □ AgentUseCase: consent → context → chat → audit
  □ POST /agent/chat
  □ Golden test suite

PHASE 5: Polish
  □ Memory elicitation (onboarding questions)
  □ Commitment follow-up proactive messages
  □ SSE streaming for agent responses
```

---

## 8. What We Consciously Deferred

| Deferred | When To Revisit |
|----------|----------------|
| TOON encoding | LLM bill > $100/month |
| pgvector / embeddings | App docs exceed ~100 files |
| Async statement processing | Users regularly upload 10+ page PDFs |
| MCP server | External AI clients need to connect to your data |
| Multi-agent orchestration | Single agent can't handle conversation complexity |
| Learned patterns per user (beyond history) | Tier 1 hit rate drops below 80% |
| SSE streaming | After chat UX validated with blocking responses |
| Few-shot examples in prompt | Golden tests show inconsistent tone |
