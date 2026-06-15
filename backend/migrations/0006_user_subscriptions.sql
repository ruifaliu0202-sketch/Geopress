BEGIN;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_platform_admin BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS subscription_tier TEXT NOT NULL DEFAULT 'free',
    ADD COLUMN IF NOT EXISTS subscription_status TEXT NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS subscription_expires_at TIMESTAMPTZ;

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_subscription_tier_check,
    ADD CONSTRAINT users_subscription_tier_check CHECK (subscription_tier IN ('free', 'vip'));

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_subscription_status_check,
    ADD CONSTRAINT users_subscription_status_check CHECK (subscription_status IN ('active', 'inactive', 'expired', 'canceled'));

COMMIT;
