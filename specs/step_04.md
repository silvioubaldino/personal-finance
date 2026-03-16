# Prompt 04

> Should we consider the toon pattern instead of json to save tokens in our tools? Should this amount/cost be significant?

---

## What TOON Is

**TOON (Token-Oriented Object Notation)** is a 2025 format that combines YAML indentation + CSV tabular layout. It's a translation layer: your Go code still works with structs/JSON, you encode to TOON right before sending to the LLM, and (if needed) decode back.

```
JSON (147 tokens)                    TOON (59 tokens)
─────────────────                    ────────────────
{                                    movements[3]{date,amount,category}:
  "movements": [                       2026-03-01,89.50,Alimentação
    {"date":"2026-03-01",              2026-03-02,45.00,Transporte
     "amount":89.50,                   2026-03-03,120.00,Lazer
     "category":"Alimentação"},
    {"date":"2026-03-02",
     "amount":45.00,
     "category":"Transporte"},
    {"date":"2026-03-03",
     "amount":120.00,
     "category":"Lazer"}
  ]
}
```

**Savings:** 30-60% on uniform arrays. Models understand it via transfer learning from YAML/CSV — no fine-tuning needed.

---

## Where TOON Can and Cannot Be Applied

This is the critical nuance most articles skip. Your agent has **four** places where data is serialized:

| Location | Format Controlled By | TOON Applicable? |
|----------|---------------------|------------------|
| **1. Tool schema definitions** | Gemini function-calling API — MUST be OpenAPI JSON Schema | ❌ No (API contract) |
| **2. Tool call arguments** (LLM → you) | LLM emits JSON per schema | ❌ No (LLM output) |
| **3. Tool results** (you → LLM) | Gemini `functionResponse.response` expects JSON object | ⚠️ Partial (see below) |
| **4. System prompt** (AgentContext) | Free text | ✅ Yes (full freedom) |

### Why Tool Schemas/Args Are Off-Limits

ADK's `functiontool.New()` generates JSON Schema from your Go struct tags. Gemini's function-calling API literally requires `FunctionDeclaration.parameters` to be a JSON Schema object. This is protocol, not convention. TOON cannot replace this.

### Tool Results: The Gray Zone

Gemini's `functionResponse.response` must be a **JSON object** (`map[string]any` in Go). You cannot send raw TOON. But you **can** send a JSON object containing a TOON-encoded string:

```go
// ❌ CANNOT do this — API rejects non-JSON response
return "movements[30]{date,amount,cat}:\n2026-03-01,89.50,Food\n..."

// ✅ CAN do this — JSON wrapper with TOON payload
return map[string]any{
    "format": "toon",
    "data":   "movements[30]{date,amount,cat}:\n2026-03-01,89.50,Food\n...",
}
```

**Trade-off:** This adds ~10 tokens of JSON wrapper but saves hundreds on the payload for large arrays. Benchmarks (GPT-5, Gemini 3 Flash, Claude Haiku 4.5) show 76.4% parsing accuracy on TOON vs 75.0% on JSON — so quality isn't the concern.

**Caveat:** When you wrap TOON in a JSON string, the tokenizer may not tokenize the inner content as efficiently (escaped newlines `\n` tokenize differently than real newlines). Real savings are closer to **30-40%** not the headline 60%.

### System Prompt: Full Freedom

Your `AgentContext` goes into the system instruction as plain text. This is where TOON shines with zero compromise:

```
You are a financial assistant. Current user context:

wallets[2]{token,balance}:
  Wallet 1,2340.50
  Wallet 2,890.00

creditCards[1]{token,limit,used,utilization}:
  Card 1,5000.00,1230.00,0.246

spendingByCategory[5]{category,amount,budget,delta}:
  Alimentação,890.00,700.00,+190.00
  Transporte,450.00,500.00,-50.00
  Lazer,380.00,300.00,+80.00
  Moradia,1200.00,1200.00,0.00
  Saúde,280.00,350.00,-70.00

memories[2]{type,content}:
  goal,"Save R$500/month for travel"
  insight,"Salary arrives on 5th each month"
```

vs JSON-in-prompt equivalent: ~40-50% more tokens.

---

## Cost Impact: The Actual Math

**Your scenario:** 400 users, ~30% active, ~5 msgs/day, Gemini 2.5 Flash on Vertex AI ($0.075/1M input, $0.30/1M output).

### Token Breakdown Per Request (Estimated)

| Component | Without TOON | With TOON | Savings |
|-----------|-------------|-----------|---------|
| System prompt (fixed instruction) | 300 | 300 | 0 |
| System prompt (AgentContext data) | 600 | 360 | 240 |
| User message | 30 | 30 | 0 |
| Conversation history (3 turns avg) | 450 | 450 | 0 |
| Tool call schemas (sent every request) | 800 | 800 | 0 ❌ locked |
| **Tool result** (when called, ~50% of reqs) | 1,200 | 720 | 480 |
| LLM response (output) | 200 | 200 | 0 |
| **Total avg** | **~2,980** | **~2,500** | **~480 (16%)** |

### Monthly Cost

```
Calls/month:      18,000  (120 users × 5 msgs × 30 days)
Input tokens:     ~50M without TOON → ~42M with TOON
Output tokens:    ~3.6M (unchanged)

                  Without TOON    With TOON     Savings
Input cost        $3.75           $3.15         $0.60
Output cost       $1.08           $1.08         $0.00
─────────────────────────────────────────────────────
Monthly total     $4.83           $4.23         $0.60/month
```

### Is $0.60/month Significant? No. But Scale Changes It.

| Scale | Monthly Savings | Annual | Worth It? |
|-------|----------------|--------|-----------|
| **400 users (MVP)** | $0.60 | $7 | ❌ Not worth the complexity |
| **4,000 users** | $6 | $72 | ❌ Still marginal |
| **40,000 users** | $60 | $720 | ⚠️ Maybe, if easy |
| **400,000 users** | $600 | $7,200 | ✅ Yes |

**At your MVP scale, TOON saves you less than a coffee per month.**

---

## The Real Token Economics of Your Agent

Let's look at what actually drives cost, because TOON attacks the wrong problem:

```
┌─────────────────────────────────────────────────┐
│   WHAT EATS TOKENS (per request average)        │
├─────────────────────────────────────────────────┤
│                                                 │
│  Tool schemas           ████████████  800  27%  │  ← Can't compress
│  Tool results (avg)     █████████     600  20%  │  ← TOON helps here
│  AgentContext           █████████     600  20%  │  ← TOON helps here
│  Conversation history   ██████        450  15%  │  ← Can truncate
│  Fixed system prompt    ████          300  10%  │  ← Can shorten
│  LLM output             ███           200   7%  │  ← Can't control
│  User message           █              30   1%  │  ← Can't control
│                                                 │
└─────────────────────────────────────────────────┘
```

**Cheaper wins that don't add complexity:**

| Optimization | Effort | Savings | How |
|--------------|--------|---------|-----|
| **Fewer tools per request** | Low | 20-30% | Only register relevant tools (e.g., don't include `get_invoice` if user has no credit cards) |
| **Truncate history** | Low | 10-15% | Keep last 3 turns + summarize older |
| **Aggressive context minimization** | Low | 10-15% | Already planned in step_03 — don't send unused data |
| **Shorter tool descriptions** | Low | 5-10% | `"Get balance for month"` not `"Retrieves the complete financial balance including all income and expense totals for the specified monthly period"` |
| **TOON in system prompt** | Medium | 5-8% | Encode AgentContext as TOON |
| **TOON in tool results** | Medium | 5-8% | Wrap in JSON, hope tokenizer cooperates |

**The first four optimizations are free and save more than TOON.**

---

## Recommendation

### For MVP: Skip TOON

| Reason | Detail |
|--------|--------|
| **Savings too small** | $0.60/month at 400 users |
| **Adds complexity** | Need a Go TOON encoder, string formatting logic, test it doesn't confuse the model |
| **Better wins available** | Dynamic tool registration + history truncation = 3-5x the savings, zero deps |
| **Already minimal context** | Step_03's data minimization already keeps prompts lean |
| **ADK doesn't help** | ADK auto-generates JSON schemas; you'd fight the framework for tool results |

### Make It Easy to Add Later

One line in the design keeps the door open:

```go
// internal/domain/agent/service/context_builder.go

// AgentContext has ONE method that produces prompt-ready text.
// Today it's readable natural language. Tomorrow it could be TOON.
// Consumers don't care.
func (c *AgentContext) ToPromptString() string {
    // v1: natural language / YAML-ish
    // v2 (if needed at scale): TOON encoding
}
```

### When to Revisit TOON

Pull the trigger when **any** of these become true:

- LLM bill exceeds ~$100/month (TOON saves ~$15-20/month at that point)
- You start returning large lists in tool results (50+ movements per call)
- You upgrade to a more expensive model where token cost 4-10x
- A mature Go TOON library emerges with zero-effort `encode(struct)` API

---

## TL;DR

| Question | Answer |
|----------|--------|
| **Can TOON replace JSON in tools?** | No for schemas/args (API contract). Partial for results (JSON wrapper). Yes for system prompt. |
| **Token savings?** | ~16% overall (40-50% on the parts it can touch) |
| **Cost savings at 400 users?** | ~$0.60/month |
| **Significant?** | No. Premature optimization. |
| **Better alternatives?** | Dynamic tool registration, history truncation, shorter descriptions — all save more, zero deps |
| **Prepare for later?** | Yes — isolate serialization behind `ToPromptString()` |

---

**Sources:**
- [TOON GitHub (spec + benchmarks)](https://github.com/toon-format/toon)
- [TOON Official Site](https://toonformat.dev/)
- [TOON vs JSON — Tensorlake](https://tensorlake.ai/blog/toon-vs-json)
- [InfoQ: TOON Hopes to Cut LLM Costs](https://www.infoq.com/news/2025/11/toon-reduce-llm-cost-tokens/)
- [LogRocket: Reduce Tokens with TOON](https://blog.logrocket.com/reduce-tokens-with-toon/)
- [Better Stack: Token-Efficient LLM Workflows with TOON](https://betterstack.com/community/guides/ai/toon-explained/)
- [Gemini Function Calling Reference](https://cloud.google.com/vertex-ai/generative-ai/docs/multimodal/function-calling)
- [Gemini API Function Calling Docs](https://ai.google.dev/gemini-api/docs/function-calling)
