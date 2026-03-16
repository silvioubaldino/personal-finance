# Prompt 03

> What about the users data security? By using the AI api how should we garantee that this AI interactions will be safe, and the user data won't be used 'to train' the AI, or be someway persisted? In this APIs we have some way to mark this data as sensible? (We have to still compliance with LGPD)

---

## Critical Finding: Step 01 Recommendation Has an LGPD Problem

In `step_01.md` I recommended **Gemini Flash via the free tier (AI Studio)** for zero cost. After researching data policies, **that recommendation is NOT LGPD-compliant** for a production app handling financial data.

Here's the fundamental distinction every provider makes:

| Access Path | Uses Data for Training? | Log Retention | LGPD Viable? |
|-------------|------------------------|---------------|--------------|
| **Gemini via AI Studio Free Tier** | **YES** — prompts used to improve models, human review possible | Unspecified | ❌ **NO** |
| **Gemini via AI Studio Paid Tier** | No — DPA applies once billing enabled | Standard | ⚠️ Maybe |
| **Gemini via Vertex AI** | **No — contractually guaranteed** | 24h in-memory (disableable) | ✅ **YES** |
| **Groq Free Tier** | No explicit guarantee | Unknown | ❌ Not documented enough |
| **OpenAI API** | No (by default) | 30 days (ZDR available) | ✅ Yes (with DPA) |
| **Anthropic API** | No (by default) | 7 days (ZDR available) | ✅ Yes (with DPA) |

**The revised cost vs compliance tradeoff is:**
- Truly free (AI Studio/Groq free tiers) → **you pay with user data** → LGPD violation
- Compliant (Vertex AI / paid API) → small cost, but actually viable

---

## The Right Answer: Vertex AI in `southamerica-east1`

Since late 2025, Google Cloud expanded Gemini to São Paulo with **full data residency**. This is the natural fit for a Brazilian SaaS under LGPD.

### What Vertex AI Guarantees by Default

```
┌────────────────────────────────────────────────────────┐
│  CONTRACTUAL GUARANTEES (no action needed)             │
│                                                        │
│  ✓ Prompts/outputs are NEVER used to train or          │
│    fine-tune foundation models                         │
│  ✓ Data stored at-rest stays in chosen region          │
│  ✓ ML processing happens in chosen region (if you      │
│    use the regional endpoint, not global)              │
│  ✓ Google Cloud DPA covers LGPD as processor           │
│                                                        │
│  DEFAULT BEHAVIORS (you should disable)                │
│                                                        │
│  ⚠ In-memory caching: 24h TTL, project-isolated        │
│    → disable via Vertex AI admin API                   │
│  ⚠ Abuse monitoring: prompts may be logged for         │
│    policy violation detection                          │
│    → request exception via ZDR form                    │
│                                                        │
│  FEATURES THAT BREAK ZERO RETENTION (avoid)            │
│                                                        │
│  ✗ Google Search Grounding → 30-day retention,         │
│    CANNOT be disabled                                  │
│  ✗ Google Maps Grounding → 30-day retention,           │
│    CANNOT be disabled                                  │
│  ✗ Gemini Live API Session Resumption → stores data    │
│    (disabled by default, keep it that way)             │
└────────────────────────────────────────────────────────┘
```

### Zero Data Retention (ZDR) Setup Checklist

| Step | Action | How |
|------|--------|-----|
| 1 | Use **Vertex AI**, not AI Studio | Different endpoint, same SDK |
| 2 | Use **regional endpoint** `southamerica-east1` | Never use `global` endpoint |
| 3 | **Disable prompt caching** | `curl` to Vertex AI admin API with `aiplatform.admin` role |
| 4 | **Request abuse monitoring opt-out** | Fill Google's ZDR request form, OR set up Invoiced Billing |
| 5 | **Don't use** Search/Maps Grounding | Simply don't pass `geminitool.GoogleSearch{}` to the agent |
| 6 | **Sign the Cloud DPA** | Standard Google Cloud onboarding |
| 7 | **Org policy** restrict regions | Optional: enforce `southamerica-east1` only |

### Realistic Cost with Vertex AI

Vertex AI has **no free tier** once you're past the 90-day trial / $300 credits. For your 400-user scenario (≈36M tokens/month):

| Model | Input | Output | Est. Monthly |
|-------|-------|--------|--------------|
| **Gemini 2.5 Flash** (Vertex) | $0.075/1M | $0.30/1M | **~$5-8/month** |
| **Gemini 2.0 Flash** (Vertex) | $0.10/1M | $0.40/1M | **~$7-10/month** |

**This is the real cost of LGPD compliance.** Not zero, but cheaper than a single cloud function cold start penalty. Given your users' financial data is the core trust contract of the product, this is non-negotiable.

---

## Alternative Providers (All API Tiers, Not Consumer Apps)

| Provider | Training on API data? | Default Retention | ZDR Available? | Data Residency | LGPD Fit |
|----------|----------------------|-------------------|----------------|----------------|----------|
| **Vertex AI (Gemini)** | No, contractual | 24h in-memory | Yes (disable cache + opt-out abuse monitoring) | `southamerica-east1` | ✅ Best |
| **Anthropic API** | No, by default | 7 days | Yes (enterprise addendum, needs approval) | US/EU only | ⚠️ No BR residency |
| **OpenAI API** | No, by default | 30 days | Yes (needs sales approval) | US/EU only | ⚠️ No BR residency |
| **Azure OpenAI** | No, contractual | Configurable | Yes | `brazilsouth` available | ✅ Good alternative |

**Note on Anthropic/OpenAI:** Both have strong no-training guarantees on API tier, but neither offers Brazilian data residency. LGPD doesn't strictly forbid international transfer (Art. 33), but requires adequate safeguards. Vertex AI in SP or Azure OpenAI in Brazil South are simpler compliance stories.

---

## There Is No "Mark as Sensitive" Flag — It's Architectural

LLM APIs don't have a `sensitive: true` field on prompts. Data protection is **not a per-request flag — it's a contractual + architectural property**:

1. **Contractual layer**: Which API tier you use determines the data policy (free = training, paid/enterprise = no training)
2. **Configuration layer**: Region selection, caching disable, abuse monitoring opt-out
3. **Architectural layer**: What you send in the first place (data minimization)

Your strongest protection is **Layer 3: don't send what you don't need to send**.

---

## Architecture-Level Data Minimization (Your Real Defense)

Regardless of provider guarantees, **LGPD Art. 6, III (necessidade)** requires you to only process data strictly necessary. This applies to what you inject into prompts.

### Data Minimization in `AgentContextBuilder`

```
┌───────────────────────────────────────────────────────────┐
│         WHAT YOUR AgentContextBuilder SHOULD SEND         │
└───────────────────────────────────────────────────────────┘

  ✓ SEND (aggregated, anonymized, useful for advice)
  ├── "Current month balance: R$ -340"
  ├── "Top 3 categories: Alimentação (R$890), Transporte (R$450)"
  ├── "Wallet 1: R$2,340 | Wallet 2: R$890"
  ├── "Credit utilization: 24.6%"
  ├── "Over budget in: Alimentação (+R$190)"
  └── "Recurrent commitments: R$2,800/month"

  ✗ NEVER SEND (unnecessary identification risk)
  ├── User email, name, or Firebase UID
  ├── Bank account numbers
  ├── Wallet real names ("Nubank" → use "Wallet 1")
  ├── Raw movement descriptions ("Uber 15/03 23:45")
  ├── Full transaction lists with timestamps
  └── CPF, phone, or any direct identifier

  ⚠ SEND ONLY ON-DEMAND VIA TOOLS (not in system prompt)
  ├── Specific movement details → only if user asks
  ├── Date ranges → only for requested period
  └── Individual transaction descriptions → pseudonymized
```

### Pseudonymization Layer in the Gateway

```go
// internal/infrastructure/gateway/adk_agent_gateway.go

// Before sending to LLM — replace identifying names with tokens
func (g *ADKAgentGateway) pseudonymize(ctx *domain.AgentContext) *domain.AgentContext {
    // Wallet real names → "Wallet 1", "Wallet 2"
    // Keep a request-scoped map to translate back on response
    walletMap := make(map[string]string)  // "Wallet 1" → "Nubank"
    for i, w := range ctx.Wallets {
        token := fmt.Sprintf("Wallet %d", i+1)
        walletMap[token] = w.Name
        w.Name = token
    }

    // Category names are generic enough (Alimentação, Transporte) — keep
    // Movement descriptions → truncate/generalize
    //   "iFood - Restaurante X" → "Delivery de comida"

    g.reverseMap = walletMap  // for post-processing response
    return ctx
}

// After receiving response — restore real names for the user
func (g *ADKAgentGateway) depseudonymize(response string) string {
    for token, real := range g.reverseMap {
        response = strings.ReplaceAll(response, token, real)
    }
    return response
}
```

This way **even if the LLM provider were breached**, the leaked prompts would contain `"Wallet 1: R$2,340"` not `"Nubank conta 12345: R$2,340"`.

### Tool Results Scrubbing

When tools return data (e.g., `list_movements`), scrub them before feeding back to the LLM:

```go
// Tool wrapper strips PII before returning to ADK/LLM
func (g *ADKAgentGateway) wrapMovementTool() tool.Tool {
    return functiontool.New(config, func(ctx tool.Context, args listMovementsArgs) (...) {
        movements, _ := g.movementUseCase.List(args.Period)

        // Strip: user_id, raw descriptions, exact timestamps
        // Keep: amount, category, day-of-month, wallet token
        return scrubMovements(movements), nil
    })
}
```

---

## LGPD Compliance Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                     LGPD COMPLIANCE LAYERS                        │
└──────────────────────────────────────────────────────────────────┘

 LAYER 1: LEGAL BASIS (Art. 7)
 ─────────────────────────────
  You already have user_consents table → extend it:
    consent_type = 'ai_assistant'
    "I consent to my aggregated financial data being processed
     by Google Cloud AI (São Paulo region) to provide
     personalized financial advice. No data used for AI training."

  Block agent access if consent not given.

 LAYER 2: CONTRACTUAL (Art. 39 — Processor obligations)
 ──────────────────────────────────────────────────────
  Google Cloud DPA → Google is Operador (processor)
  You are Controlador (controller)
  DPA must cover: purpose limitation, security, sub-processors

 LAYER 3: TECHNICAL (Art. 46 — Security measures)
 ────────────────────────────────────────────────
  ✓ Vertex AI southamerica-east1 (data residency)
  ✓ ZDR enabled (no caching, no abuse monitoring logs)
  ✓ TLS in transit (default)
  ✓ Pseudonymization before LLM call
  ✓ Data minimization in context builder
  ✓ No PII in prompts (no CPF, email, name, account numbers)

 LAYER 4: TRANSPARENCY (Art. 9 — Information to titular)
 ───────────────────────────────────────────────────────
  Privacy policy must disclose:
    - AI assistant feature processes aggregated financial data
    - Provider: Google Cloud Vertex AI (Operador)
    - Region: São Paulo, Brazil
    - Retention: Zero (ZDR enabled)
    - Not used for training (contractual)
    - User can withdraw consent → feature disabled

 LAYER 5: AUDIT TRAIL (Art. 37 — Records of processing)
 ──────────────────────────────────────────────────────
  Log on YOUR side (not provider's):
    - conversation_id, user_id, timestamp
    - which tools were called (NOT the raw prompts)
    - token counts (for cost tracking)
  Store in: agent_conversations table
  Retention: 30 days then auto-purge (or per user request)
```

---

## Updated Clean Arch Design (With LGPD Layer)

```
internal/
├── domain/
│   ├── agent.go
│   │   ├── AgentContext (only aggregated/pseudonymized data)
│   │   ├── AgentConversation
│   │   ├── AgentMemory
│   │   └── AgentConsent (or reuse UserConsent)
│   │
│   └── agent/service/
│       ├── context_builder.go       # Data minimization happens HERE
│       └── pseudonymizer.go         # Tokenize wallet names, strip PII
│
├── usecase/
│   └── agent_usecase.go
│       │
│       │  func (u *AgentUseCase) Chat(ctx, userID, msg, convID) {
│       │
│       │    // LGPD GATE — must happen before any LLM call
│       │    consent, _ := u.consentRepo.GetByType(userID, "ai_assistant")
│       │    if !consent.Granted {
│       │        return ErrAIConsentRequired
│       │    }
│       │
│       │    // Build minimized context
│       │    rawCtx := u.contextBuilder.Build(userID, period)
│       │    safeCtx, reverseMap := u.pseudonymizer.Apply(rawCtx)
│       │
│       │    // LLM call — only safe data crosses this boundary
│       │    resp, _ := u.gateway.Chat(ctx, safeCtx, msg, history)
│       │
│       │    // Restore real names for display
│       │    displayResp := u.pseudonymizer.Reverse(resp, reverseMap)
│       │
│       │    // Audit log (on YOUR Postgres, not Google's)
│       │    u.auditRepo.Log(userID, convID, toolsCalled, tokenCount)
│       │
│       │    return displayResp
│       │  }
│
├── infrastructure/
│   ├── gateway/
│   │   └── adk_agent_gateway.go
│   │       │
│   │       │  // Vertex AI config — hardcoded safety
│   │       │  model, _ := gemini.NewModel(ctx, "gemini-2.5-flash",
│   │       │      &genai.ClientConfig{
│   │       │          Project:  env.GCPProject,
│   │       │          Location: "southamerica-east1",  // NEVER "global"
│   │       │          Backend:  genai.BackendVertexAI, // NEVER AI Studio
│   │       │      })
│   │       │
│   │       │  // NO GoogleSearch tool → no 30-day retention
│   │       │  // NO Maps grounding  → no 30-day retention
│   │       │  tools := []tool.Tool{
│   │       │      balanceTool, movementsTool,  // only YOUR tools
│   │       │  }
│   │
│   └── repository/
│       ├── agent_memory_repository.go
│       ├── agent_session_repository.go
│       └── agent_audit_repository.go    # LGPD Art. 37 records
│
└── bootstrap/agent/setup.go
    # Wire consent check as precondition
```

---

## DB Migrations Needed

```sql
-- 018_create_agent_tables.up.sql

-- Extend consent for AI feature (or insert new consent type)
-- (reuses existing user_consents table structure)

-- Conversations (YOUR audit trail, not Google's)
CREATE TABLE agent_conversations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         TEXT NOT NULL,
    title           TEXT,                    -- auto-generated summary
    created_at      TIMESTAMP DEFAULT now(),
    updated_at      TIMESTAMP DEFAULT now(),
    expires_at      TIMESTAMP DEFAULT now() + interval '30 days'
);
CREATE INDEX idx_agent_conv_user ON agent_conversations(user_id);
CREATE INDEX idx_agent_conv_expires ON agent_conversations(expires_at);

-- Messages (store to rebuild context, but NOT the pseudonymized data)
CREATE TABLE agent_messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES agent_conversations(id) ON DELETE CASCADE,
    role            TEXT NOT NULL,  -- 'user' | 'assistant'
    content         TEXT NOT NULL,  -- DISPLAY version (depseudonymized)
    created_at      TIMESTAMP DEFAULT now()
);
CREATE INDEX idx_agent_msg_conv ON agent_messages(conversation_id);

-- Persistent memories (cross-session insights)
CREATE TABLE agent_memories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     TEXT NOT NULL,
    memory_type TEXT NOT NULL,      -- 'goal' | 'preference' | 'insight'
    content     TEXT NOT NULL,      -- NO PII allowed here
    created_at  TIMESTAMP DEFAULT now(),
    expires_at  TIMESTAMP
);
CREATE INDEX idx_agent_mem_user ON agent_memories(user_id);

-- Audit log (LGPD Art. 37 — processing records)
CREATE TABLE agent_audit_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         TEXT NOT NULL,
    conversation_id UUID,
    tools_called    TEXT[],          -- ['get_balance', 'list_movements']
    input_tokens    INT,
    output_tokens   INT,
    provider        TEXT DEFAULT 'vertex-ai',
    region          TEXT DEFAULT 'southamerica-east1',
    created_at      TIMESTAMP DEFAULT now()
);
CREATE INDEX idx_agent_audit_user ON agent_audit_log(user_id);

-- Auto-purge job (run via /jobs endpoint)
-- DELETE FROM agent_conversations WHERE expires_at < now();
-- DELETE FROM agent_audit_log WHERE created_at < now() - interval '90 days';
```

---

## Summary Checklist

| Item | Decision |
|------|----------|
| **LLM Provider** | Vertex AI (NOT AI Studio free tier) |
| **Region** | `southamerica-east1` regional endpoint (NOT global) |
| **Model** | `gemini-2.5-flash` (~$5-8/month at 400 users) |
| **ZDR** | Enable: disable caching + request abuse monitoring opt-out |
| **Grounding** | Disable Google Search/Maps grounding (30-day retention trap) |
| **Consent** | Extend `user_consents` with `ai_assistant` type — gate all calls |
| **Pseudonymization** | Wallet names → tokens, strip PII, before LLM call |
| **Data minimization** | System prompt = aggregates only. Details on-demand via tools. |
| **Audit trail** | Log conversation metadata in YOUR Postgres (Art. 37) |
| **Auto-purge** | 30-day conversation expiry + purge job via `/jobs` |
| **Privacy policy** | Disclose Google as Operador, region, ZDR, no-training |
| **DPA** | Google Cloud DPA covers LGPD processor obligations |

**The cost of compliance: ~$5-8/month.** The free tier shortcut would have meant shipping user financial data into Google's training pipeline — a direct LGPD Art. 6/46 violation and a trust-killer for a finance app.

---

**Sources:**
- [Vertex AI Zero Data Retention](https://docs.cloud.google.com/vertex-ai/generative-ai/docs/vertex-ai-zero-data-retention)
- [Vertex AI Data Residency](https://docs.cloud.google.com/vertex-ai/generative-ai/docs/learn/data-residency)
- [Vertex AI Locations (southamerica-east1)](https://docs.cloud.google.com/vertex-ai/generative-ai/docs/learn/locations)
- [Google Cloud: Generative AI Data Residency Guarantees](https://cloud.google.com/blog/products/ai-machine-learning/google-cloud-generative-ai-data-residency-guarantees-for-data-stored-at-rest)
- [Google Cloud GenAI & TPUs in Brazil](https://www.intelligentcio.com/latam/2025/11/14/google-cloud-making-a-major-ai-push-in-brazil-with-tpus-gemini-expansion-and-new-tools/)
- [Gemini Consumer vs Enterprise Training Policies](https://i10x.ai/news/google-gemini-training-data-consumer-vs-enterprise)
- [Gemini API Terms 2025 — Redact.dev](https://redact.dev/blog/gemini-api-terms-2025)
- [Migrate from AI Studio to Vertex AI](https://docs.cloud.google.com/vertex-ai/generative-ai/docs/migrate/migrate-google-ai)
- [Vertex AI CMEK](https://docs.cloud.google.com/vertex-ai/docs/general/cmek)
- [How to Setup ZDR with Gemini](https://goabego.medium.com/no-data-left-behind-how-to-setup-zdr-with-gemini-a9ff5caf1c71)
- [Anthropic Zero Data Retention Agreement](https://privacy.claude.com/en/articles/8956058-i-have-a-zero-data-retention-agreement-with-anthropic-what-products-does-it-apply-to)
- [Claude: Data Retention Policies Overview](https://www.datastudios.org/post/claude-data-retention-policies-storage-rules-and-compliance-overview)
- [OpenAI Data Controls](https://developers.openai.com/api/docs/guides/your-data/)
- [Google Cloud GenAI Privacy Whitepaper](https://services.google.com/fh/files/misc/genai_privacy_google_cloud_202308.pdf)
