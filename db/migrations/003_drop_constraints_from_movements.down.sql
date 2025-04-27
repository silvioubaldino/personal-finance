ALTER TABLE IF EXISTS movements
    DROP COLUMN IF EXISTS type_payment;

ALTER TABLE IF EXISTS recurrent_movements
    DROP COLUMN IF EXISTS type_payment;

DO $$
BEGIN
    IF EXISTS (
        SELECT FROM information_schema.columns 
        WHERE table_name = 'movements' AND column_name = 'status_id'
    ) AND EXISTS (
        SELECT FROM information_schema.tables 
        WHERE table_name = 'movement_status'
    ) THEN
        ALTER TABLE movements 
        ADD CONSTRAINT transaction_fk_transaction_status
        FOREIGN KEY (status_id) REFERENCES movement_status(id);
    END IF;
    
    IF EXISTS (
        SELECT FROM information_schema.columns 
        WHERE table_name = 'movements' AND column_name = 'type_payment_id'
    ) AND EXISTS (
        SELECT FROM information_schema.tables 
        WHERE table_name = 'type_payments'
    ) THEN
        ALTER TABLE movements
        ADD CONSTRAINT transaction_fk_type_payment
        FOREIGN KEY (type_payment_id) REFERENCES type_payments(id);
    END IF;
    
    IF EXISTS (
        SELECT FROM information_schema.columns 
        WHERE table_name = 'recurrent_movements' AND column_name = 'type_payment_id'
    ) AND EXISTS (
        SELECT FROM information_schema.tables 
        WHERE table_name = 'type_payments'
    ) THEN
        ALTER TABLE recurrent_movements
        ADD CONSTRAINT recurrent_movements_type_payments_id_fk
        FOREIGN KEY (type_payment_id) REFERENCES type_payments(id);
    END IF;
END
$$;