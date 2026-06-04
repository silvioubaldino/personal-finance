# Spec: Importação de extratos em PDF com senha

> Status: implementado no backend · Última atualização: 2026-06-04

## Contexto

A importação de extratos (`POST /v2/statements/extract`) envia o arquivo para um
modelo de visão (Gemini), que faz o parsing do PDF. PDFs com **senha de abertura**
têm o conteúdo criptografado e o modelo não consegue lê-los — a extração falha.

Um LLM **não descriptografa** PDF. Portanto a descriptografia acontece no
**backend**, em memória, antes dos bytes chegarem ao modelo.

## Decisão

- **Backend descriptografa** (biblioteca `pdfcpu`), nunca a IA.
- Dois tipos de proteção de PDF, tratados de forma diferente:

  | Tipo | Comportamento |
  |---|---|
  | **Owner password** (só permissões; abre com senha vazia) | Descriptografado de forma **transparente**. O usuário não percebe. |
  | **User password** (senha de abertura) | **Exige a senha do usuário.** Sem ela é impossível ler. |

- A senha trafega uma única vez, é usada em memória e descartada. **Nunca é
  persistida nem logada.**
- O backend é a **fonte da verdade**: valida e descriptografa de qualquer forma,
  mesmo que o cliente não tenha detectado a senha.

## Contrato da API

`POST /v2/statements/extract` — `multipart/form-data`

| Campo | Obrigatório | Descrição |
|---|---|---|
| `file` | sim | PDF, JPEG ou PNG (máx. 10MB) |
| `password` | não | Senha de abertura do PDF, quando houver |

### Respostas relevantes

- **200** — extração concluída (PDF sem senha, owner-only, ou senha correta).
- **422** — PDF protegido. O corpo traz um `type` legível por máquina para o
  cliente decidir o que fazer:

```jsonc
// PDF exige senha e nenhuma foi enviada
{ "error": { "code": 422, "type": "statement_password_required",
             "message": "This PDF is password protected. Please provide the password." } }

// senha enviada está incorreta
{ "error": { "code": 422, "type": "statement_wrong_password",
             "message": "The password provided is incorrect." } }
```

> O campo `type` só aparece nesses casos (`omitempty`); as demais respostas de
> erro permanecem inalteradas. **Os clientes devem ramificar pelo `type`, nunca
> pela `message`** (que é texto livre / localizável).

## Comportamento esperado no Front / Mobile

O objetivo de UX é **evitar idas e vindas**: detectar a senha localmente e enviá-la
junto com o arquivo numa única requisição.

### Fluxo recomendado

1. Usuário seleciona o arquivo.
2. Se for PDF, o cliente verifica **localmente** se há senha de abertura
   (sem upload):
   - **Web:** `pdf.js` (`getDocument`) dispara o callback `onPassword` com
     `PasswordResponses.NEED_PASSWORD` quando há user password; abre normalmente
     (sem callback) para PDFs sem senha ou owner-only.
   - **Mobile:** PDFKit (iOS) — `CGPDFDocument.isEncrypted` /
     `unlockWithPassword`; Android — PdfRenderer falha ao abrir, ou usar uma lib
     equivalente para detectar/validar.
3. Se exigir senha, mostrar um campo e, idealmente, **validar a senha localmente**
   (tentar abrir com a senha digitada) antes de enviar — feedback imediato e
   sem upload desperdiçado.
4. Enviar `file` **+** `password` na mesma requisição `extract`.
5. Se não for possível detectar localmente, basta enviar só o `file`: o backend
   responde `422 statement_password_required` e o cliente então pede a senha e
   reenvia.

### Tratamento das respostas

| Resposta | Ação no cliente |
|---|---|
| `200` | Segue o fluxo normal (revisão das movimentações). |
| `422 statement_password_required` | Solicitar a senha ao usuário e reenviar `file` + `password`. |
| `422 statement_wrong_password` | Informar "senha incorreta" e permitir nova tentativa. |
| `413` / arquivo > 10MB | Avisar sobre o limite de tamanho. |

### Pontos importantes

- O cliente **detecta e coleta** a senha; **não descriptografa** o arquivo. O PDF
  enviado continua sendo o original — a descriptografia é do backend.
- PDFs **owner-only** abrem sem senha; o cliente **não deve** pedir senha nesses
  casos (o backend resolve sozinho).
- A senha digitada **não deve ser armazenada** (sem cache, sem persistência);
  usar só para validação local e envio imediato.

## Fora de escopo

Detecção/validação client-side é responsabilidade do front/mobile. O backend não
depende dela — apenas a aproveita como otimização de UX.
