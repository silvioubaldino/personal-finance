# Diagrama — Infraestrutura de Monitoramento (Observabilidade)

> Companheiro visual do [`AyD-monitoramento.md`](./AyD-monitoramento.md). Reflete as decisões já
> fechadas: OpenTelemetry como contrato comum, **sidecar OTel Collector** no Cloud Run roteando
> por tipo de sinal (**histogramas de saúde → Grafana Cloud (14d)**, **KPIs `biz_*` → Cloud
> Monitoring (24 meses)**), **logs no Cloud Logging**, e **Grafana como single pane of glass**
> lendo tudo por *datasource*. Web/mobile fazem *bridge* só dos KPIs canônicos `biz_*`.

---

## 1. Visão geral — fluxo de ponta a ponta

```mermaid
flowchart TB
    %% ===================== CLIENTES =====================
    subgraph CLI["Clientes (RUM + correlação)"]
        direction TB
        WEB["Web — frontend-v2<br/>Next.js 15 / Vercel<br/>@vercel/otel + web-vitals"]
        MOB["Mobile — Expo SDK 52 / RN<br/>Crashlytics + GA4 (a adotar)<br/>eventos OTel biz_*"]
    end

    %% ===================== CLOUD RUN =====================
    subgraph CR["Cloud Run — serviço 'personal-finance' (multi-contêiner)"]
        direction TB
        APP["Contêiner APP (Go + Gin)<br/>ingress :PORT<br/>OTel SDK + Zap (JSON)"]
        COL["Sidecar — OTel Collector<br/>(Google-built p/ Cloud Run)<br/>recebe OTLP em localhost:4318<br/>pipelines + routing por prefixo"]
        APP -- "OTLP localhost:4318<br/>histogramas RED/USE + biz_*" --> COL
    end

    %% ===================== GOOGLE CLOUD =====================
    subgraph GCP["Google Cloud"]
        direction TB
        LOG["Cloud Logging<br/>(logs JSON, 50 GiB grátis)"]
        MON["Cloud Monitoring<br/>custom metrics biz_* (24 meses)<br/>+ métricas nativas Cloud Run"]
        BQ["BigQuery<br/>(export GA4 do mobile)"]
        SM["Secret Manager<br/>(token Grafana Cloud)"]
    end

    %% ===================== FIREBASE =====================
    subgraph FB["Firebase / GA4 (mobile)"]
        direction TB
        CRASH["Crashlytics<br/>(crash-free %)"]
        GA4["Analytics GA4<br/>(funil / DAU / retenção)"]
    end

    %% ===================== VERCEL =====================
    subgraph VC["Vercel (web)"]
        VAN["Vercel Analytics +<br/>Speed Insights (Web Vitals)"]
    end

    %% ===================== GRAFANA =====================
    subgraph GRA["Grafana Cloud Free — SINGLE PANE OF GLASS"]
        direction TB
        GMET["Prometheus<br/>(histogramas saúde, 14d)"]
        GTEMPO["Tempo<br/>(traces — fase futura)"]
        DASH["Dashboards unificados<br/>Saúde RED · Negócio · Dependências<br/>· Mobile · Web"]
        GMET --> DASH
        GTEMPO --> DASH
    end

    %% ===================== CHAMADAS DE API =====================
    WEB -- "HTTPS + user_token<br/>+ X-Request-ID (correlação)" --> APP
    MOB -- "HTTPS + user_token<br/>+ X-Request-ID (correlação)" --> APP

    %% ===================== ROTEAMENTO DO COLLECTOR =====================
    COL -- "otlphttp<br/>histogramas de saúde (14d)" --> GMET
    COL -- "exporter googlecloud<br/>biz_* counters/gauges (24 meses)" --> MON
    COL -. "exporter (fase futura)<br/>traces" .-> GTEMPO
    SM -- "token via env var" --> COL

    %% ===================== LOGS E INFRA =====================
    APP -- "stdout JSON<br/>(severity, trace_id)" --> LOG
    CR -- "métricas nativas<br/>(req count, cold start, CPU/mem)" --> MON

    %% ===================== BRIDGES WEB/MOBILE =====================
    WEB -- "OTLP (Web Vitals + biz_*)" --> GMET
    WEB -- "RUM nativo" --> VAN
    MOB -- "crashes" --> CRASH
    MOB -- "eventos GA4" --> GA4
    GA4 -- "export nativo" --> BQ
    MOB -. "biz_* (opcional, via OTLP)" .-> COL

    %% ===================== GRAFANA LÊ POR DATASOURCE =====================
    MON -- "datasource Cloud Monitoring" --> DASH
    LOG -- "datasource Cloud Logging" --> DASH
    BQ  -- "datasource BigQuery (GA4)" --> DASH
    CRASH -. "opcional (export BigQuery)" .-> BQ

    %% ===================== ESTILOS =====================
    classDef gcp fill:#E8F0FE,stroke:#4285F4,color:#1a1a1a;
    classDef grafana fill:#FFF3E0,stroke:#F46800,color:#1a1a1a;
    classDef firebase fill:#FFF8E1,stroke:#FFCA28,color:#1a1a1a;
    classDef vercel fill:#F5F5F5,stroke:#000000,color:#1a1a1a;
    classDef run fill:#E6F4EA,stroke:#34A853,color:#1a1a1a;
    classDef client fill:#EDE7F6,stroke:#673AB7,color:#1a1a1a;

    class LOG,MON,BQ,SM gcp;
    class GMET,GTEMPO,DASH grafana;
    class CRASH,GA4 firebase;
    class VAN vercel;
    class APP,COL run;
    class WEB,MOB client;
```

---

## 2. Detalhe — roteamento de sinais no sidecar Collector

> Regra central: **histograma nunca vai para o Cloud Monitoring** (custa 1 ponto/bucket e estoura
> o free tier). A separação é por **prefixo de nome de métrica** (`biz_*`) e por **tipo** (histogram
> vs counter/gauge), em **pipelines distintos** do Collector.

```mermaid
flowchart LR
    IN["OTLP receiver<br/>localhost:4318<br/>(app Go envia tudo aqui)"]

    subgraph PIPE["Collector — pipelines"]
        direction TB
        P1["Pipeline SAÚDE<br/>filter: histogramas RED/USE<br/>(http/db/external _duration_*)"]
        P2["Pipeline NEGÓCIO<br/>routing: nome com prefixo biz_<br/>(counters/gauges, baixa cardinalidade)"]
        P3["Pipeline TRACES<br/>(fase futura)"]
    end

    IN --> P1
    IN --> P2
    IN -.-> P3

    P1 -- "exporter otlphttp" --> G["Grafana Cloud<br/>Prometheus (14 dias)"]
    P2 -- "exporter googlecloud" --> M["Cloud Monitoring<br/>custom metrics (24 meses)"]
    P3 -. "exporter otlp" .-> T["Grafana Tempo"]

    classDef g fill:#FFF3E0,stroke:#F46800;
    classDef m fill:#E8F0FE,stroke:#4285F4;
    class G,T g;
    class M m;
```

---

## 3. Legenda de destinos por sinal

| Sinal | Origem | Caminho | Destino final | Retenção / custo |
|---|---|---|---|---|
| **Logs** | App Go (stdout JSON) | Cloud Run → Cloud Logging | Cloud Logging | 50 GiB/mês grátis |
| **Histogramas de saúde** (RED/USE) | App Go → OTLP | Collector (`otlphttp`) | Grafana Cloud (Prometheus) | 14 dias / grátis (10k séries) |
| **KPIs de negócio `biz_*`** | App Go → OTLP | Collector (`googlecloud`) | Cloud Monitoring | 24 meses / ~26% do free tier |
| **Métricas de infra** | Cloud Run nativo | direto | Cloud Monitoring | grátis |
| **Web Vitals + `biz_*` web** | Web (`@vercel/otel`) | OTLP direto | Grafana Cloud | grátis |
| **RUM web (visão rápida)** | Web | nativo | Vercel Analytics | nativo |
| **Crashes mobile** | Mobile | nativo | Crashlytics (Firebase) | grátis |
| **Eventos/funil mobile** | Mobile | GA4 → export | BigQuery → datasource Grafana | sandbox grátis |
| **Traces** (fase futura) | App/clientes → OTLP | Collector | Grafana Tempo | 50 GB / grátis |

> **Single pane:** o **Grafana** abre todos os dashboards lendo Cloud Monitoring + Cloud Logging +
> BigQuery por *datasource*, além do que recebe direto via OTLP (saúde/traces). Os consoles nativos
> (Vercel, Firebase) permanecem como visão complementar — sem duplicar custo.
