# Pacote de Logs com Zap

Estrutura de logging estruturado baseada em Zap para a aplicação.

## Principais Funcionalidades

- Logger estruturado de alta performance
- Middleware para injeção automática do logger no contexto de requisições
- Suporte para diferentes níveis de log e formatos (JSON, texto)
- Extração automática do logger do contexto

## Como Usar

### Inicialização Básica

```go
// Configurar o logger global
log.Initialize(
    log.WithLevel("info"),
    log.WithFormat("json"),
)
```

### Configuração do Middleware

```go
// No setup do Gin (já implementado em main.go)
r.Use(log.GinLoggerMiddleware(log.Global))
```

### Uso em Handlers

```go
func Handler(c *gin.Context) {
    // O logger já está disponível no contexto da requisição
    ctx := c.Request.Context()
    
    // Usar o logger do contexto
    log.InfoContext(ctx, "Processando requisição")
    
    // Log com campos estruturados
    log.InfoContext(ctx, "Operação concluída", 
        log.String("user_id", "123"),
        log.Int("duration_ms", 45)
    )
    
    // Log de erro
    if err := processarAlgo(); err != nil {
        log.ErrorContext(ctx, "Falha no processamento", log.Err(err))
    }
}
```

### Uso Direto do Logger Global

```go
// Em locais sem acesso ao contexto
log.Info("Aplicação inicializada")
log.Error("Erro crítico", log.Err(err))
```

## Tipos de Campos Disponíveis

- `log.String(key, value)`: Campo de texto
- `log.Int(key, value)`: Campo de inteiro
- `log.Bool(key, value)`: Campo booleano
- `log.Err(err)`: Campo de erro
- `log.Time(key, value)`: Campo de data/hora
- `log.Duration(key, value)`: Campo de duração
- `log.Any(key, value)`: Campo com qualquer valor

## Níveis de Log

- `Debug`: Detalhes para desenvolvimento
- `Info`: Informações operacionais normais
- `Warn`: Situações inesperadas mas não críticas
- `Error`: Erros que afetam operações específicas
- `Fatal`: Erros críticos (também chama os.Exit(1))

## Middleware

O middleware implementado usa o padrão de closure para inicializar o logger apenas uma vez e injetá-lo no contexto de cada requisição. Informações como request_id, método, path e outras são automaticamente incluídas nos logs. 