# Application logs and user-facing errors

Solace keeps technical failures separate from the language shown in the interface. People see short, contextual guidance such as “Your latest changes couldn’t be saved” or “That archive couldn’t be verified.” The underlying error is recorded for troubleshooting instead of being exposed as database, filesystem, or Wails terminology.

## Location and retention

Native application logs live beneath the private Solace data directory:

- Linux: `~/.local/share/solace/logs/solace.log`, or `$XDG_DATA_HOME/solace/logs/solace.log` when `XDG_DATA_HOME` is set.
- macOS and Windows: the `logs` folder beneath the platform-specific Solace data directory shown in Settings → About.

The active log rotates at 2 MiB. Solace retains at most five numbered backups (`solace.log.1` through `solace.log.5`), bounding normal log storage to approximately 12 MiB. Rotation occurs during writes and startup; it does not require a background service. Log directories use private owner-only permissions, and log files are created as `0600` where the operating system supports Unix permissions.

## What is recorded

The JSON log includes application startup and shutdown information, Wails runtime messages, scheduled-backup failures, journal health-check failures, unexpected Go panics with a stack trace, returned startup/runtime errors, and handled frontend operation failures. Frontend reports use a short area label such as `save entry` or `inspect archive` so a failure can be located without showing implementation language to the user.

Solace must never intentionally send journal text, entry titles, passwords, attachment contents, or other private form values to the log. The frontend reports only the technical error supplied by an operation. Browser-preview errors go to the browser developer console because no native data directory is open.

## Support workflow

If an operation fails, first retry it using the guidance shown in Solace. If the problem repeats, note the time and copy the relevant recent JSON lines from `solace.log`; do not share the entire Solace data directory. A log can contain local file paths and operating-system details, so review it before sharing.
