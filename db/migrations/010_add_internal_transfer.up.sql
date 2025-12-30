-- Add pair_id column to movements for linking transfer pairs
ALTER TABLE movements ADD COLUMN pair_id UUID;

-- Create index for pair_id to improve lookup performance
CREATE INDEX idx_movements_pair_id ON movements(pair_id);

-- Insert default categories for internal transfer
INSERT INTO categories (id, description, is_income, user_id)
VALUES 
    ('c1a2b3c4-d5e6-f7a8-b9c0-d1e2f3a4b5c6', 'Transferência interna - saída', false, 'default_category_id'),
    ('c2b3c4d5-e6f7-a8b9-c0d1-e2f3a4b5c6d7', 'Transferência interna - entrada', true, 'default_category_id')
ON CONFLICT (id) DO NOTHING;

