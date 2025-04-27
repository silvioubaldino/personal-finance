create table if not exists wallets
(
    id              uuid                                                                          not null
        primary key,
    description     varchar(255)                                                                  not null,
    balance         double precision,
    user_id         varchar                                                                       not null,
    initial_balance double precision         default 0                                            not null,
    initial_date    timestamp with time zone default (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) not null,
    date_create     timestamp with time zone default (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) not null,
    date_update     timestamp with time zone default (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) not null
);

alter table if exists wallets
    owner to silvioubaldino;

create table if not exists categories
(
    id          uuid                                                                          not null
        primary key,
    description varchar(255)                                                                  not null,
    is_income   boolean                  default false                                        not null,
    user_id     varchar                                                                       not null,
    date_create timestamp with time zone default (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) not null,
    date_update timestamp with time zone default (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) not null
);

alter table if exists categories
    owner to silvioubaldino;

create table if not exists type_payments
(
    id          integer generated always as identity
        primary key,
    description varchar(255)                                                                  not null,
    user_id     varchar                                                                       not null,
    date_create timestamp with time zone default (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) not null,
    date_update timestamp with time zone default (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) not null
);

alter table if exists type_payments
    owner to silvioubaldino;

create table if not exists sub_categories
(
    id          uuid                                                                          not null
        primary key,
    description varchar(255)                                                                  not null,
    category_id uuid                                                                          not null
        constraint sub_categories_fk_categories
            references categories,
    user_id     varchar                                                                       not null,
    date_create timestamp with time zone default (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) not null,
    date_update timestamp with time zone default (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) not null
);

alter table if exists sub_categories
    owner to silvioubaldino;

create table if not exists estimate_categories
(
    id          uuid             not null
        primary key,
    category_id uuid             not null,
    month       integer          not null,
    year        integer          not null,
    amount      double precision not null,
    user_id     varchar          not null
);

alter table if exists estimate_categories
    owner to silvioubaldino;

create table if not exists estimate_sub_categories
(
    id                   uuid             not null
        primary key,
    sub_category_id      uuid             not null,
    month                integer          not null,
    year                 integer          not null,
    amount               double precision not null,
    user_id              varchar          not null,
    estimate_category_id uuid             not null
        constraint estimate_sub_categories_fk_estimate_categories
            references estimate_categories
);

alter table if exists estimate_sub_categories
    owner to silvioubaldino;

create table if not exists schema_migrations
(
    version bigint  not null
        primary key,
    dirty   boolean not null
);

alter table if exists schema_migrations
    owner to silvioubaldino;

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

alter table if exists recurrent_movements
    owner to silvioubaldino;

create table if not exists movements
(
    id              uuid                                                                          not null
        primary key,
    description     varchar(255)                                                                  not null,
    amount          double precision                                                              not null,
    date            date                                                                          not null,
    user_id         varchar                                                                       not null,
    is_paid         boolean,
    category_id     uuid                                                                          not null
        constraint transaction_fk_categories
            references categories,
    sub_category_id uuid
        constraint transaction_fk_sub_category_id_fkey
            references sub_categories,
    wallet_id       uuid                                                                          not null
        constraint trasaction_fk_wallet
            references wallets,
    type_payment_id integer                                                                       not null
        constraint transaction_fk_type_payment
            references type_payments,
    date_create     timestamp with time zone default (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) not null,
    date_update     timestamp with time zone default (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) not null,
    recurrent_id    uuid
        constraint movements_recurrent_id_fk
            references recurrent_movements
);

alter table if exists movements
    owner to silvioubaldino; 