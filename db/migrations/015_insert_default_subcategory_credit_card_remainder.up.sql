-- Insert default subcategory for credit card remainder
INSERT INTO sub_categories (id, description, category_id, user_id)
VALUES ('3ef4b1a5-6e5d-4f4d-9f0b-2f7a941c4f62', 'Remanescente de cartão de crédito', '1d50405e-c42a-4991-b480-bc628d9b8713', 'default_category_id')
ON CONFLICT (id) DO NOTHING;
