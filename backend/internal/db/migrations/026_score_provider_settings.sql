-- Score provider settings: moves the library rating-provider configuration
-- (which provider serves each media type, plus per-provider API keys) from
-- environment variables into the database so it can be managed from the admin
-- UI at runtime instead of requiring a restart.
--
-- Secrets: score_providers.api_key is stored server-side and is deliberately
-- excluded from the backup module (see backend/internal/backup/service.go) and
-- never returned by the admin API (only a masked api_key_set boolean).

CREATE TABLE IF NOT EXISTS score_providers (
    name       TEXT PRIMARY KEY,              -- 'omdb' | 'rawg' (scoring registry names)
    api_key    TEXT NOT NULL DEFAULT '',      -- secret; '' = not configured
    base_url   TEXT NOT NULL DEFAULT '',      -- '' = use the provider default
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Singleton row mapping media type -> provider. '' or 'none' disables lookup
-- for that media type. saved_at is NULL until the user explicitly saves via the
-- admin UI; until then the service falls back to environment defaults so
-- existing .env-based deployments keep working.
CREATE TABLE IF NOT EXISTS score_provider_config (
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
