CREATE TABLE entries (
    id TEXT PRIMARY KEY NOT NULL,
    title TEXT,
    content_json TEXT NOT NULL,
    content_text TEXT NOT NULL DEFAULT '',
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    mood TEXT,
    is_favorite INTEGER NOT NULL DEFAULT 0 CHECK (is_favorite IN (0, 1)),
    is_draft INTEGER NOT NULL DEFAULT 1 CHECK (is_draft IN (0, 1)),
    location_text TEXT,
    display_order INTEGER,
    deleted_at TEXT,
    version INTEGER NOT NULL DEFAULT 1,
    CHECK (start_date <= end_date)
);

CREATE INDEX idx_entries_journal_dates ON entries(start_date, end_date);
CREATE INDEX idx_entries_updated_at ON entries(updated_at DESC);
CREATE INDEX idx_entries_deleted_at ON entries(deleted_at);
CREATE INDEX idx_entries_favorite ON entries(is_favorite, start_date DESC);

CREATE TABLE tags (
    id TEXT PRIMARY KEY NOT NULL,
    name TEXT NOT NULL COLLATE NOCASE UNIQUE,
    created_at TEXT NOT NULL
);

CREATE TABLE entry_tags (
    entry_id TEXT NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (entry_id, tag_id)
);

CREATE INDEX idx_entry_tags_tag_id ON entry_tags(tag_id, entry_id);
