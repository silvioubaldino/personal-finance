DROP INDEX IF EXISTS idx_movements_idempotency_hash;
ALTER TABLE movements DROP COLUMN IF EXISTS idempotency_hash;
