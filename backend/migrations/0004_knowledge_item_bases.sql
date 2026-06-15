BEGIN;

CREATE TABLE IF NOT EXISTS knowledge_item_bases (
    knowledge_item_id TEXT NOT NULL REFERENCES knowledge_items(id) ON DELETE CASCADE,
    knowledge_base_id TEXT NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (knowledge_item_id, knowledge_base_id)
);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'knowledge_items'
          AND column_name = 'knowledge_base_id'
    ) THEN
        EXECUTE '
            INSERT INTO knowledge_item_bases (knowledge_item_id, knowledge_base_id, workspace_id)
            SELECT id, knowledge_base_id, workspace_id
            FROM knowledge_items
            WHERE knowledge_base_id IS NOT NULL
            ON CONFLICT DO NOTHING
        ';

        ALTER TABLE knowledge_items DROP COLUMN knowledge_base_id;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_knowledge_item_bases_workspace_base ON knowledge_item_bases(workspace_id, knowledge_base_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_item_bases_item_id ON knowledge_item_bases(knowledge_item_id);

COMMIT;
