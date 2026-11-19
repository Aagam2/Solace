# Testing

## Frontend

Vitest and Testing Library cover shell rendering, navigation, theme application, local preference persistence, entry creation, Tiptap JSON/plain-text synchronization, rapid edits, debounced/manual-save race prevention, pasted-image and explicit drag-and-drop import, multi-day calendar counts, timeline display, partial-word search, direct result opening, tag rename propagation, compact data/safety tab navigation, integrity-check states, restoring an earlier saved entry revision, global shortcuts, writing-preference persistence, live word/character counts, and exiting focus mode. TypeScript strict checks, ESLint, and a production Vite build are also expected to pass.

## Backend

`internal/platform` tests directory creation, private permissions, and an unavailable root. `internal/database` tests first initialization, migration application, and reopening an existing database.

The database tests open temporary SQLite databases and cover initial migration, reopening, settings defaults, persistence, validation, and private file modes. Diary tests cover single and multi-day entries, multiple/overlapping entries, inclusive range lookup, tags, draft/favorite updates, invalid ranges, stale writes, trash/restore/permanent deletion, duplication, retroactive dates, month/year boundaries, and persistence after a database restart. Permission to run these local tests was granted on 2026-11-17.

Attachment tests cover images, multiple/duplicate filenames, video, PDF, unusual Unicode filenames, safe generated paths, checksums, captions, reordering, removal, generated thumbnails, restart persistence, secure media serving, missing files, interrupted reads, large-file rejection, and owned-media cleanup when an entry is permanently deleted.

Navigation tests cover calendar expansion across month/year boundaries, multiple entries on one date, the same multi-day entry ID on multiple timeline dates, historical date jumps, empty years, pinned summaries, prior-year “On This Day” results, and multi-day dashboard matches. A performance fixture creates 10,000 entries plus thousands of tag and attachment records and verifies the bounded timeline page completes within five seconds.

Search tests cover create/edit/trash/permanent-delete lifecycle, partial and multiple words, no results, special characters, inclusive overlapping date ranges, tag and combined filters, draft/published state, favorites, moods, image/video metadata, attachment captions, tag CRUD, index rebuild, restart persistence, and a 5,000-entry stress fixture.

Portable-data tests cover Markdown, HTML, JSON, and complete archive exports; empty journals; full metadata and media; duplicate/special titles; inclusive date and selected-entry scopes; manifests; checksums; unsafe ZIP paths; invalid archives without database mutation; backup retention; explicit replacement; and recovery archives. The critical round-trip fixture exports and restores 100 varied entries into a fresh temporary database, preserving stable IDs, dates, tags, attachment metadata, checksums, and media bytes.

Part 7 tests cover private pre-migration snapshots, transaction rollback for invalid migration SQL, integrity/foreign-key damage detection, unwritable database locations, PBKDF2 password configuration, wrong-password rejection, lock/unlock/disable behavior, password and inactivity validation, a 20-version revision bound, revision recovery, and revision-aware archive round trips. Existing restart, missing-media, trash, interrupted import/export, rapid-save, and checksum tests remain part of the full suite.

Logging tests force multiple small rotations, assert that only the configured backup count remains, verify the newest message stays in the active log, and confirm every retained file is private. Frontend checks ensure handled failures pass through the shared friendly-message boundary; the native reporting method records technical details without requiring raw errors in the interface.

## Native verification

`wails3 build` produces `bin/solace`. Native production and development builds are exercised against isolated temporary data roots. The Part 8 development build was created at `/tmp/solace-part8-dev`; the final production binary was built and launched twice at `/tmp/solace-part8-native`, serving the final editor/settings assets without network access. `bin/solace.deb` was generated, inspected with `dpkg-deb`, and contains the binary as `0755` plus desktop/icon files as `0644`. SQLite integrity returned `ok`; all seven migrations, the FTS5 table, nine synchronization triggers, and insert/update/delete index maintenance were verified. Foreign-key checks were clean, owned directories were `0700`, and database/log/storage files were `0600`.

The 2026-11-19 branding/reliability pass repeated browser rendering at 1240 × 780 and 800 × 600, inspected the icon at 1024 px and 32 px, rebuilt and launched the production binary with an isolated data root, verified the rotated JSON log is `0600`, rebuilt `bin/solace.deb`, and visually inspected its installed 128 px icon and generated desktop entry.

## Manual Part 8 checklist

- In the packaged native application, create and save an entry with `Cmd/Ctrl+N` and `Cmd/Ctrl+S`, restart, and confirm it reopens from the same local profile.
- Exercise global search, Today, sidebar, reading mode, Settings, and privacy-lock shortcuts. Confirm browser-reserved behavior does not occur while Solace handles them.
- Import a photo by picker, drop, and paste; move the original; restart and confirm the private copy still opens.
- Export a complete archive with attachments from the package, inspect it, and restore it into a separate temporary profile.
- Resize through 1240 × 780, 800 × 600, and a wide window in light and dark themes. Confirm no page-level horizontal scrollbar appears.
- Change text size, prose width, line spacing, autosave pace, date format, and export defaults; restart and confirm persistence.
- Disconnect networking before launch and repeat writing, search, attachment opening, and export.
- On Windows and macOS, repeat native window, dialog, font rendering, shortcut-modifier, signing, and package-install checks before publishing those targets.

## Manual Part 7 checklist

- Enable the privacy lock with a password of at least eight characters, lock manually, reject a wrong password, and unlock with the correct password.
- Enable startup locking, restart Solace, and confirm the journal is covered before interaction. Repeat with each inactivity option and `Ctrl+Shift+L`.
- Confirm the lock screen explicitly says it is not encryption; inspect SQLite only if you intend to verify that no plaintext password is present.
- Edit and save one entry several times, open History, restore an older version, and confirm the previously current version remains recoverable.
- Run Database health and confirm the result is visible and read-only.
- Start a copied older profile with a pending migration and confirm a private `pre-migration-*.db` snapshot is created before schema changes.
- Repeat existing restart, missing-media, trash/restore, full-disk/write-error, interrupted export/import, and rapid-save checks.
- Disconnect networking and exercise writing, search, attachments, export, restore inspection, revision recovery, and lock/unlock.

## Manual Part 6 checklist

- Export Markdown, HTML, JSON, and a complete archive from a populated journal; inspect each result offline.
- Repeat with an empty journal, attachments excluded, an inclusive date range, and selected entries.
- Cancel a large export and confirm no partial destination file remains.
- Inspect a complete archive, then restore it into an empty temporary profile and compare entry, tag, date, and attachment counts.
- Attempt restore into a non-empty profile without replacement and confirm it is refused; enable replacement and confirm a recovery archive is reported.
- Tamper with a copy of an archive and confirm checksum validation refuses it without changing the journal.
- Configure manual, daily, and weekly settings, choose a destination, set a small retention count, and confirm old Solace backup archives are pruned.
- Drop a photo, video, PDF, and generic file onto Details & media and confirm each imports; restart after moving the originals and confirm the stored copies still open.

## Manual Part 5 checklist

- Create the four prompt entries containing bookstore, interview, new-book, and technical-interview text.
- Search for `interview` and confirm exactly the two relevant entries appear with highlighted context.
- Type a partial word such as `interv` and confirm the same matches.
- Tag the book entries `Books`, select that tag filter, and confirm unrelated entries disappear.
- Combine text, tag, mood, favorite, image/video, draft, and overlapping date filters.
- Rename a tag and confirm existing assignments and search results update; delete it and confirm entries remain.
- Edit, trash, restore, and permanently delete a searchable entry and confirm results update at each step.
- Rebuild the search index, restart Solace offline, and confirm search results remain available.

## Manual Part 4 checklist

- Create two entries on one day and confirm the calendar count and date panel show both.
- Create a range crossing a month boundary and another crossing December 31; confirm every covered date appears in calendar/timeline views with one stable entry.
- Jump to a historical month and year, use previous/next period and previous/next entry controls, and open an entry into the editor.
- Select a year with no entries and confirm the empty state is clear.
- Pin and favorite an entry, reopen Solace, and confirm the states persist and appear in summary views.
- Create a prior-year entry covering today’s month/day and confirm it appears under “On This Day”.
- Enter reading mode, confirm editing controls are hidden/disabled, and exit back to the normal editor.

## Manual Part 3 checklist

- Start the Vite frontend and confirm the Today page renders.
- Navigate through every sidebar item.
- Toggle and collapse the sidebar.
- Select light, system, and dark themes; reload and verify persistence.
- Resize the window down to an 800px desktop width.
- Create multiple independent entries on November 15 and 16, including ranges ending November 16 and November 20.
- Edit one ranged entry and confirm the others do not change.
- Confirm the visible save state progresses through unsaved/saving/saved.
- Favorite, duplicate, draft/publish, trash, restore, and permanently delete test entries.
- Close and reopen Solace and confirm saved entries and settings remain.
- Create “My Weekend” and exercise every compact rich-text toolbar action plus undo/redo and keyboard shortcuts.
- Attach five images, two videos, and one PDF through the picker; paste an additional image and drop another file onto the attachment area.
- Insert images between formatted paragraphs, add captions, reorder the gallery, and enlarge an image.
- Play, pause, and seek supported video/audio. Confirm a deliberately unsupported codec shows the external-open fallback.
- Close Solace, move the original source files elsewhere, reopen it, and confirm every imported copy remains available.
- Remove one attachment and confirm no other attachment changes; trash/restore the entry, then permanently delete it and confirm its owned media is cleaned up.
