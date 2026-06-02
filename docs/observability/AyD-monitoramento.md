# AyD — Análise e Desenho de Monitoramento (Observabilidade)

> **Status:** Proposta para discussão
> **Escopo:** `personal-finance` (backend Go/Cloud Run), `personal-finance-front-v2` (web/Vercel), `personal-finance-mobile` (mobile/Firebase)
> **Objetivo:** Padronizar **logs** e **métricas** entre os três apps, com o **menor custo possível** (idealmente $0), consolidando tudo em **dashboards unificados**. Trace é desejável, porém **baixa prioridade** nesta fase.

---

## 1. Sumário executivo

| Tema | Decisão proposta |
|---|---|
| **Padrão de instrumentação** | **OpenTelemetry (OTel)** como contrato comum nos 3 apps. Já é dependência transitiva do backend (via Google ADK / `go.opentelemetry.io/otel`). Vendor-neutral evita lock-in. |
| **Backend / "single pane of glass"** | **Grafana Cloud (Free tier)** como camada única de visualização. Conecta dados de GCP, Firebase e Vercel sem precisar migrar tudo. |
| **Coleta no Cloud Run** | **Sidecar** (Grafana Alloy **ou** OTel Collector) recebendo OTLP em `localhost` e exportando para o Grafana Cloud — confirma a sua hipótese de pesquisa. Mantém credenciais fora do código da app e centraliza batching/retry. |
| **Logs do backend** | Já estão **bons** (JSON estruturado via Zap → stdout). No Cloud Run isso já cai no **Cloud Logging de graça**. Mantemos lá e expomos no Grafana via *datasource*, sem reprocessar. |
| **Firebase / Vercel** | **Ajudam** (grátis, nativos, ótimos no que fazem) mas **silam** dados em consoles separados. Estratégia: **manter** Crashlytics/Web Vitals e **fazer bridge** apenas dos KPIs canônicos para o backend unificado. Detalhe na §7. |

**TL;DR da arquitetura recomendada:**

```
┌─────────────────┐   OTLP    ┌──────────────────────┐
│ Backend (Go)    │──────────▶│ Sidecar Alloy/OTelCol│──┐
│ Cloud Run       │ localhost │ (mesmo serviço CR)   │  │
└─────────────────┘           └──────────────────────┘  │
        │ stdout (JSON)                                  │ remote_write / OTLP
        ▼                                                ▼
   Cloud Logging  ◀── datasource ───────────┐   ┌──────────────────────┐
   Cloud Monitoring ◀── datasource ─────────┼──▶│  GRAFANA CLOUD (Free) │
                                            │   │  Dashboards unificados│
   Vercel (web) ──@vercel/otel──────────────┤   └──────────────────────┘
   Mobile ──OTel RN + Crashlytics/GA4───────┘
```

---

## 2. Estado atual dos logs (mapeamento)

### 2.1 Backend (`personal-finance`)

**Stack de logging:** wrapper próprio em `pkg/log/` sobre **Zap** (`go.uber.org/zap v1.27.0`).

| Arquivo | Papel |
|---|---|
| `pkg/log/logger.go` | Interface `Logger`, logger global (`log.Info/Error/...`) e funções package-level. |
| `pkg/log/zap.go` | Implementação Zap. Encoder JSON ou console; chaves: `timestamp`, `level`, `message`, `caller`, `stacktrace`. Tempo ISO8601. |
| `pkg/log/config.go` | Opções `WithLevel/WithFormat/WithOutput`. Default: nível `info`, formato **JSON**, saída `stdout`. |
| `pkg/log/context.go` | Propagação do logger via `context.Context` (`InfoContext`, `ErrorContext`, ...). |
| `pkg/log/field.go` | Campos tipados (`String`, `Int`, `Err`, ...) — abstração sobre `zap.Field`. |
| `pkg/log/middleware.go` | `GinLoggerMiddleware`: injeta `request_id` (header `X-Request-ID` ou UUID novo), loga `method`+`path` no início (`"request"`) e `status` no fim (`"response"`). |
| `pkg/log/writer.go` | Adapter `io.Writer` para capturar saída do Gin (`gin.DefaultWriter/DefaultErrorWriter`) com `component=gin`. |

**Configuração em runtime (`cmd/api/main.go`):**
- `LOG_LEVEL` (default `info`) e `LOG_FORMAT` (default `text` no main, **mas JSON no default da lib** — ver §2.3 inconsistência).
- Logger global setado em `configureLogger()`; middleware aplicado em `setupGin()`.
- `gin.Recovery()` ativo, mas **panics não são logados de forma estruturada** pelo nosso logger (vão pelo writer do Gin).

**Volume/cobertura de logs (contagem por chamada):**

| Chamada | Ocorrências |
|---|---|
| `log.InfoContext` | 11 |
| `log.ErrorContext` | 9 |
| `log.Info` | 5 |
| `log.Debug` | 5 |
| `log.Error` | 4 |
| `log.Fatal` | 3 |
| `log.WarnContext` | 1 |
| **Total** | **~38 pontos de log** em ~módulos de negócio |

> Cobertura **baixa e irregular**: a maior parte da observabilidade hoje vem do middleware HTTP (1 log de entrada + 1 de saída por request). Camadas de repositório, jobs e integrações externas (Firebase, agente de IA, push) têm pouco log.

### 2.2 Métricas / Health / Trace — **inexistentes hoje**

- **Métricas:** não há instrumentação. Nenhum `/metrics`, nenhum Prometheus/OTel SDK ativo. As únicas métricas disponíveis hoje são as **nativas do Cloud Run** (request count, latências, CPU/mem, instâncias) já visíveis no Cloud Monitoring — porém **ninguém olha** e não há dashboard.
- **Health check:** existe apenas `GET /ping` → `"pong"` (sem checagem de DB/dependências). Não há `/healthz`, `/readyz` nem `/livez`.
- **Trace:** ausente. OTel está no `go.mod` apenas como dependência **indireta**.

### 2.3 Problemas/gaps identificados no backend

1. **Inconsistência de formato:** `main.go` usa default `text`, mas produção precisa de **JSON** para o Cloud Logging parsear campos. Em produção deve ser **sempre JSON**.
2. **Sem `severity` no padrão do Cloud Logging:** Zap emite `level` em lowercase (`info`, `error`). O Cloud Logging espera o campo **`severity`** (`INFO`, `ERROR`) para colorir/filtrar corretamente. Hoje os logs entram como `INFO` "burros".
3. **Sem `trace`/`span_id` nos logs** → impossível correlacionar log ↔ request ↔ futura trace.
4. **`request_id` não vira label de métrica nem é propagado para o cliente de forma consistente** (é setado no header só quando ausente).
5. **Sem log de panic estruturado** (recovery silencioso do ponto de vista do nosso logger).
6. **Sem campos canônicos** padronizados (ex.: `user_id`, `feature`, `latency_ms`) — dificulta dashboards e alertas por feature.
7. **Health check raso** — `/ping` não detecta DB fora do ar; Cloud Run não consegue tirar instância ruim de rotação.

### 2.4 Web e Mobile (premissas)

> Não temos os repositórios `personal-finance-front-v2` e `personal-finance-mobile` no escopo desta sessão. As linhas abaixo são **premissas a validar** (o desenho não depende delas em detalhe):

- **Web (`front-v2`)**: app **Next.js/React deploy na Vercel**. Observabilidade nativa = **Vercel Analytics + Speed Insights** (Core Web Vitals/RUM) e logs de função/runtime na Vercel.
- **Mobile (`mobile`)**: app **React Native / Expo** com **Firebase**. Observabilidade nativa = **Crashlytics** (crashes), **Performance Monitoring** e **Google Analytics for Firebase (GA4)** para eventos de produto.

---

## 3. Princípios de desenho

1. **Vendor-neutral primeiro:** instrumentar com **OpenTelemetry**. Trocar de backend (Grafana ↔ GCP ↔ outro) deve ser mudança de config, não de código.
2. **O mais grátis possível:** preferir tiers gratuitos e dados que **já existem** (Cloud Logging, métricas nativas do Cloud Run, Crashlytics, Web Vitals) antes de gerar custo novo.
3. **Uma camada de visualização:** um único lugar para abrir dashboards (Grafana), mesmo que os dados morem em backends diferentes — *via datasources*, não migração.
4. **Padrão > volume:** poucos campos/métricas **canônicos e consistentes** nos 3 apps valem mais que muitos sinais divergentes.
5. **Esforço incremental:** entregar valor em fases; nada de big-bang.

---

## 4. Comparativo de backends (a decisão central de custo)

| Critério | **A) Google Cloud Operations** (Logging + Monitoring) | **B) Grafana Cloud Free** | **C) OTel Collector + stack self-hosted** |
|---|---|---|---|
| Custo | Logs: **50 GiB/mês grátis**/projeto; métricas GCP grátis; custom metrics têm franquia pequena (50 MiB). | **Free:** 10k séries de métrica, 50 GB logs, 50 GB traces, 14 dias retenção, 3 usuários. | "Grátis" em licença, mas **paga em infra + operação** (VM/k8s, backups). |
| Esforço inicial | **~zero** no backend (logs do stdout já entram; métricas do Cloud Run já existem). | Baixo/médio (criar conta, sidecar, tokens). | Alto. |
| Unifica web+mobile+backend? | **Fraco** — pensado p/ GCP; Vercel/Firebase ficam de fora dos mesmos painéis. | **Forte** — recebe OTLP de qualquer origem + datasources p/ GCP/BigQuery. | Forte, mas você opera tudo. |
| Retenção | Logs configurável (custo); métricas até 24 meses. | **14 dias** no free (limitação real p/ análise trimestral). | Você decide (e paga). |
| Alertas | Bom; **atenção:** GCP passa a cobrar alerting a partir de ~set/2026 ($0,35/métrica referenciada). | Incluso no free (com limites). | Você monta (Alertmanager). |
| Lock-in | Alto (GCP). | Baixo (OSS/OTel). | Nenhum. |

**Recomendação:** **B como camada de visualização unificada**, **reaproveitando A como fonte barata de logs/infra do backend** (via datasource do Grafana para Cloud Monitoring/Logging). Ou seja, **híbrido A+B**: não pagamos para mover logs do backend (ficam no Cloud Logging grátis), e o Grafana vira o "single pane" que junta GCP + Vercel + Firebase. **C fica descartado** por contrariar o objetivo "mais grátis/menos esforço" (operar stack próprio custa tempo).

> **Decisão a confirmar:** se a unificação dos 3 apps **não** for prioridade agora, a Opção **A pura** entrega 80% do valor no backend com esforço quase zero. A Opção **B** é o que viabiliza "todos nos mesmos dashboards".

---

## 5. Arquitetura proposta — Backend (Cloud Run + sidecar)

### 5.1 Por que sidecar (e não exportar direto do app)

O Cloud Run suporta **múltiplos contêineres no mesmo serviço** (sidecars), compartilhando rede via `localhost`. O sidecar de coleta:

- Recebe **OTLP** do app em `localhost:4317` (gRPC) / `4318` (HTTP) — o app **não** conhece o backend final nem guarda tokens.
- Faz **batching, retry, buffering** e adiciona *resource attributes* (versão, revisão do Cloud Run, região).
- Centraliza o **único ponto** que tem o token do Grafana Cloud (via Secret Manager).
- Permite trocar o destino (Grafana ↔ GCP ↔ outro) sem recompilar a app.

**Imagem do sidecar:** recomendo **Grafana Alloy** (distribuição da Grafana, fala OTLP, Prometheus `remote_write`, Loki push) ou o **OpenTelemetry Collector contrib**. Para Grafana Cloud, Alloy tende a dar menos atrito de config.

### 5.2 Fluxo de dados

| Sinal | Caminho | Backend final | Custo |
|---|---|---|---|
| **Logs** | App → `stdout` (JSON) → Cloud Run → **Cloud Logging** | Cloud Logging (grátis até 50 GiB) + datasource no Grafana | $0 (dentro da franquia) |
| **Métricas** | App → OTel SDK → **sidecar** → Grafana Cloud (`remote_write`) | Grafana Cloud (Prometheus) | $0 (até 10k séries) |
| **Métricas de infra** | Cloud Run nativo → Cloud Monitoring → datasource no Grafana | Cloud Monitoring | $0 |
| **Trace** (fase futura) | App → OTel SDK → sidecar → Grafana Tempo | Grafana Cloud | $0 (até 50 GB) |

> **Nota técnica importante (logs):** no Cloud Run, um sidecar **não lê o stdout** do contêiner da app. Por isso a estratégia de logs é **deixar no Cloud Logging** (já gratuito e automático) e **consultá-los no Grafana via datasource Google Cloud Logging** — sem duplicar nem pagar. Se no futuro quisermos logs **dentro** do Loki (Grafana), aí sim adicionamos um *exporter* OTLP de logs no Zap (core customizado), mas isso é **opcional** e fora do MVP.

### 5.3 Mudanças necessárias no código do backend

1. **Forçar JSON em produção** e adicionar **`severity`** compatível com Cloud Logging (mapear `level` Zap → `severity` GCP). Pequena alteração em `pkg/log/zap.go`/`config.go`.
2. **Correlação:** incluir `trace_id`/`span_id` (e `logging.googleapis.com/trace`) nos campos do log quando houver contexto OTel — habilita "log ↔ trace" no futuro sem retrabalho.
3. **Pacote `pkg/metrics/`** (novo) inicializando o **OTel MeterProvider** com exporter OTLP para `localhost:4318`. Endereço e on/off via env (`OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_SDK_DISABLED`).
4. **Middleware de métricas HTTP** (irmão do `GinLoggerMiddleware`) emitindo as métricas RED por rota (§6.1). ~1 arquivo, plugado em `setupGin()`.
5. **Health checks reais:** `GET /healthz` (liveness) e `GET /readyz` (checa `db.Ping()` + dependências críticas). Configurar no Cloud Run como *startup/liveness probe*.
6. **Helper de métricas de negócio** (`metrics.BusinessCounter(...)`) para os times instrumentarem KPIs sem tocar no SDK (§6.2). Pontos de injeção naturais: os `Setup(...)` em `internal/bootstrap/*` e os usecases.
7. **Log de panic estruturado:** custom recovery que loga via `log.Error` com stacktrace antes de devolver 500.

> Esforço estimado do MVP backend: **pequeno-médio** (a base de log já é boa; o trabalho concentra-se no `pkg/metrics`, middleware e health checks).

---

## 6. Métricas padrão (catálogo)

### 6.1 Saúde / "Golden Signals" (RED + USE)

Aplicar o método **RED** por endpoint e **USE** para recursos. Nomes seguem convenção OTel/Prometheus.

| Métrica | Tipo | Labels | Para quê |
|---|---|---|---|
| `http_server_requests_total` | counter | `method`, `route`, `status_class` (2xx/4xx/5xx) | **Rate** + **Errors** (taxa de erro) |
| `http_server_request_duration_seconds` | histogram | `method`, `route` | **Duration** (p50/p90/p99 de latência) |
| `http_server_active_requests` | gauge | `route` | concorrência / saturação |
| `db_query_duration_seconds` | histogram | `operation`, `table` | latência de DB (GORM) |
| `db_errors_total` | counter | `operation` | erros de persistência |
| `db_pool_in_use` / `db_pool_idle` | gauge | — | saturação do pool de conexões |
| `external_call_duration_seconds` | histogram | `dependency` (firebase, agent/genai, push) | latência de dependências externas |
| `external_call_errors_total` | counter | `dependency` | falhas em dependências externas |
| `app_panics_total` | counter | `route` | estabilidade |
| **Infra (Cloud Run nativo)** | — | — | request count, instance count, **cold starts**, CPU/mem utilization, billable time |

**SLOs sugeridos (iniciais):**
- Disponibilidade da API: **≥ 99,5%** (erro 5xx / total).
- Latência: **p95 < 500 ms** nos endpoints de leitura; **p95 < 1 s** nos de escrita.
- Erro de jobs internos: **< 1%**.

### 6.2 Métricas de negócio (KPIs do produto)

Estas são as que diferenciam um app financeiro e devem ser **idênticas nos 3 clientes** (backend é a fonte da verdade; front/mobile complementam com eventos de UX). Mapeadas às features existentes em `internal/bootstrap/*`:

| KPI | Métrica | Labels | Fonte (feature) |
|---|---|---|---|
| Cadastros / ativação | `users_provisioned_total` | — | `LazyProvisionUser` / `user` |
| Usuários ativos (DAU/WAU/MAU) | derivado de `request`/eventos | `user_id` (cardinalidade!) | middleware / GA4 |
| Lançamentos criados | `movements_created_total` | `type` (income/expense) | `movement` |
| Carteiras criadas | `wallets_created_total` | — | `wallet` |
| Faturas de cartão | `invoices_generated_total`, `invoices_paid_total` | — | `creditcard`/`invoice` |
| **Assinaturas** | `subscriptions_active`, `subscription_trials_started_total`, `subscription_conversions_total`, `subscription_churn_total` | `plan` | `subscription` |
| **Receita (MRR)** | `mrr_amount` (gauge) | `plan`, `currency` | `subscription` |
| Cupons resgatados | `coupons_redeemed_total` | `coupon` | `coupon` |
| **Uso do agente de IA** | `agent_requests_total`, `agent_tokens_total`, `agent_request_duration_seconds`, `agent_errors_total` | `model` | `agent` (genai/ADK) |
| Push notifications | `push_sent_total`, `push_failed_total` | `type` | `pushnotifications`/`device` |
| Limites de plano atingidos | `plan_limit_hits_total` | `limit_type` | `limits` (forte sinal de conversão!) |
| Exports / exclusão de conta | `exports_total`, `account_deletions_total` | — | `export`/`deleteaccount` |

> **Atenção a cardinalidade:** **não** usar `user_id` como label de métrica (explode séries → estoura o free tier de 10k). DAU/MAU faz-se via **eventos/logs** (Cloud Logging ou GA4), não via labels de métrica.

### 6.3 Web (RUM) e Mobile

| Sinal | Web (Vercel) | Mobile (Firebase) |
|---|---|---|
| Performance/UX | **Core Web Vitals** (LCP, INP, CLS) via Speed Insights / `web-vitals` → OTLP | **Performance Monitoring** (app start, telas, traces de rede) |
| Erros/crashes | erros JS (Sentry-like ou OTel) | **Crashlytics** (crash-free users %) |
| Negócio | mesmos KPIs canônicos (§6.2) via eventos | GA4 + eventos canônicos via OTel |
| Engajamento | page views, funil | DAU/MAU, retenção (GA4) |

---

## 7. Firebase e Vercel: interfere, ajuda ou atrapalha?

**Resposta curta: ajudam muito no que são bons, mas atrapalham a unificação por silarem dados.** A estratégia é **não brigar com eles** — usá-los para o que fazem melhor e **fazer bridge só dos KPIs canônicos**.

### 7.1 Firebase (mobile)
- **Ajuda:** **Crashlytics** é best-in-class em crash e é **grátis** — não há motivo para substituir. **GA4 for Firebase** é grátis e ótimo para funil/retenção/eventos de produto. **Performance Monitoring** cobre app-start e rede sem código pesado.
- **Atrapalha:** tudo vive no **console do Firebase/GA4**, separado dos dashboards do backend. Métricas de produto não conversam nativamente com as de saúde da API.
- **Bridge recomendada:** habilitar **export do GA4 para o BigQuery** (grátis no sandbox) e plugar o **datasource BigQuery no Grafana** → os KPIs do mobile aparecem ao lado dos do backend. Crashlytics permanece no Firebase (e, se quiser alertas unificados, exportar via BigQuery/Functions também).

### 7.2 Vercel (web)
- **Ajuda:** **Speed Insights** (Core Web Vitals reais de usuários) e **Web Analytics** são plug-and-play. Logs de runtime/edge úteis para debug.
- **Atrapalha:** no plano **Hobby/Free** as integrações de *drain*/OTel para exportar telemetria são **limitadas** (recursos de export costumam ser de planos pagos); retenção curta; dados presos no painel da Vercel.
- **Bridge recomendada:** instrumentar o app com **`@vercel/otel` + `web-vitals`** e **exportar OTLP** direto para o **Grafana Cloud** (não depende do drain pago da Vercel). Assim Web Vitals e eventos de negócio do front entram nos **mesmos dashboards**. Manter o Vercel Analytics como visão rápida complementar.

### 7.3 Veredito
| Ferramenta | Manter? | Papel | Bridge p/ Grafana |
|---|---|---|---|
| Crashlytics | ✅ Sim | Crashes mobile | Opcional (BigQuery) |
| GA4 / Firebase | ✅ Sim | Funil/retenção/produto | **Sim** (export BigQuery → datasource) |
| Vercel Speed Insights | ✅ Sim | Visão rápida Web Vitals | Complementar |
| Vercel telemetry export pago | ❌ Não | — | Substituído por `@vercel/otel` → OTLP |

> Conclusão: **não interferem na escolha do backend** — eles **complementam**. O único cuidado é **não duplicar custo** (ex.: não pagar Vercel para exportar algo que `@vercel/otel` faz de graça) e **definir a fonte da verdade** de cada KPI para evitar números divergentes.

---

## 8. Dashboards unificados propostos (no Grafana)

1. **Visão Geral / Saúde (RED)** — tráfego, taxa de erro e latência da API; cold starts e instâncias do Cloud Run; crash-free % (mobile) e Web Vitals (web) lado a lado.
2. **Negócio / Produto** — lançamentos, carteiras, faturas, cupons; DAU/MAU; funil de assinatura (trial → conversão → churn) e **MRR**; `plan_limit_hits` (sinal de upsell).
3. **Dependências** — latência/erros de Firebase, agente de IA (tokens/custo) e push.
4. **Mobile (Firebase/BigQuery)** — crashes, performance, eventos GA4.
5. **Web (Vercel/OTel)** — Web Vitals, erros JS, eventos de negócio.

Tudo com **variáveis de template** (`environment`, `app`, `route`) e **filtro por ambiente** (prod/staging).

---

## 9. Plano de implementação faseado

| Fase | Entregas | Dependências | Esforço |
|---|---|---|---|
| **0 — Quick wins (sem código novo de métrica)** | Ativar dashboard padrão do Cloud Run no Cloud Monitoring; criar projeto Grafana Cloud Free; conectar datasources **Cloud Monitoring + Cloud Logging**. Já dá visão de saúde do backend. | Acesso GCP/Grafana | XS |
| **1 — Padronizar logs do backend** | Forçar JSON em prod; mapear `severity` p/ Cloud Logging; adicionar `trace_id`/`span_id`; log de panic; campos canônicos (`feature`, `user_id`, `latency_ms`). | `pkg/log` | S |
| **2 — Métricas de saúde + sidecar** | `pkg/metrics` (OTel SDK); middleware RED; `/healthz` e `/readyz`; **sidecar Alloy** no Cloud Run + Secret do token Grafana; dashboards RED. | Fase 1 | M |
| **3 — Métricas de negócio** | Instrumentar KPIs (§6.2) nos usecases/bootstrap; dashboard de Negócio; export GA4→BigQuery; datasource BigQuery. | Fase 2 | M |
| **4 — Web + Mobile** | `@vercel/otel`+`web-vitals` no front; OTel RN + manter Crashlytics no mobile; consolidar dashboards cross-app. | Fases 2-3 | M |
| **5 — Alertas & SLOs** | Definir SLOs (§6.1); alertas no Grafana (erro 5xx, p95 latência, crash-free %, falha de job, MRR drop). **Atenção** à cobrança de alerting do GCP (set/2026) — preferir alertas no Grafana. | Fase 2+ | S |
| **6 — Trace (opcional, baixa prioridade)** | Habilitar OTel tracing → Tempo via sidecar; correlação log↔trace já preparada na Fase 1. | Fase 2 | M |

---

## 10. Riscos e mitigações

| Risco | Mitigação |
|---|---|
| **Cardinalidade** estoura 10k séries do Grafana Free | Proibir `user_id`/IDs em labels; revisar labels em PR; usar exemplars/eventos p/ alta cardinalidade. |
| Retenção de **14 dias** (Grafana Free) insuficiente p/ análise trimestral | KPIs de longo prazo (MRR, retenção) via GA4/BigQuery (retenção longa) ou Cloud Monitoring (até 24m). |
| **Custo surpresa** no Cloud Logging (acima de 50 GiB) | *Log routing*/exclusion filters p/ descartar logs ruidosos (ex.: health checks); manter JSON enxuto. |
| Cobrança de **alerting do GCP** a partir de ~set/2026 | Centralizar alertas no **Grafana** (incluso no free). |
| Sidecar aumenta custo/recurso do Cloud Run | Sidecar é leve; dimensionar CPU/mem mínimos; só em prod. |
| **Números divergentes** entre fontes (Vercel vs Grafana vs GA4) | Definir **fonte da verdade** por KPI (tabela §6.2) e documentar. |
| Tokens/secrets do Grafana vazarem | Apenas no **sidecar** via Secret Manager; app nunca vê o token. |

---

## 11. Decisões em aberto (precisam de confirmação)

1. **Unificação total (Opção B/Grafana) vs baseline GCP (Opção A)** — confirmar se "mesmos dashboards p/ os 3 apps" é requisito firme agora ou pode ficar p/ Fase 4.
2. **Alloy vs OTel Collector** como sidecar — recomendo **Alloy** pela integração com Grafana Cloud.
3. **Logs no Loki** (Grafana) vs **permanecer no Cloud Logging** (datasource) — recomendo permanecer no Cloud Logging no MVP (grátis, zero código).
4. **Stack real de web/mobile** — validar premissas da §2.4 para detalhar a instrumentação de front/mobile.
5. **CI/CD do deploy** — não há `.github/workflows` no repo; confirmar como o Cloud Run é deployado hoje (gcloud manual?) para incluir o sidecar no manifesto de serviço.

---

### Apêndice A — Fontes (tiers gratuitos, jun/2026)

- Grafana Cloud Free: 10k séries de métrica, 50 GB logs, 50 GB traces, 14 dias de retenção, 3 usuários — [grafana.com/pricing](https://grafana.com/pricing/), [Free tier](https://grafana.com/products/cloud/free-tier/).
- Google Cloud Observability: Cloud Logging 50 GiB/mês grátis por projeto; métricas GCP grátis; alerting passa a ser cobrado a partir de ~set/2026 — [cloud.google.com/stackdriver/pricing](https://cloud.google.com/stackdriver/pricing).
