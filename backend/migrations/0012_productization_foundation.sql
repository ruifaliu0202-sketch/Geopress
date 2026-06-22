BEGIN;

-- 统一外部平台能力契约，后续账号矩阵、指标采集和发布适配器只读取声明过的能力。
ALTER TABLE media_platforms
    ADD COLUMN IF NOT EXISTS capabilities JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE media_platforms
SET capabilities = jsonb_build_object(
    'authorizationMethods',
        CASE
            WHEN credential_fields @> '["qrLogin"]'::jsonb THEN '["qr_login"]'::jsonb
            ELSE '["manual_only"]'::jsonb
        END,
    'publishModes',
        CASE
            WHEN supports_scheduling THEN '["manual","api"]'::jsonb
            ELSE '["manual"]'::jsonb
        END,
    'contentFormats',
        CASE
            WHEN supports_article AND supports_image THEN '["article","image"]'::jsonb
            WHEN supports_article THEN '["article"]'::jsonb
            WHEN supports_image THEN '["image"]'::jsonb
            ELSE '[]'::jsonb
        END,
    'capabilities',
        jsonb_build_array(
            jsonb_build_object(
                'name', 'authorization',
                'mode',
                    CASE
                        WHEN credential_fields @> '["qrLogin"]'::jsonb THEN 'browser'
                        ELSE 'manual'
                    END,
                'enabled', true,
                'manualFallback', true
            ),
            jsonb_build_object(
                'name', 'content_publish',
                'mode',
                    CASE
                        WHEN supports_scheduling THEN 'api'
                        ELSE 'manual'
                    END,
                'enabled', supports_article OR supports_image,
                'manualFallback', true
            )
        ),
    'rateLimits', '{}'::jsonb
)
WHERE capabilities = '{}'::jsonb;

UPDATE media_platforms
SET capabilities = '{
    "authorizationMethods": ["qr_login"],
    "publishModes": ["manual", "browser"],
    "contentFormats": ["article", "image"],
    "capabilities": [
        {
            "name": "authorization",
            "mode": "browser",
            "enabled": true,
            "manualFallback": true,
            "notes": "Server-managed browser QR login; dynamic platform request headers are not persisted."
        },
        {
            "name": "profile_sync",
            "mode": "manual",
            "enabled": false,
            "manualFallback": true,
            "notes": "Declared for the account matrix foundation; profile sync is implemented in a later module."
        },
        {
            "name": "content_publish",
            "mode": "browser",
            "enabled": true,
            "manualFallback": true,
            "notes": "Browser publishing uses the saved logged-in profile and keeps manual confirmation as fallback."
        },
        {
            "name": "metric_ingestion",
            "mode": "manual",
            "enabled": false,
            "manualFallback": true,
            "notes": "Metrics are not scraped until a stable and compliant ingestion path exists."
        },
        {
            "name": "comment_ingestion",
            "mode": "disabled",
            "enabled": false,
            "manualFallback": false,
            "notes": "Comment ingestion is disabled by default because it requires explicit platform permission."
        }
    ],
    "rateLimits": {}
}'::jsonb
WHERE id = 'plt_xiaohongshu';

-- 仅建立通用商业权益骨架，不在本迁移中落地技能包、Campaign 或达人业务表。
CREATE TABLE IF NOT EXISTS product_entitlements (
    id TEXT PRIMARY KEY,
    product_line TEXT NOT NULL,
    subject_type TEXT NOT NULL,
    subject_id TEXT NOT NULL,
    workspace_id TEXT REFERENCES workspaces(id) ON DELETE CASCADE,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    starts_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    source TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE product_entitlements
    DROP CONSTRAINT IF EXISTS product_entitlements_product_line_check,
    ADD CONSTRAINT product_entitlements_product_line_check
        CHECK (product_line IN ('workspace_core', 'media_matrix', 'creator_collaboration', 'skill_package', 'campaign', 'commercial_entitlement')),
    DROP CONSTRAINT IF EXISTS product_entitlements_subject_type_check,
    ADD CONSTRAINT product_entitlements_subject_type_check
        CHECK (subject_type IN ('workspace', 'user')),
    DROP CONSTRAINT IF EXISTS product_entitlements_resource_type_check,
    ADD CONSTRAINT product_entitlements_resource_type_check
        CHECK (resource_type IN ('skill_package', 'platform_knowledge_base', 'connector_capability', 'campaign_feature', 'creator_marketplace')),
    DROP CONSTRAINT IF EXISTS product_entitlements_status_check,
    ADD CONSTRAINT product_entitlements_status_check
        CHECK (status IN ('pending', 'active', 'expired', 'revoked', 'canceled'));

CREATE INDEX IF NOT EXISTS idx_product_entitlements_subject
    ON product_entitlements(subject_type, subject_id, status);
CREATE INDEX IF NOT EXISTS idx_product_entitlements_workspace
    ON product_entitlements(workspace_id, status);
CREATE INDEX IF NOT EXISTS idx_product_entitlements_resource
    ON product_entitlements(resource_type, resource_id, status);

-- 审核记录作为跨内容、技能包、达人交付和发布任务的共用状态基座。
CREATE TABLE IF NOT EXISTS product_review_records (
    id TEXT PRIMARY KEY,
    product_line TEXT NOT NULL,
    workspace_id TEXT REFERENCES workspaces(id) ON DELETE CASCADE,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft',
    requested_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    reviewed_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    decision_message TEXT NOT NULL DEFAULT '',
    evidence JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE product_review_records
    DROP CONSTRAINT IF EXISTS product_review_records_product_line_check,
    ADD CONSTRAINT product_review_records_product_line_check
        CHECK (product_line IN ('workspace_core', 'media_matrix', 'creator_collaboration', 'skill_package', 'campaign', 'commercial_entitlement')),
    DROP CONSTRAINT IF EXISTS product_review_records_target_type_check,
    ADD CONSTRAINT product_review_records_target_type_check
        CHECK (target_type IN ('content', 'campaign', 'creator_deliverable', 'skill_package_version', 'publish_job', 'commercial_compliance')),
    DROP CONSTRAINT IF EXISTS product_review_records_status_check,
    ADD CONSTRAINT product_review_records_status_check
        CHECK (status IN ('draft', 'pending', 'approved', 'rejected', 'changes_requested', 'canceled'));

CREATE INDEX IF NOT EXISTS idx_product_review_records_target
    ON product_review_records(target_type, target_id, status);
CREATE INDEX IF NOT EXISTS idx_product_review_records_workspace
    ON product_review_records(workspace_id, status, updated_at DESC);

-- 合规证据独立保存，避免具体业务模块各自发明不可追踪的证明字段。
CREATE TABLE IF NOT EXISTS commercial_evidence_records (
    id TEXT PRIMARY KEY,
    product_line TEXT NOT NULL,
    workspace_id TEXT REFERENCES workspaces(id) ON DELETE CASCADE,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    evidence_type TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    captured_by_type TEXT NOT NULL DEFAULT 'system',
    captured_by_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE commercial_evidence_records
    DROP CONSTRAINT IF EXISTS commercial_evidence_records_product_line_check,
    ADD CONSTRAINT commercial_evidence_records_product_line_check
        CHECK (product_line IN ('workspace_core', 'media_matrix', 'creator_collaboration', 'skill_package', 'campaign', 'commercial_entitlement')),
    DROP CONSTRAINT IF EXISTS commercial_evidence_records_resource_type_check,
    ADD CONSTRAINT commercial_evidence_records_resource_type_check
        CHECK (resource_type IN ('workspace', 'media_platform', 'media_account', 'creator', 'creator_order', 'skill_package', 'skill_package_version', 'campaign', 'content', 'publish_job', 'entitlement', 'review', 'audit_log')),
    DROP CONSTRAINT IF EXISTS commercial_evidence_records_evidence_type_check,
    ADD CONSTRAINT commercial_evidence_records_evidence_type_check
        CHECK (evidence_type IN ('disclosure', 'usage_right', 'approval_record', 'publication', 'settlement')),
    DROP CONSTRAINT IF EXISTS commercial_evidence_records_captured_by_type_check,
    ADD CONSTRAINT commercial_evidence_records_captured_by_type_check
        CHECK (captured_by_type IN ('user', 'platform', 'system'));

CREATE INDEX IF NOT EXISTS idx_commercial_evidence_records_resource
    ON commercial_evidence_records(resource_type, resource_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_commercial_evidence_records_workspace
    ON commercial_evidence_records(workspace_id, created_at DESC);

-- 扩展现有审计表，保留 user_id 兼容旧查询，同时新增 actor/product 维度。
ALTER TABLE audit_logs
    ADD COLUMN IF NOT EXISTS product_line TEXT NOT NULL DEFAULT 'workspace_core',
    ADD COLUMN IF NOT EXISTS actor_type TEXT NOT NULL DEFAULT 'user',
    ADD COLUMN IF NOT EXISTS actor_id TEXT NOT NULL DEFAULT '';

UPDATE audit_logs
SET actor_id = user_id
WHERE actor_id = ''
  AND user_id IS NOT NULL;

ALTER TABLE audit_logs
    DROP CONSTRAINT IF EXISTS audit_logs_product_line_check,
    ADD CONSTRAINT audit_logs_product_line_check
        CHECK (product_line IN ('workspace_core', 'media_matrix', 'creator_collaboration', 'skill_package', 'campaign', 'commercial_entitlement')),
    DROP CONSTRAINT IF EXISTS audit_logs_actor_type_check,
    ADD CONSTRAINT audit_logs_actor_type_check
        CHECK (actor_type IN ('user', 'platform', 'system'));

CREATE INDEX IF NOT EXISTS idx_audit_logs_product_resource
    ON audit_logs(product_line, resource_type, resource_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor
    ON audit_logs(actor_type, actor_id, created_at DESC);

COMMIT;
