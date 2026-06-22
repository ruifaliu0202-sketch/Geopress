BEGIN;

CREATE TABLE IF NOT EXISTS creators (
    id TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    legal_name TEXT NOT NULL DEFAULT '',
    bio TEXT NOT NULL DEFAULT '',
    avatar_url TEXT NOT NULL DEFAULT '',
    contact_email TEXT NOT NULL DEFAULT '',
    verticals TEXT[] NOT NULL DEFAULT '{}',
    audience_attributes JSONB NOT NULL DEFAULT '{}'::jsonb,
    base_price_cents INT NOT NULL DEFAULT 0 CHECK (base_price_cents >= 0),
    currency TEXT NOT NULL DEFAULT 'CNY',
    availability_status TEXT NOT NULL DEFAULT 'available'
        CHECK (availability_status IN ('available', 'limited', 'unavailable')),
    collaboration_policy TEXT NOT NULL DEFAULT '',
    verification_state TEXT NOT NULL DEFAULT 'unverified'
        CHECK (verification_state IN ('unverified', 'pending', 'verified', 'rejected')),
    brand_safety_level TEXT NOT NULL DEFAULT 'unknown',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS creator_media_accounts (
    id TEXT PRIMARY KEY,
    creator_id TEXT NOT NULL REFERENCES creators(id) ON DELETE CASCADE,
    platform_id TEXT REFERENCES media_platforms(id) ON DELETE SET NULL,
    platform_name TEXT NOT NULL DEFAULT '',
    handle TEXT NOT NULL DEFAULT '',
    profile_url TEXT NOT NULL DEFAULT '',
    follower_count INT NOT NULL DEFAULT 0 CHECK (follower_count >= 0),
    average_engagement_rate NUMERIC(8, 4) NOT NULL DEFAULT 0 CHECK (average_engagement_rate >= 0),
    verticals TEXT[] NOT NULL DEFAULT '{}',
    audience_attributes JSONB NOT NULL DEFAULT '{}'::jsonb,
    account_access_mode TEXT NOT NULL DEFAULT 'creator_operated'
        CHECK (account_access_mode IN ('creator_operated', 'agency_authorized', 'public_profile')),
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (creator_id, platform_name, handle)
);

CREATE TABLE IF NOT EXISTS creator_shortlists (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    creator_id TEXT NOT NULL REFERENCES creators(id) ON DELETE CASCADE,
    name TEXT NOT NULL DEFAULT 'default',
    fit_score INT NOT NULL DEFAULT 0 CHECK (fit_score >= 0 AND fit_score <= 100),
    qualification_status TEXT NOT NULL DEFAULT 'watching'
        CHECK (qualification_status IN ('watching', 'qualified', 'rejected', 'ordered')),
    brand_safety_level TEXT NOT NULL DEFAULT 'unknown',
    brand_safety_notes TEXT NOT NULL DEFAULT '',
    operator_notes TEXT NOT NULL DEFAULT '',
    created_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, creator_id, name)
);

CREATE TABLE IF NOT EXISTS creator_campaign_briefs (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    objective TEXT NOT NULL DEFAULT '',
    product_name TEXT NOT NULL DEFAULT '',
    target_audience TEXT NOT NULL DEFAULT '',
    platform_targets TEXT[] NOT NULL DEFAULT '{}',
    deliverable_requirements TEXT[] NOT NULL DEFAULT '{}',
    disclosure_requirements TEXT[] NOT NULL DEFAULT '{}',
    prohibited_claims TEXT[] NOT NULL DEFAULT '{}',
    authorization_scope TEXT NOT NULL DEFAULT '',
    content_usage_rights TEXT NOT NULL DEFAULT '',
    review_window_hours INT NOT NULL DEFAULT 72 CHECK (review_window_hours >= 0),
    deadline_at TIMESTAMPTZ,
    budget_cents INT NOT NULL DEFAULT 0 CHECK (budget_cents >= 0),
    currency TEXT NOT NULL DEFAULT 'CNY',
    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'active', 'archived')),
    created_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS creator_orders (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    brief_id TEXT NOT NULL REFERENCES creator_campaign_briefs(id) ON DELETE RESTRICT,
    creator_id TEXT NOT NULL REFERENCES creators(id) ON DELETE RESTRICT,
    status TEXT NOT NULL DEFAULT 'proposed'
        CHECK (status IN ('proposed', 'accepted', 'in_progress', 'submitted', 'approved', 'published', 'completed', 'canceled', 'disputed')),
    price_cents INT NOT NULL DEFAULT 0 CHECK (price_cents >= 0),
    deposit_cents INT NOT NULL DEFAULT 0 CHECK (deposit_cents >= 0),
    service_fee_cents INT NOT NULL DEFAULT 0 CHECK (service_fee_cents >= 0),
    currency TEXT NOT NULL DEFAULT 'CNY',
    disclosure_requirements TEXT[] NOT NULL DEFAULT '{}',
    deliverable_requirements TEXT[] NOT NULL DEFAULT '{}',
    authorization_scope TEXT NOT NULL DEFAULT '',
    content_usage_rights TEXT NOT NULL DEFAULT '',
    due_at TIMESTAMPTZ,
    accepted_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    last_message TEXT NOT NULL DEFAULT '',
    created_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS creator_deliverables (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    order_id TEXT NOT NULL REFERENCES creator_orders(id) ON DELETE CASCADE,
    creator_id TEXT NOT NULL REFERENCES creators(id) ON DELETE RESTRICT,
    type TEXT NOT NULL DEFAULT 'draft',
    title TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    asset_urls TEXT[] NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'submitted'
        CHECK (status IN ('submitted', 'revision_requested', 'approved', 'rejected', 'published')),
    external_url TEXT NOT NULL DEFAULT '',
    publication_proof_url TEXT NOT NULL DEFAULT '',
    publication_proof_note TEXT NOT NULL DEFAULT '',
    review_feedback TEXT NOT NULL DEFAULT '',
    revision INT NOT NULL DEFAULT 1 CHECK (revision >= 1),
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS creator_settlements (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    order_id TEXT NOT NULL UNIQUE REFERENCES creator_orders(id) ON DELETE CASCADE,
    creator_id TEXT NOT NULL REFERENCES creators(id) ON DELETE RESTRICT,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'invoiced', 'payable', 'paid', 'refunded', 'disputed', 'canceled')),
    price_cents INT NOT NULL DEFAULT 0 CHECK (price_cents >= 0),
    deposit_cents INT NOT NULL DEFAULT 0 CHECK (deposit_cents >= 0),
    service_fee_cents INT NOT NULL DEFAULT 0 CHECK (service_fee_cents >= 0),
    creator_payout_cents INT NOT NULL DEFAULT 0 CHECK (creator_payout_cents >= 0),
    currency TEXT NOT NULL DEFAULT 'CNY',
    invoice_id TEXT NOT NULL DEFAULT '',
    due_at TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS creator_compliance_evidence (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    order_id TEXT NOT NULL REFERENCES creator_orders(id) ON DELETE CASCADE,
    deliverable_id TEXT REFERENCES creator_deliverables(id) ON DELETE SET NULL,
    creator_id TEXT NOT NULL REFERENCES creators(id) ON DELETE RESTRICT,
    evidence_type TEXT NOT NULL
        CHECK (evidence_type IN ('ad_disclosure', 'authorization_record', 'usage_rights', 'review_log', 'publication_proof', 'dispute_record')),
    disclosure_text TEXT NOT NULL DEFAULT '',
    authorization_scope TEXT NOT NULL DEFAULT '',
    content_usage_rights TEXT NOT NULL DEFAULT '',
    external_url TEXT NOT NULL DEFAULT '',
    file_url TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    created_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_creator_media_accounts_creator
    ON creator_media_accounts(creator_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_creator_shortlists_workspace
    ON creator_shortlists(workspace_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_creator_campaign_briefs_workspace
    ON creator_campaign_briefs(workspace_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_creator_orders_workspace
    ON creator_orders(workspace_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_creator_deliverables_order
    ON creator_deliverables(workspace_id, order_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_creator_settlements_workspace
    ON creator_settlements(workspace_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_creator_compliance_evidence_order
    ON creator_compliance_evidence(workspace_id, order_id, created_at DESC);

INSERT INTO creators (
    id, display_name, legal_name, bio, avatar_url, contact_email,
    verticals, audience_attributes, base_price_cents, currency,
    availability_status, collaboration_policy, verification_state,
    brand_safety_level, created_at, updated_at
)
VALUES
    (
        'crt_lina', 'Lina 本地生活', 'Lin Na',
        '小红书本地生活探店达人，擅长门店体验和消费决策内容。',
        'https://example.com/creators/lina.png', 'lina@example.com',
        ARRAY['本地生活', '餐饮', '小红书'],
        '{"city":"上海","primaryAge":"25-34"}'::jsonb,
        120000, 'CNY', 'available',
        '只接受品牌提供素材和审核意见，不提供账号登录权限。',
        'verified', 'medium', now() - interval '4 months', now() - interval '4 hours'
    ),
    (
        'crt_mason', 'Mason SaaS 增长', '',
        'B2B SaaS 增长内容作者，适合白皮书、案例和深度测评合作。',
        'https://example.com/creators/mason.png', '',
        ARRAY['B2B SaaS', '增长', '内容营销'],
        '{"audience":"创始人/市场负责人","region":"中国"}'::jsonb,
        180000, 'CNY', 'limited',
        '达人自行发布，品牌获得约定范围内的内容使用权。',
        'verified', 'low', now() - interval '5 months', now() - interval '8 hours'
    )
ON CONFLICT (id) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    bio = EXCLUDED.bio,
    avatar_url = EXCLUDED.avatar_url,
    contact_email = EXCLUDED.contact_email,
    verticals = EXCLUDED.verticals,
    audience_attributes = EXCLUDED.audience_attributes,
    base_price_cents = EXCLUDED.base_price_cents,
    currency = EXCLUDED.currency,
    availability_status = EXCLUDED.availability_status,
    collaboration_policy = EXCLUDED.collaboration_policy,
    verification_state = EXCLUDED.verification_state,
    brand_safety_level = EXCLUDED.brand_safety_level,
    updated_at = EXCLUDED.updated_at;

INSERT INTO creator_media_accounts (
    id, creator_id, platform_id, platform_name, handle, profile_url,
    follower_count, average_engagement_rate, verticals, audience_attributes,
    account_access_mode, verified, created_at, updated_at
)
VALUES
    (
        'cma_lina_xhs', 'crt_lina', 'plt_xiaohongshu', '小红书', 'lina_local',
        'https://www.xiaohongshu.com/user/profile/lina_local',
        86000, 0.073, ARRAY['本地生活', '餐饮'],
        '{"city":"上海","gender":"女性为主"}'::jsonb,
        'creator_operated', TRUE, now() - interval '4 months', now() - interval '4 hours'
    ),
    (
        'cma_mason_xhs', 'crt_mason', 'plt_xiaohongshu', '小红书', 'mason_growth',
        'https://www.xiaohongshu.com/user/profile/mason_growth',
        42000, 0.041, ARRAY['B2B SaaS', '增长'],
        '{"audience":"市场/增长负责人"}'::jsonb,
        'creator_operated', TRUE, now() - interval '5 months', now() - interval '8 hours'
    )
ON CONFLICT (id) DO UPDATE SET
    platform_id = EXCLUDED.platform_id,
    platform_name = EXCLUDED.platform_name,
    handle = EXCLUDED.handle,
    profile_url = EXCLUDED.profile_url,
    follower_count = EXCLUDED.follower_count,
    average_engagement_rate = EXCLUDED.average_engagement_rate,
    verticals = EXCLUDED.verticals,
    audience_attributes = EXCLUDED.audience_attributes,
    account_access_mode = EXCLUDED.account_access_mode,
    verified = EXCLUDED.verified,
    updated_at = EXCLUDED.updated_at;

COMMIT;
