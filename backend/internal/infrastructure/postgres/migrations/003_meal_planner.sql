-- ============================================================================
-- Meal planner: recurring weekly template, one row per item per day and slot
-- ============================================================================

CREATE TABLE meal_plan_items (
    id         UUID PRIMARY KEY,
    day        TEXT NOT NULL CHECK (day IN ('mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun')),
    meal       TEXT NOT NULL CHECK (meal IN ('breakfast', 'lunch', 'dinner', 'snack')),
    name       TEXT NOT NULL,
    amount     TEXT NOT NULL DEFAULT '',
    calories   INT  NOT NULL DEFAULT 0 CHECK (calories >= 0),
    recipe_id  UUID REFERENCES recipes(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_meal_plan_items_day_meal ON meal_plan_items (day, meal);
