package database

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestOpenInitializesAndReopensDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "solace.db")
	for attempt := 0; attempt < 2; attempt++ {
		db, err := Open(context.Background(), path)
		if err != nil {
			t.Fatalf("Open() attempt %d error = %v", attempt+1, err)
		}
		var migrations int
		if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&migrations); err != nil {
			t.Fatal(err)
		}
		if migrations != 7 {
			t.Fatalf("migration count = %d, want 7", migrations)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("database mode = %o, want 600", got)
		}
		db.Close()
	}
}

func TestOpenBacksUpBeforePendingMigration(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "solace.db")
	db, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DROP TABLE privacy_settings; DROP TABLE entry_revisions; DELETE FROM schema_migrations WHERE version='007_safety.sql'`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db, err = Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	matches, err := filepath.Glob(filepath.Join(root, "backups", "pre-migration-007_safety-*.db"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("migration backups = %v, %v", matches, err)
	}
	if info, err := os.Stat(matches[0]); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("backup permissions = %v, %v", info, err)
	}
}

func TestFailedMigrationRollsBackItsTransaction(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "broken.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := configure(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	migrationFS := fstest.MapFS{
		"migrations/001_good.sql":   &fstest.MapFile{Data: []byte(`CREATE TABLE stable (id INTEGER PRIMARY KEY);`)},
		"migrations/002_broken.sql": &fstest.MapFile{Data: []byte(`CREATE TABLE should_rollback (id INTEGER); THIS IS NOT SQL;`)},
	}
	if err := migrateFS(context.Background(), db, migrationFS); err == nil {
		t.Fatal("broken migration succeeded")
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_schema WHERE type='table' AND name='should_rollback'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("failed migration left schema behind: %d, %v", count, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version='002_broken.sql'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("failed migration was recorded: %d, %v", count, err)
	}
}

func TestIntegrityDetectsForeignKeyDamage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "solace.db")
	db, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`PRAGMA foreign_keys=OFF`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO entry_tags(entry_id,tag_id) VALUES('missing-entry','missing-tag')`); err != nil {
		t.Fatal(err)
	}
	if err := CheckIntegrity(context.Background(), db); err == nil {
		t.Fatal("foreign key damage was not detected")
	}
}

func TestOpenReportsUnwritableDatabaseLocation(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, []byte("blocked"), 0o600); err != nil {
		t.Fatal(err)
	}
	if db, err := Open(context.Background(), filepath.Join(blocker, "solace.db")); err == nil {
		db.Close()
		t.Fatal("database opened below a regular file")
	}
}
