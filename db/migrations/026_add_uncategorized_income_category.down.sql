-- Reverte o backfill pelo mesmo par (categoria, sinal).
UPDATE movements
SET category_id = 'c1a2b3c4-d5e6-4f7a-8b9c-0d1e2f3a4b5c'
WHERE category_id = '3fad33b7-48da-467f-be49-2e50b1226b82';

DELETE FROM categories
WHERE id = '3fad33b7-48da-467f-be49-2e50b1226b82' AND user_id = 'default_category_id';
