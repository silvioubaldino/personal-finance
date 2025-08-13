alter table if exists movements
    add column if not exists invoice_id uuid;

alter table if exists movements
    add constraint movements_invoice_id_fk foreign key (invoice_id) references invoices (id);

create index if not exists idx_movements_invoice_type on movements (invoice_id, type_payment);
