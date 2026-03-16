# Prompt 05

> To clarify, we are moving on with vertex AI using a cheaper model like gemini 2.5 flash, without toon (keep json). Now let's think about each aspect of the user that's useful for the model to help, provide some aspects that we'll configure to save in user agent_memories. Remember the context is help the user to clarify their financial life, and financial health, and plan to the future. Some possible questions we want to answer to user besides the obvious (how much I spend in delivery): 'Where is my bottleneck expense, and how can I save more money'; 'How can I organize my finances to pay this debt'; 'Should it be a good idea to change my car/buy a house, next month'; 'My car broke, I need to spend 1000, how can we make this fit in my budget'.

---

## The Fundamental Split: Computed Context vs Persistent Memory

Before designing memories, we need to be strict about what goes where. Getting this wrong leads to **stale data** (memory says "spends R$800 on food" but that was 3 months ago) or **redundant storage** (saving balances that the DB already has).

```
┌────────────────────────────────────────────────────────────────┐
│                  AgentContext (COMPUTED FRESH)                  │
│            Built every request from DB queries                 │
│            NEVER stored in agent_memories                      │
├────────────────────────────────────────────────────────────────┤
│  - Wallet balances                                             │
│  - Spending by category (this month / last 3 months)           │
│  - Budget vs actual (from estimates table)                     │
│  - Credit card utilization                                     │
│  - Recurrent commitments total                                 │
│  - Income/expense trend (computed from movements)              │
│  - Invoice due dates and amounts                               │
│                                                                │
│  These CHANGE constantly. Memory would go stale.               │
└────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────┐
│              agent_memories (PERSISTENT)                       │
│         Things the DB CANNOT know on its own                   │
│     User told us once, or agent reasoned it once                │
├────────────────────────────────────────────────────────────────┤
│  - User's goals ("save R$10k by December")                     │
│  - Life facts ("salary on 5th", "partner pays half rent")      │
│  - Hidden obligations ("car loan until 2027, not in app")      │
│  - Preferences ("don't touch my coffee budget")                │
│  - Commitments user agreed to ("reduce delivery to R$200")     │
│  - Risk profile ("conservative, hates debt")                   │
│  - Behavioral insights (require REASONING, not just SUM())     │
│                                                                │
│  These are STABLE. Worth remembering across sessions.          │
└────────────────────────────────────────────────────────────────┘
```

**Rule of thumb:** if `SELECT SUM(...)` can answer it → AgentContext. If the user had to **tell** you or you had to **reason** about it → memory.

---

## Memory Taxonomy (memory_type Values)

### 1. `goal` — What the User Wants to Achieve

**Purpose:** The north star for all advice. Without goals, the agent is just a reporting tool.

| Field | Example |
|-------|---------|
| `content` | "Build emergency fund of R$15,000" |
| `metadata.target_amount` | 15000.00 |
| `metadata.target_date` | "2026-12-31" |
| `metadata.current_progress` | 4200.00 (updated periodically, or computed) |
| `metadata.priority` | "high" / "medium" / "low" |
| `metadata.goal_category` | "emergency_fund" / "debt_payoff" / "purchase" / "investment" / "travel" |

**Examples:**
- "Pay off credit card (currently R$3,400) in 6 months"
- "Save R$8,000 for apartment down payment by 2027"
- "Reduce monthly expenses by R$500 to increase savings rate"
- "Build 3 months of expenses as emergency fund"

**Lifecycle:** Has `expires_at` = target_date + 30 days. Agent should ask about progress when date approaches.

**Answers:** "Should I buy a house next month?" → agent checks if house goal exists, its timeline, competing goals.

---

### 2. `fact` — Things True About the User's Life the DB Can't See

**Purpose:** The agent's model of the user's financial reality is incomplete. The DB only knows what's entered. Facts fill the gaps.

| Subtype | Examples |
|---------|----------|
| **Income structure** | "Salary arrives on the 5th of each month" / "Self-employed, income varies R$4k-7k" / "Gets 13th salary in November + December" / "Annual bonus ~R$5k in March" |
| **Hidden obligations** | "Car financing R$890/month until March 2027, paid via debit not tracked" / "Sends R$400/month to parents" / "Pays R$300 condo fee in cash" |
| **Shared finances** | "Partner contributes R$1,500/month to household" / "Rent is split 50/50, I only track my half" |
| **Asset context** | "Owns apartment, no rent" / "Has R$20k in savings account not connected to app" / "Car is 2018, paid off" |
| **Employment** | "Contract ends in August, looking for new job" / "CLT, stable government job" / "Freelancer, no FGTS" |

**Lifecycle:** `expires_at` = NULL (permanent) OR explicit date if time-bound ("contract ends August 2026"). Agent should periodically re-confirm long-lived facts.

**Answers:** "Can I afford R$1000 emergency?" → agent knows about the hidden R$20k savings. "How to pay this debt?" → agent knows real income timing and hidden obligations.

---

### 3. `constraint` — What's Non-Negotiable

**Purpose:** Defines the boundaries of advice. Prevents the agent from suggesting things the user will reject, which erodes trust.

| Examples |
|----------|
| "Coffee/breakfast out is non-negotiable, don't suggest cutting it" |
| "Kids' school (R$1,200) is fixed, can't reduce" |
| "Wallet 2 is emergency fund — never suggest spending from it" |
| "Won't use credit card installments, prefers to save first" |
| "Refuses to track every small expense, only tracks >R$50" |

**Lifecycle:** `expires_at` = NULL. These rarely change.

**Answers:** "Where's my bottleneck?" → agent skips constrained categories. "Fit R$1000 emergency?" → agent knows which wallet is untouchable, which categories can flex.

---

### 4. `insight` — Patterns the Agent Derived (Not Obvious from Raw Data)

**Purpose:** The agent's "earned knowledge" — things that required reasoning over time, not a single query.

**Critical rule:** Insights must be **behavioral patterns**, NOT snapshots. ❌ "Spent R$890 on food last month" (stale in 30 days). ✅ "Food spending spikes 40% in the last week of every month" (stable pattern).

| Examples |
|----------|
| "Consistently overspends entertainment budget by ~R$150, for 4 months running" |
| "Delivery spending correlates with weekends — Fri-Sun is 70% of monthly delivery" |
| "Income deposit → spike in discretionary spending within 3 days (paycheck effect)" |
| "Never uses full budget on Transport — consistently R$80-100 under" |
| "Credit card usage climbing 5% month-over-month for 6 months" |
| "Subscription creep: added 3 new recurrents in last 4 months, +R$180/month" |

**Metadata:** `confidence` (high/medium/low), `observed_since` (date), `last_validated` (date — agent should re-check against fresh data).

**Lifecycle:** `expires_at` = created_at + 90 days. Forces re-validation. If the pattern still holds, re-save with fresh dates.

**Answers:** "Where's my bottleneck?" → THIS is the answer. The bottleneck isn't the biggest category, it's the one with the worst pattern.

---

### 5. `commitment` — What the User Agreed to Do

**Purpose:** Accountability. Turns a chat into a coaching relationship.

| Field | Example |
|-------|---------|
| `content` | "Reduce delivery spending to max R$250/month" |
| `metadata.agreed_on` | "2026-03-10" |
| `metadata.check_date` | "2026-04-05" |
| `metadata.status` | "active" / "kept" / "broken" / "abandoned" |
| `metadata.linked_goal_id` | (FK to a goal memory, if relevant) |

**Examples:**
- "Will cancel Spotify Family, switch to individual (save R$15/month)"
- "Move R$500 to savings on the 6th of each month, right after salary"
- "No credit card purchases above R$200 without sleeping on it"
- "Review all subscriptions by end of month"

**Lifecycle:** `expires_at` = check_date + 60 days. Agent proactively follows up: "You committed to X on March 10. How's that going?"

**Answers:** "How to save more?" → agent references past commitments: what worked, what didn't, why.

---

### 6. `risk_profile` — How the User Relates to Money Emotionally

**Purpose:** Same financial situation, wildly different correct advice depending on risk tolerance. This is a **singleton** — one per user, updated over time.

| Dimension | Spectrum |
|-----------|----------|
| **Debt tolerance** | "avoids all debt" ↔ "comfortable with strategic debt" |
| **Emergency buffer** | "needs 6 months expenses to feel safe" ↔ "1 month is fine" |
| **Spending style** | "frugal, guilt about spending" ↔ "lifestyle-first, save what's left" |
| **Planning horizon** | "thinks week-to-week" ↔ "plans 5 years out" |
| **Advice preference** | "wants gentle nudges" ↔ "wants blunt reality checks" |

**Storage:** Single memory with `memory_type = 'risk_profile'` and all dimensions in JSONB metadata.

**Lifecycle:** Never expires. Updated in place as agent learns more.

**Answers:** "Should I buy a house?" → for debt-averse user: "You'd need R$X saved to avoid large financing." For debt-comfortable user: "With your income, financing R$Y is manageable."

---

### 7. `life_event` — Major Context Shifts

**Purpose:** Life events invalidate everything. Budget advice pre-baby is useless post-baby.

| Examples |
|----------|
| "Getting married in June 2026 — expecting R$15k wedding cost" |
| "Baby expected November 2026" |
| "Moving to new apartment in April, rent increases R$400" |
| "Starting MBA in August, R$2k/month for 18 months" |
| "Job search in progress, might have income gap" |

**Metadata:** `event_date`, `financial_impact_estimate`, `status` (upcoming/happened/cancelled).

**Lifecycle:** `expires_at` = event_date + 180 days. After the event, insights derived FROM it may persist, but the event itself becomes history.

**Answers:** "Should I change my car?" → "You mentioned a baby in November. Car change + baby prep in the same year would be tight. Want to see the numbers?"

---

## Mapping: Which Memories Answer Which Questions

```
┌──────────────────────────────────────────────────────────────────┐
│  "Where is my bottleneck, how can I save more?"                  │
├──────────────────────────────────────────────────────────────────┤
│  AgentContext:  spending by category, budget deltas, trends      │
│  insight:       "overspends X by R$Y for N months"               │
│  constraint:    skip non-negotiables when suggesting cuts        │
│  commitment:    "you tried cutting X before, didn't stick"       │
│  risk_profile:  adjust tone — blunt vs gentle                    │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│  "How can I organize finances to pay this debt?"                 │
├──────────────────────────────────────────────────────────────────┤
│  AgentContext:  current balances, recurrents, avg surplus/deficit │
│  goal:          create/update debt payoff goal with target date   │
│  fact:          real income timing, hidden obligations, 13th      │
│  constraint:    what CAN'T be cut to free up payment room        │
│  risk_profile:  avalanche (highest interest) vs snowball (small  │
│                 wins) — pick strategy by personality              │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│  "Should I change my car / buy a house next month?"              │
├──────────────────────────────────────────────────────────────────┤
│  AgentContext:  savings rate, avg surplus, credit utilization    │
│  goal:          is this already a goal? competing goals?          │
│  fact:          hidden savings, current car status, job stability│
│  life_event:    anything else big coming?                        │
│  risk_profile:  "next month" + debt-averse = NO. debt-ok = maybe.│
│  insight:       "your spending has been climbing" = bad timing   │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│  "Car broke, need R$1000, how to fit in budget?"                 │
├──────────────────────────────────────────────────────────────────┤
│  AgentContext:  current balances, this month's remaining budget  │
│  fact:          "has R$20k savings not in app" → easy answer     │
│  constraint:    "Wallet 2 is untouchable" → respect it           │
│  insight:       "you under-spend Transport by R$80" → partial    │
│  commitment:    offer to pause a commitment temporarily          │
│  goal:          "this pushes emergency fund goal back 2 months"  │
└──────────────────────────────────────────────────────────────────┘
```

---

## Memory Creation Triggers

Memories don't appear by magic. Three mechanisms:

### A. Explicit (User Volunteers)

```
User: "My salary comes in on the 5th btw"
Agent: [calls save_memory tool]
       → fact: "Salary arrives on the 5th of each month"
```

System prompt instruction: *"When the user tells you something about their life, income, goals, or preferences that isn't in the data, save it with save_memory."*

### B. Elicited (Agent Asks)

The agent proactively fills gaps during natural conversation OR during an onboarding flow:

```
User: "How much can I save?"
Agent: "Before I answer — do you have any savings or debts
        outside what I see in the app?"
User: "Yeah, R$20k in Nubank savings"
Agent: [saves fact] "Got it. With that buffer, here's the picture..."
```

**Onboarding flow** (first 3-5 conversations): agent is instructed to ask **one** profile question per conversation:
- "What's your biggest financial goal right now?" → goal
- "Is your income fixed or does it vary?" → fact
- "Anything big coming up — move, wedding, job change?" → life_event
- "When you think about money, are you more 'save first' or 'enjoy now'?" → risk_profile

### C. Derived (Agent Reasons)

After answering a question, the agent may generate an insight:

```
User: "How much did I spend on delivery?"
Agent: [queries movements]
       "R$340 this month. I notice this is the 4th month above
        R$300, and 70% of it is Fri-Sun."
       [calls save_memory]
       → insight: "Delivery spending concentrated on weekends,
                   4 months of R$300+"
```

System prompt instruction: *"If you notice a multi-month pattern while analyzing data, save it as an insight. Only save patterns, never snapshots."*

---

## Refined DB Schema

```sql
-- 018_create_agent_memories.up.sql

CREATE TYPE agent_memory_type AS ENUM (
    'goal',
    'fact',
    'constraint',
    'insight',
    'commitment',
    'risk_profile',
    'life_event'
);

CREATE TABLE agent_memories (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         TEXT NOT NULL,
    memory_type     agent_memory_type NOT NULL,
    content         TEXT NOT NULL,        -- human-readable, what goes in prompt
    metadata        JSONB DEFAULT '{}',   -- structured extras per type
    source          TEXT NOT NULL,        -- 'explicit' | 'elicited' | 'derived'
    confidence      TEXT DEFAULT 'high',  -- 'high' | 'medium' | 'low' (for derived)
    created_at      TIMESTAMP DEFAULT now(),
    updated_at      TIMESTAMP DEFAULT now(),
    last_validated  TIMESTAMP DEFAULT now(),  -- when agent last confirmed still true
    expires_at      TIMESTAMP,            -- NULL = permanent

    -- Ensure only one risk_profile per user
    CONSTRAINT one_risk_profile_per_user
        EXCLUDE (user_id WITH =) WHERE (memory_type = 'risk_profile')
);

CREATE INDEX idx_agent_mem_user_type  ON agent_memories(user_id, memory_type);
CREATE INDEX idx_agent_mem_expires    ON agent_memories(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_agent_mem_content    ON agent_memories USING gin(to_tsvector('portuguese', content));

-- Soft cap: max 50 active memories per user (prevents prompt bloat)
-- Enforced in use case layer, not DB constraint
```

---

## Memory Hygiene (Preventing Rot)

Memory that isn't maintained becomes noise. Two background jobs (via existing `/jobs` pattern):

### 1. Expiry Purge (daily)
```sql
DELETE FROM agent_memories WHERE expires_at < now();
```

### 2. Validation Nudge (weekly)
For memories with `last_validated` > 90 days ago, the agent is instructed to gently re-confirm in the next conversation:

```
Agent: "Quick check — you mentioned your car loan runs until March 2027.
        Still accurate?"
```

If confirmed → update `last_validated`. If changed → update or delete.

### 3. Memory Budget in Prompt

Don't dump all 50 memories into every system prompt. **Selection strategy:**

```go
func selectMemoriesForPrompt(all []AgentMemory, userMessage string) []AgentMemory {
    // Always include (baseline identity):
    //   - risk_profile (1)
    //   - active goals, sorted by priority (max 3)
    //   - active life_events with event_date < 6 months away (max 2)
    //
    // Conditionally include (relevance):
    //   - facts + constraints: ILIKE match on user message keywords (max 5)
    //   - insights: most recently validated (max 3)
    //   - commitments: only those with check_date this month (max 2)
    //
    // Total cap: ~15 memories ≈ 400-600 tokens
}
```

---

## Domain Model

```go
// internal/domain/agent.go

type AgentMemoryType string

const (
    MemoryTypeGoal        AgentMemoryType = "goal"
    MemoryTypeFact        AgentMemoryType = "fact"
    MemoryTypeConstraint  AgentMemoryType = "constraint"
    MemoryTypeInsight     AgentMemoryType = "insight"
    MemoryTypeCommitment  AgentMemoryType = "commitment"
    MemoryTypeRiskProfile AgentMemoryType = "risk_profile"
    MemoryTypeLifeEvent   AgentMemoryType = "life_event"
)

type AgentMemorySource string

const (
    MemorySourceExplicit AgentMemorySource = "explicit"
    MemorySourceElicited AgentMemorySource = "elicited"
    MemorySourceDerived  AgentMemorySource = "derived"
)

type AgentMemory struct {
    ID            uuid.UUID
    UserID        string
    Type          AgentMemoryType
    Content       string                 // What goes into the prompt
    Metadata      map[string]any         // Type-specific structured data
    Source        AgentMemorySource
    Confidence    string                 // high | medium | low
    CreatedAt     time.Time
    UpdatedAt     time.Time
    LastValidated time.Time
    ExpiresAt     *time.Time
}

// Lifecycle rules per type
func (m *AgentMemory) DefaultExpiry() *time.Time {
    switch m.Type {
    case MemoryTypeGoal:
        if d, ok := m.Metadata["target_date"].(time.Time); ok {
            t := d.AddDate(0, 1, 0)
            return &t
        }
        t := time.Now().AddDate(1, 0, 0)
        return &t
    case MemoryTypeInsight:
        t := time.Now().AddDate(0, 3, 0) // 90 days, force revalidation
        return &t
    case MemoryTypeCommitment:
        if d, ok := m.Metadata["check_date"].(time.Time); ok {
            t := d.AddDate(0, 2, 0)
            return &t
        }
        t := time.Now().AddDate(0, 3, 0)
        return &t
    case MemoryTypeLifeEvent:
        if d, ok := m.Metadata["event_date"].(time.Time); ok {
            t := d.AddDate(0, 6, 0)
            return &t
        }
        t := time.Now().AddDate(1, 0, 0)
        return &t
    case MemoryTypeFact, MemoryTypeConstraint, MemoryTypeRiskProfile:
        return nil // permanent until contradicted
    }
    return nil
}
```

---

## Agent Tools for Memory Management

```
save_memory(type, content, metadata?)
    → Agent calls when user reveals something OR agent derives insight
    → Use case validates: no PII in content, type is valid, cap not exceeded

update_memory(id, content, metadata?)
    → Used when re-confirming or refining (e.g., goal progress update)

delete_memory(id)
    → User says "actually that's not true anymore" / goal abandoned

search_memories(query?)
    → Rarely needed explicitly — memories are pre-loaded in prompt
    → Useful for: "what goals did I set?" meta-questions
```

**Safety in use case layer:**
- Block saving if content matches PII patterns (CPF regex, email regex)
- Block `insight` type if content contains currency amount + single month reference (likely a snapshot, not a pattern)
- `risk_profile` → upsert, never duplicate
- Max 50 active memories per user → on cap, reject with message to agent: "Memory full, delete something stale first"

---

## Summary Table

| Type | Answers | Created By | Expires | Max/User |
|------|---------|-----------|---------|----------|
| `goal` | "Should I buy X?", "Am I on track?" | Elicited + Explicit | target_date + 30d | ~5 |
| `fact` | "Can I afford?", "What's my real situation?" | Explicit + Elicited | Never (re-validate) | ~10 |
| `constraint` | "Where can I cut?" (by exclusion) | Explicit | Never | ~5 |
| `insight` | "Where's my bottleneck?" | Derived | 90 days | ~10 |
| `commitment` | "How's my progress?", accountability | Elicited (after advice) | check_date + 60d | ~5 |
| `risk_profile` | Tone + strategy of ALL advice | Elicited + Derived | Never (singleton) | 1 |
| `life_event` | "Is now a good time for X?" | Explicit + Elicited | event_date + 180d | ~3 |

Total active memories per user: **~15-40**. Injected per prompt: **~10-15** after selection.
