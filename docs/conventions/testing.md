# Padrões de teste (deste repo)

> Padrão de engenharia (vivo). Decisão pontual sobre testes que mude a abordagem → vira TDR.

As regras de teste (estrutura AAA, formato table-driven, convenção de mocks `testify/mock`,
asserções `testify/assert`, template canônico) são definidas e mantidas **só** na skill
`go-unit-tests` (`.claude/skills/go-unit-tests/SKILL.md`) — fonte única da verdade; não
duplicadas aqui para não divergir dela. Ao tocar um teste, siga a skill.

O que a skill não cobre (framework de docs):
- **Cobertura:** todo critério de aceite de uma SPEC tem um teste correspondente.
