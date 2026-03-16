# Prompt 01

> This is a personal finance system, that is a SaaS to organize the cashflow and plan next month user finances. Read this repository to understand and lets discuss about a system architecture for a new feature. In this app we want to create an AI agent that is resposible to help user with everything related to its financial healthy. First of all lets only discuss how should be the AI strategy/pattern is viable, probabily MCP tools, and Dynamic user profile/Persistent memory layer strategy, or even RAG. The constraint we have is cost, currently this app runs freely (until about 400 user, using cloud run + vercel + neon postgres + firebase auth), and we want to keet like this for the MVP. First task is just understand the repo and discuss about the strategies, lattler we'll plan how would be the architecture.

---

## Repository Understanding

After a thorough exploration, here's what I found:

**Stack:** Go 1.24 backend with Gin HTTP framework, GORM + Neon PostgreSQL, Firebase Auth (with custom claims for plan/role), Expo push notifications, deployed on Cloud Run.

**Architecture:** Clean Architecture with hexagonal elements - clear layers:
- **HTTP handlers** (`internal/infrastructure/api/`)
- **Use cases** (`internal/usecase/`) - business orchestration
- **Domain** (`internal/domain/`) - entities + domain services + repository interfaces
- **Infrastructure** (`internal/infrastructure/repository/`, `gateway/`, `push/`)
- **Bootstrap/DI** (`internal/bootstrap/`) via a Registry pattern

**Core domains:** Movements (transactions), Wallets, Categories/Subcategories, Credit Cards, Invoices, Recurrent Movements, Estimates (budget planning), Balance, User Preferences, Devices, User Consent.

**Authorization model:** Free plan (2 wallets, 1 credit card, 50 movements/month, 3 recurrences) vs Plus plan. Firebase custom claims carry `plan`, `role`, `mp_subscription_id`.

**Current external services:** Firebase Auth, Expo Push, MercadoPago (subscription billing).

---

## AI Agent Strategy Discussion

Given the constraints (free tier for ~400 users, Cloud Run + Neon Postgres + Firebase), here are the viable strategies ranked by cost-effectiveness:

### 1. LLM Provider Strategy (The Cost Question)

| Option | Cost for MVP | Viability |
|--------|-------------|-----------|
| **OpenAI gpt-4o-mini** | ~$0.15/1M input, $0.60/1M output | Best cost/quality ratio for MVP |
| **Claude 3.5 Haiku** | ~$0.25/1M input, $1.25/1M output | Excellent reasoning, slightly more |
| **Google Gemini Flash** | Free tier available (15 RPM) | Zero cost but rate-limited |
| **Self-hosted (Ollama)** | Needs a GPU server | Kills the free-tier constraint |
| **Groq (Llama 3)** | Free tier available | Fast, free, but limited |

**Recommendation:** Start with **Gemini Flash free tier** or **Groq free tier** for MVP. Migrate to gpt-4o-mini or Haiku when monetizing. Abstract the LLM client behind an interface so swapping is trivial.

### 2. Tool-Use Pattern (Function Calling) vs MCP vs RAG

#### Option A: Direct Function Calling (Recommended for MVP)

The simplest and most cost-effective approach. The LLM gets a system prompt with tool definitions, and your Go backend executes them.

```
User question → LLM (with tool definitions) → Tool call → Your Go backend → Result → LLM → Response
```

**Why this fits:**
- You already have clean use case layer with well-defined operations (get balance, list movements, get estimates, etc.)
- No new infrastructure needed - the tools ARE your existing use cases
- Function calling is supported by all major LLM providers
- Minimal latency, minimal cost (small context windows)

**Tools you'd expose to the LLM (mapping to existing use cases):**
- `get_balance(period)` → BalanceUseCase
- `list_movements(period, category?, wallet?)` → MovementUseCase
- `get_estimates(period)` → EstimateUseCase
- `list_wallets()` → WalletUseCase
- `list_credit_cards()` → CreditCardUseCase
- `get_invoice(credit_card_id, period)` → InvoiceUseCase
- `get_spending_by_category(period)` → Custom query
- `compare_periods(period1, period2)` → Custom query

#### Option B: MCP (Model Context Protocol)

MCP is powerful but **overkill for MVP**. MCP shines when:
- You have multiple AI clients connecting to your tools
- You want a standardized protocol for tool discovery
- You're building an ecosystem of AI integrations

**For later:** If the app grows and you want to let users connect their finance data to Claude Desktop, ChatGPT, or other MCP-compatible clients, then exposing an MCP server makes sense. But for an in-app agent, direct function calling is simpler and cheaper.

#### Option C: RAG (Retrieval-Augmented Generation)

RAG is useful when you have **unstructured knowledge** the LLM needs to search. In your case:
- Financial data is **structured** (SQL tables with clear schemas)
- You don't need vector search - you need SQL queries
- RAG would add infrastructure cost (vector DB, embedding generation)

**Verdict:** RAG is **not the right pattern** for this use case. Your data is relational, not document-based. Function calling over your existing queries is far more efficient.

**Exception:** RAG could be useful later for a "financial tips knowledge base" or "tax regulation search", but not for the core agent functionality.

### 3. User Context & Memory Strategy

This is where the real value is. The agent needs to "know" the user to give personalized advice.

#### Layer 1: Session Context (Per-Request, Zero Cost)

Every agent call includes a **dynamic system prompt** built from the user's current data:

```
You are a financial assistant for {user}.
Current month: March 2026
Wallets: Nubank (R$2,340.50), Itau (R$890.00)
Credit Cards: Nubank (limit R$5,000, used R$1,230)
This month spending: R$3,200 (vs R$2,800 last month)
Top categories: Food (R$890), Transport (R$450), Entertainment (R$380)
Budget status: Over budget in Food (+R$190), Under in Transport (-R$50)
```

This is **cheap** (small token count), **always fresh** (built from DB at request time), and **requires no new infrastructure**. You'd create a new use case like `BuildAgentContext(userID, period)` that aggregates data from existing services.

#### Layer 2: Persistent Memory (New DB Table, Low Cost)

A simple table for the agent to remember things across sessions:

```sql
CREATE TABLE agent_memories (
    id UUID PRIMARY KEY,
    user_id TEXT NOT NULL,
    memory_type TEXT NOT NULL,  -- 'goal', 'preference', 'insight', 'note'
    content TEXT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    expires_at TIMESTAMP       -- auto-cleanup old memories
);
```

**Examples of what it stores:**
- "User wants to save R$500/month for travel fund"
- "User prefers detailed breakdowns over summaries"
- "User's salary arrives on the 5th of each month"
- "User is trying to reduce food delivery spending"

**The agent gets tools to manage its own memory:**
- `save_memory(type, content)` - Store an insight or user preference
- `search_memories(query)` - Simple ILIKE search (no vector DB needed)
- `delete_memory(id)` - Cleanup

This is just another Postgres table on Neon - **zero additional cost**.

#### Layer 3: Derived User Profile (Computed, Zero Cost)

A **materialized financial profile** computed periodically or on-demand:

```json
{
  "avg_monthly_income": 5500.00,
  "avg_monthly_expense": 4200.00,
  "savings_rate": 0.236,
  "top_expense_categories": ["Food", "Housing", "Transport"],
  "spending_trend": "increasing",  // last 3 months
  "recurring_commitments": 2800.00,
  "available_after_recurrents": 2700.00,
  "credit_utilization": 0.246
}
```

This can be a **Go function** that runs a few SQL queries and returns a struct. Injected into the system prompt alongside Layer 1. No new service needed.

### 4. Architecture Pattern: Agentic Loop with Guardrails

```
┌──────────────────────────────────────────────────┐
│                   FRONTEND                        │
│  Chat UI (React/Next.js on Vercel)               │
└──────────────┬───────────────────────────────────┘
               │ POST /api/agent/chat
               │ { message, conversation_id }
┌──────────────▼───────────────────────────────────┐
│              GO BACKEND (Cloud Run)               │
│                                                   │
│  ┌─────────────────────────────────┐              │
│  │      AgentUseCase               │              │
│  │                                 │              │
│  │  1. Load user context (Layer 1) │              │
│  │  2. Load memories (Layer 2)     │              │
│  │  3. Build system prompt         │              │
│  │  4. Call LLM with tools         │◄──┐          │
│  │  5. Execute tool calls          │   │          │
│  │  6. Return to LLM if needed ────┘   │ loop     │
│  │  7. Return final response           │          │
│  │  8. Store new memories if any       │          │
│  └─────────────────────────────────┘   │          │
│                                        │          │
│  Tools (existing use cases):           │          │
│  ├── BalanceUseCase.GetBalance()       │          │
│  ├── MovementUseCase.List()            │          │
│  ├── EstimateUseCase.Get()             │          │
│  ├── WalletUseCase.List()              │          │
│  ├── MemoryUseCase.Save/Search()       │          │
│  └── ProfileBuilder.Build()            │          │
│                                                   │
│  Guardrails:                                      │
│  ├── Max 5 tool calls per request                 │
│  ├── Read-only tools only (MVP)                   │
│  ├── Rate limit: 10 agent calls/day (free plan)   │
│  └── Max context: ~4K tokens                      │
└───────────────────────────────────────────────────┘
```

### 5. Cost Estimation for 400 Users

Assuming:
- 30% of users try the agent = 120 active users
- Average 5 messages/day per active user = 600 calls/day
- Average ~2K tokens per call (context + response)

| Provider | Monthly Token Usage | Monthly Cost |
|----------|-------------------|--------------|
| Gemini Flash (free) | ~36M tokens | **$0** (within free tier) |
| Groq Llama 3 (free) | ~36M tokens | **$0** (within free tier) |
| gpt-4o-mini | ~36M tokens | **~$10-15** |
| Claude Haiku | ~36M tokens | **~$15-20** |

**For true zero-cost MVP:** Use Gemini Flash or Groq with fallback to the other. Abstract the provider so you can switch later.

### 6. Summary: Recommended MVP Strategy

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **LLM Provider** | Gemini Flash (free) + Groq (fallback) | Zero cost, good quality |
| **AI Pattern** | Direct Function Calling (agentic loop) | Simplest, fits existing architecture |
| **User Context** | Dynamic system prompt from DB queries | Always fresh, zero infrastructure |
| **Memory** | New Postgres table on Neon | Zero cost, simple ILIKE search |
| **RAG** | Skip for MVP | Data is structured, not documents |
| **MCP** | Skip for MVP | Overkill, consider for v2 |
| **New Infrastructure** | None | Everything runs on existing stack |
| **Cost Control** | Rate limiting + token budgets | Keep within free tiers |

The beauty of this approach is that it **adds zero new infrastructure**. It's a new use case layer in your Go backend, a new Postgres table, and API calls to a free LLM provider. Your Clean Architecture already supports this perfectly - the agent is just another use case that orchestrates existing services.

---

**Next steps when you're ready:**
1. Plan the detailed architecture (new files, interfaces, bootstrap wiring)
2. Define the exact tool schemas for the LLM
3. Design the conversation storage model
4. Design the agent prompt engineering strategy
