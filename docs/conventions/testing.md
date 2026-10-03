# Padrões de teste (deste repo)

> Padrão de engenharia (vivo). Decisão pontual sobre testes que mude a abordagem → vira TDR.

As regras de teste (estrutura AAA, formato table-driven, convenção de mocks `testify/mock`,
asserções `testify/assert`, template canônico) são definidas e mantidas **só** na skill
`go-unit-tests` (`.claude/skills/go-unit-tests/SKILL.md`) — fonte única da verdade; não
duplicadas aqui para não divergir dela. Ao tocar um teste, siga a skill.

O que a skill não cobre (framework de docs):
- **Cobertura:** todo critério de aceite de uma SPEC tem um teste correspondente.

## Testes de aceite (Gherkin / godog)

Decisão em `TDR-001`; especificação, vocabulário de passos e matriz de cobertura em
`SPEC-008`. Os cenários ficam em `test/acceptance/features/` (inglês, um arquivo por
comportamento, agrupados por capacidade: `movement/`, `recurrence/`, `credit_card/`,
`transfer/`, `journeys/`).

- **Como rodar:** `make test-acceptance` (precisa de Docker — sobe um Postgres efêmero com as
  migrations reais — ou de `ACCEPTANCE_DATABASE_URL` apontando para um Postgres existente).
  `GODOG_TAGS='@update-all-next && @credit-card'` fatia a execução; `make
  test-acceptance-wip` roda só o que está em construção; `make
  test-acceptance-known-bugs` roda as divergências conhecidas (um cenário que passar ali
  indica bug corrigido: tirar a tag). `make test` (unit) não compila nem executa a suíte
  (build tag `acceptance`).
- **O que a suíte é:** a api sobe in-process pelo mesmo `internal/app.New` de produção,
  autenticação fake no harness e um relógio por cenário (`X-Test-Now`, só no harness). Os
  passos falam HTTP nas rotas `/v2` e nunca importam as structs de saída da api — o contrato
  JSON é o que está sob teste.
- **Como escrever:** declarativo, termos do glossário, sem HTTP no Gherkin; datas sempre
  explícitas; um comportamento por cenário; cenário nunca depende de outro — o reuso é pelo
  vocabulário de passos (`test/acceptance/steps`). Frase nova só quando nenhuma existente
  expressa o comportamento. Séries recorrentes de um mesmo cenário têm descrições distintas.
- **Invariantes:** toda execução de cenário termina checando fatura = soma dos itens, limite
  do cartão, saldo de carteira e par de transferência (`steps/invariants.go`).
- **Comportamento desejado × atual:** o cenário descreve o comportamento desejado. Onde a api
  diverge, o cenário leva `@known-bug` (sai do gate de CI, roda no job informativo) e a
  divergência é listada na SPEC-008; a correção é PR próprio, que tira a tag.
- **Resposta não asserida:** uma resposta não-2xx precisa ser verificada pelo passo seguinte
  (`the operation is rejected as "..."`); senão o cenário falha mostrando status e corpo.
