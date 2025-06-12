ALTER TABLE IF EXISTS movements
    DROP CONSTRAINT IF EXISTS transaction_fk_transaction_status;

ALTER TABLE IF EXISTS movements
    DROP CONSTRAINT IF EXISTS transaction_fk_type_payment;

ALTER TABLE IF EXISTS recurrent_movements
    DROP CONSTRAINT IF EXISTS recurrent_movements_type_payments_id_fk;

-- Verifica se as colunas existem antes de tentar alterá-las
DO $$
BEGIN
    -- Verifica se a coluna type_payment_id existe na tabela movements
    IF EXISTS (
        SELECT FROM information_schema.columns 
        WHERE table_name = 'movements' AND column_name = 'type_payment_id'
    ) THEN
        ALTER TABLE movements ALTER COLUMN type_payment_id DROP NOT NULL;
    END IF;
    
    -- Verifica se a coluna status_id existe na tabela movements
    IF EXISTS (
        SELECT FROM information_schema.columns 
        WHERE table_name = 'movements' AND column_name = 'status_id'
    ) THEN
        ALTER TABLE movements ALTER COLUMN status_id DROP NOT NULL;
    END IF;
    
    -- Verifica se a coluna type_payment_id existe na tabela recurrent_movements
    IF EXISTS (
        SELECT FROM information_schema.columns 
        WHERE table_name = 'recurrent_movements' AND column_name = 'type_payment_id'
    ) THEN
        ALTER TABLE recurrent_movements ALTER COLUMN type_payment_id DROP NOT NULL;
    END IF;
END
$$;

ALTER TABLE IF EXISTS movements
    ADD COLUMN IF NOT EXISTS type_payment VARCHAR(255);

ALTER TABLE IF EXISTS recurrent_movements
    ADD COLUMN IF NOT EXISTS type_payment VARCHAR(255);