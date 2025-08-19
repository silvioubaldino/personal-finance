create table if not exists credit_cards
(
    id               uuid                                                                          not null
        primary key,
    name             varchar(255)                                                                  not null,
    credit_limit     double precision,
    closing_day      integer                                                                       not null,
    due_day          integer                                                                       not null,
    default_wallet_id uuid,
    user_id          varchar                                                                       not null,
    date_create      timestamp with time zone default (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) not null,
    date_update      timestamp with time zone default (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) not null,
    constraint credit_cards_fk_wallets
        foreign key (default_wallet_id) references wallets
);

alter table if exists credit_cards
    owner to silvioubaldino;
