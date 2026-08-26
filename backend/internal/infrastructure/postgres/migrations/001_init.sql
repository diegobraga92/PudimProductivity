-- 001_init.sql — consolidated baseline schema for fresh installs.

-- ============================================================================
-- Core: feature flags & users
-- ============================================================================

CREATE TABLE feature_flags (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
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

-- Enforce case-insensitive email uniqueness (the UNIQUE above is case-sensitive).
CREATE UNIQUE INDEX idx_users_email_lower ON users (LOWER(email));

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
-- Incremental sync deletes: WHERE deleted_at IS NOT NULL AND deleted_at > $1.
CREATE INDEX idx_task_lists_deleted ON task_lists (deleted_at) WHERE deleted_at IS NOT NULL;

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
    deleted_at      TIMESTAMPTZ,
    -- Mirrors the domain validation (task package): a time window is either
    -- absent or start < end, and an alarm can't be negative. NULLs pass.
    CONSTRAINT tasks_time_window
        CHECK (start_time IS NULL OR end_time IS NULL OR end_time > start_time),
    CONSTRAINT tasks_alarm_non_negative
        CHECK (alarm_minutes IS NULL OR alarm_minutes >= 0)
);

CREATE INDEX idx_tasks_list_id ON tasks (list_id);
-- Habit list query: list_id IS NULL AND recurrence_days IS NOT NULL, ORDER BY created_at DESC.
CREATE INDEX idx_tasks_habits ON tasks (created_at DESC) WHERE recurrence_days IS NOT NULL;
-- Inbox list query: list_id IS NULL AND deleted_at IS NULL, ORDER BY created_at DESC.
CREATE INDEX idx_tasks_inbox ON tasks (created_at DESC) WHERE list_id IS NULL AND deleted_at IS NULL;
CREATE INDEX idx_tasks_updated_at ON tasks (updated_at);
-- Incremental sync deletes: WHERE deleted_at IS NOT NULL AND deleted_at > $1.
CREATE INDEX idx_tasks_deleted ON tasks (deleted_at) WHERE deleted_at IS NOT NULL;

CREATE TABLE task_completions (
    id              UUID PRIMARY KEY,
    task_id         UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    completed_date  DATE NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

-- Batch completion queries (habit screens / heatmap week range).
CREATE INDEX idx_task_completions_date ON task_completions (completed_date);
CREATE INDEX idx_task_completions_created_at ON task_completions (created_at);
-- Uniqueness applies to active rows only; soft-deleted tombstones don't block re-completion.
-- Also serves active-row lookups by (task_id, completed_date), so a separate
-- non-partial (task_id, completed_date) index would be redundant.
CREATE UNIQUE INDEX idx_task_completions_active_task_date
    ON task_completions (task_id, completed_date)
    WHERE deleted_at IS NULL;
-- Incremental sync deletes: WHERE deleted_at IS NOT NULL AND deleted_at > $1.
CREATE INDEX idx_task_completions_deleted
    ON task_completions (deleted_at) WHERE deleted_at IS NOT NULL;

CREATE TABLE task_list_shares (
    list_id     UUID NOT NULL REFERENCES task_lists(id) ON DELETE CASCADE,
    shared_with TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'viewer' CHECK (role IN ('editor', 'viewer')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    PRIMARY KEY (list_id, shared_with)
);

CREATE INDEX idx_task_list_shares_shared_with ON task_list_shares (shared_with);
-- Incremental sync deletes: WHERE deleted_at IS NOT NULL AND deleted_at > $1.
CREATE INDEX idx_task_list_shares_deleted
    ON task_list_shares (deleted_at) WHERE deleted_at IS NOT NULL;

-- ============================================================================
-- Planner & audit log
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
    -- UNIQUE already provides the leading-column index on recipe_id, so no
    -- separate idx_recipe_steps_recipe is needed.
    UNIQUE (recipe_id, step_number)
);

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
    completed_at  TIMESTAMPTZ NOT NULL,
    CONSTRAINT pomodoro_sessions_durations_non_negative
        CHECK (focus_minutes >= 0 AND elapsed_s >= 0),
    CONSTRAINT pomodoro_sessions_completed_after_started
        CHECK (completed_at >= started_at)
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

-- Singleton row mapping media type -> provider. '' or 'none' disables lookup
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
