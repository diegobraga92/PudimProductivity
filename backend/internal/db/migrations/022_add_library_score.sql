-- Library: optional score + source (e.g. Metacritic for games, IMDb for films).
-- Both columns are nullable so existing rows and the CSV import keep working.
ALTER TABLE library_items
    ADD COLUMN IF NOT EXISTS score NUMERIC NULL,
    ADD COLUMN IF NOT EXISTS score_source TEXT NULL;

-- One global bound covers IMDb (0-10), Metacritic (0-100) and RT (0-100%).
-- Idempotent so re-running after a partial failure cannot double-add the constraint.
ALTER TABLE library_items
    DROP CONSTRAINT IF EXISTS library_items_score_range;
ALTER TABLE library_items
    ADD CONSTRAINT library_items_score_range
        CHECK (score IS NULL OR (score >= 0 AND score <= 100));

-- Seed the feature flag gating the score-lookup feature (off by default; the
-- flag service already treats unknown flags as disabled).
INSERT INTO feature_flags (id, name, description, enabled)
VALUES (gen_random_uuid(), 'library.score_lookup_enabled', 'Library score lookup feature', false)
ON CONFLICT (name) DO NOTHING;
