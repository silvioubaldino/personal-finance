-- Remove default categories for internal transfer
DELETE FROM categories WHERE id IN (
    'c1a2b3c4-d5e6-f7a8-b9c0-d1e2f3a4b5c6',
    'c2b3c4d5-e6f7-a8b9-c0d1-e2f3a4b5c6d7'
);

-- Drop index for pair_id
DROP INDEX IF EXISTS idx_movements_pair_id;

-- Remove pair_id column from movements
ALTER TABLE movements DROP COLUMN IF EXISTS pair_id;

