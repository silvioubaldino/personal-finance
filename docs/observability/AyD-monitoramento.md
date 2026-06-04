# AyD — Análise e Desenho de Monitoramento (Observabilidade)

> **Status:** Proposta para discussão
> **Escopo:** `personal-finance` (backend Go/Cloud Run), `personal-finance-front-v2` (web/Vercel), `personal-finance-mobile` (mobile/Firebase)
> **Objetivo:** Padronizar **logs** e **métricas** entre os três apps, com o **menor custo possível** (idealmente $0), consolidando tudo em **dashboards unificados**. Trace é desejável, porém **baixa prioridade** nesta fase.

---

## 1. Sumário executivo

| Tema | Decisão proposta |
|---|---|
| **Padrão de instrumentação** | **OpenTelemetry (OTel)** como contrato comum nos 3 apps. Já é dependência transitiva do backend (via Google ADK / `go.opentelemetry.io/otel`). Vendor-neutral evita lock-in. |
| **"Single pane of glass"** | **Grafana Cloud (Free tier)** como camada única de visualização. Conecta dados de GCP, Firebase e Vercel via *datasource*, sem migrar tudo. |
| **Métricas de SAÚDE técnica** | **Grafana Cloud Free** (RED/USE, histogramas de latência). Retenção de 14 dias é suficiente — a pergunta é "como está agora/esta semana". |
| **KPIs de NEGÓCIO** ⭐ **(decidido)** | **Cloud Monitoring (custom metrics)** — retenção de **24 meses**, cabe no **free tier (150 MiB/mês)** por serem counters/gauges de baixa cardinalidade e push infrequente. Resolve o limite de 14 dias do Grafana para análise de tendência (MRR, churn, retenção). Lidos no Grafana via datasource. Detalhe na §6.2. |
| **Coleta no Cloud Run** | **Sidecar OpenTelemetry Collector** (preferência: distribuição **Google-built para Cloud Run**) recebendo OTLP em `localhost` e roteando: histogramas → Grafana; KPIs de negócio → Cloud Monitoring. Mantém credenciais fora do código e centraliza batching/retry. |
| **Logs do backend** | Já estão **bons** (JSON estruturado via Zap → stdout). No Cloud Run isso já cai no **Cloud Logging de graça**. Mantemos lá e expomos no Grafana via *datasource*, sem reprocessar. |
| **Firebase / Vercel** | **Ajudam** (grátis, nativos, ótimos no que fazem) mas **silam** dados em consoles separados. Estratégia: **manter** Crashlytics/Web Vitals e **fazer bridge** apenas dos KPIs canônicos para o backend unificado. Detalhe na §7. |

**TL;DR da arquitetura recomendada:**

```
┌─────────────────┐   OTLP    ┌──────────────────────┐ histogramas/saúde
│ Backend (Go)    │──────────▶│ Sidecar OTel Collector│─────────────┐
│ Cloud Run       │ localhost │ (mesmo serviço CR)   │              │
└─────────────────┘           └──────────┬───────────┘              │
        │ stdout (JSON)        KPIs negócio│                         ▼
        ▼                                  ▼              ┌──────────────────────┐
   Cloud Logging                    Cloud Monitoring      │  GRAFANA CLOUD (Free) │
   (logs)                           (custom metrics,      │  Dashboards unificados│
        │                            KPIs, 24 meses)      │  (single pane)        │
        └────── datasource ──────────────┴───────────────▶│                       │
   Vercel (web) ──@vercel/otel─────────────────────────── │                       │
   Mobile ──OTel RN + Crashlytics/GA4(→BigQuery)─────────▶└──────────────────────┘
```
> Roteamento de métrica no sidecar: **histogramas de saúde técnica → Grafana** (14d basta); **counters/gauges de negócio → Cloud Monitoring** (24 meses). Grafana lê Cloud Monitoring/Logging/BigQuery por *datasource* e vira o painel único.

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
| Custo | Logs: **50 GiB/mês grátis**/projeto; métricas GCP grátis; custom metrics: **150 MiB/mês grátis** por billing account, depois ~$0,258/MiB (80 bytes/ponto). | **Free:** 10k séries de métrica, 50 GB logs, 50 GB traces, 14 dias retenção, 3 usuários. | "Grátis" em licença, mas **paga em infra + operação** (VM/k8s, backups). |
| Esforço inicial | **~zero** no backend (logs do stdout já entram; métricas do Cloud Run já existem). | Baixo/médio (criar conta, sidecar, tokens). | Alto. |
| Unifica web+mobile+backend? | **Fraco** — pensado p/ GCP; Vercel/Firebase ficam de fora dos mesmos painéis. | **Forte** — recebe OTLP de qualquer origem + datasources p/ GCP/BigQuery. | Forte, mas você opera tudo. |
| Retenção | Logs configurável (custo); métricas até 24 meses. | **14 dias** no free (limitação real p/ análise trimestral). | Você decide (e paga). |
| Alertas | Bom; **atenção:** GCP passa a cobrar alerting a partir de ~set/2026 ($0,35/métrica referenciada). | Incluso no free (com limites). | Você monta (Alertmanager). |
| Lock-in | Alto (GCP). | Baixo (OSS/OTel). | Nenhum. |

**Recomendação (decidida):** **híbrido A+B**, dividindo as métricas por **horizonte temporal da pergunta**:
- **Saúde técnica** (histogramas de latência, RED/USE) → **Grafana Free**. 14 dias de retenção bastam para "está saudável agora/esta semana".
- **KPIs de negócio** (counters/gauges: MRR, churn, lançamentos, DAU) → **Cloud Monitoring (custom metrics)**, que tem **24 meses de retenção** e cabe no **free tier (150 MiB/mês)** porque são poucas séries e com push infrequente (1–5 min). Isso resolve o limite de 14 dias do Grafana, que é o ponto fraco para análise de negócio.
- **Logs** ficam no **Cloud Logging** (grátis, automático).
- O **Grafana é o "single pane"**: lê Cloud Monitoring + Cloud Logging + BigQuery(GA4) + OTLP(Vercel) por *datasource* e junta GCP + Vercel + Firebase num só lugar.

**C fica descartado** por contrariar o objetivo "mais grátis/menos esforço" (operar stack próprio custa tempo).

> **Por que não tudo no Grafana?** Métrica no Grafana Free tem **14 dias** de retenção (vale para tudo, inclusive métricas — não só logs). Inviável para tendência de negócio.
> **Por que não tudo no Cloud Monitoring?** Custom metrics são cobradas por volume (80 bytes/ponto); **histogramas de latência cobram 1 ponto por bucket** e estouram rápido o free tier (ex.: 10 rotas × 15 buckets a 60s ≈ 6M pontos/mês ≈ ~US$ 89/mês). Por isso histograma fica no Grafana. Detalhe e orçamento na §6.2.

---

## 5. Arquitetura proposta — Backend (Cloud Run + sidecar)

### 5.1 Por que sidecar (e não exportar direto do app)

O Cloud Run suporta **múltiplos contêineres no mesmo serviço** (sidecars), compartilhando rede via `localhost`. O sidecar de coleta:

- Recebe **OTLP** do app em `localhost:4317` (gRPC) / `4318` (HTTP) — o app **não** conhece o backend final nem guarda tokens.
- Faz **batching, retry, buffering** e adiciona *resource attributes* (versão, revisão do Cloud Run, região).
- Centraliza o **único ponto** que tem o token do Grafana Cloud (via Secret Manager).
- Permite trocar o destino (Grafana ↔ GCP ↔ outro) sem recompilar a app.

**Imagem do sidecar:** recomendo **OpenTelemetry Collector** — de preferência a distribuição **"Google-built OpenTelemetry Collector for Cloud Run"** (ou o `otelcol-contrib`). Motivo decisivo: o exporter para **Cloud Monitoring** (`googlecloud`/`googlemanagedprometheus`) é **first-class** no Collector, enquanto no Grafana Alloy o `otelcol.exporter.googlecloud` é *community component* (exige a flag `--feature.community-components.enabled=true` e não tem suporte oficial). Como os KPIs de negócio vão para o Cloud Monitoring, esse exporter é peça central. O lado Grafana não sofre: exporta-se via `otlphttp` para o endpoint OTLP do Grafana Cloud. **Alloy só compensaria** se fôssemos *all-in* no ecossistema Grafana (Loki/Pyroscope/Faro/Fleet Management) — o que não é o caso deste desenho.

### 5.2 Fluxo de dados

| Sinal | Caminho | Backend final | Custo |
|---|---|---|---|
| **Logs** | App → `stdout` (JSON) → Cloud Run → **Cloud Logging** | Cloud Logging (grátis até 50 GiB) + datasource no Grafana | $0 (dentro da franquia) |
| **Métricas de saúde** (histogramas RED/USE) | App → OTel SDK → **sidecar** → Grafana Cloud (`remote_write`) | Grafana Cloud (Prometheus, 14d) | $0 (até 10k séries) |
| **KPIs de negócio** (counters/gauges) | App → OTel SDK → **sidecar** → **Cloud Monitoring** (custom metrics) → datasource no Grafana | Cloud Monitoring (**24 meses**) | $0 (dentro de 150 MiB/mês a 1–5 min) |
| **Métricas de infra** | Cloud Run nativo → Cloud Monitoring → datasource no Grafana | Cloud Monitoring | $0 |
| **Trace** (fase futura) | App → OTel SDK → sidecar → Grafana Tempo | Grafana Cloud | $0 (até 50 GB) |

> O **sidecar roteia por destino**: histogramas de saúde para o Grafana (`otlphttp`) e os instrumentos de negócio para o Cloud Monitoring (`googlecloud`). No OTel Collector isso se faz com **pipelines distintos** + `routing`/`filter` processor por nome/prefixo de métrica (ex.: prefixo `biz_`); opcionalmente via *views* do MeterProvider no próprio app.

> **Nota técnica importante (logs):** no Cloud Run, um sidecar **não lê o stdout** do contêiner da app. Por isso a estratégia de logs é **deixar no Cloud Logging** (já gratuito e automático) e **consultá-los no Grafana via datasource Google Cloud Logging** — sem duplicar nem pagar. Se no futuro quisermos logs **dentro** do Loki (Grafana), aí sim adicionamos um *exporter* OTLP de logs no Zap (core customizado), mas isso é **opcional** e fora do MVP.

### 5.3 Mudanças necessárias no código do backend

1. **Forçar JSON em produção** e adicionar **`severity`** compatível com Cloud Logging (mapear `level` Zap → `severity` GCP). Pequena alteração em `pkg/log/zap.go`/`config.go`.
2. **Correlação:** incluir `trace_id`/`span_id` (e `logging.googleapis.com/trace`) nos campos do log quando houver contexto OTel — habilita "log ↔ trace" no futuro sem retrabalho.
3. **Pacote `pkg/metrics/`** (novo) inicializando o **OTel MeterProvider** com exporter OTLP para `localhost:4318`. Endereço e on/off via env (`OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_SDK_DISABLED`).
4. **Middleware de métricas HTTP** (irmão do `GinLoggerMiddleware`) emitindo as métricas RED por rota (§6.1). ~1 arquivo, plugado em `setupGin()`.
5. **Health checks reais:** `GET /healthz` (liveness) e `GET /readyz` (checa `db.Ping()` + dependências críticas). Configurar no Cloud Run como *startup/liveness probe*.
6. **Helper de métricas de negócio** (`metrics.BusinessCounter(...)`) para os times instrumentarem KPIs sem tocar no SDK (§6.2). Convenção: prefixo **`biz_`** no nome (permite ao sidecar rotear esses para o **Cloud Monitoring**, 24 meses). Pontos de injeção naturais: os `Setup(...)` em `internal/bootstrap/*` e os usecases.
7. **Log de panic estruturado:** custom recovery que loga via `log.Error` com stacktrace antes de devolver 500.

> Esforço estimado do MVP backend: **pequeno-médio** (a base de log já é boa; o trabalho concentra-se no `pkg/metrics`, middleware e health checks).

### 5.4 Modelo operacional de deploy (manual, via console)

**Como é hoje:** não há CI/CD automatizado. Um **trigger do Cloud Build** gera uma nova imagem a cada **tag nova no GitHub** que casa com um regex; o **deploy é manual no console** do Cloud Run, com as configs feitas pela interface. O sidecar e suas configs serão criados/mantidos por aí.

**O que isso exige do desenho do sidecar:**

| Item | Decisão |
|---|---|
| Imagem do sidecar | Imagem **própria** do OTel Collector (Google-built + nossa config), **versionada no repo** (ex.: `deploy/otel-collector/config.yaml`) e **construída pelo mesmo padrão de tag→Cloud Build** que já usamos. Mantém a config como código e reproduzível, sem editar YAML solto no console. |
| Config do Collector | **Baked na imagem** (reprodutível) **ou** montada via **Secret Manager** como volume (permite editar no console sem rebuild). Recomendo *baked* para versionar; o token do Grafana **nunca** vai junto. |
| Segredos (token Grafana, etc.) | **Secret Manager** referenciado como **env var** no contêiner do sidecar, configurado na interface do Cloud Run. App e imagem nunca contêm o token. |
| Multi-contêiner no Cloud Run | Cloud Run suporta **sidecar pelo console** ("Editar e implantar nova revisão" → adicionar contêiner). O contêiner de **ingress continua sendo o app** (escuta `$PORT`); o Collector é sidecar **sem ingress**, recebendo OTLP em `localhost:4318`. |
| Ordem de inicialização | Opcional: `depends_on`/startup do app no Collector. **Não é crítico** — o exporter OTel do app faz *retry/buffer* se o Collector ainda não subiu; evita acoplar o boot. |
| Recursos | Sidecar enxuto (ex.: ~128–256 MiB / fração de vCPU). Lembrar que o Cloud Run **fatura a soma** dos contêineres. |
| Probes | `/readyz` e `/healthz` do **app** como startup/liveness probe (config na interface). |

> **Fluxo de release resultante:** tag no GitHub → Cloud Build gera (a) imagem do app e (b) imagem do sidecar → no console do Cloud Run, nova revisão referencia as duas imagens + env de Secret Manager. Continua **manual**, mas com **config de observabilidade versionada** no repo.
>
> *Evolução futura (fora do escopo agora):* este é um ponto natural para introduzir `gcloud run services replace service.yaml` (manifesto versionado) e, mais adiante, automação de deploy — mas mantemos o fluxo manual atual por ora.

---

## 6. Métricas padrão (catálogo)

### 6.1 Saúde / "Golden Signals" (RED + USE)

Aplicar o método **RED** por endpoint e **USE** para recursos. Nomes seguem convenção OTel/Prometheus. **Destino: Grafana Cloud Free** (retenção de 14 dias é suficiente para saúde técnica). **Os histogramas (`*_duration_*`) ficam fora do Cloud Monitoring** — lá custam 1 ponto por bucket e estouram o free tier.

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

Estas são as que diferenciam um app financeiro e devem ser **idênticas nos 3 clientes** (backend é a fonte da verdade; front/mobile complementam com eventos de UX). **Destino: Cloud Monitoring (custom metrics), retenção de 24 meses** — é o que viabiliza análise de tendência (MRR mês a mês, churn trimestral) impossível nos 14 dias do Grafana. Convenção de nome: prefixo **`biz_`** para o sidecar rotear ao Cloud Monitoring. Mapeadas às features existentes em `internal/bootstrap/*`:

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

**Orçamento de custo no Cloud Monitoring (free tier = 150 MiB/mês ≈ 1,97M pontos a 80 bytes):**

| | Valor |
|---|---|
| Séries estimadas (todos os KPIs acima) | **~60** |
| Frequência de push (negócio não muda a cada segundo) | **5 min** → 8.640 amostras/mês por série |
| Volume mensal | 60 × 8.640 × 80 bytes ≈ **~40 MiB/mês** |
| **% do free tier (150 MiB)** | **~26%** → cabe com folga, $0 |

> Conta de margem: a 5 min sobram ~227 séries no free tier; mesmo dobrando o catálogo de KPIs continua grátis. Subir a frequência para 60s (~43,2k amostras/mês) reduziria isso para ~45 séries — **por isso KPI de negócio usa push de 1–5 min, nunca 10–60s**.

**Guard-rails de cardinalidade/custo (viram regra de revisão de PR):**
- ❌ **Nunca** usar `user_id`, ID de cupom, ID de entidade ou rota com ID como **label** de métrica (séries ilimitadas → estoura tanto os 10k do Grafana quanto os 150 MiB do Cloud Monitoring). DAU/WAU/MAU faz-se via **eventos/logs** (Cloud Logging) ou **GA4**, não via labels.
- ❌ **Histograma só no Grafana**, nunca no Cloud Monitoring (1 ponto por bucket).
- ✅ KPI de negócio = counter/gauge de **baixa cardinalidade** + push **infrequente** (1–5 min).
- ⚠️ `coupons_redeemed_total{coupon}`: se a quantidade de cupons crescer muito, trocar `coupon` por uma categoria/agrupamento para limitar séries.

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

1. **Visão Geral / Saúde (RED)** — *fonte: Grafana (14d)* — tráfego, taxa de erro e latência da API; cold starts e instâncias do Cloud Run; crash-free % (mobile) e Web Vitals (web) lado a lado.
2. **Negócio / Produto** — *fonte: Cloud Monitoring (24 meses) via datasource* — lançamentos, carteiras, faturas, cupons; DAU/MAU; funil de assinatura (trial → conversão → churn) e **MRR**; `plan_limit_hits` (sinal de upsell). Tendência mês a mês/trimestral viável pela retenção longa.
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
| **2 — Métricas de saúde + sidecar** | `pkg/metrics` (OTel SDK); middleware RED; `/healthz` e `/readyz`; **sidecar OTel Collector** (Google-built) no Cloud Run + Secret do token Grafana; dashboards RED. | Fase 1 | M |
| **3 — Métricas de negócio** | Instrumentar KPIs (§6.2) nos usecases/bootstrap com prefixo `biz_`; **rotear `biz_*` para o Cloud Monitoring** (view/pipeline no sidecar); datasource Cloud Monitoring no Grafana; dashboard de Negócio. (Mobile/GA4→BigQuery entra na Fase 4.) | Fase 2 | M |
| **4 — Web + Mobile** | `@vercel/otel`+`web-vitals` no front; OTel RN + manter Crashlytics no mobile; consolidar dashboards cross-app. | Fases 2-3 | M |
| **5 — Alertas & SLOs** | Definir SLOs (§6.1); alertas no Grafana (erro 5xx, p95 latência, crash-free %, falha de job, MRR drop). **Atenção** à cobrança de alerting do GCP (set/2026) — preferir alertas no Grafana. | Fase 2+ | S |
| **6 — Trace (opcional, baixa prioridade)** | Habilitar OTel tracing → Tempo via sidecar; correlação log↔trace já preparada na Fase 1. | Fase 2 | M |

---

## 10. Riscos e mitigações

| Risco | Mitigação |
|---|---|
| **Cardinalidade** estoura 10k séries do Grafana / 150 MiB do Cloud Monitoring | Proibir `user_id`/IDs em labels; revisar labels em PR; usar eventos/logs p/ alta cardinalidade (guard-rails §6.2). |
| Retenção de **14 dias** (Grafana Free) insuficiente p/ análise de negócio | **Resolvido (decidido):** KPIs de negócio vão para o **Cloud Monitoring (24 meses)**, lidos no Grafana via datasource. Grafana fica só com saúde técnica (14d basta). |
| **Custo surpresa no Cloud Monitoring** (histograma/frequência alta acima de 150 MiB) | Histograma **só no Grafana**; KPI de negócio com push 1–5 min e baixa cardinalidade; monitorar volume de ingestão (orçamento §6.2 ≈ 26% do free). |
| **Custo surpresa** no Cloud Logging (acima de 50 GiB) | *Log routing*/exclusion filters p/ descartar logs ruidosos (ex.: health checks); manter JSON enxuto. |
| Cobrança de **alerting do GCP** a partir de ~set/2026 | Centralizar alertas no **Grafana** (incluso no free). |
| Sidecar aumenta custo/recurso do Cloud Run | Sidecar é leve; dimensionar CPU/mem mínimos; só em prod. |
| **Números divergentes** entre fontes (Vercel vs Grafana vs GA4) | Definir **fonte da verdade** por KPI (tabela §6.2) e documentar. |
| Tokens/secrets do Grafana vazarem | Apenas no **sidecar** via Secret Manager; app nunca vê o token. |

---

## 11. Decisões em aberto (precisam de confirmação)

1. ✅ **Divisão de métricas — DECIDIDO:** saúde técnica no **Grafana Free** (14d) e **KPIs de negócio no Cloud Monitoring** (24 meses), unificados no Grafana via datasource. (Resta confirmar apenas o cronograma da unificação de web/mobile — Fase 4.)
2. ✅ **Sidecar — DECIDIDO:** **OTel Collector** (distribuição Google-built para Cloud Run), pelo exporter Cloud Monitoring first-class. Alloy descartado por ter esse exporter em tier *community*/flag. (Detalhe na §5.1.)
3. **Logs no Loki** (Grafana) vs **permanecer no Cloud Logging** (datasource) — recomendo permanecer no Cloud Logging no MVP (grátis, zero código).
4. **Stack real de web/mobile** — validar premissas da §2.4 para detalhar a instrumentação de front/mobile.
5. ✅ **Deploy — ESCLARECIDO:** sem CI/CD automatizado. Cloud Build trigger gera imagem por **tag** (regex) no GitHub; **deploy manual no console** do Cloud Run. Sidecar e configs criados/mantidos por lá. Modelo operacional detalhado na **§5.4** (imagem do sidecar versionada no repo + token via Secret Manager + multi-contêiner pelo console).

---

### Apêndice A — Fontes (tiers gratuitos, jun/2026)

- Grafana Cloud Free: 10k séries de métrica, 50 GB logs, 50 GB traces, **14 dias de retenção (vale também para métricas)**, 3 usuários — [grafana.com/pricing](https://grafana.com/pricing/), [Free tier](https://grafana.com/products/cloud/free-tier/).
- Google Cloud Observability: Cloud Logging 50 GiB/mês grátis por projeto; métricas GCP grátis; **custom metrics: 150 MiB/mês grátis por billing account, depois ~$0,258/MiB, 80 bytes por ponto (histograma = 1 ponto/bucket), retenção até 24 meses**; alerting passa a ser cobrado a partir de ~set/2026 — [cloud.google.com/stackdriver/pricing](https://cloud.google.com/stackdriver/pricing).
- Como o volume de custom metrics é calculado (fórmula, 80 bytes/ponto, cardinalidade × frequência) — [Reduzir custos de Cloud Monitoring](https://cloud.google.com/blog/products/management-tools/learn-to-understand-and-reduce-cloud-monitoring-costs).
