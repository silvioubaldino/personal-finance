-- Revert invoice remainder movements to previous category and no subcategory
UPDATE movements
SET
    category_id = 'd47cc960-f08d-480e-bf01-f4ec5ddfcb8b',
    sub_category_id = NULL
WHERE type_payment = 'invoice_remainder';
