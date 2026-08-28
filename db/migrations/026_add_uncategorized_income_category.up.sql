-- Flavor de receita do fallback do import (AYD-006@context).
-- O id NÃO segue o padrão hex de 'Sem categoria' / 'Transferência interna - saída',
-- que diferem entre si por um único nibble.
INSERT INTO categories (id, description, is_income, user_id)
VALUES ('3fad33b7-48da-467f-be49-2e50b1226b82', 'Sem categoria (receita)', true, 'default_category_id')
ON CONFLICT (id) DO NOTHING;

-- Backfill: entradas gravadas no fallback de despesa passam para o de receita.
-- Idempotente e reversível pelo par (categoria, sinal).
UPDATE movements
SET category_id = '3fad33b7-48da-467f-be49-2e50b1226b82'
WHERE category_id = 'c1a2b3c4-d5e6-4f7a-8b9c-0d1e2f3a4b5c'
  AND amount > 0;
