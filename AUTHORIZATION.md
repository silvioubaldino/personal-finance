# Documentação da Feature de Autorização e Limites

Esta documentação descreve o funcionamento da feature de autorização e controle de limites do sistema, fornecendo as informações necessárias para que o Front-end implemente o tratamento adequado.

## 1. Visão Geral

A autorização no sistema é baseada em dois pilares principais:
- **Papéis (Roles):** Controlam o acesso a funcionalidades administrativas (ex: `user`, `admin`).
- **Planos e Limites:** Controlam o uso de recursos do sistema (como carteiras, cartões e movimentações) com base no plano do usuário (`free`, `plus`).

O sistema utiliza o **Firebase Authentication** para autenticação, extraindo o plano e o papel do usuário diretamente dos *custom claims* do token JWT.

---

## 2. Autenticação

Todas as requisições autenticadas devem enviar o token do Firebase no header:
- **Header:** `user_token`
- **Valor:** `<FIREBASE_ID_TOKEN>`

---

## 3. Papéis (Roles)

- **`user`**: Papel padrão para todos os usuários. Permite acesso às funcionalidades básicas de finanças pessoais.
- **`admin`**: Papel com permissões elevadas. Permite consultar e alterar planos e papéis de outros usuários através dos endpoints de `/admin`.

---

## 4. Planos e Limites

Atualmente, o sistema suporta dois planos:
1.  **`free`**: Plano gratuito com limites de uso.
2.  **`plus`**: Plano pago sem limites (ilimitado).

### Limites Atuais (Plano Free)

| Recurso | Limite | Descrição |
| :--- | :--- | :--- |
| **Carteiras** | 2 | Máximo de carteiras ativas que o usuário pode criar. |
| **Cartões de Crédito** | 1 | Máximo de cartões de crédito que o usuário pode possuir. |
| **Movimentações** | 50 | Limite de movimentações por mês. |
| **Recorrências** | 3 | Limite de movimentações recorrentes ativas por mês. |

*Nota: Estes valores podem ser configurados via variáveis de ambiente no servidor.*

---

## 5. Endpoints Disponíveis

### Consultar Limites Atuais
Retorna o plano do usuário, os limites permitidos e o uso atual.
- **Método:** `GET`
- **Path:** `/me/limits`
- **Sucesso (200 OK):**
```json
{
  "plan": "free",
  "limits": {
    "wallets": 2,
    "credit_cards": 1,
    "movements_per_month": 50,
    "recurrences_per_month": 3
  },
  "usage": {
    "wallets": 1,
    "credit_cards": 0,
    "movements_per_month": 10,
    "recurrences_per_month": 1
  },
  "reset_at": "2024-04-01T00:00:00Z"
}
```

### Endpoints com Verificação de Limites
Os seguintes endpoints validarão se o usuário ainda possui cota disponível antes de processar a criação:
- `POST /wallets`: Criação de nova carteira.
- `POST /credit-cards`: Criação de novo cartão de crédito.
- `POST /v2/movements`: Criação de movimentação (comum ou recorrente).

---

## 6. Tratamento de Erros (Limits Ultrapassados)

Quando um limite de plano é atingido, o servidor retornará um erro **403 Forbidden**. O Front-end deve capturar este status e exibir uma mensagem adequada ao usuário (ex: sugerindo upgrade para o plano Plus).

### Formato do Erro
- **Status Code:** `403 Forbidden`
- **Body:**
```json
{
  "error": {
    "code": 403,
    "message": "mensagem de erro específica"
  }
}
```

### Mensagens de Erro Possíveis
Dependendo do recurso que atingiu o limite, a `message` será uma das seguintes:

| Mensagem | Recurso Atingido |
| :--- | :--- |
| `wallet limit reached for your plan` | Limite de Carteiras |
| `credit card limit reached for your plan` | Limite de Cartões de Crédito |
| `movement limit reached for your plan this month` | Limite de Movimentações Mensais |
| `recurrence limit reached for your plan this month` | Limite de Recorrências Mensais |

---

## 7. Endpoints Administrativos (Admin Only)

Estes endpoints exigem o papel de `admin` no token:
- `GET /admin/users/:userID/claims`: Consulta o plano e papel de um usuário.
- `PATCH /admin/users/:userID/plan`: Altera o plano de um usuário (`free`/`plus`).
- `PATCH /admin/users/:userID/role`: Altera o papel de um usuário (`user`/`admin`).

Caso um usuário normal tente acessar estes paths, o servidor retornará `403 Forbidden` com a mensagem `"forbidden"`.
