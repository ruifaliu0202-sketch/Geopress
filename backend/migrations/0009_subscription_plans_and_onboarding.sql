BEGIN;

CREATE TABLE IF NOT EXISTS subscription_plans (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    tier TEXT NOT NULL CHECK (tier IN ('free', 'vip')),
    price_cents INT NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'USD',
    monthly_token_budget_cents INT NOT NULL DEFAULT 0,
    input_token_price_per_1k INT NOT NULL DEFAULT 1,
    output_token_price_per_1k INT NOT NULL DEFAULT 4,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO subscription_plans (
    id, name, tier, price_cents, currency, monthly_token_budget_cents,
    input_token_price_per_1k, output_token_price_per_1k, enabled
)
VALUES
    ('free', 'Free', 'free', 0, 'USD', 0, 1, 4, TRUE),
    ('vip', 'VIP', 'vip', 10000, 'USD', 10000, 1, 4, TRUE)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    tier = EXCLUDED.tier,
    price_cents = EXCLUDED.price_cents,
    currency = EXCLUDED.currency,
    monthly_token_budget_cents = EXCLUDED.monthly_token_budget_cents,
    input_token_price_per_1k = EXCLUDED.input_token_price_per_1k,
    output_token_price_per_1k = EXCLUDED.output_token_price_per_1k,
    enabled = EXCLUDED.enabled,
    updated_at = now();

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS subscription_plan_id TEXT NOT NULL DEFAULT 'free',
    ADD COLUMN IF NOT EXISTS monthly_token_budget_cents INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS monthly_token_used_cents INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS monthly_token_input_used INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS monthly_token_output_used INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS subscription_current_period TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS onboarding_completed BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS onboarding_completed_at TIMESTAMPTZ;

UPDATE users
SET
    subscription_plan_id = CASE WHEN subscription_tier = 'vip' THEN 'vip' ELSE 'free' END,
    monthly_token_budget_cents = CASE WHEN subscription_tier = 'vip' THEN 10000 ELSE 0 END,
    subscription_current_period = CASE
        WHEN subscription_current_period = '' THEN to_char(now(), 'YYYY-MM')
        ELSE subscription_current_period
    END,
    onboarding_completed = CASE
        WHEN id IN ('usr_demo', 'usr_growth') THEN TRUE
        ELSE onboarding_completed
    END,
    onboarding_completed_at = CASE
        WHEN id IN ('usr_demo', 'usr_growth') AND onboarding_completed_at IS NULL THEN now()
        ELSE onboarding_completed_at
    END;

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_subscription_plan_id_fkey,
    ADD CONSTRAINT users_subscription_plan_id_fkey
        FOREIGN KEY (subscription_plan_id) REFERENCES subscription_plans(id);

CREATE TABLE IF NOT EXISTS ai_token_usage_events (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    generation_request_id TEXT NOT NULL DEFAULT '',
    provider TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    subscription_plan_id TEXT NOT NULL REFERENCES subscription_plans(id),
    input_tokens INT NOT NULL DEFAULT 0,
    output_tokens INT NOT NULL DEFAULT 0,
    total_tokens INT NOT NULL DEFAULT 0,
    input_cost_cents INT NOT NULL DEFAULT 0,
    output_cost_cents INT NOT NULL DEFAULT 0,
    total_cost_cents INT NOT NULL DEFAULT 0,
    billing_period TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ai_token_usage_events_user_period ON ai_token_usage_events(user_id, billing_period);
CREATE INDEX IF NOT EXISTS idx_ai_token_usage_events_workspace ON ai_token_usage_events(workspace_id);

COMMIT;
