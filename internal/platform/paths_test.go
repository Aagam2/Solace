package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDirectories(t *testing.T) {
	paths := PathsForRoot(filepath.Join(t.TempDir(), "solace"))
	if err := EnsureDirectories(paths); err != nil {
		t.Fatalf("EnsureDirectories() error = %v", err)
	}
	for _, path := range []string{paths.Root, paths.Images, paths.Videos, paths.Audio, paths.Documents, paths.Thumbnails, paths.Backups, paths.Logs} {
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			t.Fatalf("expected directory %q: %v", path, err)
		}
		if got := info.Mode().Perm(); got != 0o700 {
			t.Errorf("directory %q mode = %o, want 700", path, got)
		}
	}
}

func TestEnsureDirectoriesRejectsFileRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(root, []byte("occupied"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureDirectories(PathsForRoot(root)); err == nil {
		t.Fatal("EnsureDirectories() expected error")
	}
}
