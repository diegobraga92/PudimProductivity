CREATE TABLE IF NOT EXISTS notifications (
    id         UUID PRIMARY KEY,
    event_id   TEXT NOT NULL,
    channel    TEXT NOT NULL CHECK (channel IN ('email', 'push')),
    event_type TEXT NOT NULL,
    task_id    TEXT,
    sent_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (event_id, channel)
);

CREATE INDEX IF NOT EXISTS idx_notifications_event
    ON notifications(event_id);
