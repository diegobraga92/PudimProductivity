-- Library: the Go domain models score_source as a non-nullable string (empty
-- string = "no score source"), but migration 022 declared the column TEXT NULL,
-- leaving pre-existing rows as NULL. The list query scans it into a plain
-- *string and fails on NULL rows ("cannot scan NULL into *string"), which broke
-- the whole Library page (GET /api/v1/library → 500). Backfill the NULLs and
-- enforce NOT NULL DEFAULT '' so the column matches the domain model — the same
-- contract as notes and subtype.
UPDATE library_items SET score_source = '' WHERE score_source IS NULL;

ALTER TABLE library_items
    ALTER COLUMN score_source SET DEFAULT '';

ALTER TABLE library_items
    ALTER COLUMN score_source SET NOT NULL;
