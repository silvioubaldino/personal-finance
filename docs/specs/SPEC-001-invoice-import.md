---
id: SPEC-001
type: spec
title: "Invoice Import — api (Phases 1–3, 5 e 6)"
status: review
created: 2026-06-30
updated: 2026-09-12
owner: Silvio Ubaldino
parents: [AYD-004@context]
children: []
related: [GLO]
tags: [invoice, statement, import, ai]
superseded_by: null
---

# Spec: Invoice Import — api (Phases 1–3, 5 e 6)

> Detalha O QUÊ este repo (api) faz para cumprir o AYD-004 nas Fases 1, 2, 3, 5 e 6.
> Congela ao virar `approved`.

## Objetivo

Generalizar o pipeline de import de documentos financeiros (`/v2/statements`) para suportar
tanto extrato bancário (`Statement`) quanto fatura de cartão de crédito (`Invoice`), sem
quebrar os clientes existentes. Inclui:

- **Fase 1:** `/extract` aceita `source_type`; resposta ganha `document_type`, `confidence`,
  `warnings`; substitui hard-fail `ErrStatementNotAStatement` por `document_type: unknown` +
  warning `low_confidence`.
- **Fase 2:** Prompt dedicado para fatura (`invoice`), parsing de parcelas
  (`installment_number`/`total_installments`), `invoice_meta` (fechamento, vencimento, total).
  Feature label `invoice_extract` para tracking de tokens separado.
- **Fase 3:** Novo endpoint `POST /v2/statements/confirm-invoice` com
  `StatementUseCase.ConfirmInvoice` — cria `Movement`s com `TypePayment=credit_card`,
  `IsPaid=false`, reutiliza `InvoiceUseCase` para resolver/criar fatura e atualizar limite.
  Hash de idempotência escopado por `credit_card_id`.
- **Fase 5 (parcial):** Exclusão determinística do pagamento da fatura anterior
  (`IsInvoicePaymentDescription`) e checksum do total (`total_amount_mismatch`); gravação
  atômica de item/série (movimento + total da fatura + limite numa transação só).
- **Fase 6 — reconciliação de parcelas:** `/extract` aceita `credit_card_id` (opcional) e
  passa a (a) marcar como `future_installment` o item **parcelado** datado depois do fim do
  período da fatura alvo e (b) sugerir o vínculo com séries de parcelas já registradas
  (`installment_match`, pré-vinculado em `installment_group_id` quando a confiança é alta).
  No `confirm-invoice`, item com `installment_group_id` **vincula em vez de criar**:
  atualiza só o valor da parcela daquela competência, ajusta fatura e limite pelo delta e
  pula a série inteira. Inclui as validações de limite do cartão
  (`ErrInsufficientCreditLimit`, 403) e de carteira default (`ErrCreditCardNoDefaultWallet`,
  400) no caminho de fatura.

## Critérios de aceite

```gherkin
Cenário: Extração com source_type=invoice retorna campos de fatura
  Dado um arquivo PDF ou imagem de fatura de cartão
  E source_type="invoice" no form-data
  Quando POST /v2/statements/extract é chamado
  Então a resposta 200 contém document_type="invoice"
  E confidence entre 0.0 e 1.0
  E movements com type_payment="credit_card"
  E invoice_meta quando detectável no documento

Cenário: Extração com source_type=invoice mas IA detecta statement gera warning
  Dado um PDF de extrato bancário
  E source_type="invoice" no form-data
  Quando POST /v2/statements/extract é chamado
  Então a resposta 200 contém document_type="statement"
  E warnings inclui {type: "document_type_mismatch", expected: "invoice", detected: "statement"}
  E nenhum erro HTTP 4xx é retornado

Cenário: Extração sem source_type usa detecção automática
  Dado um arquivo PDF ou imagem
  E source_type ausente no form-data
  Quando POST /v2/statements/extract é chamado
  Então a resposta 200 contém document_type detectado pela IA
  E a resposta é retrocompatível com clientes que não enviam source_type

Cenário: Documento ambíguo retorna document_type=unknown e warning, nunca 4xx
  Dado um documento que a IA não consegue classificar com confiança
  Quando POST /v2/statements/extract é chamado
  Então a resposta é 200 (não 4xx)
  E document_type="unknown"
  E warnings inclui {type: "low_confidence"}
  E movements pode ser vazio

Cenário: confirm-invoice cria movimentos com TypePayment=credit_card e IsPaid=false
  Dado credit_card_id válido
  E movements com date, description e amount
  Quando POST /v2/statements/confirm-invoice é chamado
  Então cada movement é criado com TypePayment=credit_card e IsPaid=false
  E a resposta contém created > 0

Cenário: confirm-invoice deduplica por credit_card_id
  Dado que um movimento já foi importado anteriormente para o mesmo cartão
  (mesmo hash: userID + creditCardID + date + amount + description normalizada)
  Quando POST /v2/statements/confirm-invoice é chamado com o mesmo movimento
  Então o movimento é ignorado (skipped)
  E created=0, skipped=1

Cenário: confirm-invoice com parcelas chama geração de série de installments
  Dado um movimento com installment_number=3 e total_installments=12
  Quando POST /v2/statements/confirm-invoice é chamado
  Então são gerados 10 movimentos (restantes da parcela 3 até 12)
  E cada parcela vai para a fatura do mês correspondente

Cenário: confirm-invoice em fatura já paga retorna erro
  Dado credit_card_id cujo invoice alvo já está pago (IsPaid=true)
  Quando POST /v2/statements/confirm-invoice é chamado
  Então a resposta é 422
  E o body contém o erro ErrInvoiceAlreadyPaid

Cenário: confirm-invoice com cartão inexistente retorna erro
  Dado credit_card_id que não pertence ao usuário autenticado
  Quando POST /v2/statements/confirm-invoice é chamado
  Então a resposta é 404 ou 400

Cenário: Pagamento da fatura anterior é marcado, não removido
  Dado uma fatura cujos itens incluem a linha "PAGAMENTO ON LINE"
  Quando POST /v2/statements/extract é chamado com source_type="invoice"
  Então o item continua presente em movements
  E esse item tem excluded=true e exclusion_reason="invoice_payment"
  E warnings inclui {type: "invoice_payment_excluded"}

Cenário: Estabelecimento cujo nome contém "pag" não é marcado
  Dado uma fatura com os itens "PAGUE MENOS 1234" e "PAGSEGURO *LOJA"
  Quando POST /v2/statements/extract é chamado com source_type="invoice"
  Então nenhum dos dois é marcado com excluded=true

Cenário: Soma divergente do total declarado gera warning informativo
  Dado uma fatura cujo invoice_meta.total_amount não bate com a soma dos itens não marcados
  Quando POST /v2/statements/extract é chamado
  Então a resposta é 200 (nunca bloqueia)
  E warnings inclui {type: "total_amount_mismatch", expected, detected}

Cenário: Extrato bancário não sofre marcação de pagamento de fatura
  Dado um documento com document_type="statement" contendo "PAGAMENTO ON LINE"
  Quando POST /v2/statements/extract é chamado
  Então nenhum item é marcado com excluded=true

Cenário: confirm-invoice ignora o pagamento mesmo se o cliente enviá-lo
  Dado um cliente que ignora o campo excluded e envia a linha "PAGAMENTO ON LINE"
  Quando POST /v2/statements/confirm-invoice é chamado
  Então o item não é criado
  E é contabilizado em skipped
  E invoice.Amount e o limite do cartão não são alterados por ele

Cenário: Item de competência futura é marcado, não removido
  Dado uma fatura cujo cartão fecha no dia 3
  E itens majoritariamente na competência de maio/2026
  E um item "DROGARIA SP PARCELA 03 DE 03" datado de 2026-09-03
  Quando POST /v2/statements/extract é chamado com source_type="invoice" e credit_card_id
  Então o item continua presente em movements
  E esse item tem excluded=true e exclusion_reason="future_installment"
  E warnings inclui {type: "future_installment_excluded"}

Cenário: Compra comum depois do fechamento não é marcada como competência futura
  Dado uma fatura cujo cartão fecha no dia 3
  E itens majoritariamente na competência de maio/2026
  E um item "POSTO SHELL" datado de 2026-09-03, sem installment_number
  Quando POST /v2/statements/extract é chamado com source_type="invoice" e credit_card_id
  Então o item NÃO é marcado com excluded=true
  E nenhum warning future_installment_excluded é emitido
  # a regra vale só para PARCELAS (AYD-004 decisão 7); compra comum depois do
  # fechamento pertence à fatura seguinte e é parenteada pela resolução por data
  # (decisão 2) — marcá-la a faria sumir do import

Cenário: Item dentro do período da fatura não é marcado como competência futura
  Dado uma fatura cujo cartão fecha no dia 3
  E itens datados entre 2026-05-04 e 2026-06-03
  Quando POST /v2/statements/extract é chamado com source_type="invoice" e credit_card_id
  Então nenhum item é marcado com excluded=true

Cenário: Parcela já registrada com confiança alta vem pré-vinculada
  Dado um item "MERCADO LIVRE PARCELA 03/12" de -119,90
  E uma parcela 3/12 de -119,90 já registrada no mesmo cartão com a mesma raiz de descrição
  Quando POST /v2/statements/extract é chamado com credit_card_id
  Então o item traz installment_match preenchido
  E installment_group_id preenchido com o grupo da série existente
  E warnings inclui {type: "installment_match_found"}

Cenário: Parcela já registrada com confiança média vem só como sugestão
  Dado um item "MAGALU*MAGAZINELUIZA PARC 03/12" de -119,90
  E uma parcela 3/12 de -119,90 registrada como "TV da sala" no mesmo cartão
  Quando POST /v2/statements/extract é chamado com credit_card_id
  Então o item traz installment_match preenchido
  E installment_group_id vazio

Cenário: Extração sem credit_card_id permanece retrocompatível
  Dado uma fatura com item de competência futura e item parcelado já registrado
  Quando POST /v2/statements/extract é chamado sem credit_card_id
  Então nenhum item é marcado com exclusion_reason="future_installment"
  E nenhum item traz installment_match

Cenário: credit_card_id inválido no extract retorna 400
  Dado um credit_card_id que não é um uuid
  Quando POST /v2/statements/extract é chamado
  Então a resposta é 400 e o servidor não entra em pânico

Cenário: confirm-invoice vincula a parcela em vez de criar duplicatas
  Dado um item parcelado 3/12 de -120,50 com installment_group_id de uma série existente
  E a parcela 3/12 daquela série registrada hoje com -120,00
  Quando POST /v2/statements/confirm-invoice é chamado
  Então apenas o valor da parcela existente é atualizado para -120,50
  E description, date e is_paid da parcela permanecem inalterados
  E invoice.Amount e o limite do cartão são ajustados pelo delta (-0,50)
  E created=0 e skipped=10 (12 − 3 + 1, a série inteira)

Cenário: Vínculo inválido cai no caminho normal de criação
  Dado um installment_group_id inexistente, de outro usuário ou de outro cartão
  Quando POST /v2/statements/confirm-invoice é chamado
  Então o vínculo é rejeitado
  E a série é criada normalmente (created=10)

Cenário: confirm-invoice reaplica a regra de competência futura
  Dado um cliente que envia um item datado depois do fim do período da fatura alvo
  Quando POST /v2/statements/confirm-invoice é chamado
  Então o item não é criado
  E é contabilizado em skipped

Cenário: Estouro de limite do cartão retorna 403
  Dado um cartão com limite de R$ 100
  E um item de -5.000,00
  Quando POST /v2/statements/confirm-invoice é chamado
  Então a resposta é 403 com ErrInsufficientCreditLimit
  E nenhum movimento do item é criado

Cenário: Cartão sem carteira default retorna 400
  Dado um credit_card_id cujo cartão não tem default_wallet_id
  Quando POST /v2/statements/confirm-invoice é chamado
  Então a resposta é 400 com ErrCreditCardNoDefaultWallet

Cenário: Extract com ErrStatementNotAStatement legado vira soft-fail
  Dado que o gateway retorna ErrStatementNotAStatement
  Quando StatementUseCase.Extract é chamado
  Então o resultado é StatementExtractResult com document_type=unknown
  E warnings inclui low_confidence
  E nenhum erro é propagado ao handler
```

## Contratos consumidos/expostos

Contratos definidos em AYD-004@context. Este repo NÃO os redefine.

### Endpoint: `POST /v2/statements/extract` (estendido)
- Novo campo de form-data: `source_type` (opcional, `"statement"` | `"invoice"`)
- Novo campo de form-data: `credit_card_id` (opcional, uuid) — habilita os enriquecimentos
  de fatura (competência futura e vínculo de parcela); uuid inválido → 400
- Resposta 200 ganha campos aditivos: `document_type`, `confidence`, `warnings[]`, `invoice_meta`
- `ExtractedMovement` ganha `installment_number` e `total_installments` (opcionais)
- `ExtractedMovement` ganha `excluded` e `exclusion_reason` (opcionais) — itens que não
  pertencem à fatura, marcados e nunca removidos
- `ExtractedMovement` ganha `installment_match` (sugestão do servidor) e
  `installment_group_id` (decisão do cliente), ambos opcionais
- `warnings[].type` ganha `invoice_payment_excluded`, `total_amount_mismatch`,
  `future_installment_excluded` e `installment_match_found`
- `exclusion_reason` ganha o valor `future_installment`
- Retrocompatível: clientes sem `source_type`/`credit_card_id` continuam recebendo o mesmo
  comportamento

### Endpoint: `POST /v2/statements/confirm-invoice` (novo)
- Request: `{ credit_card_id, invoice_id?, movements[] }`; cada movimento aceita
  `installment_group_id` (vincula a uma série já registrada em vez de criar)
- Response: `{ created, skipped, errors[] }` (mesmo shape do `/confirm`)
- Erros: `ErrInvoiceAlreadyPaid` (422), cartão não encontrado (404),
  `ErrCreditCardNoDefaultWallet` (400), `ErrInsufficientCreditLimit` (403)

### `POST /v2/statements/classify` — inalterado
### `POST /v2/statements/confirm` — inalterado (caminho statement)

## Modelo de dados / componentes afetados

- `internal/domain/statement.go`:
  - `DocumentType` enum (`statement` | `invoice` | `unknown`)
  - `ExtractWarning` struct
  - `InvoiceMeta` struct
  - `InvoiceConfirmInput` struct
  - `ExtractedMovement` ganha `InstallmentNumber`, `TotalInstallments`, `Excluded`,
    `ExclusionReason`, `InstallmentMatch` e `InstallmentGroupID`
  - `InstallmentMatch` struct (Fase 6) — sugestão de vínculo com uma série já registrada
  - Constantes `Warning*` (tipos de aviso), `ExclusionReasonInvoicePayment` e
    `ExclusionReasonFutureInstallment`
  - `IsInvoicePaymentDescription` — detecção determinística do pagamento de fatura anterior
  - `StripInstallmentSuffix` (Fase 6) — raiz estável da descrição, removendo o sufixo de
    parcela (`03/12`, `PARC 3/12`, `PARCELA 03 DE 12`…) antes de normalizar. Só remove
    quando `1 ≤ n ≤ total` e `total ≥ 2`, para não mutilar nomes como "POSTO 24/7"
  - `InstallmentMatchConfidence` (Fase 6) — assinatura do match: mesmo total de parcelas,
    mesmo número de parcela, valor dentro de `InstallmentAmountTolerance` (R$ 0,05) e raiz
    da descrição → alta (`0.95`); raiz divergente → média (`0.7`); resto → nenhuma
  - `ResolveTargetInvoicePeriodEnd` (Fase 6) — fim do período da fatura alvo, pela **moda**
    dos períodos dos itens sobre o dia de fechamento do cartão
  - `StatementExtractResult` ganha `DocumentType`, `Confidence`, `Warnings`, `InvoiceMeta`
  - `ComputeIdempotencyHash` generalizado: aceita `scopeKey string` (walletID.String() ou creditCardID.String())

- `internal/usecase/statement_usecase.go`:
  - `markItemsNotBelongingToInvoice` / `appendTotalMismatchWarning` — camadas 2 e 3
  - `ConfirmInvoice` ignora itens marcados e reaplica a detecção
  - `StatementVisionGateway.ExtractMovements` agora recebe `sourceType string`
  - `StatementInvoiceUseCase` interface (estreita, declarada aqui) — reduzida a
    `FindOrCreateInvoiceForMovement`: o total da fatura passou a ser escrito pelo repositório,
    dentro da transação
  - `StatementInvoiceRepository` interface (estreita, declarada aqui) — `UpdateAmount` com `tx`
  - `StatementCreditCardRepository` interface (estreita, declarada aqui)
  - `StatementUseCase` ganha `invoiceUseCase`, `invoiceRepo`, `creditCardRepo` e `txManager`
  - `persistInvoiceMovements` — grava movimentos + total da fatura + limite do cartão numa
    transação única; deltas acumulados por fatura (um update por fatura)
  - `saveInstallmentSeries` — resolve as faturas fora da transação e grava a série inteira
    dentro de uma só; checagem explícita de fatura paga (antes vinha da `InvoiceUseCase`)
  - `Extract` recebe `sourceType string` e `creditCardID *uuid.UUID`; soft-fail para
    `ErrStatementNotAStatement`
  - `ConfirmInvoice` método novo
  - `enrichInvoiceExtraction` / `markFutureInstallments` / `suggestInstallmentMatches`
    (Fase 6) — enriquecimentos do `/extract` quando há cartão de destino; falha de leitura
    do cartão ou do histórico é no-op, nunca derruba a extração
  - `linkInstallmentSeries` / `findLinkedInstallment` (Fase 6) — caminho de vínculo do
    `confirm-invoice`, com revalidação do grupo (usuário, cartão, número e total de
    parcelas) antes de escrever
  - `creditLimitGuard` (Fase 6) — limite do cartão debitado em memória ao longo da
    importação, sobre `domain.CreditCard.HasSufficientLimit` (mesma regra do lançamento
    manual); estouro aborta a requisição com `ErrInsufficientCreditLimit`
  - `invoiceMovementItem` ganha `linkTo`/`delta`: `persistInvoiceMovements` passa a
    atualizar o valor de uma parcela existente, em vez de inserir, quando há vínculo

- `internal/infrastructure/gateway/gemini_vision_gateway.go`:
  - Prompts: `statementExtractionPrompt` (atualizado), `invoiceExtractionPrompt` (novo), `autoDetectionPrompt` (novo)
  - Seleção de prompt por `sourceType`
  - Feature label `invoice_extract` para tokens de fatura
  - Novo shape de resposta JSON: objeto com `document_type`, `confidence`, `invoice_meta`, `movements`

- `internal/infrastructure/repository/movement_repository.go` (Fase 6):
  - `FindInstallmentCandidatesByCreditCard` — candidatos a vínculo por cartão (join com
    `invoices`, onde mora o `credit_card_id`) + `total_installments`, escopado por usuário
    via `BuildBaseQuery`
  - `UpdateAmount` — atualização **estreita** do valor de um movimento.
    `UpdateStatementLink` não serve: sobrescreve descrição, data e carteira e força
    `is_paid = true`

- `internal/infrastructure/api/statement_api.go`:
  - `StatementUsecase` interface ganha `ConfirmInvoice`
  - Handler `ConfirmInvoice` registrado em `POST /v2/statements/confirm-invoice`
  - Handler `Extract` lê os campos `source_type` e `credit_card_id` do form-data

- `internal/infrastructure/api/errors_handler.go`:
  - `ErrInsufficientCreditLimit` mapeado para 403 (antes caía no 500 genérico). É o estouro
    do limite do próprio cartão — o "403 `ErrCreditCardLimitReached`" do AYD-004 —, distinto
    de `ErrCreditCardLimitReached`, que é o teto de cartões do plano do usuário

- `internal/bootstrap/statement/setup.go`:
  - Injeta `InvoiceUseCase`, `InvoiceRepository`, `CreditCardRepository` e o `txManager` no
    `StatementUseCase` (todos já estavam no escopo do `Setup`)

- `db/migrations/027_add_movements_indexes.{up,down}.sql`:
  - `idx_movements_idempotency_hash` sobre `(user_id, idempotency_hash)` — a query do dedup
    (`FindExistingHashes`) filtra pelas duas colunas; a `019` criou a coluna e o seu `down` já
    dropava um índice que o `up` nunca criou
  - `idx_movements_installment_group` sobre `(installment_group_id, installment_number)` —
    ordem das colunas segue o filtro + `ORDER BY` de `FindByInstallmentGroupFromNumber`
  - Ambos parciais (`WHERE ... IS NOT NULL`): a maioria dos movements não tem hash nem grupo

## Casos de borda & fora de escopo

- **Borda:** documento ambíguo nunca retorna 4xx — sempre 200 com `document_type=unknown` + warning
- **Borda:** `source_type` ausente = modo auto-detecção (retrocompatível)
- **Borda:** importar em fatura já paga → 422 ao encontrar o primeiro item com invoice paga
- **Borda:** movimento parcelado gera série a partir da parcela `installment_number` até `total_installments`
- **Fora de escopo:** Fase 4 e a UI da Fase 6 (web/mobile — bifurcação do confirm, entrada
  "Importar fatura", chip de vínculo de parcela e vínculo manual)
- **Borda:** o pagamento da fatura anterior é detectado em três camadas (prompt, guarda
  determinística por primeira palavra, checksum do total), **marcado e não removido**; o
  `confirm-invoice` reaplica a detecção e não confia no cliente
- **Borda:** o checksum compara **módulos** — o modelo é inconsistente no sinal do
  `total_amount` (já devolveu `"6035.06"` positivo para fatura de itens negativos)
- **Borda:** a persistência de um item (movimento + total da fatura + limite do cartão) é
  atômica; para item parcelado a unidade é a **série inteira** — série pela metade infla a
  fatura sem representar a compra. A granularidade continua por item, não por requisição:
  sucesso parcial segue sendo o comportamento do `confirm-invoice`
- **Borda:** a resolução da fatura (`FindOrCreateInvoiceForMovement`) roda **fora** da
  transação — ela abre a própria, e aninhá-la criaria uma transação paralela que commitaria
  por fora do rollback. Fatura criada sem itens é inofensiva
- **Borda:** a fatura alvo da regra de competência futura é a **moda** dos períodos dos
  itens: usar a menor data seria frágil (vários bancos datam a parcela pela data da compra
  original) e a maior também (a fatura lista parcelas de competência futura). Empate resolve
  pelo período mais antigo. Sem dia de fechamento válido no cartão, a regra é no-op
- **Borda:** vínculo de parcela é idempotente — reimportar a mesma fatura aplica delta zero
  na fatura e no limite; a série converge para os valores reais mês a mês, e as parcelas
  futuras seguem com o valor estimado até a fatura correspondente chegar
- **Borda:** vínculo inválido (grupo inexistente, de outro usuário ou de outro cartão) não é
  erro: o item cai no caminho normal de criação. O grupo de outro usuário nem chega ao
  matcher, porque a consulta é escopada por `BuildBaseQuery`
- **Fora de escopo:** métricas `biz_invoice_imports_total` avançadas (resto da Fase 5;
  a validação de total foi implementada junto com a exclusão do pagamento)
- **Fora de escopo:** heurísticas estruturais de texto (mencionar vs. depender da IA apenas)
- **Fora de escopo:** `POST /v2/statements/classify` — inalterado, sem mudanças nesta SPEC
