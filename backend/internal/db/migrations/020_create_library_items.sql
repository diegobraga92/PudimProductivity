-- Library: generic media tracking (movies, series, books, games) with a done
-- flag and optional notes. Replaces the book-specific books table (Phase 5):
-- existing books are migrated as media_type='book' with done = (status = 'read').
CREATE TABLE IF NOT EXISTS library_items (
    id           UUID PRIMARY KEY,
    name         TEXT NOT NULL,
    media_type   TEXT NOT NULL DEFAULT 'book'
        CHECK (media_type IN ('movie', 'series', 'book', 'game')),
    release_year INT NULL CHECK (release_year IS NULL OR (release_year BETWEEN 1800 AND 2100)),
    done         BOOLEAN NOT NULL DEFAULT false,
    notes        TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Migrate existing books into library items. release_year is derived from the
-- free-text Google Books published_date (first 4-digit year found, if any).
INSERT INTO library_items (id, name, media_type, release_year, done, created_at, updated_at)
SELECT
    id,
    title,
    'book',
    (regexp_match(published_date, '\d{4}'))[1]::INT,
    status = 'read',
    created_at,
    updated_at
FROM books;

-- The book-specific table is fully replaced by library_items.
DROP TABLE books;

-- Type + done filters are the two hot queries on the list page.
CREATE INDEX IF NOT EXISTS idx_library_items_media_type ON library_items (media_type);
CREATE INDEX IF NOT EXISTS idx_library_items_done ON library_items (done);
