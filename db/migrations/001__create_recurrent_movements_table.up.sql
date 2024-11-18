create table if not exists recurrent_movements
(
    id              uuid             not null
        constraint recurrent_movements_pk
            primary key,
    description     varchar,
    amount          double precision not null,
    initial_date    date             not null,
    end_date        date,
    category_id     uuid             not null
        constraint recurrent_movements_categories_id_fk
            references categories,
    sub_category_id uuid             not null
        constraint recurrent_movements_sub_categories_id_fk
            references sub_categories,
    wallet_id       uuid             not null
        constraint recurrent_movements_wallets_id_fk
            references wallets,
    type_payment_id integer
        constraint recurrent_movements_type_payments_id_fk
            references type_payments,
    user_id         varchar          not null
);