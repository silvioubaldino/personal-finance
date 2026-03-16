# Prompt 02

> Clarify the google ADK (AI development kit), we should use it here to manage this agent? How should be the system design in clean arch? (this repo is in a migration process from DDD to clean arch, DDD will be deprecated).

---

## What is Google ADK?

Google's **Agent Development Kit (ADK)** is an open-source, code-first framework for building AI agents. Released at Google Cloud NEXT 2025, it now supports **Python, TypeScript, Java, and Go** (`google.golang.org/adk`).

### Key ADK Features

- **LLM Agent with Tool Calling**: Define agents with system instructions + tools, ADK handles the agentic loop (LLM call → tool execution → LLM again)
- **Custom Function Tools**: Wrap any Go function as a tool via `functiontool.New()` with typed args/results structs
- **Session Management**: Built-in `SessionService` (in-memory or database-backed) tracks conversation history per user
- **State Management**: Per-session, per-user (`user:` prefix), and per-app (`app:` prefix) state accessible from tools via `tool.Context`
- **Memory Service**: Long-term knowledge that spans multiple sessions (in-memory or Vertex AI Memory Bank)
- **Multi-Agent Orchestration**: Sequential, Parallel, Loop agents + agent-to-agent transfer
- **MCP Integration**: Native `mcptoolset` package to consume MCP servers
- **Deployment**: REST API server (`server/adkrest`), A2A protocol, or Cloud Run — deployment-agnostic
- **Model-Agnostic**: Optimized for Gemini but supports LiteLLM for Anthropic, OpenAI, Meta, etc.

### ADK Go API at a Glance

```go
// Define a tool
type getBalanceArgs struct {
    Month int `json:"month" jsonschema:"The month number (1-12)"`
    Year  int `json:"year"  jsonschema:"The year"`
}
type getBalanceResult struct {
    Income   float64 `json:"income"`
    Expense  float64 `json:"expense"`
    Balance  float64 `json:"balance"`
}
func getBalance(ctx tool.Context, args getBalanceArgs) (getBalanceResult, error) {
    // call your existing use case here
}

// Wrap as ADK tool
balanceTool, _ := functiontool.New(functiontool.Config{
    Name:        "get_balance",
    Description: "Get income, expense and balance for a given month",
}, getBalance)

// Create agent
agent, _ := llmagent.New(llmagent.Config{
    Name:        "finance_assistant",
    Model:       geminiModel,
    Instruction: "You are a personal finance assistant...",
    Tools:       []tool.Tool{balanceTool, movementsTool, ...},
})

// Run with session tracking
runner, _ := runner.New(runner.Config{
    AppName:        "personal_finance",
    Agent:          agent,
    SessionService: sessionService,
})
```

---

## Should We Use ADK Here? Analysis

### ADK Gives Us for Free

| What | Without ADK (DIY) | With ADK |
|------|-------------------|----------|
| **Agentic loop** | You build: LLM call → parse tool calls → execute → loop back | Built-in via `runner.Run()` |
| **Tool schema generation** | Manual JSON schema for each tool | Auto-generated from Go struct tags |
| **Session/conversation history** | Build your own table + logic | `SessionService` with DB backend |
| **State management** | Manual per-user/per-session state | `tool.Context.State()` with prefixes |
| **Memory (cross-session)** | Build your own memory table + search | `MemoryService` interface |
| **Streaming responses** | Manual SSE implementation | Built-in streaming modes |
| **Multi-agent routing** | Custom orchestration | `TransferToAgent`, workflow agents |
| **Model switching** | Abstract yourself | LiteLLM integration built-in |

### ADK Concerns

| Concern | Assessment |
|---------|------------|
| **Maturity** | Go SDK is ~4 months old (Nov 2025). API may change. |
| **DB Session backend in Go** | Python has `DatabaseSessionService` with SQLite/Postgres. Go currently has `InMemoryService` + Vertex AI. Custom DB backend may need implementation. |
| **Vendor lock-in** | Optimized for Gemini. LiteLLM support exists but is secondary. |
| **Complexity** | Adds a dependency with its own abstractions on top of your clean arch. |
| **Cost** | ADK itself is free. The LLM calls remain the cost driver. |

### Verdict: YES, Use ADK — But Wrapped Behind Your Own Interfaces

ADK saves significant boilerplate (agentic loop, tool schema gen, session management, streaming). The risk is coupling your clean architecture to ADK's types. The solution: **use ADK as an infrastructure detail**, never let it leak into your domain or use case layers.

---

## System Design in Clean Architecture

Here's how the AI agent feature fits into your existing clean architecture, following the same patterns already in the repo:

### Layer Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     INFRASTRUCTURE LAYER                     │
│                                                              │
│  ┌──────────────────┐  ┌─────────────────────────────────┐  │
│  │  agent_api.go     │  │  ADK Adapter (adkagent/)        │  │
│  │  (Gin HTTP handler│  │                                 │  │
│  │   POST /agent/chat│  │  - Wraps ADK runner             │  │
│  │   SSE streaming)  │  │  - Converts domain tools → ADK  │  │
│  │                   │──│  - Manages ADK sessions         │  │
│  │                   │  │  - Implements AgentGateway iface │  │
│  └──────────────────┘  └─────────────────────────────────┘  │
│                                                              │
│  ┌──────────────────┐  ┌─────────────────────────────────┐  │
│  │  agent_memory_    │  │  agent_session_                 │  │
│  │  repository.go    │  │  repository.go                  │  │
│  │  (Postgres/GORM)  │  │  (Postgres/GORM)                │  │
│  │  Implements        │  │  Implements                     │  │
│  │  AgentMemoryRepo   │  │  AgentSessionRepo               │  │
│  └──────────────────┘  └─────────────────────────────────┘  │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                       USE CASE LAYER                         │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  agent_usecase.go                                     │   │
│  │                                                       │   │
│  │  - Chat(userID, message, conversationID) Response     │   │
│  │  - Orchestrates: build context → call gateway → store │   │
│  │  - Depends on interfaces only (AgentGateway,          │   │
│  │    AgentMemoryRepo, existing use cases)               │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  agent_context_builder.go                             │   │
│  │                                                       │   │
│  │  - BuildContext(userID, period) → AgentContext         │   │
│  │  - Aggregates: wallets, balances, credit cards,       │   │
│  │    spending by category, estimates, recurrents         │   │
│  │  - Depends on existing repository interfaces          │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
│  Existing use cases (read-only access by agent):             │
│  ├── BalanceUseCase                                          │
│  ├── MovementUseCase                                         │
│  ├── WalletUseCase                                           │
│  ├── CreditCardUseCase                                       │
│  ├── InvoiceUseCase                                          │
│  └── EstimateUseCase                                         │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                       DOMAIN LAYER                           │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  agent.go (new domain entities)                        │ │
│  │                                                        │ │
│  │  type AgentConversation struct {                        │ │
│  │      ID        uuid.UUID                               │ │
│  │      UserID    string                                  │ │
│  │      Messages  []AgentMessage                          │ │
│  │      CreatedAt time.Time                               │ │
│  │      UpdatedAt time.Time                               │ │
│  │  }                                                     │ │
│  │                                                        │ │
│  │  type AgentMessage struct {                            │ │
│  │      Role    string  // "user" | "assistant" | "tool"  │ │
│  │      Content string                                    │ │
│  │  }                                                     │ │
│  │                                                        │ │
│  │  type AgentMemory struct {                             │ │
│  │      ID         uuid.UUID                              │ │
│  │      UserID     string                                 │ │
│  │      MemoryType string  // goal, preference, insight   │ │
│  │      Content    string                                 │ │
│  │      CreatedAt  time.Time                              │ │
│  │      ExpiresAt  *time.Time                             │ │
│  │  }                                                     │ │
│  │                                                        │ │
│  │  type AgentContext struct {                             │ │
│  │      UserID              string                        │ │
│  │      Wallets             []WalletSummary               │ │
│  │      CreditCards         []CreditCardSummary           │ │
│  │      CurrentBalance      BalanceSummary                │ │
│  │      SpendingByCategory  []CategorySpending            │ │
│  │      Estimates           []EstimateSummary             │ │
│  │      RecurrentTotal      float64                       │ │
│  │      Memories            []AgentMemory                 │ │
│  │  }                                                     │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Interfaces (ports) — defined in domain layer          │ │
│  │                                                        │ │
│  │  type AgentGateway interface {                         │ │
│  │      Chat(ctx, userID, message, context, history)      │ │
│  │          → (response string, toolCalls []ToolCall, err)│ │
│  │  }                                                     │ │
│  │                                                        │ │
│  │  type AgentMemoryRepository interface {                │ │
│  │      Save(ctx, memory AgentMemory) error               │ │
│  │      SearchByUser(ctx, userID, query string)           │ │
│  │          → ([]AgentMemory, error)                      │ │
│  │      DeleteByID(ctx, id uuid.UUID) error               │ │
│  │  }                                                     │ │
│  │                                                        │ │
│  │  type AgentSessionRepository interface {               │ │
│  │      SaveConversation(ctx, conv) error                 │ │
│  │      GetConversation(ctx, id) → (AgentConversation, e) │ │
│  │      ListByUser(ctx, userID) → ([]AgentConversation,e) │ │
│  │  }                                                     │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Where ADK Lives (Critical Decision)

```
ADK lives ONLY in: internal/infrastructure/gateway/adkagent/

It implements: domain.AgentGateway interface

Your use cases NEVER import ADK packages.
Your domain NEVER knows ADK exists.
```

This means if ADK Go becomes unmaintained or you want to switch to a raw OpenAI/Anthropic SDK, you only change **one package** — the gateway adapter.

### ADK Session vs Your Own Session Storage

There's a design tension here:

| Approach | Pros | Cons |
|----------|------|------|
| **Use ADK SessionService** (let ADK manage sessions) | Less code, ADK handles history automatically | Go SDK only has InMemory + Vertex AI backends. You'd need to implement a custom Postgres `SessionService`. ADK sessions own the data format. |
| **Own session storage** (pass history to ADK each call) | Full control, Postgres/GORM like everything else, clean arch compliant | You manage conversation history yourself, pass it as context each call |

**Recommendation: Hybrid approach.** Use ADK's `InMemoryService` for the agentic loop within a single request (it needs session to track multi-step tool calls). But persist conversations yourself in Postgres via `AgentSessionRepository`. On each new request, load history from your DB → feed to ADK as context in the system prompt. This keeps your data in your DB, not locked in ADK's format.

### File Structure (Following Existing Patterns)

```
internal/
├── domain/
│   ├── agent.go                          # Domain entities + interfaces
│   └── agent/
│       ├── service/
│       │   ├── agent.go                  # Domain service (context builder)
│       │   └── mock.go
│       └── repository/
│           └── (none — repos defined as interfaces in domain)
│
├── usecase/
│   ├── agent_usecase.go                  # Orchestration: chat flow
│   └── agent_usecase_test.go
│
├── infrastructure/
│   ├── api/
│   │   └── agent_api.go                  # Gin handler: POST /agent/chat
│   ├── gateway/
│   │   └── adk_agent_gateway.go          # ADK adapter (implements AgentGateway)
│   └── repository/
│       ├── agent_memory_repository.go    # GORM implementation
│       └── agent_session_repository.go   # GORM implementation
│
└── bootstrap/
    └── agent/
        └── setup.go                      # Wire everything together
```

### Data Flow for a Single Chat Request

```
1. POST /api/agent/chat { message: "How much did I spend on food?", conversation_id: "abc" }
       │
2. agent_api.go (Gin handler)
       │ extracts userID from Firebase JWT (existing middleware)
       │
3. AgentUseCase.Chat(ctx, userID, message, conversationID)
       │
       ├─ 4a. AgentSessionRepo.GetConversation(conversationID)
       │       → loads previous messages
       │
       ├─ 4b. AgentContextBuilder.Build(userID, currentPeriod)
       │       → calls existing repos: wallets, balance, credit cards, etc.
       │       → returns AgentContext struct
       │
       ├─ 4c. AgentMemoryRepo.SearchByUser(userID, message)
       │       → loads relevant persistent memories
       │
5. AgentGateway.Chat(ctx, userID, message, agentContext, history)
       │
       │  ┌─── ADK ADAPTER (infrastructure detail) ──────────┐
       │  │                                                    │
       │  │  - Builds system prompt from AgentContext           │
       │  │  - Creates ADK InMemory session                    │
       │  │  - Registers tools (each tool calls back into      │
       │  │    your use cases/repos via closure)                │
       │  │  - Runs ADK runner.Run() — handles the loop:       │
       │  │    LLM → tool call → execute → LLM → ...          │
       │  │  - Returns final text response                     │
       │  │                                                    │
       │  └────────────────────────────────────────────────────┘
       │
6. AgentUseCase stores the new messages (user + assistant)
       │  AgentSessionRepo.SaveConversation(...)
       │
7. If the agent called save_memory tool during the loop,
       │  AgentMemoryRepo.Save(...) was already called
       │
8. Return response to user
```

### Tools Registration (Inside ADK Adapter)

The ADK adapter creates tools as closures over your existing services:

```go
// Inside internal/infrastructure/gateway/adk_agent_gateway.go

func (g *ADKAgentGateway) buildTools() []tool.Tool {
    // Each tool is a closure that calls your existing use cases

    balanceTool, _ := functiontool.New(
        functiontool.Config{
            Name:        "get_balance",
            Description: "Get income/expense/balance for a month",
        },
        func(ctx tool.Context, args getBalanceArgs) (getBalanceResult, error) {
            // Calls your existing BalanceUseCase
            balance, err := g.balanceUseCase.GetBalance(ctx, args.UserID, period)
            return mapToResult(balance), err
        },
    )

    // ... more tools wrapping your existing use cases

    return []tool.Tool{balanceTool, movementsTool, ...}
}
```

**The tools don't duplicate logic** — they're thin wrappers that delegate to your existing use cases and services. ADK handles the LLM-tool loop, your code handles the business logic.

### Bootstrap Wiring (Following Existing Pattern)

```go
// internal/bootstrap/agent/setup.go

func Setup(r *gin.Engine, reg *registry.Registry) {
    // Repositories (new)
    memoryRepo  := repository.NewAgentMemoryRepository(reg.GetDB())
    sessionRepo := repository.NewAgentSessionRepository(reg.GetDB())

    // Context builder (uses existing repos)
    contextBuilder := service.NewAgentContextBuilder(
        reg.GetWalletRepository(),
        reg.GetCategoryRepository(),
        reg.GetCreditCardRepository(),
        reg.GetInvoiceRepository(),
        reg.GetEstimateRepository(),
        reg.GetMovementRepository(),
        reg.GetRecurrentMovementRepository(),
    )

    // ADK gateway (infrastructure)
    agentGateway := gateway.NewADKAgentGateway(
        contextBuilder,
        /* existing use cases for tools */
    )

    // Use case
    agentUseCase := usecase.NewAgentUseCase(
        agentGateway,
        memoryRepo,
        sessionRepo,
        contextBuilder,
    )

    // API handler
    handler := api.NewAgentAPI(agentUseCase)

    // Routes
    agentGroup := r.Group("/agent")
    agentGroup.Use(reg.GetAuthenticator().AuthMiddleware())
    agentGroup.POST("/chat", handler.Chat)
}
```

---

## ADK vs DIY Decision Matrix

| Factor | ADK | DIY (raw LLM SDK) |
|--------|-----|--------------------|
| **Time to MVP** | ~1 week | ~2-3 weeks |
| **Agentic loop** | Free | Build yourself (~200 lines) |
| **Tool schema** | Auto from Go structs | Manual JSON schema |
| **Session in loop** | Free | Build yourself |
| **Multi-agent** | Free (if needed later) | Build yourself |
| **Streaming** | Free | Build yourself (SSE) |
| **Dependency risk** | New Go SDK, may change | Only LLM provider SDK |
| **Clean arch fit** | Wrapped in gateway, clean | Direct, fully controlled |
| **Testability** | Mock the `AgentGateway` interface | Same |
| **Model flexibility** | Gemini first, LiteLLM for others | Any SDK directly |

---

## Recommendation Summary

1. **Use ADK Go** as an infrastructure adapter behind your `AgentGateway` interface
2. **ADK never touches** your domain or use case layers — it's an infrastructure detail like GORM or Firebase
3. **Session management**: Hybrid — ADK InMemory for single-request tool loops, your Postgres for conversation persistence
4. **Memory**: Your own Postgres table with ILIKE search (simple, free, no vector DB)
5. **Tools**: Thin ADK `functiontool` wrappers that delegate to existing use cases
6. **Model**: Start with Gemini Flash (free tier), ADK makes switching easy later
7. **File structure**: Follows existing bootstrap/registry patterns exactly

The key insight: **ADK is to your agent what GORM is to your database** — a useful framework that lives in the infrastructure layer and is hidden behind interfaces. Your clean architecture stays clean.

---

**Sources:**
- [Google ADK Documentation](https://google.github.io/adk-docs/)
- [ADK Go Getting Started](https://google.github.io/adk-docs/get-started/go/)
- [ADK Go GitHub Repository](https://github.com/google/adk-go)
- [ADK Go Package (pkg.go.dev)](https://pkg.go.dev/google.golang.org/adk)
- [ADK Custom Tools Documentation](https://google.github.io/adk-docs/tools-custom/)
- [ADK Sessions & Memory](https://google.github.io/adk-docs/sessions/)
- [Google Blog: ADK for Go Announcement](https://developers.googleblog.com/announcing-the-agent-development-kit-for-go-build-powerful-ai-agents-with-your-favorite-languages/)
- [Google Blog: ADK Introduction](https://developers.googleblog.com/en/agent-development-kit-easy-to-build-multi-agent-applications/)
- [Google Cloud: Agent State and Memory with ADK](https://cloud.google.com/blog/topics/developers-practitioners/remember-this-agent-state-and-memory-with-adk)
