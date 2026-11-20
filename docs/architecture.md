# Architecture

## Boundaries

Solace is organized around a small Go application core and a React desktop UI:

- `internal/platform` resolves and creates private, platform-appropriate data directories.
- `internal/database` owns SQLite connection policy and ordered embedded migrations.
- `internal/settings` owns persistent user preferences.
- `internal/diary` owns entry validation, inclusive range queries, calendar aggregation, paginated timeline expansion, dashboard summaries, tag normalization, optimistic updates, trash, restore, deletion, and duplication.
- `internal/attachments` owns staged imports, generated storage names, MIME detection, checksums, thumbnails, ordering, captions, cleanup, secure media serving, and external-open fallback.
- `internal/portable` owns Markdown/HTML/JSON/archive export, archive manifests and validation, transactional restore, SQLite snapshots, scheduled backup settings, and retention.
- `internal/privacy` owns password derivation and verification plus the in-memory application-lock state. It does not claim or implement file encryption.
- `internal/app` is the stable service boundary intended for Wails-generated bindings.
- `frontend/src/lib/bridge.ts` isolates React from its host. Browser preview uses a local adapter; the native adapter uses generated Wails v3 bindings.
- `frontend/src/features/entries` contains the host-neutral diary bridge, entry types, and journal workspace.
- `frontend/src/features/navigation` contains host-neutral calendar/timeline/dashboard adapters and the Part 4 browsing UI.
- `frontend/src/features/search` contains the native/browser search adapter, filtered result UI, and tag manager.
- `frontend/src/features/data` contains the tabbed export, verified restore, and local backup interface plus its native-dialog bridge.
- `frontend/src/features/privacy` contains the privacy/safety settings panel, inactivity/manual lock controller, and modal unlock surface.
- `frontend/src/features/settings` contains the compact application-settings workspace and the read-only runtime/storage summary.

The core does not expose a network service. SQLite is the source of truth for journal and attachment metadata; media lives as private files beneath the Solace data root. A same-origin Wails asset-handler route resolves media by attachment ID, validates the database path, and never accepts a filesystem path from the URL.

## Startup sequence

The native startup sequence is designed to be:

1. Resolve the OS data directory.
2. Create private Solace directories.
3. Open SQLite with foreign keys, WAL, a busy timeout, and normal synchronous mode.
4. Apply embedded migrations transactionally.
5. Construct the application/settings, diary, attachment, portable-data, and privacy services, then check whether a scheduled backup is due.
6. Register all services with Wails v3, install the secure media handler, and open the main window with native file-drop support.

All six steps are implemented in `main.go`. Wails embeds `frontend/dist`, exposes the application service through generated TypeScript interfaces, creates the native window, and closes SQLite when the application exits.

## Filesystem privacy

On Unix-like systems Solace applies a process umask of `0077` before SQLite or WebKit starts. Owned directories are created as `0700`; the database, logs, and WebKit storage files are `0600`. The native acceptance test verifies these modes in an isolated data root.

## Error strategy

Backend operations wrap errors with the failing operation and preserve the underlying error for inspection. Entry updates require the version last read by the editor, so a stale save cannot silently overwrite newer content. The bridge returns rejected promises to the UI, which preserves the form and displays a visible failure state.

## Entry flow

The desktop UI calls generated Wails bindings through a small diary bridge. Browser preview uses the same interface with local storage, allowing UI development without touching SQLite. Native mode keeps SQLite as the source of truth. Autosave uses the validated 400, 700, or 1,500 ms writing preference, skips clean forms, saves before entry selection changes, and attempts to flush pending edits when leaving an entry view.

Tiptap JSON is the canonical rich-text representation. The editor simultaneously derives plain text for previews, future FTS indexing, export, and optional future processing. Documents are validated as Tiptap `doc` roots; later schema changes must use forward migrations rather than rewriting historical migrations.

## Navigation flow

Entries retain one stable database row even when their inclusive date range spans many days. Calendar aggregation counts an entry on each covered date. Timeline queries first select entries overlapping a bounded month or year, then expand only the requested date window into occurrences that point back to the same entry ID. Results are paginated in fixed-size pages, so the UI never renders an unbounded archive at once.

Dashboard queries return compact entry summaries rather than rich documents. “On This Day” checks the current month/day against prior-year dates inside each inclusive range. Browser preview implements the same contract over disposable local storage; native mode uses indexed SQLite queries.

## Search flow

SQLite FTS5 stores one search document per stable entry ID with separate title, plain-text content, tag, and attachment-caption columns. Database triggers keep the document synchronized when entries, assignments, tag names, captions, or attachments change. Soft-deleted entries remain indexed for inexpensive restore but are excluded by every search query; permanent deletion removes the FTS row.

User search text is reduced to Unicode letter/number tokens and passed to `MATCH` as a parameterized prefix query. Controlled filter clauses are assembled from fixed SQL fragments, while all user values remain parameters. Search results return compact summaries, relevance ranking, highlighted title/snippet text, and pagination. The frontend treats highlight markers as delimiters and renders their content as text, avoiding HTML injection.

## Attachment flow

Native picker and OS drop operations pass source paths to Go, while pasted files pass encoded bytes through the binding. Imports enforce a 1 GiB maximum, use a private temporary file, compute SHA-256 while copying, sync and atomically move the result, then insert metadata. A failure before the database insert removes the staged/final file. Original files are never modified or deleted.

Gallery thumbnails use generated local JPEGs where the standard decoder supports the source; formats such as WebP safely fall back to the original local image. Video and audio use native HTML media controls, and decoding failures expose the operating-system external-open action. Unsupported documents are always available through the same external fallback.

## Portable-data flow

Exports load an explicitly scoped logical journal and write a private temporary destination before an atomic rename. Markdown and HTML archives provide human-readable copies; JSON provides the versioned interchange model; complete archives add a consistent SQLite snapshot, media bytes, and a checksummed manifest.

Restore never trusts ZIP paths or the included database snapshot. It validates versions, names, sizes, checksums, relationships, and attachment paths before mutation. A non-empty journal requires explicit replacement and receives a recovery archive first. Media is staged and swapped with rollback protection while canonical tables are replaced inside one SQLite transaction. Backup scheduling reuses the same complete archive path and runs only when Solace starts and finds a daily or weekly interval due.

## Safety and lock flow

Before pending migrations, existing databases receive a private SQLite snapshot. Every migration remains transactional, and integrity/foreign-key checks run after migration without automatic repair. Meaningful entry updates can snapshot the previous saved state in the same transaction, at most once per ten-minute window. Application logic and a database trigger bound recovery history to five versions per entry.

The privacy service stores only a salted PBKDF2-HMAC-SHA-256 derived hash and configuration. Lock state is process-local. React covers the journal with a focus-contained modal on startup, inactivity, or manual lock and exposes `Ctrl+Shift+L`. This is intentionally documented as a privacy screen, not encryption; native services and on-disk files are not cryptographically sealed.

## Desktop settings and shortcuts

The existing key/value settings table stores a validated application-settings model: theme, sidebar state, editor font size, prose width, line spacing, autosave delay, export defaults, and date display. No schema migration is needed when a new preference uses the same table; missing keys receive safe defaults, while the service rejects invalid writes.

One global React keyboard listener maps Command on macOS and Control on Linux/Windows to stable application actions. Entry-specific save and reading actions are dispatched only to the mounted workspace. Focus mode changes layout only; it does not change entry data or Tiptap semantics. CSS variables apply prose width, size, and spacing without rebuilding the editor.

## AI boundary

There is no LLM, embedding, vector database, model download, background indexer, external AI API, or RAG code in the application. The normal journal must remain fully usable without any AI component. Detailed AI architecture is intentionally deferred until future Part 9.
