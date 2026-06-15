BEGIN;

CREATE TABLE IF NOT EXISTS platform_knowledge_item_bases (
    platform_knowledge_item_id TEXT NOT NULL REFERENCES platform_knowledge_items(id) ON DELETE CASCADE,
    platform_knowledge_base_id TEXT NOT NULL REFERENCES platform_knowledge_bases(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (platform_knowledge_item_id, platform_knowledge_base_id)
);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'platform_knowledge_items'
          AND column_name = 'knowledge_base_id'
    ) THEN
        EXECUTE '
            INSERT INTO platform_knowledge_item_bases (platform_knowledge_item_id, platform_knowledge_base_id)
            SELECT id, knowledge_base_id
            FROM platform_knowledge_items
            WHERE knowledge_base_id IS NOT NULL
            ON CONFLICT DO NOTHING
        ';

        ALTER TABLE platform_knowledge_items DROP COLUMN knowledge_base_id;
    END IF;
END $$;

DROP INDEX IF EXISTS idx_platform_knowledge_items_base_id;
CREATE INDEX IF NOT EXISTS idx_platform_knowledge_item_bases_base_id ON platform_knowledge_item_bases(platform_knowledge_base_id);
CREATE INDEX IF NOT EXISTS idx_platform_knowledge_item_bases_item_id ON platform_knowledge_item_bases(platform_knowledge_item_id);

COMMIT;
