-- Phase 9c: offline-first sync — soft-delete + change-tracking.
--
-- 1. Soft-delete columns: offline clients need to know what was deleted since
--    their last sync, so DELETE becomes `UPDATE ... SET deleted_at = NOW()`.
--    Every read query filters `deleted_at IS NULL`.
-- 2. Indexes on updated_at/created_at make the `GET /api/v1/sync?since=...`
--    incremental queries cheap.

ALTER TABLE tasks             ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE task_lists        ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE task_list_shares  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE task_completions  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_tasks_updated_at      ON tasks (updated_at);
CREATE INDEX IF NOT EXISTS idx_task_lists_updated_at ON task_lists (updated_at);
CREATE INDEX IF NOT EXISTS idx_task_completions_created_at ON task_completions (created_at);
