# Development status

Last updated: 2026-11-19

## Current checkpoint: Part 8 desktop experience and packaging (complete)

### Completed

- Read and decomposed the full master development prompt.
- Confirmed the repository began as an empty initial commit.
- Established Go package boundaries for platform paths, SQLite, settings, and the application service.
- Added private OS data-directory creation and the planned media/thumbnail/backup/log layout.
- Added transactional embedded migrations and initial settings schema.
- Added a React + strict TypeScript/Vite application shell.
- Added sidebar navigation, Today empty state, collapsed mode, light/dark/system themes, and browser preference persistence.
- Added frontend and backend foundational tests.
- Installed and configured Tailwind CSS v4 through the Vite plugin.
- Configured the `@/*` TypeScript and Vite import alias required by shadcn/ui.
- Installed Wails CLI v3.0.0 and verified the Linux GTK4/WebKitGTK 6.0 toolchain.
- Initialized shadcn/ui 4.23.0 with Base UI and the Nova preset.
- Documented architecture, database decisions, testing, prerequisites, and roadmap.
- Explicitly kept all LLM, embedding, semantic-search, vector-store, and RAG functionality out of the runtime.
- Added the Wails v3 native entry point, generated interface bindings, embedded assets, and Linux build scaffold.
- Connected React settings to the real Go/SQLite service in native mode while preserving browser preview mode.
- Added structured JSON file/stderr logging and clean database shutdown.
- Enforced private process, directory, database, log, and WebKit storage permissions.
- Renamed all runtime, module, build, binding, storage, master-prompt, and documentation identifiers from the old working name to Solace.
- Added the diary schema, stable UUID entry IDs, inclusive journal date ranges, soft deletion, optimistic versions, tags, and supporting indexes.
- Added native create/read/update/list/duplicate/trash/restore/permanent-delete operations.
- Added untitled, draft, favorite, mood, location, tag, retroactive, overlapping, multi-day, and multiple-same-day support.
- Added Today, All entries, Favorites, and Trash workspaces with date-range editing and newest/oldest/updated/favorites sorting.
- Added 700 ms debounced autosave, visible save/error states, clean-form write suppression, navigation flushing, and stale-write rejection.
- Added a minimal Tiptap editor with headings, bold, italic, underline, strike, lists, checklists, blockquotes, links, horizontal rules, undo/redo, keyboard shortcuts, and synchronized plain text.
- Added the attachment migration and service with UUID filenames, portable relative paths, MIME detection, SHA-256 checksums, 1 GiB validation, atomic staged imports, and failure cleanup.
- Added picker, OS drag/drop, clipboard-paste, inline images, image gallery, captions, reordering, generated thumbnails, enlargement, and attachment removal.
- Added video/audio controls, PDF and arbitrary-file cards, missing-file states, and external-open fallbacks for unsupported playback or formats.
- Added secure attachment-ID media routes, private `0600` media/thumbnail files, and complete owned-file cleanup on permanent entry deletion.
- Added migration `004_navigation.sql` with persisted pinned-entry state and an index for pinned browsing.
- Added calendar aggregation that expands inclusive entry ranges across every covered day, including cross-month and cross-year ranges, without duplicating entries in storage.
- Added a monthly calendar with adjacent-month navigation, direct month/today jumps, entry counts, mood and attachment indicators, and a date-specific entry list.
- Added a bounded, paginated timeline for month/year browsing. Multi-day entries appear on each covered date while retaining one stable entry ID.
- Added historical month/year jumps, previous/next period controls, previous/next entry controls, empty-year handling, and direct opening in the editor.
- Added a Today dashboard for today’s pages, recently edited pages, recent memories, and “On This Day” matches from prior years, including multi-day memories.
- Added persisted pin/unpin actions and visible pinned state in entry lists, dashboard cards, and timeline rows.
- Added distraction-free reading mode and direct navigation from the calendar, timeline, and dashboard to the underlying entry.
- Added a browser-preview adapter for all Part 4 flows without touching SQLite.
- Completed a code review of Parts 1–4 and recorded the scope, findings, fixes, checks, and residual tooling observations in `docs/code-review.md`.
- Fixed a confirmed autosave race by cancelling pending debounce timers, coalescing concurrent saves, and preserving edits made during an in-flight save.
- Fixed attachment reorder validation so duplicate attachment IDs cannot produce inconsistent ordering.
- Added SQLite FTS5 migration `005_search.sql` with title, plain-text content, tag, and attachment-caption columns, prefix indexes, initial backfill, and synchronization triggers.
- Added safe partial-word and multi-word search with parameterized FTS input, relevance/date sorting, highlighted titles, context snippets, and pagination.
- Added inclusive overlap-date, multi-tag, mood, favorite, image, video, and draft/published filters that combine with text search.
- Added transactional search-index rebuild and tests for edit/delete/restart/rebuild behavior.
- Added tag creation, rename, deletion, counts, assignment, filtering, and one-click tag-to-search navigation without hardcoded tag names.
- Added native and browser-preview Part 5 bridges and a full Search/Tags interface.
- Completed a screenshot-based visual review of every primary screen in light and dark themes at 1240 x 780 and the 800 x 600 Wails minimum size.
- Repaired the light-theme contrast regression by separating muted background and foreground tokens.
- Unified Solace and shadcn semantic tokens around the warm paper, ink, and brown brand palette in both themes.
- Added a compact icon rail below 820 px so the minimum desktop window no longer has horizontal overflow.
- Added consistent keyboard focus rings and prevented the Tags creation action from wrapping.
- Added five lightweight project-local UI/review skills totaling approximately 584 KB; no browser runtime or other large dependency was downloaded.
- Recorded the visual findings, fixes, skill footprint, verification, and remaining platform checks in `docs/ui-review.md`.
- Reduced avoidable scrolling with tabbed Today summaries and tabbed Writing/Details & media editor surfaces.
- Removed the duplicate Today editor, kept navigation and the top bar stable, and contained long scrolling inside the active work surface.
- Added accessible tab semantics and keyboard navigation, plus a regression test for switching editor sections without leaving the entry.
- Added Markdown, offline HTML, versioned JSON, and complete archive exports for the whole active journal, inclusive date ranges, or selected entries.
- Added optional attachment inclusion, collision-safe readable filenames, stable-ID preservation, atomic destination writes, cancellation, and SQLite-supported snapshots.
- Added versioned archive manifests with SHA-256 checksums and validation against missing files, duplicate or unsafe paths, unsupported versions, invalid relationships, and oversized expanded archives.
- Added inspection before restore, explicit replacement for non-empty journals, automatic recovery archives, staged media, and transactional database replacement with rollback protection.
- Added manual, daily, and weekly local backup settings, user-selected destinations, attachment inclusion, last-success tracking, and retention pruning.
- Added the compact tabbed Data Management interface under Settings with Export, Restore, and Backups panels, keyboard tab navigation, native file/folder dialogs, progress states, cancellation, and browser-preview guidance.
- Added explicit frontend regression coverage for media drag-and-drop and confirmed the existing picker, native OS drop, and paste workflows remain available.
- Documented the archive format, restore safety model, backup behavior, testing matrix, and Part 6 checkpoint.
- Added migration `007_safety.sql` with bounded entry revisions and privacy-lock configuration.
- Added private SQLite snapshots before pending migrations on existing profiles, transactional failure tests, startup integrity/foreign-key checks, and a read-only health action in Settings.
- Added the newest 20 saved revisions per entry, captured in the same transaction as updates, with accessible History browsing and revision restore that preserves the current saved version.
- Upgraded portable export format to version 2 so complete archives and JSON include revision history while version 1 archives remain accepted.
- Added an optional password privacy screen with random salts, standard PBKDF2-HMAC-SHA-256 derivation, constant-time verification, startup locking, 1–60 minute inactivity locking, manual locking, and `Ctrl+Shift+L`.
- Clearly labeled the application lock as a privacy screen rather than database/media encryption; no plaintext password is stored.
- Added the compact Privacy & safety tab with lock configuration, native-only controls, integrity status, responsive light/dark styling, and accessible focus containment on the unlock dialog.
- Added focused tests for migration backup/failure, integrity damage, unwritable storage, lock lifecycle/validation, bounded revision recovery, browser revision restore, and revision-aware archive round trips.
- Documented privacy boundaries, recovery behavior, migration safety, reliability coverage, and remaining encryption work.
- Added platform-aware global shortcuts for new entry, save, search, Today, sidebar, reading mode, and Settings, with discoverable shortcut references.
- Added editable focus mode, preserved read-only reading mode, Escape dismissal, smooth/reduced-motion-aware scrolling, and live word/character counts.
- Added persisted editor font size, bounded prose width, line spacing, autosave pace, date display, and export defaults with backend validation and safe defaults.
- Replaced the data-only Settings destination with compact General, Writing, Data & safety, and About tabs while retaining the existing export, restore, backup, and privacy workflows.
- Added read-only native storage information and an explicit explanation that relocation is unavailable without a verified migration process.
- Updated Linux desktop and Debian metadata, runtime dependencies, package permissions, application version information, and Office/Utility categories.
- Added Linux-first build, install, data retention, backup/restore, and safe-uninstall documentation plus Windows/macOS preparation notes.
- Replaced the generated Wails placeholder artwork with a lightweight Solace journal-and-spark vector identity, regenerated PNG/ICO/ICNS assets, matched the in-app mark and favicon, and corrected Linux, Windows, and macOS product metadata.
- Replaced raw frontend exception text with shared, contextual user-facing guidance while reporting technical details to the native log without journal text or passwords.
- Added a private size-based JSON log writer with a 2 MiB active-file limit, five retained backups, startup rotation, fatal-error reporting, and panic stack capture.
- Documented the log location, retention bound, privacy expectations, and support workflow in `docs/logging.md`.
- Curated `.gitignore` for local agent/Codex/AWS state, dependencies, tool caches, compiled packages, generated platform icons, runtime journal data, test output, and editor clutter; restored `master_prompt.txt` as a trackable implementation checkpoint while keeping bindings, lockfiles, icon sources, packaging templates, and documentation trackable.

### Verification completed

- No external service or database was accessed; verification used only temporary local project SQLite databases under test directories and `/tmp`.
- Node.js: 22.15.1.
- npm: 11.4.0.
- Go: 1.27.4; the normal project test and build commands run successfully in the current environment.
- Legacy Wails v2.10.1 remains installed as the separate `wails` command.
- Wails CLI v3.0.0: installed alongside the existing v2 CLI.
- GTK 4: 4.14.5; WebKitGTK 6.0: 2.52.6.
- `wails3 doctor`: passed with no issues; system ready for Wails development.
- shadcn/ui: Base UI 1.9.0, Nova preset, Lucide icons, CSS variables, and aliases verified.

- The initial frontend dependency set was installed from the local npm cache; Tailwind, shadcn/ui, and their dependencies were subsequently installed by the user.
- Frontend strict type-check: passed.
- Frontend unit tests: 18 passed, including autosave race prevention, tabbed editor navigation, explicit media drag-and-drop, revision recovery, lock-screen disclosure, friendly error translation, hidden technical details, partial-word search/direct opening, tag rename propagation, data/safety tabs, database-health states, global shortcuts, persisted writing preferences, writing counts, and focus-mode dismissal.
- Frontend ESLint: passed.
- Frontend production build: passed with React and Tiptap/ProseMirror split into cacheable vendor chunks.
- Branding/error-copy browser review: passed at 1240 x 780 and 800 x 600; the mark remains legible in the full and collapsed sidebars and at 32 px launcher size.
- Go platform filesystem tests: passed.
- All Go packages and database test sources compile (`go test ./internal/... -run '^$'` with network disabled).
- Full Go test suite passed after database-test permission was granted; first run, reopen, migrations, settings persistence, validation, and the frontend contract are covered.
- Log-rotation tests passed for rollover, retention limits, active-file continuity, private modes, and invalid configuration.
- Diary/backend tests passed for entry lifecycle, date overlap, month/year boundaries, invalid ranges, retroactive entries, restart persistence, and stale versions.
- Attachment/backend tests passed for media types, duplicate/unusual names, reordering, captions, thumbnails, restart/open behavior, missing files, interrupted imports, size limits, removal, and permanent cleanup.
- Navigation/backend tests passed for month/year boundaries, cross-month and cross-year expansion, multiple entries on one date, timeline identity, historical jumps, empty years, pinned summaries, and “On This Day” matching.
- The navigation performance fixture passed with 10,000 entries, 2,000 tags, and 2,000 attachment records; timeline queries remain bounded and paginated.
- Search/backend tests passed for creation, editing, deletion, multiple words, empty/special queries, overlap ranges, combined filters, tag CRUD, rebuild, restart, attachment captions, and 5,000 entries.
- Portable-data/backend tests passed for all four export formats, empty and populated journals, inclusive and selected scopes, attachments, manifests/checksums, unsafe paths, invalid-restore isolation, recovery backups, retention, and a 100-entry export/restore round trip.
- Part 7 backend tests passed for migration snapshots and rollback, integrity/foreign-key damage, unwritable storage, password hashing and lock lifecycle, 20-version retention and restore, and revision-aware archive round trips.
- `go vet ./...`: passed.
- Go race detector: app, attachments, and database packages passed; the synthetic diary package was stopped because race instrumentation made its large fixtures impractically slow. The full normal suite passes.
- Production `wails3 build`: passed after Part 3; native binary created at `bin/solace`.
- Native production launch: passed using an isolated data root; embedded JavaScript and CSS were served.
- Native development launch: passed; Vite became ready and the Wails window connected to it.
- Part 2 native production launch: passed using `/tmp/solace-part2-native`; current embedded assets were served.
- Part 3 native production launch: passed using `/tmp/solace-part3-native`; current embedded Tiptap/media assets were served.
- Part 4 native production build and launch: passed using `/tmp/solace-part4-native`; current calendar/timeline/dashboard assets were served.
- Part 5 native production build and isolated launch: passed using `/tmp/solace-part5-native`; the search/tag bundle and current embedded assets were served.
- Pre-Part 6 production build and isolated native launch: passed using `/tmp/solace-visual-final`; the visually reviewed assets were served by GTK4/WebKitGTK 6.0.
- Scrolling/density follow-up production build and isolated native launch: passed using `/tmp/solace-tabs-final`; the tabbed assets were served by GTK4/WebKitGTK 6.0.
- Part 6 production build and isolated native launch: passed using `/tmp/solace-part6-native`; the portable-data service, sixth migration, generated bindings, and tabbed Data Management assets were served by GTK4/WebKitGTK 6.0.
- Part 7 production build and isolated native launch: passed using `/tmp/solace-part7-native`; the safety/privacy service, seventh migration, generated bindings, and history/privacy assets were served by GTK4/WebKitGTK 6.0.
- Part 8 explicit development build: passed at `/tmp/solace-part8-dev`.
- Part 8 production build and two isolated launches: passed using `/tmp/solace-part8-native`; the final settings/writing assets were served by GTK4/WebKitGTK 6.0 without network access.
- Linux Debian package: generated at `bin/solace.deb` (approximately 6.4 MB), inspected successfully, with declared GTK4/WebKitGTK 6.0 dependencies and hardened packaged file modes.
- Branded production build and isolated native launch: passed using `/tmp/solace-brand-native.cJJeGk`; `solace.log` was created as `0600`, embedded assets loaded, and the rebuilt Debian package contained the branded 128 px icon plus user-facing desktop metadata.
- SQLite integrity and foreign-key checks: clean; migrations `001_foundation.sql` through `007_safety.sql`, the FTS5 table, all nine synchronization triggers, and insert/update/delete index maintenance were verified.
- Native filesystem check: all owned directories `0700`, database/log/storage files `0600`.
- Production npm dependency audit: 0 vulnerabilities after updating Vite to 6.4.6 and classifying build tools as development dependencies.
- Development-only tooling still reports advisories during installation. They are not shipped in the runtime; rerun the full audit before release and avoid forced breaking dependency changes without a dedicated compatibility pass.

### Manual checks for the user

- Theme and normal/minimum-size visual checks are complete for Linux. Before publishing Windows or macOS packages, repeat the platform-specific pass there.
- Toggle light, dark, and system themes, restart, and confirm the selection persists in the native app.
- Collapse the sidebar, restart, and confirm its state persists.
- Complete the Part 3 “My Weekend” rich-text/media matrix in `docs/testing.md`, then restart after moving the original source files.
- Complete the Part 4 calendar/timeline matrix in `docs/testing.md`, including cross-year entries and “On This Day”.
- Complete the packaged native create/import/export/restart checklist in `docs/testing.md`; automated browser/domain tests cover those flows, but this environment cannot drive native GTK file dialogs.

## Planned roadmap

1. Part 1 complete: desktop foundation.
2. Part 2 complete: diary CRUD, inclusive date ranges, drafts, trash, and autosave.
3. Part 3 complete: Tiptap rich text and safe local media import.
4. Part 4 complete: calendar, timeline, dashboard, pinning, reading mode, and historical navigation.
5. Part 5 complete: offline FTS5 search, filters, tag management, and organization.
6. Part 6 complete: export, verified restore/import, and local backups.
7. Part 7 complete: privacy lock, revision recovery, migration safety, integrity checks, and reliability hardening.
8. Part 8 complete: keyboard shortcuts, writing polish, application settings, Linux-first packaging, and desktop experience.
9. Part 9 remains future-only: optional AI/RAG architecture. No implementation until machine and user requirements change.

## Update protocol

At every development checkpoint, update this file with completed scope, verification evidence, blockers, remaining tasks, and the next approved part. Corresponding architecture/database/testing documentation must change with the code.
