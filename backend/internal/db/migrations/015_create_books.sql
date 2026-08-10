-- Phase 5: Book tracking. Books are added by ISBN (looked up via the Google
-- Books API) or manually, then tracked through want_to_read / reading / read.
CREATE TABLE IF NOT EXISTS books (
    id             UUID PRIMARY KEY,
    isbn           TEXT NOT NULL UNIQUE,
    title          TEXT NOT NULL,
    authors        TEXT[] NOT NULL DEFAULT '{}',
    publisher      TEXT NOT NULL DEFAULT '',
    published_date TEXT NOT NULL DEFAULT '',
    description    TEXT NOT NULL DEFAULT '',
    page_count     INT  NOT NULL DEFAULT 0 CHECK (page_count >= 0),
    thumbnail_url  TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'want_to_read'
        CHECK (status IN ('want_to_read', 'reading', 'read')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ISBN lookup + status filter are the two hot queries.
CREATE INDEX IF NOT EXISTS idx_books_isbn ON books (isbn);
CREATE INDEX IF NOT EXISTS idx_books_status ON books (status);
