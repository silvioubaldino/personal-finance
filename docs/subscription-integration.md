# Assinaturas — Integração Front ↔ Backend

> Backend da migração para **Stripe (web)** + **RevenueCat (mobile)**. Escopo deste doc: o que o front precisa para integrar.

## Decisões

- **Web = Stripe** (Checkout hospedado, redirect). **Mobile = IAP nativo via RevenueCat** (App Store / Play).
- **MercadoPago** descontinuado para novos checkouts; webhook/cancel mantidos só para assinantes legados.
- **Fonte de verdade do plano = custom claims do Firebase** (`plan`, `plan_expires_at`, `subscription_source`). O backend atualiza o claim via webhooks (Stripe no web, RC no mobile).
- **Cupom = só web (Stripe) por enquanto.** O front manda só o **código**; o backend valida nas suas regras (`coupons`) e resolve o `promotion_code` pelo **mesmo código** no Stripe. (Cupom no mobile fica para a fase mobile.)
- **Pós-checkout:** o plano só vira `plus` quando o webhook do Stripe chega. O front deve **forçar refresh do ID token** ao voltar do checkout para enxergar o claim novo (pode haver pequeno atraso → re-tentar).

## Autenticação

Rotas `/me/*` e `/v2/*` exigem o **ID token do Firebase** no header **`user_token`**. O plano do usuário vem dos **custom claims** do próprio token: `plan` (`free`|`plus`), `plan_expires_at` (unix, opcional), `subscription_source` (`stripe`|`iap`|`mp`).

## Contratos (web)

### 1. Listar planos
`GET /subscription/plan` — público.
```json
[{ "id": "plus_monthly", "name": "Plus Mensal", "price": 9.90, "currency": "BRL", "frequency": 1, "frequency_type": "months", "is_active": true }]
```

### 2. (Opcional) Validar cupom
`POST /v2/subscriptions/coupons/preview` — auth.
```jsonc
// req
{ "plan_id": "plus_monthly", "code": "LANCAMENTO50" }
// resp
{ "valid": true, "original_price": 9.90, "discounted_price": 4.95, "currency": "BRL" }
// inválido: { "valid": false, "reason": "coupon outside validity window" }
```
- `discounted_price` é só para **exibição** (o desconto real é aplicado pelo Stripe).

### 3. Criar checkout (web)
`POST /me/subscription/checkout` — auth.
```jsonc
// req  (success_url/cancel_url e coupon_code são opcionais)
{ "plan_id": "plus_monthly", "success_url": "https://app/success", "cancel_url": "https://app/cancel", "coupon_code": "LANCAMENTO50" }
// resp: string JSON com a URL do Stripe Checkout
"https://checkout.stripe.com/c/pay/cs_test_..."
```
- O front **redireciona** para a URL retornada. Com cupom: mandar só `coupon_code` (o backend resolve o `promotion_code`). Erros possíveis: plano sem preço Stripe, cupom inválido/indisponível no web.

### 4. Cancelar
`POST /me/subscription/cancel` — auth → `200`. Cancela no fim do período (mantém Plus até lá; downgrade automático depois).

### Webhooks (somente backend, não o front)
`POST /webhooks/stripe`, `/webhooks/revenuecat`, `/webhooks/mercadopago`.

## Responsabilidades do front (web)

1. Listar planos → (opcional) validar cupom → criar checkout → **redirecionar** para a URL.
2. Páginas de retorno `success_url` / `cancel_url`.
3. Ao voltar do checkout, **dar refresh no ID token** e ler `plan` do claim (re-tentar por alguns segundos, pois depende do webhook).
4. Botão de cancelar → `POST /me/subscription/cancel`.

## O que precisa existir para funcionar (ops)

- **Stripe:** produtos/preços (`price_...`) → cadastrar o `stripe_price_id` em `subscription_plans`. Para cupom: criar `Coupon` + `Promotion Code` no Stripe **com o mesmo código** do cupom no nosso banco (ex. `LANCAMENTO50`); o backend valida nas suas regras e resolve o `promotion_code` pelo próprio código.
- **Stripe webhooks (2 destinos):** o backend `/webhooks/stripe` (principal) **e** o endpoint do RevenueCat (só métricas).
- **RevenueCat:** integração Stripe ligada (lê `app_user_id` da metadata); RC continua dono do mobile.
- **Env backend:** `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET`, `STRIPE_SUCCESS_URL`, `STRIPE_CANCEL_URL`, `REVENUECAT_WEBHOOK_AUTH_KEY`.

## Fora de escopo (outro time)
Front mobile (SDK/paywall do RevenueCat, offerings, produto promo de cupom) e a criação dos recursos no Stripe/RC/App Store/Play.
