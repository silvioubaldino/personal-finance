ALTER TABLE movements ADD COLUMN idempotency_hash VARCHAR(64);

INSERT INTO categories (id, description, is_income, user_id)
VALUES ('c1a2b3c4-d5e6-4f7a-8b9c-0d1e2f3a4b5c', 'Sem categoria', false, 'default_category_id')
ON CONFLICT (id) DO NOTHING;