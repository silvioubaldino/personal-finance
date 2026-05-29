ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS mp_preapproval_plan_id VARCHAR,
    ADD COLUMN IF NOT EXISTS is_public BOOLEAN NOT NULL DEFAULT true;

CREATE UNIQUE INDEX IF NOT EXISTS idx_subscription_plans_mp_preapproval_plan_id
    ON subscription_plans (mp_preapproval_plan_id)
    WHERE mp_preapproval_plan_id IS NOT NULL;

ALTER TABLE coupons
    ADD COLUMN IF NOT EXISTS target_plan_id VARCHAR;
