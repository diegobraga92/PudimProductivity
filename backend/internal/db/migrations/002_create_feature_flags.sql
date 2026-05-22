CREATE TABLE IF NOT EXISTS feature_flags (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    enabled     BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed default feature flags
INSERT INTO feature_flags (id, name, description, enabled)
VALUES
    (gen_random_uuid(), 'tasks', 'Task CRUD feature', true),
    (gen_random_uuid(), 'habits', 'Habit tracking feature', false),
    (gen_random_uuid(), 'focus_timer', 'Focus timer feature', false),
    (gen_random_uuid(), 'meal_planning', 'Meal planning feature', false),
    (gen_random_uuid(), 'book_tracking', 'Book tracking feature', false),
    (gen_random_uuid(), 'collaboration', 'Collaboration feature', false),
    (gen_random_uuid(), 'ai_insights', 'AI-powered insights feature', false)
ON CONFLICT (name) DO NOTHING;
