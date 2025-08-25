alter table if exists movements
    add column if not exists installment_group_id uuid;

alter table if exists movements
    add column if not exists installment_number integer;

alter table if exists movements
    add column if not exists total_installments integer;
