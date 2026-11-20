# Database foundation

Solace uses SQLite through the pure-Go `modernc.org/sqlite` driver. The database path is `<data-root>/solace.db`; large media will not be stored as database blobs.

## Connection policy

- Foreign keys enabled for every connection.
- WAL journal mode for desktop read/write concurrency.
- `synchronous=NORMAL`, balancing WAL durability and interactive performance.
- Five-second busy timeout.
- One open connection, making per-connection pragmas deterministic for the desktop process.

## Migrations

SQL migrations are embedded from `internal/database/migrations` and applied by filename order inside transactions. Applied filenames are recorded in `schema_migrations`, making startup idempotent.

Migration `001_foundation.sql` creates the settings table and two defaults.

Migration `002_entries.sql` adds:

- `entries`, using stable UUID text IDs and separate `YYYY-MM-DD` journal dates.
- Inclusive `start_date`/`end_date` ranges with a database constraint enforcing `start_date <= end_date`.
- Plain content plus a Tiptap-compatible JSON document for forward migration to the Part 3 rich-text editor.
- Optional title, mood, location, display order, favorite and draft flags, timestamps, soft-deletion time, and an optimistic-lock version.
- Case-insensitive `tags` and the many-to-many `entry_tags` join table.
- Date, update-time, trash, favorite, and tag lookup indexes.

Soft deletion keeps entries recoverable until the explicitly confirmed permanent-delete operation.

Migration `003_attachments.sql` adds:

- Stable attachment IDs and an owning entry foreign key with cascading metadata cleanup.
- Original display names separated from generated UUID storage names.
- Portable paths relative to the Solace data root, MIME type, byte size, timestamps, caption, display order, SHA-256 checksum, and extensible metadata JSON.
- An entry/order index for deterministic gallery loading.

Attachment bytes are never stored in SQLite. Files are staged privately, copied with a size limit and checksum, synchronized, atomically moved into their media category, and only then recorded in the database. Supported images also receive deterministic JPEG thumbnails beneath `thumbnails/`. Permanent entry deletion removes both attachment metadata and owned files.

Migration `004_navigation.sql` adds:

- `entries.is_pinned`, a persisted boolean used by quick navigation and chronological summaries.
- `idx_entries_pinned`, supporting pinned-first access without changing entry identity or date semantics.

Calendar and timeline views do not materialize duplicate rows. They query the existing inclusive `start_date`/`end_date` ranges and expand occurrences only inside a requested month or year. Timeline requests are capped at 500 results per page; the UI normally requests 100.

Migration `005_search.sql` adds:

- The `entry_search` FTS5 virtual table, with entry ID, title, plain text, tags, and attachment captions.
- Unicode case-insensitive tokenization, diacritic normalization, and 2/3/4-character prefix indexes for practical partial-word queries.
- Initial backfill for every existing entry.
- Triggers for entry create/edit/delete, tag assignment/removal/rename, and attachment caption/create/delete changes.

Search filters use inclusive range overlap (`entry.start_date <= filter.end AND entry.end_date >= filter.start`). Text and filter values are bound parameters. Rebuilding the index is a transactional service operation that regenerates search documents from canonical entry, tag, and attachment tables.

Migration `006_backups.sql` adds the singleton `backup_settings` row with:

- manual, daily, or weekly scheduling;
- an absolute destination selected by the user, falling back to Solace's private `backups/` directory;
- retention from 1 to 100 archives;
- attachment inclusion and the last successful backup timestamp.

Complete archives contain a consistent SQLite snapshot created with SQLite's `VACUUM INTO`. Restore uses the versioned logical JSON document and a transaction rather than blindly replacing the live database file, allowing full validation while preserving the migration-managed structure.

Migration `007_safety.sql` adds:

- `entry_revisions`, a cascading per-entry recovery table containing the prior saved rich-text document, plain text, dates, metadata, tags, source version, and save time. Solace retains at most five rows per entry. A database trigger enforces the bound for imports as well as ordinary saves.
- `privacy_settings`, a singleton containing only the application-lock state, random salt, password-derived hash, work factor, startup preference, and inactivity timeout. No plaintext password is stored.

Before applying a pending migration to an existing database, startup writes a private SQLite snapshot beneath `backups/`. Migrations remain individually transactional. After migrations, startup runs `PRAGMA quick_check` and `pragma_foreign_key_check`; damage is reported without automatic deletion or repair.

## Settings

The persisted settings are theme (`system`, `light`, or `dark`) and sidebar state. Settings and entry mutations use parameterized statements; multi-table entry/tag writes are transactional.

Local database-test permission was granted on 2026-11-17. Automated tests have verified all eight migrations, pre-migration snapshots, failed-migration rollback, entry/revision and attachment persistence, calendar/timeline/search queries, FTS rebuild and restart behavior, migration idempotency, backup/privacy settings, integrity failure detection, foreign keys, and private file permissions.
