BEGIN;

CREATE TABLE IF NOT EXISTS system_settings (
    key TEXT PRIMARY KEY,
    value JSONB NOT NULL DEFAULT '{}'::jsonb,
    value_type TEXT NOT NULL DEFAULT 'json',
    is_secret BOOLEAN NOT NULL DEFAULT FALSE,
    description TEXT NOT NULL DEFAULT '',
    updated_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS system_secrets (
    key TEXT PRIMARY KEY,
    encrypted_value TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT 'db',
    description TEXT NOT NULL DEFAULT '',
    updated_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_system_settings_updated_at ON system_settings(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_system_secrets_updated_at ON system_secrets(updated_at DESC);

COMMIT;
