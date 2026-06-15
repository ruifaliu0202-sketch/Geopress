BEGIN;

ALTER TABLE publish_jobs DROP CONSTRAINT IF EXISTS publish_jobs_status_check;

ALTER TABLE publish_jobs
    ADD COLUMN IF NOT EXISTS external_url TEXT NOT NULL DEFAULT '';

ALTER TABLE publish_jobs
    ADD CONSTRAINT publish_jobs_status_check
    CHECK (status IN ('queued', 'running', 'manual_pending', 'succeeded', 'failed', 'retrying'));

COMMIT;
