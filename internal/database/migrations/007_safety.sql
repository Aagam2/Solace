CREATE TABLE entry_revisions (
    id TEXT PRIMARY KEY NOT NULL,
    entry_id TEXT NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    title TEXT,
    content_json TEXT NOT NULL,
    content_text TEXT NOT NULL DEFAULT '',
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL,
    mood TEXT,
    is_favorite INTEGER NOT NULL CHECK (is_favorite IN (0, 1)),
    is_pinned INTEGER NOT NULL CHECK (is_pinned IN (0, 1)),
    is_draft INTEGER NOT NULL CHECK (is_draft IN (0, 1)),
    location_text TEXT,
    display_order INTEGER,
    tags_json TEXT NOT NULL DEFAULT '[]',
    source_version INTEGER NOT NULL,
    saved_at TEXT NOT NULL
);

CREATE INDEX idx_entry_revisions_entry_saved ON entry_revisions(entry_id, saved_at DESC, id DESC);

CREATE TABLE privacy_settings (
    id INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
    enabled INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1)),
    password_salt TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL DEFAULT '',
    password_iterations INTEGER NOT NULL DEFAULT 0,
    lock_on_start INTEGER NOT NULL DEFAULT 0 CHECK (lock_on_start IN (0, 1)),
    inactivity_minutes INTEGER NOT NULL DEFAULT 0 CHECK (inactivity_minutes BETWEEN 0 AND 120)
);

INSERT INTO privacy_settings (id) VALUES (1) ON CONFLICT (id) DO NOTHING;
