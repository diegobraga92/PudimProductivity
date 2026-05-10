ALTER TABLE tasks ADD COLUMN IF NOT EXISTS list_id UUID REFERENCES task_lists(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_tasks_list_id ON tasks (list_id);
