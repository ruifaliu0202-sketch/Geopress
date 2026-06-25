BEGIN;

CREATE TABLE IF NOT EXISTS knowledge_assets (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    asset_type TEXT NOT NULL DEFAULT 'document',
    mime_type TEXT NOT NULL DEFAULT '',
    original_filename TEXT NOT NULL DEFAULT '',
    storage_key TEXT NOT NULL DEFAULT '',
    checksum TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'ready', 'failed', 'archived')),
    error_message TEXT NOT NULL DEFAULT '',
    progress INT NOT NULL DEFAULT 0 CHECK (progress >= 0 AND progress <= 100),
    extracted_text TEXT NOT NULL DEFAULT '',
    ai_enhancement_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ai_enhancement_status TEXT NOT NULL DEFAULT 'disabled' CHECK (ai_enhancement_status IN ('disabled', 'pending', 'processing', 'succeeded', 'failed', 'skipped')),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS knowledge_asset_bases (
    asset_id TEXT NOT NULL REFERENCES knowledge_assets(id) ON DELETE CASCADE,
    knowledge_base_id TEXT NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (asset_id, knowledge_base_id)
);

CREATE TABLE IF NOT EXISTS knowledge_chunks (
    id TEXT PRIMARY KEY,
    asset_id TEXT NOT NULL REFERENCES knowledge_assets(id) ON DELETE CASCADE,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    chunk_index INT NOT NULL CHECK (chunk_index >= 0),
    title TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    search_text TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    tags TEXT[] NOT NULL DEFAULT '{}',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    embedding VECTOR(1536),
    embedding_status TEXT NOT NULL DEFAULT 'pending' CHECK (embedding_status IN ('pending', 'processing', 'ready', 'failed', 'skipped')),
    embedding_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (asset_id, chunk_index)
);

CREATE TABLE IF NOT EXISTS knowledge_chunk_bases (
    chunk_id TEXT NOT NULL REFERENCES knowledge_chunks(id) ON DELETE CASCADE,
    knowledge_base_id TEXT NOT NULL REFERENCES knowledge_bases(id) ON DELETE CASCADE,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (chunk_id, knowledge_base_id)
);

CREATE TABLE IF NOT EXISTS knowledge_processing_tasks (
    id TEXT PRIMARY KEY,
    asset_id TEXT NOT NULL REFERENCES knowledge_assets(id) ON DELETE CASCADE,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    task_type TEXT NOT NULL DEFAULT 'extract',
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'queued', 'running', 'succeeded', 'failed', 'canceled')),
    progress INT NOT NULL DEFAULT 0 CHECK (progress >= 0 AND progress <= 100),
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_knowledge_assets_workspace_status ON knowledge_assets(workspace_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_knowledge_asset_bases_workspace_base ON knowledge_asset_bases(workspace_id, knowledge_base_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_asset_bases_asset_id ON knowledge_asset_bases(asset_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_workspace_enabled ON knowledge_chunks(workspace_id, enabled, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_asset_index ON knowledge_chunks(asset_id, chunk_index);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_embedding_status ON knowledge_chunks(workspace_id, embedding_status);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunk_bases_workspace_base ON knowledge_chunk_bases(workspace_id, knowledge_base_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunk_bases_chunk_id ON knowledge_chunk_bases(chunk_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_processing_tasks_asset_status ON knowledge_processing_tasks(asset_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_knowledge_processing_tasks_workspace_status ON knowledge_processing_tasks(workspace_id, status, created_at DESC);

COMMIT;
