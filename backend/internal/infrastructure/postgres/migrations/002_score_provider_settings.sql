-- 002_score_provider_settings.sql — IGDB support.

ALTER TABLE score_providers
    ADD COLUMN settings JSONB NOT NULL DEFAULT '{}'::jsonb;

-- Register the IGDB provider (games -> Metacritic/OpenCritic aggregated rating).
INSERT INTO score_providers (name) VALUES ('igdb')
ON CONFLICT (name) DO NOTHING;
