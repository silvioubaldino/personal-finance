# Spec: Associação Manual de Importação com Movimentos Recorrentes

## Contexto

O sistema já possui dois fluxos independentes:

1. **Importação de extrato (PDF/Imagem)** — O usuário faz upload de um extrato bancário. A IA extrai as transações (descrição, valor, data) e apresenta uma lista para revisão. Após confirmação, as transações são salvas como novos `movements` com `is_paid: true`.

2. **Movimentos Recorrentes** — O usuário pode cadastrar um `RecurrentMovement` (ex: aluguel, Spotify, academia). A cada mês, o sistema gera automaticamente um `Movement` vinculado ao template recorrente (`recurrent_id` preenchido, `is_recurrent: true`).

**Problema:** hoje esses dois fluxos são completamente separados. Quando o usuário importa um extrato que contém o pagamento de uma recorrente (ex: "SPOTIFY *PREMIUM - R$21,90"), o sistema cria um novo movimento avulso em vez de reconhecer que aquele pagamento corresponde ao movimento recorrente já existente naquele mês. O resultado é dados duplicados e a recorrente do mês permanece marcada como não paga.

---

## Objetivo

Permitir que, durante a **revisão da importação** (entre o Step 1 e Step 2 do fluxo de extrato), o usuário possa **associar manualmente** uma linha importada a um movimento recorrente existente.

Ao fazer essa associação, em vez de criar um novo movimento, o sistema **atualiza o movimento recorrente do mês** com o `amount` e `date` reais vindos do extrato e o marca como **pago**.

---

## Fluxo Atual de Importação (Referência)

### Step 1 — Extração (Preview)
```
POST /v2/statements/extract
Content-Type: multipart/form-data
Body: file (PDF, JPEG ou PNG. Máx: 10MB, 20 páginas)

Response 200:
{
  "movements": [
    { "date": "2026-03-01", "description": "PIX FULANO", "amount": 150.00 },
    { "date": "2026-03-10", "description": "SPOTIFY *PREMIUM", "amount": 21.90 }
  ],
  "errors": []
}
```

A UI apresenta essa lista para o usuário revisar. Ele pode excluir linhas localmente antes de confirmar.

### Step 2 — Confirmação (Save)
```
POST /v2/statements/confirm
Content-Type: application/json
Body:
{
  "wallet_id": "uuid-da-carteira",
  "movements": [
    { "date": "2026-03-01", "description": "PIX FULANO", "amount": 150.00 }
  ]
}

Response 200:
{
  "created": 1,
  "skipped": 0,
  "errors": []
}
```

---

## Nova Funcionalidade: Botão "Associar a Recorrente"

### Comportamento esperado na tela de revisão

Cada linha da lista de movimentos extraídos deve ter, além da opção de excluir, um botão/ação **"Associar a recorrente"**.

Ao clicar nesse botão:

1. Abre um **modal ou drawer** listando os `RecurrentMovements` ativos do usuário.
2. O usuário seleciona qual recorrente corresponde àquela linha importada.
3. A linha importada muda de estado visualmente (ex: badge "Associado a: Spotify") e sai do fluxo normal de criação.
4. Na confirmação (Step 2), essa linha **não é enviada** no array `movements` do `POST /v2/statements/confirm`. Em vez disso, o frontend faz uma chamada separada para atualizar e pagar o movimento recorrente do mês.

---

## API que o Frontend Deve Chamar para Concluir a Associação

### 1. Buscar os movimentos recorrentes ativos do usuário

Para popular a lista do modal, use o endpoint existente de listagem de movements por período, filtrando os que são recorrentes — **ou** peça ao backend um endpoint de listagem de `RecurrentMovements` ativos. O objeto `RecurrentMovement` tem a seguinte estrutura:

```json
{
  "id": "uuid",
  "description": "Spotify",
  "amount": 21.90,
  "initial_date": "2025-01-10T00:00:00Z",
  "end_date": null,
  "category_id": "uuid",
  "sub_category_id": "uuid",
  "wallet_id": "uuid",
  "type_payment": "credit_card"
}
```

### 2. Buscar o Movement do mês correspondente à recorrente selecionada

Quando o usuário selecionar uma `RecurrentMovement`, o frontend precisa encontrar o `Movement` gerado para o mês atual vinculado a ela. Use o endpoint:

```
GET /v1/movements/period?from=2026-03-01&to=2026-03-31
```

Filtre no lado do cliente o movimento cujo `recurrent_id === recurrentMovement.id` e que ainda **não está pago** (`is_paid: false`). Esse é o `Movement` alvo que será atualizado.

### 3. Atualizar o valor e data do Movement

```
PUT /v1/movements/:id
Content-Type: application/json
Body:
{
  "amount": 21.90,
  "date": "2026-03-10"
}
```

> Use o `amount` e `date` vindos da linha importada do extrato para sobrescrever os valores do template recorrente (que podem ter variado).

### 4. Marcar o Movement como pago

```
POST /v1/movements/:id/pay?date=2026-03-10
```

> O `date` no query param deve ser a data efetiva do pagamento (a data vinda do extrato).

---

## Estados Visuais na Tela de Revisão

| Estado da linha | Visual sugerido |
|---|---|
| Normal (vai criar novo) | Linha padrão com botão "Associar a recorrente" |
| Associada a uma recorrente | Badge verde "Associado a: [nome da recorrente]" + botão "Desfazer" |
| Excluída pelo usuário | Linha riscada ou oculta |

---

## Lógica de Confirmação Final (Step 2)

Ao clicar em "Confirmar Importação":

1. **Linhas normais** (não associadas, não excluídas): enviadas no `POST /v2/statements/confirm` como hoje.
2. **Linhas associadas**: para cada uma, executar em sequência:
   - `PUT /v1/movements/:id` → atualiza `amount` e `date`
   - `POST /v1/movements/:id/pay` → marca como pago
3. **Linhas excluídas**: ignoradas.

As chamadas das linhas associadas podem ser feitas em paralelo entre si (Promise.all), mas o `PUT` de cada linha deve preceder seu próprio `POST /pay`.

---

## Casos de Borda

- **Recorrente não tem Movement gerado para o mês ainda:** pode acontecer se o movimento recorrente do mês não foi gerado. Nesse caso, exibir mensagem ao usuário: *"O movimento deste mês ainda não foi gerado. Confirme a importação normalmente e associe depois."*
- **Movimento recorrente já está pago:** exibir aviso no modal ao lado da recorrente: *"Já pago neste mês"*. Permitir associação mesmo assim caso o usuário queira corrigir o valor.
- **Usuário não tem recorrentes cadastradas:** ocultar o botão "Associar a recorrente" ou desabilitá-lo com tooltip explicativo.

---

## O que NÃO muda

- O fluxo de Step 1 (extração) é **idêntico** ao atual. Nenhuma mudança de API ou payload.
- O Step 2 (`POST /v2/statements/confirm`) continua enviando apenas as linhas que não foram associadas.
- A lógica de deduplicação e idempotência do backend permanece intacta.
