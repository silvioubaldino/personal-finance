ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS stripe_price_id VARCHAR;

CREATE UNIQUE INDEX IF NOT EXISTS idx_subscription_plans_stripe_price_id
    ON subscription_plans (stripe_price_id)
    WHERE stripe_price_id IS NOT NULL;
