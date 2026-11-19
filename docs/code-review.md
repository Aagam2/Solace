# Code review report

Reviewed: 2026-11-17, after Part 4 and before Part 5 implementation.

## Scope

The review covered the Go service boundaries, migrations, SQLite query construction, entry and attachment lifecycle, React editor state, host bridges, calendar/timeline/dashboard navigation, generated bindings, dependency posture, and automated checks. No `.env` file or real Solace database was read. Database checks used temporary test databases and the isolated native acceptance root.

## Findings resolved

### Autosave could submit the same optimistic version twice

An explicit Save click did not cancel the active 700 ms debounce timer. Both writes could use the same entry version, causing the second write to report a false stale-update error. The editor now cancels the timer, coalesces concurrent saves, tracks edit revisions, and preserves edits made while a save is in flight. A frontend regression test verifies one version increment and no delayed failure.

### Attachment reorder accepted duplicate IDs

A malformed reorder list with the correct length could repeat an attachment ID, update that row twice, and leave another row at an inconsistent order. The service now rejects duplicate IDs before starting the transaction. Backend regression coverage was added.

## Review checks

- `go vet ./...`: clean.
- Full normal Go test suite: clean.
- Go race detector: app, attachment, and database packages passed. The synthetic diary package was stopped after several minutes because race instrumentation makes its 10,000-entry fixtures disproportionately slow; the normal diary suite and all search stress tests pass.
- TypeScript strict check: clean.
- ESLint: clean, with no warnings.
- Frontend tests: clean.
- Production build and dependency audit: clean; production npm dependencies report zero known vulnerabilities.
- SQL search text is parameterized and converted to a restricted FTS token expression; user text is never concatenated into SQL.
- Search highlighting is rendered as React text fragments rather than injected HTML.

## Residual observations

The full npm development-tool audit reports advisories in Vitest, ESLint transitive packages, source-map tooling, and the shadcn CLI. These packages are not shipped in the runtime bundle. A compatible Vitest patch installation was attempted, but npm 11.4.0 failed internally with `Cannot read properties of null (reading 'edgesOut')`; no forced breaking downgrade was applied. Revisit the development lockfile when npm or the affected packages are updated. Production audit remains zero vulnerabilities.

No unresolved high-severity application-runtime defect was found in the reviewed scope.
