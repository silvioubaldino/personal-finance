---
id: SPEC-NNN
type: spec
title: 
status: draft
created: 2025-01-01
updated: 2025-01-01
owner: <nome>
parents: [AYD-NNN@context]   # obrigatório: o AYD que originou esta spec
children: []                 # normalmente vazio — o plano vive aqui dentro
related: [GLO]               # ADR/TDR relevantes
tags: []
superseded_by: null
---

# Spec: <feature> (parte deste repo)

> **Doc único deste repo para a feature:** o *o quê* (objetivo, critérios, contratos) e o
> *como* (plano de implementação). A parte "o quê" **congela ao virar `approved`**; a
> checklist do plano segue sendo marcada durante a execução, sem mudar o `status`.

## Objetivo
_O papel deste repo nesta feature (conforme o AYD)._

## Critérios de aceite
```gherkin
Cenário: <nome>
  Dado <contexto>
  Quando <ação>
  Então <resultado observável>
```

## Contratos consumidos/expostos
_Referencie os contratos do AYD. Este repo NÃO os redefine._

## Modelo de dados / componentes afetados
- 

## Casos de borda & fora de escopo
- Borda:
- Fora:

---

## Plano de implementação

> Parte de trabalho deste doc. Decisão técnica não trivial vira TDR, não fica só aqui.

### Abordagem técnica
_Resumo da estratégia._

### Passos
1. 
2. 

### Arquivos / módulos afetados
- 

### Testes (ver `docs/conventions/testing.md`)
- **Aceite (mapeia os critérios acima):**
- **Unit/integração:**

### Checklist
- [ ] 
