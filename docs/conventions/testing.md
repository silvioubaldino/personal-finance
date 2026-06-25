# Padrões de teste (deste repo)

> Padrão de engenharia (vivo). Decisão pontual sobre testes que mude a abordagem → vira TDR.
> Regras completas e template canônico: skill `go-unit-tests`
> (`.claude/skills/go-unit-tests/SKILL.md`).

- **Estrutura:** AAA (Arrange, Act, Assert), com comentário marcando cada seção.
- **Localização:** `*_test.go` no mesmo diretório do código testado, `package <feature>_test`
  (external test package).
- **Formato:** um teste table-driven por função, tabela `map[string]struct{...}` (chave = nome
  do caso, ex.: `"should return ErrNotFound when movement does not exist"`). Sem `if`/`switch`
  no corpo do teste — toda variação entra pela tabela, não por controle de fluxo.
- **Cobertura:** todo critério de aceite de uma SPEC tem um teste correspondente.
- **Mocks:** `testify/mock`, só na fronteira (repository, gateway), nunca a unidade testada.
  Reusar mock existente (`mock_test.go`) antes de criar um novo; nunca `mock.Anything`.
- **Asserções:** `testify/assert`; comparar erros sempre com `assert.ErrorIs`, inclusive no
  caminho de sucesso (nunca `assert.NoError`/`err == nil`).
- **Comando:** `go test ./internal/path/to/package/...` (pacote) · `make test` (suite
  completa, com `-race` e cobertura).
