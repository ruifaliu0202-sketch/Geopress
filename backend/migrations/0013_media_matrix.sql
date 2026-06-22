BEGIN;

ALTER TABLE media_accounts
    ADD COLUMN IF NOT EXISTS account_group TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ownership_type TEXT NOT NULL DEFAULT 'owned',
    ADD COLUMN IF NOT EXISTS operating_role TEXT NOT NULL DEFAULT 'primary',
    ADD COLUMN IF NOT EXISTS persona TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS positioning TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS target_audience TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS content_categories TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS health_status TEXT NOT NULL DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS health_notes TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS authorization_scopes TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS sync_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS last_sync_job_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_sync_status TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_sync_message TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS last_profile_synced_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_metrics_synced_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS next_sync_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS matrix_metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE media_accounts
SET health_status = CASE
        WHEN status = 'connected' THEN 'healthy'
        WHEN status IN ('pending_login', 'qr_waiting') THEN 'needs_authorization'
        WHEN status = 'expired' THEN 'expired'
        ELSE health_status
    END,
    authorization_scopes = CASE
        WHEN array_length(authorization_scopes, 1) IS NULL AND credentials ? 'loginMethod' THEN ARRAY['profile:read']
        ELSE authorization_scopes
    END
WHERE health_status = 'unknown'
   OR array_length(authorization_scopes, 1) IS NULL;

CREATE TABLE IF NOT EXISTS media_account_metric_snapshots (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    media_account_id TEXT NOT NULL REFERENCES media_accounts(id) ON DELETE CASCADE,
    platform_id TEXT NOT NULL REFERENCES media_platforms(id),
    source TEXT NOT NULL DEFAULT 'manual',
    captured_at TIMESTAMPTZ NOT NULL,
    follower_count INT NOT NULL DEFAULT 0,
    following_count INT NOT NULL DEFAULT 0,
    content_count INT NOT NULL DEFAULT 0,
    total_like_count INT NOT NULL DEFAULT 0,
    total_favorite_count INT NOT NULL DEFAULT 0,
    total_comment_count INT NOT NULL DEFAULT 0,
    total_share_count INT NOT NULL DEFAULT 0,
    engagement_rate NUMERIC(10,4) NOT NULL DEFAULT 0,
    audience_signals JSONB NOT NULL DEFAULT '{}'::jsonb,
    profile JSONB NOT NULL DEFAULT '{}'::jsonb,
    raw_metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
    freshness_status TEXT NOT NULL DEFAULT 'fresh',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, media_account_id, captured_at, source)
);

CREATE TABLE IF NOT EXISTS content_metrics (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    content_id TEXT NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    publish_job_id TEXT REFERENCES publish_jobs(id) ON DELETE SET NULL,
    media_account_id TEXT NOT NULL REFERENCES media_accounts(id) ON DELETE CASCADE,
    platform_id TEXT NOT NULL REFERENCES media_platforms(id),
    external_content_id TEXT NOT NULL DEFAULT '',
    external_url TEXT NOT NULL DEFAULT '',
    metric_date DATE NOT NULL,
    captured_at TIMESTAMPTZ NOT NULL,
    impression_count INT NOT NULL DEFAULT 0,
    view_count INT NOT NULL DEFAULT 0,
    like_count INT NOT NULL DEFAULT 0,
    comment_count INT NOT NULL DEFAULT 0,
    share_count INT NOT NULL DEFAULT 0,
    favorite_count INT NOT NULL DEFAULT 0,
    click_count INT NOT NULL DEFAULT 0,
    engagement_rate NUMERIC(10,4) NOT NULL DEFAULT 0,
    attribution_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    raw_metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS media_account_sync_jobs (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    media_account_id TEXT NOT NULL REFERENCES media_accounts(id) ON DELETE CASCADE,
    platform_id TEXT NOT NULL REFERENCES media_platforms(id),
    requested_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    sync_type TEXT NOT NULL DEFAULT 'metrics',
    status TEXT NOT NULL DEFAULT 'queued',
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    idempotency_key TEXT NOT NULL,
    request_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    result_summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, media_account_id, idempotency_key)
);

ALTER TABLE contents
    ADD COLUMN IF NOT EXISTS attributed_media_account_id TEXT REFERENCES media_accounts(id) ON DELETE SET NULL;

ALTER TABLE publish_jobs
    ADD COLUMN IF NOT EXISTS attribution_metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE contents AS c
SET attributed_media_account_id = latest.media_account_id
FROM (
    SELECT DISTINCT ON (content_id)
        content_id,
        media_account_id
    FROM publish_jobs
    WHERE content_id IS NOT NULL
    ORDER BY content_id, scheduled_at DESC, id DESC
) AS latest
WHERE c.id = latest.content_id
  AND c.attributed_media_account_id IS NULL;

UPDATE publish_jobs AS pj
SET attribution_metadata = jsonb_build_object(
    'contentId', COALESCE(pj.content_id, ''),
    'mediaAccountId', pj.media_account_id,
    'scheduleId', COALESCE(pj.schedule_id, ''),
    'attributionSource', 'publish_job',
    'attributedAt', COALESCE(pj.updated_at, now())
)
WHERE attribution_metadata = '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_media_accounts_workspace_group ON media_accounts(workspace_id, account_group);
CREATE INDEX IF NOT EXISTS idx_media_accounts_workspace_health ON media_accounts(workspace_id, health_status);
CREATE INDEX IF NOT EXISTS idx_media_accounts_sync_status ON media_accounts(workspace_id, last_sync_status, next_sync_at);
CREATE INDEX IF NOT EXISTS idx_media_account_metric_snapshots_account_time ON media_account_metric_snapshots(workspace_id, media_account_id, captured_at DESC);
CREATE INDEX IF NOT EXISTS idx_media_account_metric_snapshots_freshness ON media_account_metric_snapshots(workspace_id, freshness_status, captured_at DESC);
CREATE INDEX IF NOT EXISTS idx_content_metrics_account_date ON content_metrics(workspace_id, media_account_id, metric_date DESC);
CREATE INDEX IF NOT EXISTS idx_content_metrics_content_date ON content_metrics(workspace_id, content_id, metric_date DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_content_metrics_external_day
    ON content_metrics(workspace_id, media_account_id, external_content_id, metric_date)
    WHERE external_content_id <> '';
CREATE INDEX IF NOT EXISTS idx_media_account_sync_jobs_account_time ON media_account_sync_jobs(workspace_id, media_account_id, requested_at DESC);
CREATE INDEX IF NOT EXISTS idx_media_account_sync_jobs_status ON media_account_sync_jobs(workspace_id, status, requested_at DESC);
CREATE INDEX IF NOT EXISTS idx_contents_attributed_media_account ON contents(workspace_id, attributed_media_account_id);

COMMIT;
