DROP INDEX IF EXISTS idx_subscription_plans_apple_product_id;
DROP INDEX IF EXISTS idx_subscription_plans_google_product_id;

ALTER TABLE subscription_plans
    DROP COLUMN IF EXISTS apple_product_id,
    DROP COLUMN IF EXISTS google_product_id;
