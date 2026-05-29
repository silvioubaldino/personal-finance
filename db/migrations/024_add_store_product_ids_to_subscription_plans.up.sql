ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS apple_product_id  VARCHAR,
    ADD COLUMN IF NOT EXISTS google_product_id VARCHAR;

CREATE UNIQUE INDEX IF NOT EXISTS idx_subscription_plans_apple_product_id
    ON subscription_plans (apple_product_id)
    WHERE apple_product_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_subscription_plans_google_product_id
    ON subscription_plans (google_product_id)
    WHERE google_product_id IS NOT NULL;
