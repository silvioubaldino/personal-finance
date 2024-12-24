ALTER TABLE movements
DROP CONSTRAINT transaction_fk_transaction_status;
ALTER TABLE movements
DROP COLUMN status_id;
DROP TABLE movement_status;

ALTER TABLE movements
DROP CONSTRAINT IF EXISTS transaction_fk_transaction_id;
ALTER TABLE movements
DROP COLUMN IF EXISTS transaction_id;