-- Library: optional subtype for items — e.g. "Console" for games, or "Genre"
-- for movies/series/books. Free text; empty string means not set.
ALTER TABLE library_items ADD COLUMN IF NOT EXISTS subtype TEXT NOT NULL DEFAULT '';
