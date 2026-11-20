# Privacy, reliability, and data safety

Solace is designed to work without a network connection. It has no account system, analytics, telemetry, automatic crash reporting, remote fonts, AI runtime, or background upload path. File access occurs only inside Solace-owned storage or after an explicit picker/drop action.

## Application privacy lock

The optional lock is a local privacy screen for casual access while Solace is running. It supports:

- locking when the application starts;
- locking after 1, 5, 15, 30, or 60 minutes of inactivity;
- manual locking with the visible action or `Ctrl+Shift+L`;
- changing or disabling the lock only after verifying the current password.

Passwords are never stored. Solace derives a 256-bit hash using standard PBKDF2-HMAC-SHA-256 with a random 128-bit salt and a work factor stored alongside the hash. Comparison is constant-time.

The lock is **not database or media encryption**. Someone who can access the operating-system account and Solace data directory may still read the underlying files. Full encryption requires a separately designed key-management and recovery system and remains future work.

## Database and migration safety

- SQLite foreign keys, WAL, a busy timeout, and migration transactions remain enabled.
- Startup runs SQLite `quick_check` and `foreign_key_check`. Failure stops startup and reports the problem; Solace does not attempt destructive automatic repair.
- When an existing database has a pending migration, Solace first creates a private `0600` SQLite snapshot under `backups/pre-migration-*.db` using `VACUUM INTO`.
- A failed migration transaction is rolled back and is not recorded as applied.
- Filesystem errors, full-disk/write errors, missing media, and import failures are surfaced instead of silently deleting data.

## Entry recovery

Trash remains recoverable until explicit permanent deletion. For entry recovery, Solace stores the previous saved state only when writing or meaningful details change and at least ten minutes have elapsed since the last automatic recovery point. Favorite, pin, draft-status, and ordering changes do not create noise. Solace retains at most five versions per entry; people can delete one version or clear an entry's recovery history without changing the current entry. Restoring always preserves the current saved state first when it differs.

Restoring a revision goes through the normal optimistic update path, so a different current saved state becomes another recoverable revision. Complete archives use export format version 2 and include revision history; version 1 archives remain accepted with an empty history.

## Reliability boundaries

Atomic attachment import, atomic export destinations, transactional restore, WAL recovery, optimistic versions, coalesced autosaves, checksum validation, and pre-restore archives cover the primary interruption cases. Scheduled backups run when Solace starts and finds an interval due; Solace does not install a background daemon.
