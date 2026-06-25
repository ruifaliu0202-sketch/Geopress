BEGIN;

ALTER TABLE knowledge_assets
    ADD COLUMN IF NOT EXISTS source_data BYTEA;

COMMIT;
