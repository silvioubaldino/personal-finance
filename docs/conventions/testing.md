# Padrões de teste (deste repo)

> Padrão de engenharia (vivo). Decisão pontual sobre testes que mude a abordagem → vira TDR.

As regras de teste (estrutura AAA, formato table-driven, convenção de mocks `testify/mock`,
asserções `testify/assert`, template canônico) são definidas e mantidas **só** na skill
`go-unit-tests` (`.claude/skills/go-unit-tests/SKILL.md`) — fonte única da verdade; não
duplicadas aqui para não divergir dela. Ao tocar um teste, siga a skill.

O que a skill não cobre (framework de docs):
- **Cobertura:** todo critério de aceite de uma SPEC tem um teste correspondente.

## Testes de aceite (Gherkin / godog)

Decisão em `TDR-001`; vocabulário, matriz de cobertura e divergências conhecidas em `SPEC-008`.
Cenários em `test/acceptance/features/` (inglês, um arquivo por comportamento).

- **Rodar:** `make test-acceptance` (Docker, ou `ACCEPTANCE_DATABASE_URL`);
  `GODOG_TAGS='@update-all-next && @credit-card'` fatia; `make test-acceptance-wip` roda o que
  está em construção; `make test-acceptance-known-bugs` roda as divergências conhecidas (passou =
  bug corrigido, tirar a tag). `make test` não compila a suíte (build tag `acceptance`).
- **Escrever:** declarativo, termos do glossário, sem HTTP no Gherkin, datas explícitas, um
  comportamento por cenário, cenários independentes — reuso pelos passos de `test/acceptance/steps`
  (frase nova só se nenhuma serve). Séries recorrentes de um cenário têm descrições distintas.
- **Desejado × atual:** o cenário descreve o comportamento desejado; onde a api diverge, leva
  `@known-bug` e entra na SPEC-008. A correção é PR próprio e tira a tag.
- **Resposta não-2xx** precisa ser verificada pelo passo seguinte
  (`the operation is rejected as "..."`), senão o cenário falha.
- **Invariantes** (fatura, limite, saldo, par de transferência) são checadas ao fim de todo cenário.
