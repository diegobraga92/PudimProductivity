-- 001_init.sql — consolidated baseline schema for fresh installs.
--
-- This single migration replaces the original 001..026 series. Since the
-- database is always created from scratch, the schema is declared in its final
-- state: every historical intermediate step (ALTER TABLE additions, data
-- backfills, tables created then dropped, idempotency workarounds) is folded
-- directly into the CREATE statements below.
--
-- Folded-in sources (original migration numbers):
--   001, 003, 006, 010, 011     tasks
--   002, 022                    feature_flags + seeds
--   004, 013, 024               task_completions (+ indexes, partial unique)
--   005, 017, 019               task_lists / task_list_shares (+ soft delete)
--   007                         users + seeds
--   008                         audit_log
--   009                         planner_entries
--   012                         notifications
--   014                         recipes (+ tags / ingredients / steps)
--   015, 020                    library_items (books never created)
--   018                         pomodoro_sessions / insight_reports
--   021                         no-op for fresh installs (meal plans never created)
--   022, 023, 025               library_items score / subtype columns
--   026                         score_providers / score_provider_config

-- ============================================================================
-- Core: feature flags & users
-- ============================================================================

CREATE TABLE feature_flags (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    enabled     BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed default feature flags.
INSERT INTO feature_flags (id, name, description, enabled)
VALUES
    (gen_random_uuid(), 'tasks', 'Task CRUD feature', true),
    (gen_random_uuid(), 'habits', 'Habit tracking feature', false),
    (gen_random_uuid(), 'focus_timer', 'Focus timer feature', false),
    (gen_random_uuid(), 'book_tracking', 'Book tracking feature', false),
    (gen_random_uuid(), 'collaboration', 'Collaboration feature', false),
    (gen_random_uuid(), 'library.score_lookup_enabled', 'Library score lookup feature', false)
ON CONFLICT (name) DO NOTHING;

CREATE TABLE users (
    id          UUID PRIMARY KEY,
    email       TEXT NOT NULL UNIQUE,
    role        TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed a default admin user (replace email in production).
INSERT INTO users (id, email, role)
VALUES (gen_random_uuid(), 'admin@pudimproductivity.com', 'admin')
ON CONFLICT (email) DO NOTHING;

-- Seed a default regular user for local development.
INSERT INTO users (id, email, role)
VALUES (gen_random_uuid(), 'user@pudimproductivity.com', 'user')
ON CONFLICT (email) DO NOTHING;

-- ============================================================================
-- Tasks: lists, tasks, completions, sharing (offline-first soft delete)
-- ============================================================================

CREATE TABLE task_lists (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    owner_id    TEXT NOT NULL DEFAULT 'dev-user',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_task_lists_owner_id ON task_lists (owner_id);
CREATE INDEX idx_task_lists_updated_at ON task_lists (updated_at);

CREATE TABLE tasks (
    id              UUID PRIMARY KEY,
    title           TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'todo'
                        CHECK (status IN ('todo', 'done')),
    recurrence_days TEXT[],
    list_id         UUID REFERENCES task_lists(id) ON DELETE SET NULL,
    start_time      TIME,
    end_time        TIME,
    color           TEXT,
    scheduled_date  DATE,
    alarm_minutes   INTEGER,
    updated_by      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_tasks_status ON tasks (status);
CREATE INDEX idx_tasks_list_id ON tasks (list_id);
-- Habit list query: list_id IS NULL AND recurrence_days IS NOT NULL, ORDER BY created_at DESC.
CREATE INDEX idx_tasks_habits ON tasks (created_at DESC) WHERE recurrence_days IS NOT NULL;
CREATE INDEX idx_tasks_updated_at ON tasks (updated_at);

CREATE TABLE task_completions (
    id              UUID PRIMARY KEY,
    task_id         UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    completed_date  DATE NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_task_completions_task_date ON task_completions (task_id, completed_date);
-- Batch completion queries (habit screens / heatmap week range).
CREATE INDEX idx_task_completions_date ON task_completions (completed_date);
CREATE INDEX idx_task_completions_created_at ON task_completions (created_at);
-- Uniqueness applies to active rows only; soft-deleted tombstones don't block re-completion.
CREATE UNIQUE INDEX idx_task_completions_active_task_date
    ON task_completions (task_id, completed_date)
    WHERE deleted_at IS NULL;

CREATE TABLE task_list_shares (
    list_id     UUID NOT NULL REFERENCES task_lists(id) ON DELETE CASCADE,
    shared_with TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'viewer' CHECK (role IN ('editor', 'viewer')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    PRIMARY KEY (list_id, shared_with)
);

CREATE INDEX idx_task_list_shares_shared_with ON task_list_shares (shared_with);

-- ============================================================================
-- Planner, audit log & notifications
-- ============================================================================

CREATE TABLE planner_entries (
    id         UUID PRIMARY KEY,
    title      TEXT NOT NULL,
    days       TEXT[] NOT NULL,
    start_time TIME NOT NULL,
    end_time   TIME NOT NULL,
    color      TEXT NOT NULL DEFAULT '#3B82F6',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE audit_log (
    id          UUID PRIMARY KEY,
    actor_id    TEXT NOT NULL,
    action      TEXT NOT NULL,
    resource    TEXT NOT NULL,
    resource_id TEXT,
    old_values  JSONB,
    new_values  JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_log_actor ON audit_log (actor_id, created_at);
CREATE INDEX idx_audit_log_resource ON audit_log (resource, resource_id);
CREATE INDEX idx_audit_log_action ON audit_log (action, created_at);

CREATE TABLE notifications (
    id         UUID PRIMARY KEY,
    event_id   TEXT NOT NULL,
    channel    TEXT NOT NULL CHECK (channel IN ('email', 'push')),
    event_type TEXT NOT NULL,
    task_id    TEXT,
    sent_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (event_id, channel)
);

CREATE INDEX idx_notifications_event ON notifications (event_id);

-- ============================================================================
-- Recipes module
-- ============================================================================

CREATE TABLE recipes (
    id                 UUID PRIMARY KEY,
    title              TEXT NOT NULL,
    description        TEXT NOT NULL DEFAULT '',
    difficulty         TEXT NOT NULL DEFAULT 'easy'
        CHECK (difficulty IN ('easy', 'medium', 'hard')),
    prep_time_minutes  INT  NOT NULL DEFAULT 0 CHECK (prep_time_minutes >= 0),
    cook_time_minutes  INT  NOT NULL DEFAULT 0 CHECK (cook_time_minutes >= 0),
    servings           INT  NOT NULL DEFAULT 1 CHECK (servings >= 1),
    image_url          TEXT,
    source_url         TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE recipe_tags (
    recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    tag       TEXT NOT NULL,
    PRIMARY KEY (recipe_id, tag)
);

CREATE INDEX idx_recipe_tags_tag ON recipe_tags (tag);

CREATE TABLE recipe_ingredients (
    id         UUID PRIMARY KEY,
    recipe_id  UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    quantity   TEXT NOT NULL DEFAULT '',
    unit       TEXT NOT NULL DEFAULT '',
    sort_order INT  NOT NULL DEFAULT 0
);

CREATE INDEX idx_recipe_ingredients_recipe ON recipe_ingredients (recipe_id);

CREATE TABLE recipe_steps (
    id          UUID PRIMARY KEY,
    recipe_id   UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    step_number INT  NOT NULL,
    instruction TEXT NOT NULL,
    UNIQUE (recipe_id, step_number)
);

CREATE INDEX idx_recipe_steps_recipe ON recipe_steps (recipe_id);

-- ============================================================================
-- Library: generic media tracking (replaces the book-specific table)
-- ============================================================================

CREATE TABLE library_items (
    id           UUID PRIMARY KEY,
    name         TEXT NOT NULL,
    media_type   TEXT NOT NULL DEFAULT 'book'
        CHECK (media_type IN ('movie', 'series', 'book', 'game')),
    release_year INT NULL
        CHECK (release_year IS NULL OR (release_year BETWEEN 1800 AND 2100)),
    done         BOOLEAN NOT NULL DEFAULT false,
    notes        TEXT NOT NULL DEFAULT '',
    subtype      TEXT NOT NULL DEFAULT '',
    score        NUMERIC,
    score_source TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- One global bound covers IMDb (0-10), Metacritic (0-100) and RT (0-100%).
    CONSTRAINT library_items_score_range
        CHECK (score IS NULL OR (score >= 0 AND score <= 100))
);

CREATE INDEX idx_library_items_media_type ON library_items (media_type);
CREATE INDEX idx_library_items_done ON library_items (done);

-- ============================================================================
-- Pomodoro focus history + generated reports
-- ============================================================================

CREATE TABLE pomodoro_sessions (
    id            UUID PRIMARY KEY,
    user_id       TEXT NOT NULL,
    focus_minutes INT NOT NULL,
    elapsed_s     INT NOT NULL,
    started_at    TIMESTAMPTZ NOT NULL,
    completed_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_pomodoro_sessions_user_completed
    ON pomodoro_sessions (user_id, completed_at);


-- ============================================================================
-- Score provider settings (admin-configurable at runtime)
-- ============================================================================

CREATE TABLE score_providers (
    name       TEXT PRIMARY KEY,           -- 'omdb' | 'rawg' (scoring registry names)
    api_key    TEXT NOT NULL DEFAULT '',   -- secret; '' = not configured
    base_url   TEXT NOT NULL DEFAULT '',   -- '' = use the provider default
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Singleton row mapping media type -> provider. '' or 'none' disables lookup for
-- that media type. saved_at is NULL until the user explicitly saves via the
-- admin UI; until then the service falls back to environment defaults.
CREATE TABLE score_provider_config (
    id              SMALLINT PRIMARY KEY CHECK (id = 1),
    movie_provider  TEXT NOT NULL DEFAULT '',
    series_provider TEXT NOT NULL DEFAULT '',
    game_provider   TEXT NOT NULL DEFAULT '',
    book_provider   TEXT NOT NULL DEFAULT '',
    saved_at        TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO score_providers (name) VALUES ('omdb'), ('rawg')
ON CONFLICT (name) DO NOTHING;

INSERT INTO score_provider_config (id) VALUES (1)
ON CONFLICT (id) DO NOTHING;
