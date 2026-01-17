-- Update existing invoice remainder movements to use Taxas > Remanescente de cartão de crédito
UPDATE movements
SET
    category_id = '1d50405e-c42a-4991-b480-bc628d9b8713',
    sub_category_id = '3ef4b1a5-6e5d-4f4d-9f0b-2f7a941c4f62'
WHERE type_payment = 'invoice_remainder';
