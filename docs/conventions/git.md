# Convenções de Git (deste repo)

- **Branches:** `feature/<slug>` (ex.: `feature/api-create-authorization`) ou
  `bugfix/<slug>` (ex.: `bugfix/kubernetes`), a partir de `develop`. Ao trabalhar a partir de
  uma SPEC, inclua o ID no slug (ex.: `feature/SPEC-012-credit-card-limit`).
- **Commits:** mensagem curta no imperativo (ex.: `Fix 500 on credit card movement when
  default wallet is nil`); `feat(scope): ...` / `fix(scope): ...` (Conventional Commits) em
  mudanças maiores. Referencie o ID da SPEC quando aplicável.
- **PRs:** merge via squash contra `develop` (GitHub anexa `(#NNN)` à mensagem
  automaticamente); cada PR soma uma linha ao `CHANGELOG.md` (raiz do repo, formato Keep a
  Changelog) — regras abaixo. PR não altera contrato — isso é PR no repo de contexto
  (`personal-finance-context`).
- **Antes do PR:** `make all` (format + lint + test); se a SPEC depender do contexto
  compartilhado, rode `docs/scripts/sync-context.sh` para validar contra o contexto atual.

## CHANGELOG (`CHANGELOG.md` at the repo root)

These rules are the source of truth — `CHANGELOG.md` only points here. When adding a line,
follow them without needing to open the top of that file:

- **Order:** most recent on top; new entries go **above** the previous ones.
- **Unreleased:** unreleased work accrues under `## Unreleased` (always the top block), with
  no date/version. On release, `## Unreleased` becomes `## Release - vX.Y.Z - dd/MM/yyyy` and
  a new empty `## Unreleased` is opened above it.
- **One line per PR:** each PR adds a single line stating **what** was delivered — not how it
  was built, not why, and no docs-framework housekeeping (SPEC/PLAN/AYD). Reference the PR
  (e.g. `[PR#227](url)`). One PR = one line: if the PR already has a line, edit it instead of
  adding another.
- **Limit:** **350 characters per line, URL included.** Doesn't fit? The line is describing
  the how or the why — cut those, not the fact. Detail lives in the SPEC/AYD and in git.
- **SPEC-only PRs:** if a PR only adds a SPEC, summarize the feature it opens (the SPEC itself
  is tracked by its own file and by git).

The most common mistake is describing the **how**. Compare:

- BAD: `Fixed the analytics summary counting the credit card twice: money aggregates now use
  the canonical realized cut, itemizing card spend into its real categories and classifying
  by the category's is_income flag instead of amount sign [PR#227](url)`
- GOOD: `Fixed the analytics summary counting credit card spend twice [PR#227](url)`

- BAD: `Merged the docs framework's SPEC and PLAN into a single per-repo doc` (docs
  housekeeping, and no PR — this does not belong in the changelog)
- GOOD: nothing — that change does not earn a line.
