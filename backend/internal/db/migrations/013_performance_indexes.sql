-- Phase 6 database performance review: add indexes for the two most frequent
-- queries that were doing sequential scans.
--
-- 1. Habit list (tasks WHERE list_id IS NULL AND recurrence_days IS NOT NULL
--    ORDER BY created_at DESC) — partial index gives index-ordered scan.
CREATE INDEX IF NOT EXISTS idx_tasks_habits
    ON tasks (created_at DESC)
    WHERE recurrence_days IS NOT NULL;

-- 2. Batch completions (task_completions WHERE completed_date BETWEEN $1 AND $2)
--    — used by the habit screens / heatmap week range.
CREATE INDEX IF NOT EXISTS idx_task_completions_date
    ON task_completions (completed_date);
