BEGIN;

CREATE TABLE IF NOT EXISTS skill_packages (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT 'general',
    target_platform TEXT NOT NULL DEFAULT '',
    target_industry TEXT NOT NULL DEFAULT '',
    supported_content_formats TEXT[] NOT NULL DEFAULT '{}',
    author_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    author_name TEXT NOT NULL DEFAULT '',
    listing_status TEXT NOT NULL DEFAULT 'draft' CHECK (listing_status IN ('draft', 'in_review', 'approved', 'published', 'rejected', 'deprecated')),
    price_cents INT NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'USD',
    revenue_share_bps INT NOT NULL DEFAULT 0 CHECK (revenue_share_bps >= 0 AND revenue_share_bps <= 10000),
    latest_version_id TEXT NOT NULL DEFAULT '',
    published_version_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS skill_package_versions (
    id TEXT PRIMARY KEY,
    package_id TEXT NOT NULL REFERENCES skill_packages(id) ON DELETE CASCADE,
    version TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'submitted', 'approved', 'rejected', 'published', 'deprecated')),
    prompt_contract TEXT NOT NULL DEFAULT '',
    output_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    quality_rules TEXT NOT NULL DEFAULT '',
    qa_rules TEXT NOT NULL DEFAULT '',
    publish_prep_rules TEXT NOT NULL DEFAULT '',
    change_note TEXT NOT NULL DEFAULT '',
    submitted_at TIMESTAMPTZ,
    reviewed_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (package_id, version)
);

CREATE TABLE IF NOT EXISTS skill_package_assets (
    id TEXT PRIMARY KEY,
    package_id TEXT NOT NULL REFERENCES skill_packages(id) ON DELETE CASCADE,
    version_id TEXT NOT NULL REFERENCES skill_package_versions(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('prompt', 'schema', 'rule', 'example', 'qa', 'publish')),
    title TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS skill_package_examples (
    id TEXT PRIMARY KEY,
    package_id TEXT NOT NULL REFERENCES skill_packages(id) ON DELETE CASCADE,
    version_id TEXT NOT NULL REFERENCES skill_package_versions(id) ON DELETE CASCADE,
    title TEXT NOT NULL DEFAULT '',
    input TEXT NOT NULL DEFAULT '',
    expected_output TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS skill_package_reviews (
    id TEXT PRIMARY KEY,
    package_id TEXT NOT NULL REFERENCES skill_packages(id) ON DELETE CASCADE,
    version_id TEXT NOT NULL REFERENCES skill_package_versions(id) ON DELETE CASCADE,
    reviewer_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    decision TEXT NOT NULL CHECK (decision IN ('submitted', 'approved', 'rejected')),
    comment TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS workspace_skill_entitlements (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    package_id TEXT NOT NULL REFERENCES skill_packages(id) ON DELETE CASCADE,
    version_id TEXT NOT NULL REFERENCES skill_package_versions(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'expired', 'canceled', 'uninstalled')),
    source TEXT NOT NULL DEFAULT 'purchase' CHECK (source IN ('trial', 'purchase', 'subscription', 'admin_grant')),
    seats INT NOT NULL DEFAULT 1 CHECK (seats > 0),
    price_cents INT NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'USD',
    current_period TEXT NOT NULL DEFAULT '',
    current_period_started_at TIMESTAMPTZ,
    current_period_ends_at TIMESTAMPTZ,
    installed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, package_id, version_id)
);

CREATE TABLE IF NOT EXISTS skill_package_usage_metrics (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    package_id TEXT NOT NULL REFERENCES skill_packages(id) ON DELETE CASCADE,
    version_id TEXT NOT NULL REFERENCES skill_package_versions(id) ON DELETE CASCADE,
    generation_request_id TEXT NOT NULL DEFAULT '',
    content_id TEXT NOT NULL DEFAULT '',
    metric_type TEXT NOT NULL CHECK (metric_type IN ('generation', 'formatting', 'qa', 'publish_prep')),
    count INT NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS skill_package_revenue_metrics (
    id TEXT PRIMARY KEY,
    package_id TEXT NOT NULL REFERENCES skill_packages(id) ON DELETE CASCADE,
    version_id TEXT NOT NULL REFERENCES skill_package_versions(id) ON DELETE CASCADE,
    workspace_id TEXT REFERENCES workspaces(id) ON DELETE SET NULL,
    entitlement_id TEXT REFERENCES workspace_skill_entitlements(id) ON DELETE SET NULL,
    metric_type TEXT NOT NULL CHECK (metric_type IN ('purchase', 'subscription', 'refund', 'payout')),
    amount_cents INT NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'USD',
    author_revenue_cents INT NOT NULL DEFAULT 0,
    platform_fee_cents INT NOT NULL DEFAULT 0,
    billing_period TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE generation_requests
    ADD COLUMN IF NOT EXISTS skill_package_version_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_skill_packages_listing ON skill_packages(listing_status, category);
CREATE INDEX IF NOT EXISTS idx_skill_package_versions_package_status ON skill_package_versions(package_id, status);
CREATE INDEX IF NOT EXISTS idx_skill_package_assets_version ON skill_package_assets(version_id);
CREATE INDEX IF NOT EXISTS idx_skill_package_examples_version ON skill_package_examples(version_id);
CREATE INDEX IF NOT EXISTS idx_skill_package_reviews_version ON skill_package_reviews(version_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_workspace_skill_entitlements_workspace_status ON workspace_skill_entitlements(workspace_id, status);
CREATE INDEX IF NOT EXISTS idx_workspace_skill_entitlements_version ON workspace_skill_entitlements(version_id);
CREATE INDEX IF NOT EXISTS idx_skill_package_usage_metrics_workspace ON skill_package_usage_metrics(workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_skill_package_usage_metrics_package ON skill_package_usage_metrics(package_id, version_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_skill_package_revenue_metrics_package ON skill_package_revenue_metrics(package_id, version_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_generation_requests_skill_package_version ON generation_requests(skill_package_version_id);

COMMIT;
