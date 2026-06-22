# Backend — `POST /v1/telemetry` (ingestão de telemetria do mobile)

> **Status:** Especificação para implementação no backend (`personal-finance`, Go/Gin).
> **Origem:** consumido pelo `src/lib/telemetry/` do app mobile (Expo/RN).
> **Documento irmão:** `docs/AyDmonitoramento.md` (desenho geral de observabilidade).

---

## 1. Por que esse endpoint precisa existir

O mobile precisa mandar **métricas e logs** para o mesmo lugar que o backend e o web
(Grafana / Cloud Monitoring / Cloud Logging, via o **sidecar OTel Collector** já existente).
Mas o app **não pode** falar direto com esses destinos:

1. **Segurança de credenciais.** Exportar direto para o Grafana Cloud ou Cloud Monitoring
   exigiria embutir tokens/credenciais GCP no binário do app — qualquer pessoa extrai do
   APK/IPA. O token do Grafana mora **só no sidecar** (Secret Manager); ninguém mais o vê.
2. **O sidecar não tem ingress.** O Collector no Cloud Run escuta apenas em `localhost:4318`
   e não é acessível pela internet. Algo com ingress (o app Go) precisa receber o tráfego
   externo e repassar para o Collector em `localhost`.
3. **Reaproveitar autenticação.** O app já manda o header `user_token` (Firebase ID token)
   em toda chamada. Um endpoint no backend reusa esse middleware — o Collector não tem como
   autenticar usuários.
4. **Roteamento já existe.** O Collector já separa sinais por tipo/prefixo
   (`app_*` → Grafana 14d, `biz_*` → Cloud Monitoring 24m, logs → Cloud Logging). O endpoint
   só precisa traduzir o payload do mobile para OTLP e jogar no Collector — **sem inventar
   roteamento novo**.

**Resumo:** o endpoint é a ponte autenticada `mobile → backend → sidecar`, mantendo
credenciais fora do app e reusando toda a infra de roteamento já desenhada.

```
Mobile (buffer OTLP-like)
   │ POST /v1/telemetry  (HTTPS + user_token)
   ▼
Backend Go  ──── traduz p/ OTLP ────►  Collector (localhost:4318)
                                          ├─ app_*  → Grafana Cloud (14d)
                                          ├─ biz_*  → Cloud Monitoring (24m)
                                          └─ logs   → Cloud Logging
```

---

## 2. Contrato HTTP

| Item | Valor |
|---|---|
| Método | `POST` |
| Path | `/v1/telemetry` |
| Auth | **Obrigatória** — mesmo middleware das demais rotas (header `user_token`). 401 se ausente/inválido. |
| Content-Type | `application/json` |
| Resposta de sucesso | **`202 Accepted`** com corpo vazio (ou `204`). É ingestão best-effort: aceitar e processar async. |
| Idempotência | Não precisa. Eventos são contadores/medições; reenvio duplicado é tolerável (mobile não faz retry hoje). |

### Corpo da requisição

O app envia um **batch** com contexto + arrays de métricas e logs:

```jsonc
{
  "context": {
    "app_version": "1.1.0",        // versão da loja (expo-constants)
    "runtime_version": "54.0.0",   // bundle EAS Update (expo-updates)
    "update_id": "abc-123",        // id do update OTA, pode ser null
    "os": "ios",                   // "ios" | "android"
    "os_version": "17.4",
    "session_id": "uuid-v4"        // correlaciona eventos da mesma sessão
  },
  "metrics": [
    {
      "kind": "counter",                       // "counter" | "histogram"
      "name": "app_request_failed_total",
      "value": 1,                              // counter: incremento; histogram: medição
      "labels": { "reason": "timeout", "method": "GET" },
      "timestamp": 1719000000000               // epoch ms (cliente)
    },
    {
      "kind": "histogram",
      "name": "app_start_duration_seconds",
      "value": 0.84,                           // unidade do nome (segundos)
      "labels": { "type": "cold" },
      "timestamp": 1719000000123
    }
  ],
  "logs": [
    {
      "level": "error",                        // "error" | "warn" | "info"
      "message": "Cannot read property 'x' of undefined",
      "fields": {
        "stack": "...",
        "source": "error_boundary",
        "component_stack": "..."
      },
      "timestamp": 1719000000200
    }
  ]
}
```

> `metrics` e `logs` podem vir vazios (mas não os dois ao mesmo tempo — o cliente só
> envia se houver algo). O `context` é sempre presente e aplica-se a todos os eventos do batch.

---

## 3. Catálogo de sinais que o mobile envia

O backend **não precisa** validar nome por nome, mas é útil saber o que chega para
definir o roteamento e os guard-rails.

### Métricas técnicas — prefixo `app_*` → **Grafana Cloud (14d)**

| Nome | Tipo | Labels | Significado |
|---|---|---|---|
| `app_request_failed_total` | counter | `reason` (`timeout`/`network`/`server`/`unknown`), `method` | Request que falhou no device — invisível ao backend |
| `app_start_duration_seconds` | histogram | `type` (`cold`/`warm`) | Tempo de inicialização do app |
| `app_screen_render_seconds` | histogram | `screen` (nome da rota) | Tempo de render percebido por tela |

### Métricas de negócio — prefixo `biz_*` → **Cloud Monitoring (24m)**

| Nome | Tipo | Labels | Significado |
|---|---|---|---|
| `biz_app_sessions_total` | counter | `app_version`, `runtime_version`, `os` | 1 por sessão — mede adoção de versão/OTA |
| `biz_app_push_opened_total` | counter | `type` | Push aberto pelo usuário (funil de engajamento) |

### Logs → **Cloud Logging**

Erros JS estruturados (ErrorBoundary, handler global, unhandledrejection). `level` +
`message` + `fields` (inclui `stack`, `source`, e `component_stack` quando houver).

---

## 4. Comportamento esperado do endpoint

1. **Autenticar** via `user_token` (middleware existente). 401 se inválido.
2. **Parsear e validar superficialmente** o batch. Validação deve ser **leniente**:
   campos desconhecidos ignorados; evento malformado individual descartado sem rejeitar o
   batch inteiro. Telemetria nunca deve causar erro 4xx por causa de um evento ruim.
3. **Enriquecer com atributos de recurso** ao traduzir para OTLP:
   - `service.name = personal-finance-mobile`
   - `app.version`, `runtime.version`, `os`, `os.version`, `session.id` (do `context`)
   - `user_id` do token **apenas em logs** (Cloud Logging), **nunca como label de métrica**
     (cardinalidade — ver §5). Pode ir como log field para correlação/auditoria.
4. **Traduzir para OTLP e exportar para o Collector** em `localhost:4318` (mesmo destino do
   app Go). Counters → OTLP Sum; histograms → OTLP Histogram; logs → OTLP Logs. O Collector
   já roteia por prefixo (`app_*` vs `biz_*`) e tipo.
   - *Alternativa simples se preferir não montar OTLP na mão:* counters/gauges de negócio
     podem ir direto via cliente Cloud Monitoring, e logs via Cloud Logging client. Mas o
     caminho recomendado é **reusar o Collector** para manter um único ponto de roteamento.
5. **Responder `202` imediatamente.** O processamento/export deve ser **assíncrono** (fila/
   goroutine com buffer) — não bloquear a resposta esperando o Collector. Se o export
   falhar, descartar (best-effort); não propagar erro ao cliente.
6. **Correlação `X-Request-ID`:** o app já manda esse header em toda chamada (inclusive
   nesta). O middleware de log do backend (`pkg/log/middleware.go`) já o consome — então os
   logs desta própria ingestão ficam correlacionáveis. Não é preciso nada extra aqui.

---

## 5. Guard-rails (importante — viram regra de revisão)

- ❌ **Nunca** transformar `user_id`, `session_id`, `update_id` ou qualquer ID em **label de
  métrica**. Estoura cardinalidade (10k séries do Grafana / 150 MiB do Cloud Monitoring).
  Eles podem ir como **log fields** (alta cardinalidade é ok em log), nunca em métrica.
- ❌ **Histograma só no Grafana**, nunca no Cloud Monitoring (custa 1 ponto por bucket).
  Como o Collector já roteia por prefixo, basta garantir que histogramas tenham nomes
  `app_*` (e não `biz_*`).
- ✅ Validar/limitar o tamanho do batch (ex.: rejeitar payloads absurdos > N KB ou com
  milhares de eventos) para evitar abuso, mesmo autenticado.
- ✅ Considerar **rate limiting** leve por usuário/IP — o endpoint é público (autenticado) e
  recebe de muitos devices.
- ⚠️ `timestamp` vem do **cliente** (relógio do device, pode estar torto). Usar como
  best-effort; se o destino exigir, o backend pode sobrescrever/limitar com o tempo de
  recebimento. Para counters não importa; para histogramas/logs, o tempo do servidor é mais
  confiável.

---

## 6. Notas de segurança

- Endpoint **autenticado** — não aceitar anônimo (evita spam de telemetria forjada).
- Não confiar cegamente em `message`/`fields` de log: podem conter dados do device. Tratar
  como não confiável ao exibir/alertar (escapar). Não logar PII sensível.
- O `user_id` derivado do token (servidor) é mais confiável que qualquer coisa no corpo —
  preferir o do token para atribuição.

---

## 7. Definition of done (backend)

- [ ] Rota `POST /v1/telemetry` autenticada, retornando `202`.
- [ ] Parsing leniente do batch (evento ruim não derruba o lote).
- [ ] Tradução para OTLP + export assíncrono para o Collector (`localhost:4318`).
- [ ] Resource attributes preenchidos a partir do `context`.
- [ ] Guard-rail de cardinalidade garantido (nenhum ID vira label de métrica).
- [ ] Limite de tamanho de payload + rate limit básico.
- [ ] Dashboard Grafana "Mobile" lendo os `app_*`/`biz_*`/logs do mobile.

> **Lembrete de ativação:** o mobile só começa a chamar este endpoint quando a env
> `EXPO_PUBLIC_TELEMETRY_ENABLED=true` for definida no build. Até lá ele bufferiza e
> descarta — dá para subir o backend e ligar o mobile em momentos separados, sem acoplar
> os deploys.
