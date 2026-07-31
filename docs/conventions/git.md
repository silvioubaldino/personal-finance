# Convenções de Git (deste repo)

- **Branches:** `feature/<slug>` (ex.: `feature/api-create-authorization`) ou
  `bugfix/<slug>` (ex.: `bugfix/kubernetes`), a partir de `develop`. Ao trabalhar a partir de
  uma SPEC/PLAN, inclua o ID no slug (ex.: `feature/SPEC-012-credit-card-limit`).
- **Commits:** mensagem curta no imperativo (ex.: `Fix 500 on credit card movement when
  default wallet is nil`); `feat(scope): ...` / `fix(scope): ...` (Conventional Commits) em
  mudanças maiores. Referencie o ID (SPEC/PLAN) quando aplicável.
- **PRs:** merge via squash contra `develop` (GitHub anexa `(#NNN)` à mensagem
  automaticamente); cada PR soma uma linha ao `CHANGELOG.md` (raiz do repo, formato Keep a
  Changelog). PR não altera contrato — isso é PR no repo de contexto
  (`personal-finance-context`).
- **Antes do PR:** `make all` (format + lint + test); se a SPEC depender do contexto
  compartilhado, rode `docs/scripts/sync-context.sh` para validar contra o contexto atual.
