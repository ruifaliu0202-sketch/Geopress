BEGIN;

CREATE TABLE IF NOT EXISTS brand_assets (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    type TEXT NOT NULL DEFAULT 'brand',
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    channels JSONB NOT NULL DEFAULT '[]'::jsonb,
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    source TEXT NOT NULL DEFAULT 'manual',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS brand_guardrails (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    asset_id TEXT REFERENCES brand_assets(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    category TEXT NOT NULL DEFAULT 'brand',
    channel TEXT NOT NULL DEFAULT '',
    source_type TEXT NOT NULL DEFAULT 'manual',
    source_id TEXT NOT NULL DEFAULT '',
    severity TEXT NOT NULL DEFAULT 'medium' CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
    rules JSONB NOT NULL DEFAULT '[]'::jsonb,
    action TEXT NOT NULL DEFAULT 'review',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS approval_workflows (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('draft', 'active', 'completed', 'canceled')),
    stages JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS approval_tasks (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    workflow_id TEXT NOT NULL REFERENCES approval_workflows(id) ON DELETE CASCADE,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL DEFAULT '',
    stage_name TEXT NOT NULL DEFAULT '',
    assignee_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    assignee_role TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'skipped', 'canceled')),
    decision TEXT NOT NULL DEFAULT '',
    comment TEXT NOT NULL DEFAULT '',
    processed_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    due_at TIMESTAMPTZ,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS compliance_checks (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL DEFAULT '',
    channel TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'completed' CHECK (status IN ('queued', 'running', 'completed', 'failed')),
    risk_level TEXT NOT NULL DEFAULT 'low' CHECK (risk_level IN ('none', 'low', 'medium', 'high', 'critical')),
    summary TEXT NOT NULL DEFAULT '',
    created_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS compliance_findings (
    id TEXT PRIMARY KEY,
    check_id TEXT NOT NULL REFERENCES compliance_checks(id) ON DELETE CASCADE,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    severity TEXT NOT NULL DEFAULT 'low' CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
    category TEXT NOT NULL DEFAULT 'general',
    evidence TEXT NOT NULL DEFAULT '',
    finding TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL DEFAULT '',
    source_type TEXT NOT NULL DEFAULT '',
    source_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS agency_client_relations (
    id TEXT PRIMARY KEY,
    agency_workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    client_workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    client_name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'ended')),
    scopes JSONB NOT NULL DEFAULT '[]'::jsonb,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (agency_workspace_id, client_workspace_id)
);

CREATE TABLE IF NOT EXISTS report_packages (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    report_type TEXT NOT NULL DEFAULT 'monthly',
    audience TEXT NOT NULL DEFAULT 'management',
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    status TEXT NOT NULL DEFAULT 'generated' CHECK (status IN ('draft', 'generated', 'exported', 'archived')),
    sections JSONB NOT NULL DEFAULT '[]'::jsonb,
    metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
    summary TEXT NOT NULL DEFAULT '',
    generated_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS strategy_recommendations (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    source_type TEXT NOT NULL DEFAULT 'rule_placeholder',
    recommendation_type TEXT NOT NULL DEFAULT 'topic',
    title TEXT NOT NULL,
    rationale TEXT NOT NULL DEFAULT '',
    evidence JSONB NOT NULL DEFAULT '[]'::jsonb,
    action TEXT NOT NULL DEFAULT '',
    confidence NUMERIC(5, 4) NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'accepted', 'dismissed', 'archived')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_brand_assets_workspace_updated
    ON brand_assets(workspace_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_brand_guardrails_workspace_enabled
    ON brand_guardrails(workspace_id, enabled, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_approval_workflows_workspace_resource
    ON approval_workflows(workspace_id, resource_type, resource_id);

CREATE INDEX IF NOT EXISTS idx_approval_tasks_workspace_status
    ON approval_tasks(workspace_id, status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_compliance_checks_workspace_resource
    ON compliance_checks(workspace_id, resource_type, resource_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_compliance_findings_check
    ON compliance_findings(check_id);

CREATE INDEX IF NOT EXISTS idx_agency_client_relations_agency
    ON agency_client_relations(agency_workspace_id, status);

CREATE INDEX IF NOT EXISTS idx_agency_client_relations_client
    ON agency_client_relations(client_workspace_id, status);

CREATE INDEX IF NOT EXISTS idx_report_packages_workspace_period
    ON report_packages(workspace_id, period_end DESC);

CREATE INDEX IF NOT EXISTS idx_strategy_recommendations_workspace_status
    ON strategy_recommendations(workspace_id, status, updated_at DESC);

COMMIT;
