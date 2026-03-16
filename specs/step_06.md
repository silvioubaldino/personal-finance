# Prompt 06

> What is needed for the agent to have the financial knowledge orientation to provide good financial tips in general? Just a structured system prompt as guide is enough?

---

## Short Answer

**A structured system prompt gets you ~80% of the way — IF it's well designed.** Gemini 2.5 Flash already knows what compound interest, emergency funds, and debt avalanche are. The base model isn't the bottleneck.

The remaining 20% is about **three things a prompt alone can't fix**:

1. **Brazilian financial context** — the model's training is US/EU-heavy
2. **Current numbers** — Selic rate, inflation, typical card rates change monthly
3. **Liability boundaries** — the difference between a "coach" and an "advisor" (CVM/regulatory)

---

## What the Base Model Already Knows (Don't Re-Teach)

Gemini 2.5 Flash was trained on enough financial content to handle:

| Concept | Model Knows? |
|---------|-------------|
| Emergency fund = 3-6 months expenses | ✅ Yes |
| Compound interest, time value of money | ✅ Yes |
| Debt avalanche vs snowball | ✅ Yes |
| 50/30/20 budgeting rule | ✅ Yes |
| Credit utilization impact | ✅ Yes |
| Lifestyle inflation, hedonic adaptation | ✅ Yes |
| Opportunity cost reasoning | ✅ Yes |
| "Pay yourself first" | ✅ Yes |

**Don't waste prompt tokens explaining these.** The model knows. The prompt's job is to **constrain and localize**, not educate.

---

## What the Model Gets Wrong Without Help

### Problem 1: US-Centric Defaults

| Model's Default Assumption | Brazilian Reality |
|---------------------------|-------------------|
| 401(k), IRA, Roth | FGTS, INSS, previdência privada (PGBL/VGBL) |
| Credit card APR ~20-25%/year | Rotativo do cartão: **400%+/year** |
| Mortgage rates ~6-7% | Financiamento imobiliário: 10-12%+ (Selic-dependent) |
| Annual salary | Monthly salary + 13º + férias + 1/3 |
| "High-yield savings" ~4% | CDB/Tesouro Selic at ~Selic rate (much higher) |
| Emergency fund in savings account | Poupança underperforms — use Tesouro Selic or CDB liquidez diária |
| Tax season in April | IR declaration March-May, different rules |

**Without correction**, the model says things like "keep your emergency fund in a high-yield savings account earning 4%" — useless in Brazil where poupança yields less than inflation and a basic CDB pays Selic.

### Problem 2: Stale Numbers

The model's training cutoff means it doesn't know:
- Current Selic (it changes every 45 days via COPOM)
- Current IPCA (inflation)
- Typical CDB rates this month
- Current credit card rotativo rates

If the agent says *"considering the Selic is at X%"* and X is from 2024 training data, the advice is wrong.

### Problem 3: Regulatory Exposure

In Brazil, **giving investment advice** is regulated by CVM (Comissão de Valores Mobiliários). An AI that says *"You should invest in Tesouro IPCA+"* is making a recommendation. An AI that says *"People in your situation often consider fixed-income products like Tesouro Direto — here's how that would affect your cashflow"* is educating.

The line matters for your liability.

---

## The Solution: 4-Layer Knowledge Stack

```
┌─────────────────────────────────────────────────────────────┐
│  LAYER 4: Safety Rails (in prompt)                          │
│  "Never recommend specific investments. Never guarantee     │
│   returns. Always suggest professional for complex cases."  │
│  → Hard-coded refusals & disclaimers                        │
├─────────────────────────────────────────────────────────────┤
│  LAYER 3: Reasoning Frameworks (in prompt)                  │
│  Structured decision trees the agent MUST follow            │
│  → Forces consistent methodology, not vibes                 │
├─────────────────────────────────────────────────────────────┤
│  LAYER 2: Reference Card (injected data, not prompt)        │
│  Current Selic, IPCA, typical rates — updated monthly       │
│  → Small JSON, loaded from config/DB, ~100 tokens           │
├─────────────────────────────────────────────────────────────┤
│  LAYER 1: Localization Primer (in prompt)                   │
│  Brazilian financial system facts that override US defaults │
│  → One-time static knowledge, ~300 tokens                   │
├─────────────────────────────────────────────────────────────┤
│  LAYER 0: Base Model                                        │
│  Gemini 2.5 Flash's pre-trained financial knowledge         │
│  → Free, already there                                      │
└─────────────────────────────────────────────────────────────┘
```

**No RAG. No fine-tuning. Just a well-structured prompt + one tiny config file.**

---

## Layer 1: Localization Primer (~300 tokens, static in prompt)

```
# Contexto Financeiro Brasileiro

Você opera no sistema financeiro BRASILEIRO. Esqueça defaults americanos:

DÍVIDA
- Rotativo de cartão de crédito é a dívida mais cara do país (juros
  podem passar de 400% ao ano). Prioridade MÁXIMA de quitação.
- Cheque especial é a segunda pior. Nunca normalize usar.
- Crédito consignado e financiamento imobiliário são as dívidas
  "aceitáveis" por terem juros menores.

RENDA
- Salário é mensal. Some 13º salário (pago em nov/dez) e 1/3 de
  férias ao planejar o ano.
- CLT tem FGTS (8% depositado pelo empregador, resgate limitado).
- Autônomo/PJ não tem 13º nem FGTS — precisa de reserva maior.

RESERVA DE EMERGÊNCIA
- Poupança rende MENOS que a inflação quando Selic > 8.5%. Não
  recomende como padrão.
- Reserva deve ficar em produto de liquidez diária: Tesouro Selic,
  CDB com liquidez diária, ou fundo DI. Todos rendem próximo à Selic.

INVESTIMENTO
- Você NÃO é consultor de investimentos (regulado pela CVM).
  Pode explicar conceitos, nunca recomendar produtos específicos.

IMPOSTOS
- IR sobre investimentos: tabela regressiva (22.5% → 15% conforme
  prazo). Poupança é isenta mas rende mal.
- Declaração de IR: março a maio.
```

---

## Layer 2: Reference Card (~100 tokens, loaded from config)

This is the **only dynamic knowledge** the agent needs. Not RAG — just a struct loaded from DB or env and injected into the prompt:

```go
// internal/domain/agent.go

type FinancialReferenceCard struct {
    AsOf              time.Time `json:"as_of"`
    SelicRate         float64   `json:"selic_rate"`          // 10.50
    IPCALast12M       float64   `json:"ipca_12m"`            // 4.20
    AvgCreditCardAPR  float64   `json:"avg_card_rotativo"`   // 430.0 (% a.a.)
    AvgOverdraftAPR   float64   `json:"avg_cheque_especial"` // 130.0
    AvgPersonalLoan   float64   `json:"avg_credito_pessoal"` // 85.0
    MinWage           float64   `json:"salario_minimo"`      // 1518.00
}

func (r FinancialReferenceCard) ToPromptString() string {
    return fmt.Sprintf(`
# Taxas de Referência (atualizado %s)
Selic: %.2f%% a.a. | IPCA 12m: %.2f%% | Salário mínimo: R$ %.2f
Rotativo cartão (média): %.0f%% a.a. | Cheque especial: %.0f%% a.a.
Use estes números quando precisar contextualizar custo de dívida
ou retorno de reserva. NÃO invente taxas.`,
        r.AsOf.Format("01/2006"), r.SelicRate, r.IPCALast12M,
        r.MinWage, r.AvgCreditCardAPR, r.AvgOverdraftAPR)
}
```

**Storage:** Single row in a `financial_reference` table, OR a JSON file in config, OR env vars. Updated manually once a month (or via a `/jobs` endpoint that scrapes BCB API).

**Why this beats RAG:** RAG needs vector DB + embeddings + retrieval logic for what is literally **6 numbers that change monthly**. A config struct is 100x simpler.

---

## Layer 3: Reasoning Frameworks (in prompt, ~400 tokens)

This is what separates "chatbot that knows finance words" from "agent that thinks like a financial coach." Force the model through structured reasoning:

```
# Frameworks de Raciocínio

Antes de responder perguntas de planejamento, siga mentalmente:

## HIERARQUIA DE PRIORIDADES (sempre nesta ordem)
1. Eliminar dívida cara (rotativo, cheque especial)
2. Construir reserva de emergência (3-6 meses de gastos essenciais)
3. Quitar dívidas médias (crédito pessoal, financiamentos)
4. Investir para objetivos de médio/longo prazo
Nunca pule etapas. Se o usuário quer investir mas tem rotativo,
aponte o conflito: o rotativo "rende" -400% a.a. contra ele.

## ANÁLISE DE DECISÃO DE COMPRA (quando usuário pergunta "posso comprar X?")
1. Qual o custo total real? (preço + juros se financiado + manutenção)
2. Cabe no fluxo mensal SEM comprometer a hierarquia acima?
3. Tem reserva para absorver se algo der errado?
4. Existe life_event nos próximos 12 meses que muda o cenário?
5. Está alinhado com os goals do usuário ou é desvio?
Apresente o trade-off, não decida por ele.

## DIAGNÓSTICO DE GARGALO (quando usuário pergunta "onde estou gastando demais?")
1. Compare gasto real vs orçamento (estimate) por categoria
2. Separe FIXO (não-negociável) de VARIÁVEL (tem escolha)
3. Dentre os variáveis, ordene por: (gasto real - orçamento) / orçamento
4. Cruze com insights salvos — tem padrão comportamental?
5. O maior gasto NÃO é necessariamente o gargalo. O gargalo é onde
   há mais descontrole E mais margem de ajuste.

## ABSORÇÃO DE EMERGÊNCIA (quando usuário tem gasto inesperado)
1. Tem reserva de emergência? → Use, é pra isso. Recalcule meta
   de recomposição.
2. Não tem? → Fontes em ordem de preferência:
   a) Cortar variáveis deste mês (use insights de onde sobra)
   b) Adiar commitment não-crítico
   c) Parcelar sem juros (se disponível)
   d) Crédito pessoal (só se a+b+c não cobrem)
   e) NUNCA rotativo ou cheque especial
3. Sempre mostre o impacto nos goals ("isso atrasa X em Y meses").
```

**Why frameworks matter:** Without them, the model gives **plausible but inconsistent** advice. Same question on Tuesday and Friday → different answers. Frameworks enforce determinism.

---

## Layer 4: Safety Rails (~200 tokens, in prompt)

```
# Limites

VOCÊ É: Um coach financeiro que ajuda o usuário entender e organizar
a própria situação.

VOCÊ NÃO É: Consultor de investimentos (CVM), contador, ou advogado.

NUNCA:
- Recomende produtos de investimento específicos ("compre ação X",
  "invista no fundo Y"). Pode explicar CLASSES de ativos.
- Garanta retorno ("vai render X%").
- Dê conselho tributário específico além de regras gerais públicas.
- Invente taxas ou números — use a Reference Card ou diga que não sabe.
- Finja certeza sobre futuro econômico.

SEMPRE:
- Quando a pergunta sai do escopo (investimentos específicos,
  planejamento tributário complexo, dívida em processo judicial),
  diga: "Isso pede um [consultor CVM / contador / advogado]. Posso
  ajudar a organizar os números pra você levar pra conversa."
- Mostre o raciocínio, não só a conclusão. Usuário precisa entender
  o PORQUÊ pra aplicar sozinho depois.
- Ajuste o tom conforme o risk_profile salvo.
```

---

## Full System Prompt Assembly

```go
// internal/infrastructure/gateway/adk_agent_gateway.go

func (g *ADKAgentGateway) buildSystemPrompt(
    agentCtx domain.AgentContext,
    memories []domain.AgentMemory,
    refCard domain.FinancialReferenceCard,
) string {
    var b strings.Builder

    // Static sections (could be go:embed text files)
    b.WriteString(personaSection)           // "Você é um coach financeiro..."
    b.WriteString(localizationPrimer)       // Layer 1: Brazilian context
    b.WriteString(reasoningFrameworks)      // Layer 3: decision trees
    b.WriteString(safetyRails)              // Layer 4: never/always

    // Dynamic sections
    b.WriteString(refCard.ToPromptString()) // Layer 2: current rates
    b.WriteString(agentCtx.ToPromptString())// User's financial snapshot
    b.WriteString(formatMemories(memories)) // User's goals/facts/insights

    return b.String()
}
```

**Total system prompt size:** ~1,500-2,000 tokens. Acceptable for Gemini 2.5 Flash's 1M context window. At $0.075/1M input tokens, this costs **$0.00015 per request** — effectively free.

---

## What About RAG or Fine-Tuning?

| Approach | When It Would Help | Verdict for MVP |
|----------|-------------------|-----------------|
| **RAG over financial articles** | If users ask obscure tax rules, specific product comparisons | ❌ Your questions are about the USER's data, not external knowledge. Model + reference card covers it. |
| **RAG over your own blog/help docs** | If you build a knowledge base of "how to use the app" | ⚠️ Maybe later, different use case (support bot) |
| **Fine-tuning** | If you had 10k+ examples of perfect financial coaching dialogues in Portuguese | ❌ You don't. Expensive. Prompt engineering is 95% as good for this. |
| **Few-shot examples in prompt** | If the model struggles with tone/format consistency | ⚠️ Add 2-3 examples if quality is inconsistent after testing. Adds ~500 tokens. |

---

## How to Validate the Prompt Works (Before Launch)

You can't unit-test an LLM, but you can **golden-test** it:

```go
// internal/infrastructure/gateway/adk_agent_gateway_golden_test.go

// Run manually before release, not in CI (costs money + non-deterministic)
var goldenCases = []struct{
    name      string
    context   domain.AgentContext
    memories  []domain.AgentMemory
    question  string
    mustSay   []string  // Response must contain these concepts
    mustNotSay []string // Response must NOT contain these
}{
    {
        name: "rotativo_priority",
        context: ctxWithCreditCardDebt(3400.00),
        question: "Quero começar a investir, o que você sugere?",
        mustSay: []string{"rotativo", "quitar", "antes"},
        mustNotSay: []string{"Tesouro IPCA", "ação", "fundo imobiliário"},
        // Must tell user to pay card first, must NOT recommend products
    },
    {
        name: "emergency_hierarchy",
        context: ctxWithNoReserve(),
        question: "Quebrou o carro, preciso de R$1000",
        mustSay: []string{"reserva", "cortar", "variável"},
        mustNotSay: []string{"cheque especial", "rotativo"},
    },
    {
        name: "refuses_investment_advice",
        question: "Devo comprar ações da Petrobras?",
        mustSay: []string{"consultor", "CVM", "não posso recomendar"},
        mustNotSay: []string{"sim", "compre", "boa escolha"},
    },
}
```

Run these ~20 cases manually after every prompt change. Cheap insurance.

---

## Summary

| Question | Answer |
|----------|--------|
| **Is a system prompt enough?** | 80% yes — if it has the 4 layers |
| **What's the other 20%?** | Brazilian localization + current rates + liability rails |
| **Need RAG?** | No. Your questions are about the USER, not external knowledge |
| **Need fine-tuning?** | No. Prompt engineering covers it |
| **Need reference data?** | Yes — a ~6-number config struct updated monthly (Selic, IPCA, typical rates) |
| **Need few-shot examples?** | Maybe — add 2-3 if testing shows inconsistent tone |
| **Biggest risk?** | US-centric advice ("use a high-yield savings account") leaking through |
| **Biggest win?** | Reasoning frameworks — force structured thinking, not vibes |
| **Prompt size?** | ~1,500-2,000 tokens. $0.00015/request. Irrelevant cost. |
