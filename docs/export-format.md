# Portable exports, restore, and backups

Part 6 adds four offline export formats. Every export is written to a temporary file first, synchronized, and atomically renamed so cancellation or a write failure cannot leave a file that looks complete.

## Formats

- **Markdown** is a ZIP containing one readable Markdown file per entry plus optionally copied media. Filenames are sanitized and include a stable ID suffix, so duplicate and unusual titles cannot collide.
- **HTML** is a self-contained ZIP with an index, one offline page per entry, a local stylesheet, and optionally copied media.
- **JSON** is a versioned machine-readable document containing entries, bounded revision history, tags, relationships, attachment metadata, settings, and applied schema versions. It does not contain attachment bytes.
- **Complete archive** is the restorable ZIP format. It contains `manifest.json`, `exports/journal.json`, readable Markdown entries, a consistent SQLite snapshot at `database/solace.db`, and attachment files when requested.

Exports can include the entire active journal, entries overlapping an inclusive date range, or selected stable entry IDs. Trashed entries are excluded from normal exports. Attachments may be omitted when a smaller copy is needed.

## Archive validation and restore

`manifest.json` records the archive/export versions, creation time, counts, and a SHA-256 checksum and byte length for every payload file. Inspection and restore reject:

- unsupported or newer formats;
- absolute paths, `..` traversal, duplicate ZIP paths, and oversized expanded archives;
- missing payloads, checksum/size mismatches, invalid relationships, and unsafe attachment paths;
- partial archives with missing attachment files.

Validation completes before Solace changes the journal. Restoring into a non-empty journal requires an explicit replacement choice. Solace first creates a complete recovery archive in its private `backups/` directory, stages media separately, replaces journal data in one SQLite transaction, and rolls the media change back if the database commit fails. Entry and attachment IDs and journal dates are preserved.

Export format version 2 adds bounded entry revisions. Restore remains backward-compatible with version 1 archives, which simply contain no revision history.

## Backups

Backups use the same complete archive format and may run manually, daily, or weekly. Scheduled work is checked when Solace starts; it is not a background daemon while the application is closed. The destination, attachment inclusion, and retention count (1–100) are stored in SQLite. Expired `solace-backup-*.zip` files are pruned only from the selected backup directory.

All operations are local. This feature does not upload journal data or require an account, network connection, LLM, or RAG service.
