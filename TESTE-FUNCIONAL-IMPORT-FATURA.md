# Roteiro de teste funcional — Import de fatura (AYD-004, Fases 1–6)

> **Arquivo temporário.** Está no `.gitignore`; apague quando terminar a validação.
> Gerado em 13/set/2026, cobrindo o que foi implementado nas branches
> `claude/aydimportfatura-docs-novmzd` dos quatro repos.

A lógica que mais importa (exclusões, vínculo de parcela, limites) **não precisa de PDF** —
dá para exercitar o `confirm-invoice` direto por `curl`. Só a Parte C precisa de faturas
reais. Sugiro fazer nessa ordem: B (rápido, pega a maioria dos bugs) → C (realidade) → D (UI).

---

## Parte A — Preparação

### A.1 Subir o banco e aplicar as migrations

```bash
cd personal-finance
docker-compose up -d
# aguarde o healthcheck ficar verde
docker ps --filter name=pg-personal-finance
```

A **migration 027** é nova (índices de `idempotency_hash` e `installment_group_id`).
Se o seu banco é antigo, aplique à mão:

```bash
docker exec -i pg-personal-finance psql -U silvioubaldino -d personal_finance \
  < db/migrations/027_add_movements_indexes.up.sql

# confirme
docker exec -it pg-personal-finance psql -U silvioubaldino -d personal_finance \
  -c "\di idx_movements_idempotency_hash idx_movements_installment_group"
```

Esperado: os dois índices listados. Sem eles nada quebra, mas o dedup e a busca de
candidatos a match fazem varredura.

### A.2 Subir a API

```bash
go run ./cmd/api/main.go   # precisa do .env
```

### A.3 Variáveis do shell

```bash
export TOKEN="<seu firebase id token>"
export API="http://localhost:8080"
export H_AUTH="user_token: $TOKEN"
```

Todas as chamadas abaixo usam o header `user_token` (não `Authorization`).

### A.4 Criar o cartão de teste

Crie pela UI ou pela API um `CreditCard` com:

- **dia de fechamento = 3** (o roteiro assume isso; ajuste as datas se usar outro)
- **carteira default definida** — sem ela o import é rejeitado de propósito (caso B.11)
- **limite = R$ 2.000,00** (baixo, para o caso B.10 disparar)

```bash
export CARD_ID="<uuid do cartão>"
```

Com fechamento no dia 3, a fatura de referência de maio/2026 tem
`period_start = 2026-05-04` e `period_end = 2026-06-03`.

### A.5 Consultas de verificação (deixe à mão)

```bash
# helper: estado da fatura e do limite
docker exec -it pg-personal-finance psql -U silvioubaldino -d personal_finance -c "
SELECT i.id, i.period_start, i.period_end, i.amount, i.is_paid, c.credit_limit
FROM invoices i JOIN credit_cards c ON c.id = i.credit_card_id
WHERE i.credit_card_id = '$CARD_ID' ORDER BY i.period_start;"

# helper: movimentos importados
docker exec -it pg-personal-finance psql -U silvioubaldino -d personal_finance -c "
SELECT m.date, m.description, m.amount, m.is_paid, m.installment_number,
       m.total_installments, m.installment_group_id
FROM movements m WHERE m.credit_card_id = '$CARD_ID' ORDER BY m.date, m.description;"
```

---

## Parte B — `confirm-invoice` por `curl` (sem PDF)

> Estes casos exercitam a persistência, que é onde estavam os bugs de duplicação e de
> limite. **Zere entre os blocos** apagando os movimentos do cartão, senão o dedup por
> hash mascara o resultado:
>
> ```bash
> docker exec -it pg-personal-finance psql -U silvioubaldino -d personal_finance -c "
> DELETE FROM movements WHERE credit_card_id = '$CARD_ID';
> UPDATE invoices SET amount = 0 WHERE credit_card_id = '$CARD_ID';
> UPDATE credit_cards SET credit_limit = 2000 WHERE id = '$CARD_ID';"
> ```

### B.1 — Import simples entra como item de fatura

```bash
curl -s -X POST "$API/v2/statements/confirm-invoice" -H "$H_AUTH" \
  -H 'Content-Type: application/json' -d "{
  \"credit_card_id\": \"$CARD_ID\",
  \"movements\": [
    {\"date\":\"2026-05-12\",\"description\":\"MERCADO LIVRE\",\"amount\":-100.00},
    {\"date\":\"2026-05-20\",\"description\":\"IFOOD\",\"amount\":-60.00}
  ]}" | jq
```

- [ ] `{"created": 2, "skipped": 0, "errors": []}`
- [ ] No banco: os dois com `is_paid = false` (quem paga é a **fatura**, não o item)
- [ ] `invoices.amount` da competência de maio = **-160,00**
- [ ] `credit_cards.credit_limit` caiu de 2000 para **1840,00**

> Se o `amount` da fatura ou o limite não mexeram, a transação de persistência quebrou.

### B.2 — Reenviar a mesma fatura não duplica (idempotência)

Rode **exatamente o mesmo comando do B.1 de novo**, sem zerar.

- [ ] `{"created": 0, "skipped": 2}`
- [ ] `invoices.amount` continua **-160,00** (não virou -320)
- [ ] Limite continua **1840,00**

### B.3 — Pagamento da fatura anterior é recusado mesmo se o cliente insistir

O servidor não confia no cliente: reaplica a detecção.

```bash
curl -s -X POST "$API/v2/statements/confirm-invoice" -H "$H_AUTH" \
  -H 'Content-Type: application/json' -d "{
  \"credit_card_id\": \"$CARD_ID\",
  \"movements\": [
    {\"date\":\"2026-05-07\",\"description\":\"PAGAMENTO ON LINE\",\"amount\":-5212.59},
    {\"date\":\"2026-05-08\",\"description\":\"PAGTO FATURA\",\"amount\":-100.00},
    {\"date\":\"2026-05-09\",\"description\":\"RESTAURANTE\",\"amount\":-80.00}
  ]}" | jq
```

- [ ] `{"created": 1, "skipped": 2}` — só o restaurante entrou
- [ ] O limite **não** foi consumido pelos -5212,59 (era o bug: inflava fatura e comia limite)

### B.4 — Estabelecimento que só *começa* parecido não é confundido

```bash
curl -s -X POST "$API/v2/statements/confirm-invoice" -H "$H_AUTH" \
  -H 'Content-Type: application/json' -d "{
  \"credit_card_id\": \"$CARD_ID\",
  \"movements\": [
    {\"date\":\"2026-05-10\",\"description\":\"PAGUE MENOS 1234\",\"amount\":-45.00},
    {\"date\":\"2026-05-11\",\"description\":\"PAGSEGURO *LOJA\",\"amount\":-30.00},
    {\"date\":\"2026-05-12\",\"description\":\"PAGBANK SERVICOS\",\"amount\":-20.00}
  ]}" | jq
```

- [ ] `{"created": 3, "skipped": 0}` — nenhum foi marcado como pagamento de fatura

### B.5 — Compra parcelada nova gera a série inteira

Zere antes.

```bash
curl -s -X POST "$API/v2/statements/confirm-invoice" -H "$H_AUTH" \
  -H 'Content-Type: application/json' -d "{
  \"credit_card_id\": \"$CARD_ID\",
  \"movements\": [
    {\"date\":\"2026-05-15\",\"description\":\"MAGALU*MAGAZINELUIZA PARC 03/10\",
     \"amount\":-200.00,\"installment_number\":3,\"total_installments\":10}
  ]}" | jq
```

- [ ] `created = 8` (parcelas 3 a 10)
- [ ] No banco: 8 movimentos, **todos com o mesmo `installment_group_id`**
- [ ] Cada parcela caiu na fatura do mês correspondente (mai/2026 até dez/2026)
- [ ] Limite consumido = **1600,00** (8 × 200), não 200

```bash
export GRP=$(docker exec -it pg-personal-finance psql -U silvioubaldino \
  -d personal_finance -tAc "SELECT DISTINCT installment_group_id FROM movements
  WHERE credit_card_id='$CARD_ID' AND installment_group_id IS NOT NULL LIMIT 1;" | tr -d '\r')
echo $GRP
```

### B.6 — 🔴 O bug principal: importar a fatura do mês seguinte **vincula**, não duplica

Este é o caso que motivou a Fase 6. Sem ele, cada import mensal duplicava a série inteira.

A fatura de junho traz a **mesma compra** com o texto do banco atualizado
(`PARC 04/10`) e um valor com centavos diferentes — o banco distribui o arredondamento.

```bash
curl -s -X POST "$API/v2/statements/confirm-invoice" -H "$H_AUTH" \
  -H 'Content-Type: application/json' -d "{
  \"credit_card_id\": \"$CARD_ID\",
  \"movements\": [
    {\"date\":\"2026-06-15\",\"description\":\"MAGALU*MAGAZINELUIZA PARC 04/10\",
     \"amount\":-200.03,\"installment_number\":4,\"total_installments\":10,
     \"installment_group_id\":\"$GRP\"}
  ]}" | jq
```

- [ ] `created = 0`
- [ ] `skipped = 7` — a **série inteira** (parcelas 4 a 10), não só a parcela 4
- [ ] Continuam existindo **8** movimentos no total, não 16
- [ ] A parcela 4 teve o **valor atualizado** para **-200,03**
- [ ] A **descrição da parcela 4 NÃO mudou** — continua o texto original, não o do banco
- [ ] A **data e o `is_paid` da parcela 4 não mudaram**
- [ ] `invoices.amount` de junho ajustou só pelo **delta de -0,03**
- [ ] Limite ajustou só pelos **-0,03**

> Se `created` vier 7 e existirem 15–16 movimentos, o vínculo não pegou — é a duplicação
> voltando.

### B.7 — Vínculo inválido cai no caminho normal de criação

O servidor não confia num `installment_group_id` que não seja dele.

```bash
curl -s -X POST "$API/v2/statements/confirm-invoice" -H "$H_AUTH" \
  -H 'Content-Type: application/json' -d "{
  \"credit_card_id\": \"$CARD_ID\",
  \"movements\": [
    {\"date\":\"2026-07-01\",\"description\":\"COMPRA X PARC 01/03\",\"amount\":-50.00,
     \"installment_number\":1,\"total_installments\":3,
     \"installment_group_id\":\"00000000-0000-0000-0000-000000000000\"}
  ]}" | jq
```

- [ ] `created = 3` — grupo inexistente é ignorado e a série é criada normalmente
- [ ] Nada foi corrompido

### B.8 — Parcela de competência futura é pulada; compra comum **não** é

Este caso valida a correção de escopo que eu apliquei: a regra vale **só para parcelas**.

Zere antes.

```bash
curl -s -X POST "$API/v2/statements/confirm-invoice" -H "$H_AUTH" \
  -H 'Content-Type: application/json' -d "{
  \"credit_card_id\": \"$CARD_ID\",
  \"movements\": [
    {\"date\":\"2026-05-12\",\"description\":\"MERCADO LIVRE\",\"amount\":-100.00},
    {\"date\":\"2026-09-03\",\"description\":\"DROGARIA SP PARCELA 03 DE 03\",
     \"amount\":-198.67,\"installment_number\":3,\"total_installments\":3},
    {\"date\":\"2026-09-03\",\"description\":\"POSTO SHELL\",\"amount\":-220.00}
  ]}" | jq
```

- [ ] A **parcela** de setembro foi **pulada** (`skipped`)
- [ ] A **compra comum** de setembro **foi criada** e caiu na fatura de setembro, não na de maio
- [ ] Total: `created = 2`, `skipped = 1`

> Se o POSTO SHELL também tiver sido pulado, a regra ficou larga demais e o usuário
> **perde** a compra. Era exatamente o risco que motivou a correção.

### B.9 — Estorno reduz a fatura e devolve limite

```bash
curl -s -X POST "$API/v2/statements/confirm-invoice" -H "$H_AUTH" \
  -H 'Content-Type: application/json' -d "{
  \"credit_card_id\": \"$CARD_ID\",
  \"movements\": [
    {\"date\":\"2026-05-18\",\"description\":\"ESTORNO COMPRA LOJA\",\"amount\":75.00}
  ]}" | jq
```

- [ ] `created = 1`
- [ ] `invoices.amount` **subiu** 75,00 (a despesa diminuiu)
- [ ] Limite **voltou** 75,00
- [ ] ⚠️ **Anote a categoria que ficou.** Hoje o estorno é classificado como
      *Uncategorized **Income*** (receita). É um problema conhecido e **não corrigido**
      nesta entrega — está registrado em AYD-004 §"Achados de set/2026". Só confirme que o
      comportamento é esse, não trate como bug novo.

### B.10 — Estouro de limite é barrado com 403

Zere e deixe o limite em 2000. Envie uma fatura que passa disso.

```bash
curl -s -o /dev/null -w "%{http_code}\n" -X POST "$API/v2/statements/confirm-invoice" \
  -H "$H_AUTH" -H 'Content-Type: application/json' -d "{
  \"credit_card_id\": \"$CARD_ID\",
  \"movements\": [
    {\"date\":\"2026-05-12\",\"description\":\"COMPRA A\",\"amount\":-900.00},
    {\"date\":\"2026-05-13\",\"description\":\"COMPRA B\",\"amount\":-900.00},
    {\"date\":\"2026-05-14\",\"description\":\"COMPRA C\",\"amount\":-900.00}
  ]}"
```

- [ ] HTTP **403**
- [ ] A mensagem fala de **limite do cartão**, não de quota de plano
      (era o motivo de não reusar `ErrCreditCardLimitReached`)
- [ ] O limite acompanha o import item a item: as duas primeiras cabem, a terceira estoura.
      Antes disso, cada item passava sozinho e a fatura inteira furava o limite.

### B.11 — Cartão sem carteira default é rejeitado

Tire a carteira default do cartão e repita o B.1.

- [ ] HTTP **400**, mensagem sobre carteira default
- [ ] Devolva a carteira default antes de seguir

### B.12 — Fatura já paga

Marque a fatura de maio como paga e repita o B.1.

```bash
docker exec -it pg-personal-finance psql -U silvioubaldino -d personal_finance -c "
UPDATE invoices SET is_paid = true WHERE credit_card_id = '$CARD_ID'
AND period_start = '2026-05-04';"
```

- [ ] HTTP **422**
- [ ] ⚠️ **Confira quantos itens ficaram gravados.** Hoje o import aborta no meio, com os
      itens anteriores já persistidos e sem rollback. Conhecido, **não corrigido**,
      registrado em AYD-004 §"Achados de set/2026". Reverta o `is_paid` depois.

---

## Parte C — `/extract` com faturas reais (precisa de PDF)

Aqui é onde a IA entra. Separe **3 faturas do mesmo cartão, de meses consecutivos** —
é o cenário que expõe a duplicação de série.

### C.1 — Extração de fatura com contexto de cartão

```bash
curl -s -X POST "$API/v2/statements/extract" -H "$H_AUTH" \
  -F "file=@fatura-maio.pdf" \
  -F "source_type=invoice" \
  -F "credit_card_id=$CARD_ID" | jq
```

- [ ] `document_type = "invoice"`, `confidence` alto
- [ ] Todo item com `type_payment = "credit_card"`
- [ ] `invoice_meta` com fechamento, vencimento e total
- [ ] Parcelas detectadas com `installment_number` / `total_installments`
- [ ] A linha de pagamento da fatura anterior veio com `excluded: true` e
      `exclusion_reason: "invoice_payment"` — **presente na lista**, não sumida
- [ ] `warnings` inclui `invoice_payment_excluded`

**Guarde o JSON**: `... | jq > extract-maio.json`. Você vai reusar no C.5.

### C.2 — Checksum do total

- [ ] Se a soma dos itens não marcados bater com `invoice_meta.total_amount`,
      **não** há warning `total_amount_mismatch`
- [ ] Se não bater (OCR perdeu linha), o warning aparece com `expected` e `detected`,
      e a extração **continua 200** — nunca bloqueia

### C.3 — Sem `credit_card_id` a extração degrada, não quebra

```bash
curl -s -X POST "$API/v2/statements/extract" -H "$H_AUTH" \
  -F "file=@fatura-maio.pdf" -F "source_type=invoice" | jq '.warnings, .movements[0]'
```

- [ ] Continua 200, itens extraídos normalmente
- [ ] **Sem** `installment_match` em nenhum item
- [ ] **Sem** `exclusion_reason: "future_installment"`
- [ ] O pagamento da fatura anterior **continua** marcado (essa camada não depende do cartão)

### C.4 — Documento trocado gera warning, não erro

Suba um **extrato bancário** dizendo que é fatura:

```bash
curl -s -X POST "$API/v2/statements/extract" -H "$H_AUTH" \
  -F "file=@extrato.pdf" -F "source_type=invoice" | jq '.document_type, .warnings'
```

- [ ] HTTP **200** (nunca 4xx)
- [ ] `warnings` inclui `document_type_mismatch` com `expected: "invoice"`,
      `detected: "statement"`

E um documento que não é nenhum dos dois (uma nota fiscal, um print qualquer):

- [ ] `document_type = "unknown"` + warning `low_confidence`, ainda 200

### C.5 — 🔴 Fluxo real de dois meses consecutivos

1. Extraia e confirme a fatura de **maio** (C.1 → `confirm-invoice`).
2. Anote quantos movimentos existem e o total da fatura.
3. Extraia a fatura de **junho** com `credit_card_id`.

- [ ] As parcelas que continuam da fatura de maio vêm com `installment_match` preenchido
- [ ] As de confiança alta vêm com `installment_group_id` **já preenchido**
- [ ] As de confiança média vêm com `installment_match` mas `installment_group_id` **vazio**
- [ ] `warnings` inclui `installment_match_found`
- [ ] As parcelas datadas depois do fechamento vêm `excluded` com
      `exclusion_reason: "future_installment"`

4. Confirme junho.

- [ ] **Nenhuma série duplicada** — confira o total de movimentos por `installment_group_id`
- [ ] Os valores das parcelas da competência de junho foram atualizados
- [ ] O limite do cartão bate com o esperado, sem consumo fantasma

5. Repita para **julho**.

- [ ] A série converge: cada import corrige só a parcela da sua competência

### C.6 — PDF protegido por senha

```bash
curl -s -X POST "$API/v2/statements/extract" -H "$H_AUTH" \
  -F "file=@fatura-protegida.pdf" -F "source_type=invoice" | jq '.error'
```

- [ ] 422 com `error.type = "statement_password_required"`
- [ ] Com `-F "password=<senha correta>"` → extrai normalmente
- [ ] Com senha errada → 422 `statement_wrong_password`

---

## Parte D — UI

### D.1 Web (`personal-finance-frontend-v2`)

```bash
cd personal-finance-frontend-v2 && npm run dev
```

Vá em **Cartões** e selecione um cartão.

- [ ] O botão "Importar fatura" aparece **na área do cartão selecionado**, não mais no card
      de total agregado
- [ ] Com um cartão selecionado, o botão de **confirmar habilita** (antes ficava
      permanentemente desabilitado — o caminho de fatura não completava no web)
- [ ] Fechamento, vencimento e total da fatura aparecem no topo da revisão
- [ ] Warnings informativos aparecem, incluindo total divergente com esperado × detectado
- [ ] O pagamento da fatura anterior aparece **esmaecido e desmarcado**, com o motivo
- [ ] Dá para **reincluir** o item marcado
- [ ] Parcela pré-vinculada mostra o chip "já registrada" e **desvincula com um clique**
- [ ] Desvincular volta a criar a série
- [ ] Sugestão mostra a compra **como está registrada no app** (ex.: "TV da sala"),
      não o texto do banco — é o ponto todo do caso que você levantou
- [ ] Existe vinculação manual para o que o matcher não pegou
- [ ] Resumo no topo: "N compras parceladas já registradas serão vinculadas"
- [ ] No DevTools → Network, o `/extract` leva o header **`X-Request-ID`**

### D.2 Mobile (`personal-finance-mobile`)

```bash
cd personal-finance-mobile && npm start
```

Vá em **Cartões** → abra a fatura → "Importar fatura".

- [ ] Mesma lista de verificações do web (D.1), da terceira à décima primeira
- [ ] **PDF protegido**: o campo de senha aparece em vez de um erro sem saída, e o reenvio
      funciona (antes era beco sem saída, e fatura de banco vem protegida por padrão)
- [ ] Senha errada mantém o campo aberto para nova tentativa
- [ ] Strings aparecem certas em pt-BR; confira também en-US e pt-PT

---

## Parte E — Watchlist: comportamentos conhecidos e **não** corrigidos

Se você topar com algum destes, **não é regressão** — está mapeado em
AYD-004 §"Achados de set/2026" esperando decisão:

| # | O que você vai ver | Onde |
|---|---|---|
| 1 | Estorno positivo classificado como **receita** (Uncategorized Income) | B.9 |
| 2 | Usuário `free` **fura o teto mensal** de movimentos importando fatura — o import não valida limite de plano | qualquer import grande |
| 3 | Com `invoice_id` no body, **todo item não parcelado** cai naquela fatura mesmo datado fora do período (o mobile sempre manda `invoice_id`) | D.2 |
| 4 | Fatura paga **aborta o import no meio**, com itens já gravados e sem rollback | B.12 |
| 5 | Reimport de correção (OCR perdeu linha, reenvia) funciona, mas não está no contrato nem coberto por teste | C.5 |
| 6 | Interação entre fatura importada e o pagamento dela (AYD-003) não descrita | — |

Também **pendente e alheio a esta feature**:

- `npm run lint` **não roda** em web nem mobile — a config do ESLint está pendente de
  migração para flat config nos dois repos. Pré-existente.
- No api, `go test ./...` completo falha em `TestPushNotifications_SendDailyUnpaidPush`
  e nos pacotes legacy que não compilam (`internal/domain/movement/repository`).
  Pré-existente. Para rodar só o que importa aqui:
  `go test ./internal/domain/ ./internal/usecase/... -run 'Statement|Invoice'`

---

## Onde reportar

Se algo da Parte B ou C falhar, é **bug da entrega** — mande o comando, a resposta e o
resultado das duas queries de verificação da A.5. Se for da Parte E, é **decisão pendente**,
não bug.
