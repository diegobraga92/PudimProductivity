-- Phase 8: Collaboration & Multi-User
--
-- 1. Task lists gain an owner (the user who created them). Existing rows are
--    assigned to 'dev-user' — the identity the web client has always presented
--    in dev mode (see web/src/api/client.ts).
-- 2. task_list_shares grants editor/viewer access to another user.
-- 3. tasks gain updated_by so CRDT merges can break timestamp ties
--    deterministically (LWW tie-break: newest updated_at wins; on exact ties
--    the lexicographically greater updated_by wins — see docs/adr/010).

ALTER TABLE task_lists ADD COLUMN IF NOT EXISTS owner_id TEXT NOT NULL DEFAULT 'dev-user';
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS updated_by TEXT;

CREATE TABLE IF NOT EXISTS task_list_shares (
    list_id     UUID NOT NULL REFERENCES task_lists(id) ON DELETE CASCADE,
    shared_with TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'viewer' CHECK (role IN ('editor', 'viewer')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (list_id, shared_with)
);

CREATE INDEX IF NOT EXISTS idx_task_list_shares_shared_with ON task_list_shares (shared_with);
CREATE INDEX IF NOT EXISTS idx_task_lists_owner_id ON task_lists (owner_id);
