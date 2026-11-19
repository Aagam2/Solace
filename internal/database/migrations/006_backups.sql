CREATE TABLE backup_settings (
    id INTEGER PRIMARY KEY NOT NULL CHECK (id = 1),
    schedule TEXT NOT NULL DEFAULT 'manual' CHECK (schedule IN ('manual', 'daily', 'weekly')),
    destination TEXT NOT NULL DEFAULT '',
    retention INTEGER NOT NULL DEFAULT 5 CHECK (retention BETWEEN 1 AND 100),
    include_attachments INTEGER NOT NULL DEFAULT 1 CHECK (include_attachments IN (0, 1)),
    last_backup_at TEXT NOT NULL DEFAULT ''
);

INSERT INTO backup_settings (id) VALUES (1) ON CONFLICT (id) DO NOTHING;
