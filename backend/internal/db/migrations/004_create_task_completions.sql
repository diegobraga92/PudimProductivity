CREATE TABLE IF NOT EXISTS task_completions (
    id              UUID PRIMARY KEY,
    task_id         UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    completed_date  DATE NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(task_id, completed_date)
);

CREATE INDEX IF NOT EXISTS idx_task_completions_task_date ON task_completions (task_id, completed_date);
