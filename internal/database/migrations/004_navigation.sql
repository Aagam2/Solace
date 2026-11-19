ALTER TABLE entries ADD COLUMN is_pinned INTEGER NOT NULL DEFAULT 0 CHECK (is_pinned IN (0, 1));

CREATE INDEX idx_entries_pinned ON entries(is_pinned, start_date DESC);
