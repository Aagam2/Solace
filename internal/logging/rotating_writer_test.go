package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRotatingWriterBoundsFilesAndKeepsPrivateModes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "solace.log")
	writer, err := NewRotatingWriter(path, 24, 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"first message\n", "second message\n", "third message\n", "fourth message\n"} {
		if _, err := writer.Write([]byte(value)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"", ".1", ".2"} {
		info, err := os.Stat(path + suffix)
		if err != nil {
			t.Fatalf("stat %s: %v", suffix, err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("mode %s = %o, want 600", suffix, info.Mode().Perm())
		}
	}
	if _, err := os.Stat(path + ".3"); !os.IsNotExist(err) {
		t.Fatalf("unexpected third backup: %v", err)
	}
	latest, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(latest), "fourth message") {
		t.Fatalf("active log = %q", latest)
	}
}

func TestRotatingWriterRejectsInvalidLimits(t *testing.T) {
	if _, err := NewRotatingWriter(filepath.Join(t.TempDir(), "log"), 0, 1); err == nil {
		t.Fatal("expected invalid maximum size to fail")
	}
}
