ALTER TABLE coupons DROP COLUMN IF EXISTS target_plan_id;

DROP INDEX IF EXISTS idx_subscription_plans_mp_preapproval_plan_id;

ALTER TABLE subscription_plans DROP COLUMN IF EXISTS is_public;
ALTER TABLE subscription_plans DROP COLUMN IF EXISTS mp_preapproval_plan_id;
