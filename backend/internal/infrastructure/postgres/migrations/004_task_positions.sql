-- 004_task_positions.sql: manual ordering for tasks and habits.

ALTER TABLE tasks ADD COLUMN position INTEGER NOT NULL DEFAULT 0;

-- Seed the manual order with what users see today (newest first), so the first
-- drag starts from the familiar order. Buckets mirror the API scope: list,
-- habit-ness and status.
WITH ranked AS (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY list_id, (recurrence_days IS NOT NULL), status
               ORDER BY created_at DESC, id
           ) - 1 AS pos
    FROM tasks
    WHERE deleted_at IS NULL
)
UPDATE tasks
SET position = ranked.pos
FROM ranked
WHERE tasks.id = ranked.id;

CREATE INDEX idx_tasks_position ON tasks (list_id, position);
