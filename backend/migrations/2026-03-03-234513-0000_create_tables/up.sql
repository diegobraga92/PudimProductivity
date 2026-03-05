CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE lists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id UUID NULL REFERENCES lists(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    list_type SMALLINT NOT NULL CHECK (list_type IN (0,1,2))
);

CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    list_id UUID NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    done BOOLEAN NOT NULL DEFAULT FALSE,
    repeat_on SMALLINT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ NULL
);

CREATE TABLE task_completions (
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    date TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (task_id, date)
);

CREATE INDEX idx_tasks_list_id ON tasks(list_id);
CREATE INDEX idx_lists_parent_id ON lists(parent_id);
CREATE INDEX idx_task_completions_task_id ON task_completions(task_id);
CREATE INDEX idx_tasks_not_deleted ON tasks(list_id) WHERE deleted_at IS NULL;

-- Insert default lists
INSERT INTO lists
    (id, parent_id, name, list_type)
VALUES
    ('10000000-0000-0000-0000-000000000001', NULL, 'Daily', 1),
    ('10000000-0000-0000-0000-000000000002', NULL, 'To Do', 0);