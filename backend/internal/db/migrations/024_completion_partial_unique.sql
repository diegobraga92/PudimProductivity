-- Phase 9c follow-up: task_completions uniqueness must apply to active rows only.
--
-- Migration 004 created a full `UNIQUE(task_id, completed_date)` constraint and
-- migration 019 turned completion removal into a soft delete (`deleted_at`), so
-- the tombstone row kept occupying the unique key. Re-completing a date after
-- unchecking it failed with a false "already completed" (409), which surfaced on
-- web and mobile as "a habit unchecked on a previous date cannot be re-checked".
--
-- Replace the full unique constraint with a partial unique index that only
-- constrains active (non-soft-deleted) rows. Soft-deleted tombstones no longer
-- block re-insertion, while the "already completed for this date" contract is
-- preserved for active completions (CreateCompletion targets this index).
--
-- Safe to apply: the previous full constraint guaranteed at most one row per
-- (task_id, completed_date), so no duplicate active rows can exist here.

ALTER TABLE task_completions
    DROP CONSTRAINT IF EXISTS task_completions_task_id_completed_date_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_task_completions_active_task_date
    ON task_completions (task_id, completed_date)
    WHERE deleted_at IS NULL;
