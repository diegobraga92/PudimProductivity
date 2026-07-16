CREATE TABLE IF NOT EXISTS planner_entries (
    id         UUID PRIMARY KEY,
    title      TEXT NOT NULL,
    days       TEXT[] NOT NULL,
    start_time TIME NOT NULL,
    end_time   TIME NOT NULL,
    color      TEXT NOT NULL DEFAULT '#3B82F6',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);