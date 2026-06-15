BEGIN;

CREATE TABLE IF NOT EXISTS media_account_login_sessions (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    account_id TEXT NOT NULL REFERENCES media_accounts(id) ON DELETE CASCADE,
    platform TEXT NOT NULL DEFAULT '',
    profile_dir TEXT NOT NULL DEFAULT '',
    login_url TEXT NOT NULL DEFAULT '',
    state_file TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_media_account_login_sessions_account
    ON media_account_login_sessions(workspace_id, account_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_media_account_login_sessions_expires_at
    ON media_account_login_sessions(expires_at);

COMMIT;
