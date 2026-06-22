BEGIN;

CREATE TABLE IF NOT EXISTS campaigns (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'planned', 'active', 'paused', 'completed', 'archived')),
    goal TEXT NOT NULL DEFAULT '',
    products TEXT[] NOT NULL DEFAULT '{}',
    target_audiences TEXT[] NOT NULL DEFAULT '{}',
    channels TEXT[] NOT NULL DEFAULT '{}',
    media_account_ids TEXT[] NOT NULL DEFAULT '{}',
    start_at TIMESTAMPTZ,
    end_at TIMESTAMPTZ,
    budget_cents INT NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'CNY',
    content_quota INT NOT NULL DEFAULT 0,
    approval_policy TEXT NOT NULL DEFAULT 'manual',
    success_metrics TEXT[] NOT NULL DEFAULT '{}',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_at IS NULL OR start_at IS NULL OR end_at >= start_at)
);

CREATE TABLE IF NOT EXISTS campaign_topics (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    campaign_id TEXT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    brief TEXT NOT NULL DEFAULT '',
    content_type TEXT NOT NULL DEFAULT 'article',
    target_audience TEXT NOT NULL DEFAULT '',
    funnel_stage TEXT NOT NULL DEFAULT '',
    keywords TEXT[] NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'idea' CHECK (status IN ('idea', 'planned', 'drafted', 'approved', 'scheduled', 'published', 'canceled')),
    content_id TEXT REFERENCES contents(id) ON DELETE SET NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS campaign_calendar_items (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    campaign_id TEXT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    topic_id TEXT REFERENCES campaign_topics(id) ON DELETE SET NULL,
    content_id TEXT REFERENCES contents(id) ON DELETE SET NULL,
    publish_schedule_id TEXT REFERENCES publish_schedules(id) ON DELETE SET NULL,
    publish_job_id TEXT REFERENCES publish_jobs(id) ON DELETE SET NULL,
    media_account_id TEXT REFERENCES media_accounts(id) ON DELETE SET NULL,
    assigned_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    brief TEXT NOT NULL DEFAULT '',
    content_type TEXT NOT NULL DEFAULT 'article',
    channel TEXT NOT NULL DEFAULT '',
    publish_window_start_at TIMESTAMPTZ,
    publish_window_end_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'planned' CHECK (status IN ('planned', 'drafting', 'review', 'scheduled', 'published', 'skipped', 'canceled')),
    dependency_item_ids TEXT[] NOT NULL DEFAULT '{}',
    approval_required BOOLEAN NOT NULL DEFAULT FALSE,
    approval_status TEXT NOT NULL DEFAULT 'not_required',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (publish_window_end_at IS NULL OR publish_window_start_at IS NULL OR publish_window_end_at >= publish_window_start_at)
);

CREATE TABLE IF NOT EXISTS campaign_metrics (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    campaign_id TEXT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    calendar_item_id TEXT REFERENCES campaign_calendar_items(id) ON DELETE SET NULL,
    content_id TEXT REFERENCES contents(id) ON DELETE SET NULL,
    publish_job_id TEXT REFERENCES publish_jobs(id) ON DELETE SET NULL,
    media_account_id TEXT REFERENCES media_accounts(id) ON DELETE SET NULL,
    metric_name TEXT NOT NULL,
    metric_value DOUBLE PRECISION NOT NULL DEFAULT 0,
    metric_unit TEXT NOT NULL DEFAULT 'count',
    source TEXT NOT NULL DEFAULT 'manual',
    collected_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS campaign_rollups (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    campaign_id TEXT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    content_count INT NOT NULL DEFAULT 0,
    scheduled_count INT NOT NULL DEFAULT 0,
    published_count INT NOT NULL DEFAULT 0,
    failed_count INT NOT NULL DEFAULT 0,
    impression_count BIGINT NOT NULL DEFAULT 0,
    engagement_count BIGINT NOT NULL DEFAULT 0,
    click_count BIGINT NOT NULL DEFAULT 0,
    conversion_count BIGINT NOT NULL DEFAULT 0,
    spend_cents INT NOT NULL DEFAULT 0,
    revenue_cents INT NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (period_end >= period_start)
);

CREATE INDEX IF NOT EXISTS idx_campaigns_workspace_status ON campaigns(workspace_id, status);
CREATE INDEX IF NOT EXISTS idx_campaigns_workspace_updated ON campaigns(workspace_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_campaign_topics_campaign_status ON campaign_topics(workspace_id, campaign_id, status);
CREATE INDEX IF NOT EXISTS idx_campaign_calendar_campaign_status ON campaign_calendar_items(workspace_id, campaign_id, status);
CREATE INDEX IF NOT EXISTS idx_campaign_calendar_publish_window ON campaign_calendar_items(workspace_id, publish_window_start_at);
CREATE INDEX IF NOT EXISTS idx_campaign_metrics_campaign_collected ON campaign_metrics(workspace_id, campaign_id, collected_at DESC);
CREATE INDEX IF NOT EXISTS idx_campaign_rollups_campaign_period ON campaign_rollups(workspace_id, campaign_id, period_start DESC);

COMMIT;
