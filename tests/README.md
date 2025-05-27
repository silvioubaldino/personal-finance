# Testes Integrados

Este diretório contém os testes integrados do sistema de finanças pessoais, implementados usando BDD (Behavior Driven Development) com Godog.

## Estrutura

```
tests/
├── features/           # Arquivos .feature com cenários em Gherkin
├── steps/             # Implementação dos steps em Go
├── helpers/           # Utilitários para testes (ex: DateHelper)
├── suite/             # Configuração da suite de testes
├── suite_test.go      # Arquivo principal de execução
└── README.md          # Este arquivo
```

## Como executar

### Executar apenas testes integrados
```bash
make test-integration
```

### Executar apenas testes unitários
```bash
make test-unit
```

### Executar todos os testes
```bash
make test-all
```

### Limpar cache de testes
```bash
make clean-test
```

## Funcionalidades testadas

### Movimentações Recorrentes
- **Criação**: Validação de criação de movimentações recorrentes
- **Update One**: Atualização de uma ocorrência específica
- **Update All Next**: Atualização de todas as ocorrências futuras
- **Delete One**: Deleção de uma ocorrência específica
- **Delete All Next**: Deleção de todas as ocorrências futuras

## Infraestrutura

- **Banco**: SQLite em memória para isolamento
- **Servidor**: httptest.Server para simular API
- **Limpeza**: Banco é limpo entre cenários
- **Migrations**: Executadas automaticamente

## Padrões

- Steps genéricos e reutilizáveis
- Uso de offsets de mês (0 = atual, 2 = dois meses à frente)
- Validações específicas por cenário
- Isolamento completo entre testes 