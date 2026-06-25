BEGIN;

INSERT INTO knowledge_assets (
    id,
    workspace_id,
    title,
    asset_type,
    mime_type,
    original_filename,
    storage_key,
    status,
    progress,
    extracted_text,
    ai_enhancement_enabled,
    ai_enhancement_status,
    metadata,
    created_at,
    updated_at
)
SELECT
    'kba_legacy_' || ki.id,
    ki.workspace_id,
    ki.title,
    COALESCE(NULLIF(ki.type, ''), 'legacy_item'),
    'text/markdown',
    ki.title || '.md',
    'legacy:' || ki.id,
    'ready',
    100,
    ki.content,
    FALSE,
    'disabled',
    jsonb_build_object(
        'legacyKnowledgeItemId', ki.id,
        'legacyType', ki.type,
        'migratedFrom', 'knowledge_items'
    ),
    ki.created_at,
    ki.updated_at
FROM knowledge_items ki
ON CONFLICT (id) DO UPDATE SET
    workspace_id = EXCLUDED.workspace_id,
    title = EXCLUDED.title,
    asset_type = EXCLUDED.asset_type,
    mime_type = EXCLUDED.mime_type,
    original_filename = EXCLUDED.original_filename,
    storage_key = EXCLUDED.storage_key,
    status = EXCLUDED.status,
    progress = EXCLUDED.progress,
    extracted_text = EXCLUDED.extracted_text,
    metadata = EXCLUDED.metadata,
    updated_at = EXCLUDED.updated_at;

INSERT INTO knowledge_asset_bases (asset_id, knowledge_base_id, workspace_id)
SELECT
    'kba_legacy_' || kib.knowledge_item_id,
    kib.knowledge_base_id,
    kib.workspace_id
FROM knowledge_item_bases kib
JOIN knowledge_assets ka ON ka.id = 'kba_legacy_' || kib.knowledge_item_id
JOIN knowledge_bases kb ON kb.id = kib.knowledge_base_id AND kb.workspace_id = kib.workspace_id
ON CONFLICT DO NOTHING;

INSERT INTO knowledge_chunks (
    id,
    asset_id,
    workspace_id,
    chunk_index,
    title,
    content,
    search_text,
    metadata,
    enabled,
    embedding_status,
    created_at,
    updated_at
)
SELECT
    'kbc_legacy_' || ki.id || '_0000',
    'kba_legacy_' || ki.id,
    ki.workspace_id,
    0,
    ki.title,
    ki.content,
    trim(ki.title || E'\n' || ki.content),
    jsonb_build_object(
        'legacyKnowledgeItemId', ki.id,
        'legacyType', ki.type,
        'migratedFrom', 'knowledge_items',
        'type', ki.type
    ),
    ki.enabled,
    'skipped',
    ki.created_at,
    ki.updated_at
FROM knowledge_items ki
JOIN knowledge_assets ka ON ka.id = 'kba_legacy_' || ki.id
ON CONFLICT (id) DO UPDATE SET
    asset_id = EXCLUDED.asset_id,
    workspace_id = EXCLUDED.workspace_id,
    chunk_index = EXCLUDED.chunk_index,
    title = EXCLUDED.title,
    content = EXCLUDED.content,
    search_text = EXCLUDED.search_text,
    metadata = EXCLUDED.metadata,
    enabled = EXCLUDED.enabled,
    embedding_status = EXCLUDED.embedding_status,
    updated_at = EXCLUDED.updated_at;

INSERT INTO knowledge_chunk_bases (chunk_id, knowledge_base_id, workspace_id)
SELECT
    'kbc_legacy_' || kib.knowledge_item_id || '_0000',
    kib.knowledge_base_id,
    kib.workspace_id
FROM knowledge_item_bases kib
JOIN knowledge_chunks kc ON kc.id = 'kbc_legacy_' || kib.knowledge_item_id || '_0000'
JOIN knowledge_bases kb ON kb.id = kib.knowledge_base_id AND kb.workspace_id = kib.workspace_id
ON CONFLICT DO NOTHING;

INSERT INTO knowledge_processing_tasks (
    id,
    asset_id,
    workspace_id,
    task_type,
    status,
    progress,
    created_at,
    started_at,
    finished_at,
    updated_at
)
SELECT
    'kbpt_legacy_' || ki.id || '_extract',
    'kba_legacy_' || ki.id,
    ki.workspace_id,
    'extract',
    'succeeded',
    100,
    ki.updated_at,
    ki.updated_at,
    ki.updated_at,
    ki.updated_at
FROM knowledge_items ki
JOIN knowledge_assets ka ON ka.id = 'kba_legacy_' || ki.id
ON CONFLICT (id) DO UPDATE SET
    asset_id = EXCLUDED.asset_id,
    workspace_id = EXCLUDED.workspace_id,
    status = EXCLUDED.status,
    progress = EXCLUDED.progress,
    finished_at = EXCLUDED.finished_at,
    updated_at = EXCLUDED.updated_at;

COMMIT;
