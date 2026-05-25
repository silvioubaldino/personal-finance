CREATE TABLE IF NOT EXISTS subscriptions
(
    id                  UUID                                                                          NOT NULL
        PRIMARY KEY,
    user_id             VARCHAR                                                                       NOT NULL
        REFERENCES users (id) ON DELETE CASCADE,
    source              VARCHAR                                                                       NOT NULL,
    external_id         VARCHAR                                                                       NOT NULL,
    external_product_id VARCHAR,
    plan_id             VARCHAR
        REFERENCES subscription_plans (id),
    status              VARCHAR                                                                       NOT NULL,
    current_price       DECIMAL(10, 2),
    currency            VARCHAR(3),
    started_at          TIMESTAMP WITH TIME ZONE                                                      NOT NULL,
    current_period_end  TIMESTAMP WITH TIME ZONE,
    cancelled_at        TIMESTAMP WITH TIME ZONE,
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) NOT NULL,
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) NOT NULL,
    CONSTRAINT subscriptions_source_external_id_key UNIQUE (source, external_id)
);

ALTER TABLE IF EXISTS subscriptions
    OWNER TO silvioubaldino;

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_active
    ON subscriptions (user_id)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_subscriptions_period_end
    ON subscriptions (status, current_period_end);
