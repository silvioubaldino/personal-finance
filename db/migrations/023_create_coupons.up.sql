CREATE TABLE IF NOT EXISTS coupons
(
    id                  VARCHAR                                                                       NOT NULL
        PRIMARY KEY,
    code                VARCHAR                                                                       NOT NULL,
    description         TEXT,
    discount_type       VARCHAR                                                                       NOT NULL,
    discount_value      DECIMAL(10, 2)                                                                NOT NULL,
    valid_from          TIMESTAMP WITH TIME ZONE                                                      NOT NULL,
    valid_until         TIMESTAMP WITH TIME ZONE                                                      NOT NULL,
    max_redemptions     INTEGER,
    redemption_count    INTEGER                                                                       NOT NULL DEFAULT 0,
    applicable_plan_ids TEXT,
    is_active           BOOLEAN                                                                       NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) NOT NULL,
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) NOT NULL,
    CONSTRAINT coupons_code_key UNIQUE (code)
);

ALTER TABLE IF EXISTS coupons
    OWNER TO silvioubaldino;

CREATE INDEX IF NOT EXISTS idx_coupons_code_active
    ON coupons (code)
    WHERE is_active = TRUE;

CREATE TABLE IF NOT EXISTS coupon_redemptions
(
    id              UUID                                                                          NOT NULL
        PRIMARY KEY,
    user_id         VARCHAR                                                                       NOT NULL
        REFERENCES users (id) ON DELETE CASCADE,
    coupon_id       VARCHAR                                                                       NOT NULL
        REFERENCES coupons (id),
    plan_id         VARCHAR                                                                       NOT NULL
        REFERENCES subscription_plans (id),
    subscription_id UUID
        REFERENCES subscriptions (id) ON DELETE SET NULL,
    original_price  DECIMAL(10, 2)                                                                NOT NULL,
    locked_price    DECIMAL(10, 2)                                                                NOT NULL,
    status          VARCHAR                                                                       NOT NULL DEFAULT 'pending',
    redeemed_at     TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'::text) NOT NULL,
    cancelled_at    TIMESTAMP WITH TIME ZONE,
    CONSTRAINT coupon_redemptions_user_coupon_key UNIQUE (user_id, coupon_id)
);

ALTER TABLE IF EXISTS coupon_redemptions
    OWNER TO silvioubaldino;

CREATE INDEX IF NOT EXISTS idx_coupon_redemptions_subscription
    ON coupon_redemptions (subscription_id);
