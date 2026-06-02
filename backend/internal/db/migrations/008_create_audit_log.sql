CREATE TABLE IF NOT EXISTS audit_log (
    id          UUID PRIMARY KEY,
    actor_id    TEXT NOT NULL,
    action      TEXT NOT NULL,
    resource    TEXT NOT NULL,
    resource_id TEXT,
    old_values  JSONB,
    new_values  JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_actor
    ON audit_log(actor_id, created_at);

CREATE INDEX IF NOT EXISTS idx_audit_log_resource
    ON audit_log(resource, resource_id);

CREATE INDEX IF NOT EXISTS idx_audit_log_action
    ON audit_log(action, created_at);