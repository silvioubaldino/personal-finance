create table if not exists invoices
(
    id            uuid                                                                          not null
        primary key,
    credit_card_id uuid                                                                         not null
        constraint invoices_fk_credit_cards references credit_cards,
    period_start  date                                                                          not null,
    period_end    date                                                                          not null,
    due_day       date                                                                          not null,
    payment_date  date,
    amount        double precision           default 0                                          not null,
    is_paid       boolean                    default false                                       not null,
    wallet_id     uuid
        constraint invoices_fk_wallets references wallets,
    user_id       varchar                                                                       not null,
    date_create   timestamp with time zone default (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) not null,
    date_update   timestamp with time zone default (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) not null
);

create index if not exists idx_invoices_card_period on invoices (credit_card_id, period_start, period_end);

alter table if exists invoices
    owner to silvioubaldino;
