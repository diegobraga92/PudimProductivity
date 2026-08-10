-- Phase 9a: AI coach — focus history + generated insight reports.
--
-- 1. pomodoro_sessions persists completed focus sessions (the pomodoro module
--    is otherwise in-memory); the insights module writes them by consuming
--    pomodoro.session.completed events.
-- 2. insight_reports caches weekly generated reports per user (unique by week).

CREATE TABLE IF NOT EXISTS pomodoro_sessions (
    id            UUID PRIMARY KEY,
    user_id       TEXT NOT NULL,
    focus_minutes INT NOT NULL,
    elapsed_s     INT NOT NULL,
    started_at    TIMESTAMPTZ NOT NULL,
    completed_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_pomodoro_sessions_user_completed
    ON pomodoro_sessions (user_id, completed_at);

CREATE TABLE IF NOT EXISTS insight_reports (
    id           UUID PRIMARY KEY,
    user_id      TEXT NOT NULL,
    week_start   DATE NOT NULL,
    report_json  JSONB NOT NULL,
    report_text  TEXT NOT NULL,
    llm_summary  TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, week_start)
);
