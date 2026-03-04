-- Create lists table
CREATE TABLE lists (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    config TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Create tasks table
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    list_id TEXT NOT NULL REFERENCES lists(id),
    title TEXT NOT NULL,
    completed BOOLEAN NOT NULL,
    order_index INTEGER NOT NULL,
    due_date TIMESTAMPTZ,
    recurrence TEXT,
    streak_count INTEGER,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- Insert default lists
INSERT INTO lists
    (id, user_id, name, type, config, created_at, updated_at)
VALUES
    ('10000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'Daily', 'daily', '{"showCompleted":false,"daysOfWeek":[0,1,2,3,4,5,6],"analyticsEnabled":true}', NOW(), NOW()),
    ('10000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'To Do', 'todo', '{"showCompleted":false,"autoArchive":false}', NOW(), NOW());

-- TODO