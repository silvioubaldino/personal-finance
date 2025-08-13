drop index if exists idx_movements_invoice_type;
alter table if exists movements drop constraint if exists movements_invoice_id_fk;
alter table if exists movements drop column if exists invoice_id;
