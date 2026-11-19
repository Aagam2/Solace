# Solace

Solace is a private, local-first desktop journal and memory archive. **Parts 1–8 are complete: writing, attachments, chronological browsing, search, portable data, recovery safeguards, desktop shortcuts, configurable writing, and Linux-first packaging are working.**

## Current capabilities

- Create, edit, duplicate, favorite, draft, publish, trash, restore, and permanently delete journal entries.
- Single-day, multi-day, overlapping, untitled, and retroactive entries with inclusive date ranges.
- Debounced autosave with visible status and optimistic version checks that prevent stale overwrites.
- Entry lists for Today, All entries, Favorites, and Trash with four ordering choices.
- Monthly calendar with entry counts, mood/attachment indicators, inclusive multi-day coverage, and date drill-down.
- Paginated month/year timeline with historical jumps and previous/next-entry controls.
- Today dashboard with today’s entries, recently edited pages, recent memories, and prior-year “On This Day” matches.
- Pinned entries, direct opening from calendar/timeline/dashboard, and a distraction-free reading mode.
- Offline SQLite FTS5 search across titles, plain text, tags, and attachment captions, with highlighted snippets and partial-word matching.
- Combined overlap-date, tag, mood, favorite, image, video, and draft/published filters with relevance or journal-date ordering.
- Tag creation, rename, deletion, counts, assignment, and one-click tag-to-search filtering.
- Tiptap rich-text editing with headings, emphasis, lists, checklists, blockquotes, links, rules, undo, and redo.
- Images, video, audio, PDFs, text, and arbitrary file attachments copied into private Solace storage.
- File picker, drag-and-drop, pasted images, inline images, generated thumbnails, captions, gallery reordering, and click-to-enlarge.
- Native video/audio playback with a clear external-open fallback when the WebView cannot decode a format.
- React + TypeScript desktop UI shell with tags, search, and advanced settings ready for later parts.
- Compact Today and editor tabs that keep writing, organization, and media controls available without excessive page scrolling.
- Markdown, offline HTML, versioned JSON, and complete restorable archive exports for the full journal, date ranges, or selected entries.
- Preflight archive inspection, checksums, path-traversal protection, transactional replacement, and automatic pre-restore recovery archives.
- Manual, daily, or weekly local backups with a configurable destination, attachment inclusion, and retention.
- Recoverable trash plus the 20 most recent saved revisions per entry, including revision-aware complete archives.
- Startup database integrity/foreign-key checks, transactional migrations, and private pre-migration database snapshots.
- Optional startup, inactivity, and manual application lock using a salted standard password hash, clearly separated from future file encryption.
- Light, dark, and system theme selection with a collapsible sidebar.
- Platform-aware shortcuts for creating, saving, searching, Today, sidebar, reading mode, and Settings.
- Distraction-free writing, bounded prose widths, adjustable text size/line spacing, word and character counts, and configurable autosave pace.
- Compact General, Writing, Data & safety, and About settings with read-only storage information and export defaults.
- Linux production binary and Debian package metadata, with Windows and macOS build scaffolds prepared for target-platform verification.
- A custom Solace journal-and-spark application icon across the in-app identity, browser preview, Linux package, Windows icon, and macOS icon assets.
- Friendly, contextual interface errors backed by private JSON logs with 2 MiB rotation and five retained backups.
- Go services for platform-safe local storage, SQLite migration management, and persistent settings.
- No account, telemetry, cloud dependency, external server, or AI runtime.

LLM/RAG functionality remains a future-only feature and is not part of the current runtime.

## Run the desktop application

```bash
cd /home/aagam-linux/Desktop/Projects/Solace
wails3 dev
```

This starts Vite and opens the native Solace window. The first run creates the private application data tree and SQLite database under `~/.local/share/solace` (or `XDG_DATA_HOME/solace`). Entry metadata is written to SQLite; imported files are copied beneath `media/` and no longer depend on their original locations. Stop development mode with `Ctrl+C`.

To run only the browser preview, use `cd frontend && npm run dev`. Browser preview stores UI preferences, disposable entries, and small preview attachments in local storage; it does not open SQLite. Use the native app for real journals and large media.

Build and run the production binary:

```bash
wails3 build
./bin/solace
```

## Checks

```bash
cd frontend
npm run typecheck
npm test
npm run lint
npm run build
```

Go source checks (these use temporary local SQLite databases):

```bash
go test ./...
```

## Desktop prerequisites

This checkout requires Go 1.26+, Wails v3, a C compiler, `pkg-config`, GTK4, WebKitGTK 6.0, Node.js, and npm. Run `wails3 doctor` to verify the desktop toolchain. See [docs/development-status.md](docs/development-status.md) for the exact checkpoint.

Application data will live outside the installation directory. On Linux the default is `~/.local/share/solace`; the service also honors `XDG_DATA_HOME`.
Technical logs are stored in the private `logs/` subfolder and rotate automatically; see [Application logs and user-facing errors](docs/logging.md).

## Documentation

- [Architecture](docs/architecture.md)
- [Database foundation](docs/database.md)
- [Testing](docs/testing.md)
- [Export, restore, and backup format](docs/export-format.md)
- [Privacy, reliability, and data safety](docs/privacy-and-safety.md)
- [Desktop experience and shortcuts](docs/desktop-experience.md)
- [Packaging and installation](docs/packaging.md)
- [Application logs and user-facing errors](docs/logging.md)
- [UI review checkpoint](docs/ui-review.md)
- [Development status and roadmap](docs/development-status.md)
- [Code review report](docs/code-review.md)
