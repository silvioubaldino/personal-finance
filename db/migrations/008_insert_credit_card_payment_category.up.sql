-- Insert default category for credit card payment
INSERT INTO categories (id, description, is_income, user_id)
VALUES ('d47cc960-f08d-480e-bf01-f4ec5ddfcb8b', 'Pagamento de fatura', false, 'default_category_id')
ON CONFLICT (id) DO NOTHING;
