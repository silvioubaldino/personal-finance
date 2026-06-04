DROP INDEX IF EXISTS idx_subscription_plans_stripe_price_id;

ALTER TABLE subscription_plans
    DROP COLUMN IF EXISTS stripe_price_id;
