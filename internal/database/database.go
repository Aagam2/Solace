package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Open(ctx context.Context, path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := configure(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	if err := backupBeforeMigrations(ctx, db, path); err != nil {
		db.Close()
		return nil, err
	}
	if err := Migrate(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	if err := CheckIntegrity(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	if err := secureDatabaseFiles(path); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func secureDatabaseFiles(path string) error {
	for _, candidate := range []string{path, path + "-wal", path + "-shm"} {
		if err := os.Chmod(candidate, 0o600); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("secure database file %q: %w", candidate, err)
		}
	}
	return nil
}

func configure(ctx context.Context, db *sql.DB) error {
	for _, pragma := range []string{"PRAGMA foreign_keys = ON", "PRAGMA journal_mode = WAL", "PRAGMA synchronous = NORMAL", "PRAGMA busy_timeout = 5000"} {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			return fmt.Errorf("configure sqlite (%s): %w", pragma, err)
		}
	}
	return nil
}

func Migrate(ctx context.Context, db *sql.DB) error {
	return migrateFS(ctx, db, migrations)
}

func migrateFS(ctx context.Context, db *sql.DB, migrationFS fs.FS) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY NOT NULL, applied_at TEXT NOT NULL)`); err != nil {
		return fmt.Errorf("initialize migrations: %w", err)
	}
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		var exists int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, entry.Name()).Scan(&exists); err != nil {
			return fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if exists > 0 {
			continue
		}
		body, err := fs.ReadFile(migrationFS, "migrations/"+entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", entry.Name(), err)
		}
		if _, err = tx.ExecContext(ctx, string(body)); err == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)`, entry.Name(), time.Now().UTC().Format(time.RFC3339Nano))
		}
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", entry.Name(), err)
		}
	}
	return nil
}

// CheckIntegrity verifies SQLite's internal structures and every declared
// foreign key. It never attempts automatic repair or deletion.
func CheckIntegrity(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `PRAGMA quick_check`)
	if err != nil {
		return fmt.Errorf("run database integrity check: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var result string
		if err := rows.Scan(&result); err != nil {
			return fmt.Errorf("read database integrity check: %w", err)
		}
		if result != "ok" {
			return fmt.Errorf("database integrity check failed: %s", result)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("database integrity check: %w", err)
	}
	var table, parent string
	var rowID int64
	var foreignKey int
	err = db.QueryRowContext(ctx, `SELECT "table", rowid, parent, fkid FROM pragma_foreign_key_check LIMIT 1`).Scan(&table, &rowID, &parent, &foreignKey)
	if err == nil {
		return fmt.Errorf("foreign key check failed: table %s row %d references %s (key %d)", table, rowID, parent, foreignKey)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("run foreign key check: %w", err)
	}
	return nil
}

func backupBeforeMigrations(ctx context.Context, db *sql.DB, databasePath string) error {
	var tableExists int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_schema WHERE type='table' AND name='schema_migrations'`).Scan(&tableExists); err != nil {
		return fmt.Errorf("inspect migration state: %w", err)
	}
	if tableExists == 0 {
		return nil
	}
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations before backup: %w", err)
	}
	pending := ""
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		var applied int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=?`, entry.Name()).Scan(&applied); err != nil {
			return fmt.Errorf("inspect migration %s before backup: %w", entry.Name(), err)
		}
		if applied == 0 {
			pending = entry.Name()
			break
		}
	}
	if pending == "" {
		return nil
	}
	backupDirectory := filepath.Join(filepath.Dir(databasePath), "backups")
	if err := os.MkdirAll(backupDirectory, 0o700); err != nil {
		return fmt.Errorf("create pre-migration backup directory: %w", err)
	}
	name := fmt.Sprintf("pre-migration-%s-%s.db", strings.TrimSuffix(pending, ".sql"), time.Now().UTC().Format("20060102-150405.000000000"))
	destination := filepath.Join(backupDirectory, name)
	quoted := strings.ReplaceAll(destination, "'", "''")
	if _, err := db.ExecContext(ctx, "VACUUM INTO '"+quoted+"'"); err != nil {
		return fmt.Errorf("create pre-migration database backup: %w", err)
	}
	if err := os.Chmod(destination, 0o600); err != nil {
		return fmt.Errorf("secure pre-migration database backup: %w", err)
	}
	return nil
}
