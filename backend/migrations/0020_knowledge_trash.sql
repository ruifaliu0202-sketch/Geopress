BEGIN;

ALTER TABLE knowledge_bases
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';

ALTER TABLE knowledge_bases
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

ALTER TABLE knowledge_bases
    ADD COLUMN IF NOT EXISTS delete_expires_at TIMESTAMPTZ;

ALTER TABLE knowledge_assets
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

ALTER TABLE knowledge_assets
    ADD COLUMN IF NOT EXISTS delete_expires_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_knowledge_bases_workspace_status
    ON knowledge_bases(workspace_id, status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_knowledge_bases_delete_expires_at
    ON knowledge_bases(delete_expires_at)
    WHERE deleted_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_knowledge_assets_delete_expires_at
    ON knowledge_assets(delete_expires_at)
    WHERE deleted_at IS NOT NULL;

COMMIT;
