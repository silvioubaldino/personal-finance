#!/usr/bin/env sh
# Sincroniza a camada compartilhada (repo de contexto) para docs/shared/.
# Espelho read-only e gitignored. Rode ANTES de abrir o Claude Code
# (os @imports do CLAUDE.md carregam no início da sessão).
set -e
cd "$(dirname "$0")/../.."

CONTEXT_REPO="git@github.com:silvioubaldino/personal-finance-context.git"
CONTEXT_REF="main"   # troque por uma tag/commit para fixar o contexto

rm -rf docs/shared
git clone --depth 1 --branch "$CONTEXT_REF" "$CONTEXT_REPO" docs/shared
rm -rf docs/shared/.git   # vira cópia simples (não um repo aninhado)
echo "✓ contexto sincronizado em docs/shared/ ($CONTEXT_REF)"
