---
id: AYD-007
type: design
title: Testes de aceite determinísticos do import (extrato e fatura)
status: draft
created: 2026-11-25
updated: 2026-11-25
owner: Silvio Ubaldino
affects: [api]
parents: [REQ-001]
children: []
related: [AYD-004, AYD-006, SPEC-001@api, SPEC-008@api, TDR-001@api, GLO]
tags: [testing, acceptance, import, statement, invoice, installments]
superseded_by: null
---

# AYD-007: Testes de aceite determinísticos do import (extrato e fatura)

> **Rascunho no api.** Escrito numa branch derivada do PR #216 para ficar ao lado do código
> que ele testa. Onde este doc mora depois de aprovado é a questão Q1.

## Objetivo

O import de documento financeiro (`Statement` e `Invoice`) tem duas metades:

1. **a IA lê o documento** — probabilística, varia entre leituras;
2. **o sistema decide o que entra, onde entra e como** — deve ser determinística, e é a metade
   que corrompe dados quando erra: duplica série de parcelas, perde item, infla fatura, consome
   limite, desloca saldo.

Este AYD desenha uma camada de testes de aceite, na suíte Gherkin do SPEC-008, que valida
**só a segunda metade**. A IA é substituída por um dublê que devolve exatamente o que o
documento contém. Cada cenário responde a uma pergunta: *dado que a IA leu X, o app ficou
certo?*

Medir a qualidade da IA (prompt, OCR, leitura de layout) fica fora (ver §Fora de escopo).

## Repos afetados e papéis

| Repo | Papel | Doc gerado |
|---|---|---|
| api | Costura para trocar os gateways de IA, dublês, fixtures, passos e features | SPEC@api (ver Q1) |
| web / mobile | Nenhum código. Os PDFs de fixture servem ao teste manual da UI (roteiro do PR #216, Parte C/D) | — |

**Sem mudança de contrato.** Nenhum endpoint, payload ou enum muda. A única alteração em código
de produção é uma costura de injeção de dependência (§Costura).

## Propriedades que a suíte garante

Valem para extrato e fatura, salvo indicação. Cada cenário declara as que exercita.

| # | Propriedade | Verificação |
|---|---|---|
| **P1 Completude** | Todo item que pertence ao documento vira valor no app, **mesmo sem categoria entendida e com nome ilegível**. Na fatura, a fatura da competência no app fecha com o total do documento. No extrato, o saldo da carteira muda exatamente pela soma das linhas | asserção de cenário |
| **P2 Sem duplicação** | Reimportar não cria nada. A parcela de uma série conhecida vincula em vez de criar. Nenhuma série tem duas parcelas com o mesmo número | invariante I5 e asserção |
| **P3 Associação** | Parcela já registrada (à mão ou por import anterior) é vinculada e tem o valor corrigido; `description`, `date` e `is_paid` ficam intactos. No extrato, linha vinculada a recorrência atualiza a ocorrência do mês | asserção |
| **P4 Exclusão** | O que não pertence ao documento (pagamento da fatura anterior, parcela de competência futura) não entra, **mesmo que o cliente insista** | asserção |
| **P5 Consistência contábil** | Depois de qualquer import valem fatura = Σ itens (I1), limite (I2) e saldo (I3) | invariantes do SPEC-008 |
| **P6 Categoria nunca bloqueia** | Sem sugestão, o item vai para a categoria de fallback pelo sinal (AYD-006). Com histórico, usa a categoria do histórico | asserção |
| **P7 Determinismo** | Mesmo documento e mesmo estado dão o mesmo resultado. Sem rede e sem IA, roda no CI atual sem segredo | arquitetura |

## Arquitetura do teste

### Onde a IA é cortada

```mermaid
sequenceDiagram
    participant S as Passo Gherkin
    participant H as Handler (real)
    participant U as StatementUseCase (real)
    participant D as pdfcpu (real)
    participant V as FixtureVisionGateway (dublê)
    participant C as FixtureClassificationGateway (dublê)
    participant DB as Postgres efêmero

    S->>H: POST /extract (multipart: PDF, source_type, credit_card_id)
    H->>U: Extract
    U->>D: decripta (senha testável)
    U->>V: ExtractMovements
    V-->>U: leitura gabarito do documento
    U->>DB: enrich: competência futura, match de parcela
    U-->>S: extração (excluded, installment_match, warnings)
    S->>H: POST /classify
    U->>DB: histórico de categorias
    U->>C: itens sem histórico
    C-->>U: nenhuma sugestão (padrão)
    Note over S: revisão do usuário simulada
    S->>H: POST /confirm ou /confirm-invoice
    U->>DB: dedup, vínculo, séries, fatura, limite, saldo
    S->>DB: asserções + invariantes I1–I6
```

Só os dois gateways Gemini são trocados. Todo o resto roda como em produção: handler,
validação de tamanho e mime, decriptação pelo pdfcpu, marcação de pagamento, checksum do total,
competência futura, matcher de parcela, classificação por histórico, dedup e persistência.

### Costura no código de produção

Hoje `statement.Setup` constrói `GeminiVisionGateway` e `GeminiClassificationGateway`
diretamente, e `app.Config` não oferece como trocá-los. Proposta:

- `app.Config` ganha um campo opcional `StatementGateways` (`Vision`, `Classification`).
  `nil` significa Gemini, como hoje.
- `bootstrap.SetupCleanArchComponents` recebe o override por **opção funcional variádica**,
  então `cmd/api/main.go` não muda.
- `statement.Setup` usa o override quando presente; senão constrói o Gemini como hoje.

Isso segue a regra da skill `go-bootstrap`: gateway externo continua construído no `Setup` da
feature e nunca fica memoizado no `Registry`. As interfaces `StatementVisionGateway` e
`StatementClassificationGateway` já existem no usecase; o dublê só as implementa.

### Dublê de visão: `FixtureVisionGateway`

Devolve, para cada documento recebido, a **leitura gabarito**: o `StatementExtractResult` que
uma IA perfeita devolveria.

- **Identificação:** por um ID gravado nos metadados do PDF (`/Subject: pf-fixture:<id>`), não
  por hash dos bytes. O metadado sobrevive à decriptação do pdfcpu, então a versão com senha
  usa o mesmo gabarito.
- **Duas fontes de gabarito, mesmo mecanismo:**
  1. **Arquivo:** `extraction.json` da fixture, gerado junto com o PDF (§Fixtures). Serve às
     jornadas realistas.
  2. **Tabela no cenário:** o passo registra a leitura no dublê sob um ID único do cenário e
     sobe um PDF mínimo com esse ID. Serve aos casos de borda e à **leitura imperfeita** (item
     parcelado sem `installment_number`, nome truncado, total com sinal trocado), sem exigir
     um PDF por caso.
- **Concorrência:** o registro é por ID único, então é seguro com `GODOG_CONCURRENCY=4`.
- **Erro explícito:** PDF com ID desconhecido falha o teste. Nunca devolve vazio em silêncio.

### Dublê de classificação: `FixtureClassificationGateway`

Por padrão **não sugere nada**, o que força o caminho "a IA não entendeu a categoria" (P6). O
cenário pode declarar sugestões (`And the AI suggests category "Transport" for "UBER *TRIP"`).
A classificação **por histórico** vem do banco e roda de verdade.

### Revisão do usuário simulada

Entre `extract` e `confirm`, web e mobile mostram a revisão. O passo de import reproduz o
**cliente padrão**:

- manda tudo o que veio;
- preserva `excluded`;
- mantém o `installment_group_id` pré-preenchido;
- aplica as sugestões de categoria.

Desvios do padrão são passos explícitos:

| Passo | Efeito no payload do confirm |
|---|---|
| `I accept the suggested link for "<item>"` | `installment_group_id` ← grupo de `installment_match` |
| `I reject the link for "<item>"` | `installment_group_id` ← vazio |
| `I link "<item>" to the purchase "<M>"` | vínculo manual a uma série existente |
| `I re-include "<item>"` | `excluded` ← `false` (o servidor deve recusar mesmo assim, P4) |
| `I link the line "<item>" to the recurrent expense "<R>"` | `recurrence_id` (extrato) |

Assim o ciclo inteiro roda sem HTTP no Gherkin, como pede o SPEC-008.

### Por que PDF, se a IA é dublê

O PDF não é lido pelo dublê, mas paga o custo:

- exercita o upload multipart real, a validação de tamanho e mime e a **decriptação real**
  (cenários de senha);
- é a versão legível do gabarito: quem revisa um cenário abre o PDF e vê o que o app recebeu;
- os mesmos arquivos servem ao teste funcional manual com a IA real e à UI de web e mobile.

## Vocabulário de passos (novo)

**Import**
- `I import the invoice "<fixture>" into card "<K>"`: extract (`source_type=invoice`,
  `credit_card_id`), classify, revisão padrão e `confirm-invoice`.
- `I import the statement "<fixture>" into wallet "<W>"`: extract (`source_type=statement`),
  classify, revisão padrão e `confirm`.
- Em duas etapas, quando o cenário precisa inspecionar a extração:
  `I extract the invoice "<fixture>" for card "<K>"`, depois as asserções, depois
  `I confirm the import`.
- Inline: `I import an invoice into card "<K>" read as:` com a tabela
  `| date | description | amount | installment |` e o complemento opcional
  `declaring total <n>`. O equivalente para extrato é `a statement into wallet "<W>"`.
- Sufixo opcional no upload: `with password "<p>"`.

**Asserções**
- `the extraction marks:` com a tabela `| description | as |`, onde `as` é um de
  `invoice payment`, `future installment`, `linked to "<M>"`, `suggested "<M>"` ou `new`.
- `the import created <n> and skipped <m>`.
- `the invoice of card "<K>" due in "<yyyy-mm>" matches the document total` (P1).
- `every item of the document is in the app` (P1): cada item não excluído do gabarito tem um
  movimento correspondente, criado ou vinculado.
- `the installments of "<M>" on card "<K>" are:` com a tabela `| installment | due | amount |`
  (P2, P3).
- `"<item>" is in category "<C>"` (P6).
- Os passos existentes de fatura, limite, saldo e `the operation is rejected as` são reusados.

## Invariantes novas

Somam-se às I1–I4 do SPEC-008 e rodam no `After` de **todo** cenário da suíte, não só nos de
import:

| # | Invariante |
|---|---|
| **I5** | Em cada `installment_group_id`: `installment_number` único e dentro de `1..total_installments`; todas as parcelas com o mesmo `total_installments` e o mesmo cartão |
| **I6** | `idempotency_hash` único por usuário |

A I5 pega duplicação de série em qualquer cenário, inclusive nos que não a procuram.

## Fixtures

### Geração

- **Fonte única:** `test/acceptance/fixtures/import/<id>/document.yaml` (cartão ou carteira,
  período, itens). Dela saem:
  - `document.pdf`: layout do Inter para fatura e um extrato bancário genérico para statement;
  - `extraction.json`: a leitura gabarito que o dublê devolve.

  PDF e gabarito não divergem porque nascem do mesmo arquivo.
- **Gerador:** Python, HTML e Chromium headless (o mesmo motor que gerou a fatura usada como
  molde), em `test/acceptance/fixtures/import/generator/`. É ferramenta de dev: o CI não roda o
  gerador, e os arquivos gerados são versionados. Itens "aleatórios" usam semente fixa, então
  regerar dá o mesmo PDF.
- **Mais leves que o original:** 2 a 3 páginas, sem imagem nem QR code. Ficam só as seções que
  importam ao import:
  - resumo (total, vencimento, limite);
  - "Despesas da fatura" com **dois cartões** (titular e adicional) e subtotal de cada;
  - "Próxima fatura".
- **Convenções do Inter preservadas:**
  - data `17 de set. 2026`;
  - parcela `(Parcela 04 de 06)`, datada pela **data da compra original**;
  - crédito como `+ R$`;
  - descritores de adquirente (`UBER *TRIP`, `IFD*`, `MERCADOLIVRE*`, `MP *CONECTCAR`,
    `DL*GOOGLE`);
  - encargos (IOF, juros).
- **Dados 100% fictícios** (titular, finais de cartão, linha digitável). A fatura real usada
  como molde **não entra no repo**.
- **Teste de integridade:** valida cada fixture. O ID do metadado deve bater com o nome da
  pasta, e a Σ dos itens não excluídos deve bater com o total declarado, exceto na fixture que
  diverge de propósito.

### Linha do tempo

Um cartão "Inter" (fechamento dia 3, vencimento dia 10, limite 5.000,00) e uma carteira
"Checking", ao longo de três meses seguidos. As parcelas avançam entre as faturas como no mundo
real.

| Fixture | Conteúdo | Serve a |
|---|---|---|
| `inter-2026-08` | ~18 itens em 2 cartões; 2 parceladas novas (`01 de 03`, `SAMSUNG*LOJA 01 de 10`); 1 parcela de compra anterior ao app (`04 de 06`); `PAGAMENTO ON LINE`; estorno; 2 pedágios idênticos no mesmo dia; IOF e juros | base, exclusões, idênticos, parcelada nova |
| `inter-2026-09` | Continua as séries (`02 de 03`, `02 de 10`, `05 de 06`), uma com centavos diferentes; 1 parcelada nova; seção "Próxima fatura" | vínculo, delta |
| `inter-2026-10` | Fecha séries (`03 de 03`, `06 de 06`) e traz `03 de 10` | convergência e término |
| `inter-2026-08-protegida` | `inter-2026-08` com senha | decriptação real com o mesmo gabarito |
| `extrato-2026-08` | Conta corrente: salário, PIX, boleto, Netflix e aluguel (recorrentes), `PAG FATURA CARTAO`, 2 PIX idênticos no mesmo dia | extrato base, recorrência, idênticos |
| `extrato-2026-08-segunda-quinzena` | Período sobreposto ao anterior (dias 10 a 31) | sobreposição sem duplicar |

Os casos de borda usam tabela inline e dispensam PDF: competência futura, total divergente,
falsos positivos (`PAGUE MENOS`, `PAGSEGURO`, `PAGBANK`) e leitura imperfeita.

## Catálogo de cenários

Pastas: `features/import/invoice/`, `features/import/statement/` e `features/journeys/`.

Cada cenário descreve o comportamento **desejado**. Onde a api diverge:
- se o desejado **já está decidido**, o cenário leva `@known-bug` e entra na tabela de
  divergências do SPEC-008;
- se o desejado **ainda não foi decidido** (achados do AYD-004), leva `@wip` e vira questão
  aqui. Um teste não inventa decisão de produto.

### Fatura

| Feature | Cenários | Prop. |
|---|---|---|
| `invoice_import` | Todos os itens entram e a fatura fecha com o total do documento · item sem categoria vai para o fallback pelo sinal · nome ilegível é registrado como veio · compras dos 2 cartões físicos vão para o mesmo `CreditCard` · limite consumido = total importado | P1 P5 P6 |
| `invoice_import_idempotency` | Reimportar a mesma fatura não cria nada e não mexe em fatura nem limite · reimport de correção (o primeiro import perdeu 1 linha) cria só a que faltava · reimport com a IA lendo o nome com outra pontuação (R5) | P2 |
| `invoice_import_installments` | Parcelada nova cria a série restante · compra registrada à mão com o mesmo nome vincula sozinha (R7) · compra registrada como "Notebook" vem sugerida: aceitar vincula, recusar cria série nova · centavos de arredondamento ajustam só a parcela, e fatura e limite pelo delta · parcela de compra anterior ao app (`04 de 06`) cria só 4..6 · vínculo inválido cai na criação · parcela datada pela compra original entra na fatura importada (R1) | P2 P3 |
| `invoice_import_exclusions` | Pagamento anterior é marcado e não importado · cliente re-inclui o pagamento e o servidor recusa · `PAGUE MENOS`, `PAGSEGURO` e `PAGBANK` entram · parcela de competência futura é excluída, compra comum na mesma data entra · estorno reduz a fatura e devolve limite | P4 P5 |
| `invoice_import_identical_items` | Dois pedágios iguais no mesmo dia entram os dois · reimportar continua sem duplicar (R2) | P1 P2 |
| `invoice_import_rejections` | Estouro de limite é rejeitado e nada do item é criado · cartão sem carteira default é rejeitado · fatura já paga (`@wip`, achado 4 do AYD-004) · PDF com senha: ausente, errada, certa | P5 |
| `invoice_import_imperfect_reading` | Parcelada lida sem `installment_number`, com a parcela só no texto (R6) · total declarado com sinal trocado não gera aviso · data ilegível em 1 item (R4) | P1 P2 |

### Extrato

| Feature | Cenários | Prop. |
|---|---|---|
| `statement_import` | Todas as linhas entram pagas na carteira, com saldo = inicial + Σ · entrada sem categoria vai para o fallback de receita (AYD-006) · linha já categorizada no passado usa o histórico · plano `free` para no teto e reporta o resto | P1 P5 P6 |
| `statement_import_idempotency` | Reimport não cria nada · extratos com período sobreposto não duplicam a interseção | P2 |
| `statement_import_recurrence` | Linha vinculada a recorrência atualiza a ocorrência do mês, sem duplicar · reimportar o extrato sem refazer o vínculo não duplica a ocorrência (R3) | P2 P3 |
| `statement_import_identical_lines` | Dois PIX iguais no mesmo dia entram os dois (R2) | P1 |

### Jornada

`invoice_import_three_months`, em sequência:
1. importa a fatura de ago;
2. registra uma compra parcelada à mão;
3. importa a fatura de set;
4. paga a fatura de ago;
5. importa a fatura de out.

A cada passo, a fatura bate com o total do documento, nenhuma série tem duplicata (I5), o
limite fecha (I2) e as séries convergem e terminam no número certo.

### Exemplos

```gherkin
@import @invoice @installments
Scenario: An installment purchase registered by hand is linked, not duplicated
  Given today is "2026-09-05"
  And a credit card purchase "SAMSUNG*LOJA" in 10 installments of 349.90 starting "2026-07-20" on card "Inter" under category "Electronics"
  When I import the invoice "inter-2026-09" into card "Inter"
  Then the extraction marks:
    | description                     | as                       |
    | SAMSUNG*LOJA (Parcela 02 de 10) | linked to "SAMSUNG*LOJA" |
    | PAGAMENTO ON LINE               | invoice payment          |
  And the installments of "SAMSUNG*LOJA" on card "Inter" are:
    | installment | due     | amount  |
    | 1           | 2026-08 | -349.90 |
    | 2           | 2026-09 | -349.93 |
    | 3           | 2026-10 | -349.90 |
    | ...         | ...     | ...     |
  And the invoice of card "Inter" due in "2026-09" matches the document total

@import @invoice @identical
Scenario: Two identical tolls on the same day are both imported
  Given today is "2026-08-05"
  When I import an invoice into card "Inter" read as:
    | date       | description   | amount |
    | 2026-07-19 | MP *CONECTCAR | -4.37  |
    | 2026-07-19 | MP *CONECTCAR | -4.37  |
  Then the import created 2 and skipped 0
  And every item of the document is in the app
```

O primeiro exemplo é justamente o que expõe a R7: escrito como desejado, ele falha hoje.

## Riscos que os cenários devem expor

Levantados lendo o código do PR #216. **Confirmados** foram verificados com teste unitário
descartável no domínio; **hipóteses** vêm só da leitura do código. Todos viram cenário e são
triados na implementação.

| # | Risco | Estado | Onde |
|---|---|---|---|
| **R1** | **Parcela datada pela compra original vai para a fatura errada.** O Inter lista `EmilioAutoCenter (Parcela 04 de 06)` com a data da compra (17/jun) na fatura de out. O import trata a data do item como a data da parcela N (`GenerateInstallmentMovements`: data + i meses) e resolve a fatura por data (`saveInstallmentSeries`, sempre sem override). A parcela 4 cai na fatura de jul, ou é pulada se jul estiver paga, e as seguintes também ficam deslocadas. A fatura importada não fecha com o PDF | hipótese | `ConfirmInvoice`, `saveInstallmentSeries` |
| **R2** | **Linhas idênticas viram uma só.** O hash é usuário + escopo + data + valor + descrição. O segundo pedágio de 4,37 no mesmo dia colide com o primeiro, inclusive dentro da mesma requisição (`existingHashes[hash] = true`), e é pulado como duplicata. Fere P1. Vale para extrato e fatura | hipótese | `parseAndHashMovements`, `Confirm` |
| **R3** | **Reimport depois de vínculo de recorrência duplica.** A linha vinculada a recorrência é gravada sem `idempotency_hash`, porque o caminho de vínculo roda antes do dedup. Reimportar o extrato sem refazer o vínculo cria a linha de novo | hipótese | `Confirm` (statement) |
| **R4** | **Uma data ilegível derruba o import inteiro.** `parseAndHashMovements` devolve 400 no primeiro item com data inválida, e nenhum item entra | hipótese | `parseAndHashMovements` |
| **R5** | **Leitura com pontuação diferente fura o dedup.** `NormalizeDescription` remove `*` mas preserva o espaço: `MP *CONECTCAR` vira `mp conectcar` e `MP*CONECTCAR` vira `mpconectcar`. A IA real varia assim entre leituras, e o reimport duplica | **confirmado** | `NormalizeDescription` |
| **R6** | **Parcela sem campos de parcela vira compra avulsa.** Se a IA não preenche `installment_number`, o item entra avulso mesmo com `(Parcela 02 de 03)` no texto; se a série já existe, duplica o valor daquela competência. `StripInstallmentSuffix` já reconhece o padrão e poderia servir de fallback | hipótese | `ConfirmInvoice` |
| **R7** | **Os parênteses do Inter rebaixam o match.** `StripInstallmentSuffix("EmilioAutoCenter (Parcela 05 de 06)")` deixa `()`, e a raiz normalizada vira `"emilioautocenter "`, com espaço no fim. A compra registrada à mão como `EmilioAutoCenter` tem raiz `"emilioautocenter"`. A confiança cai de alta para média: a parcela vem só sugerida, e o cliente padrão cria a série de novo. Séries criadas pelo próprio import não sofrem (as duas raízes têm o espaço) | **confirmado** | `StripInstallmentSuffix` |

Os achados de set/2026 do AYD-004 ainda sem decisão entram como `@wip`: estorno classificado
como receita, limite de plano no import de fatura, `invoice_id` sobrepondo a data, fatura paga
abortando no meio, e pagamento da fatura × AYD-003. O reimport de correção, que era achado,
ganha cenário com o comportamento atual (cria só o que faltava) **proposto** como desejado;
a confirmação vai para o AYD-004.

## Plano faseado

| Fase | Entrega |
|---|---|
| **F0** | Costura (`app.Config`, opção em `SetupCleanArchComponents`, `statement.Setup`), os dois dublês e as invariantes I5 e I6 |
| **F1** | Gerador, fixtures (PDF + gabarito) e teste de integridade das fixtures |
| **F2** | Passos de import e features de fatura |
| **F3** | Features de extrato |
| **F4** | Jornada de três meses; triagem: cada falha vira `@known-bug` (desejado decidido) ou `@wip` (sem decisão); atualização da matriz e das divergências do SPEC-008 |

Correções de R1 a R7 vão em **PRs próprios**, que tiram a tag do cenário (regra do SPEC-008).
Esta entrega só os torna visíveis e reproduzíveis.

Para `make test-acceptance`, o padrão continua `~@known-bug && ~@wip`, e o CI não ganha segredo
nem acesso de rede.

## Fora de escopo

- **Qualidade da IA** (prompt, OCR, leitura de layout): não há perfil com Gemini real. Os PDFs
  continuam servindo ao teste manual com a IA real, fora da suíte.
- Fixtures de imagem (JPEG/PNG). O mecanismo serve igual; fica para depois.
- UI de web e mobile (bifurcação do confirm, chip de vínculo).
- Extrato contendo o pagamento da fatura × itens importados da fatura (achado 6 do AYD-004).
- `InternalTransfer` reconhecida em extrato.

## Questões em aberto

- **Q1 — Onde este doc mora.** A proposta é engenharia interna do api e não muda contrato, o
  mesmo caso do SPEC-008, que nasceu como SPEC com `parents: [TDR-001]` por exceção consciente.
  Opções: (a) mover para `personal-finance-context/design/` como AYD-007 e gerar a SPEC@api;
  (b) converter direto em SPEC@api com `parents: [TDR-001]` e `related: [AYD-004]`.
  **Recomendo (b).** As decisões de produto que os riscos pedem (Q2 a Q6) vão para o AYD-004
  de qualquer forma.
- **Q2 — R2:** proposta de incluir no hash o **ordinal da linha idêntica no documento** (a
  n-ésima linha igual), preservando a idempotência do reimport. Muda a composição do hash
  descrita no AYD-004.
- **Q3 — R1:** proposta de posicionar a parcela N na **fatura alvo** do import (a moda dos
  períodos, já calculada para a competência futura) quando a data do item for anterior ao
  período dela, e as parcelas seguintes nas faturas subsequentes. Decisão para o AYD-004.
- **Q4 — R4:** item com data ilegível deve virar `skipped` + erro por item (coerente com o
  sucesso parcial do `confirm-invoice`) em vez de 400 para a requisição inteira?
- **Q5 — R5:** mudar a normalização do hash quebra o dedup contra movimentos já importados com
  o hash antigo. Normalizar só para comparação, com janela de compatibilidade, ou aceitar a
  limitação?
- **Q6 — R6:** inferir `installment_number`/`total_installments` da descrição quando a IA não
  os devolve?
- **R7 não é questão:** o AYD-004 já decide que raiz igual significa confiança alta. É bug, e
  entra como `@known-bug`.
