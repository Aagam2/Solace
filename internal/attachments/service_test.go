package attachments

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"solace/internal/database"
	"solace/internal/diary"
	"solace/internal/platform"
)

func testAttachmentService(t *testing.T) (*Service, *sql.DB, platform.Paths, string) {
	t.Helper()
	root := t.TempDir()
	paths := platform.PathsForRoot(root)
	if err := platform.EnsureDirectories(paths); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(context.Background(), paths.Database)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := diary.New(db).CreateEntry(context.Background(), diary.EntryInput{Title: "My Weekend", StartDate: "2026-11-15"})
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	return New(db, paths), db, paths, entry.ID
}

func writeFixture(t *testing.T, directory, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestImportListReorderCaptionRemoveAndDuplicateNames(t *testing.T) {
	service, db, paths, entryID := testAttachmentService(t)
	defer db.Close()
	sources := t.TempDir()
	pngBytes, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	imageA := writeFixture(t, sources, "holiday photo.png", pngBytes)
	imageBDir := t.TempDir()
	imageB := writeFixture(t, imageBDir, "holiday photo.png", pngBytes)
	video := writeFixture(t, sources, "clip.mp4", []byte("video-data"))
	pdf := writeFixture(t, sources, "résumé [final].pdf", []byte("%PDF-1.7\nfixture"))

	imported, err := service.ImportFiles(context.Background(), entryID, []string{imageA, imageB, video, pdf})
	if err != nil {
		t.Fatal(err)
	}
	if len(imported) != 4 || imported[0].StoredFilename == imported[1].StoredFilename {
		t.Fatalf("unexpected imports: %+v", imported)
	}
	if !strings.HasPrefix(imported[0].MIMEType, "image/") || imported[2].MIMEType != "video/mp4" || imported[3].MIMEType != "application/pdf" {
		t.Fatalf("unexpected MIME types: %+v", imported)
	}
	if imported[0].ThumbnailURL == imported[0].URL {
		t.Fatalf("image thumbnail was not generated: %+v", imported[0])
	}
	for _, attachment := range imported {
		if filepath.IsAbs(attachment.RelativePath) || strings.Contains(attachment.RelativePath, "holiday photo") {
			t.Fatalf("unsafe storage path: %q", attachment.RelativePath)
		}
		info, err := os.Stat(filepath.Join(paths.Root, filepath.FromSlash(attachment.RelativePath)))
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("stored attachment permission: %v, %v", info, err)
		}
	}
	if err := service.UpdateCaption(context.Background(), imported[0].ID, "At the lake"); err != nil {
		t.Fatal(err)
	}
	ids := []string{imported[3].ID, imported[2].ID, imported[1].ID, imported[0].ID}
	if err := service.ReorderAttachments(context.Background(), entryID, ids); err != nil {
		t.Fatal(err)
	}
	if err := service.ReorderAttachments(context.Background(), entryID, []string{imported[0].ID, imported[0].ID, imported[2].ID, imported[3].ID}); err == nil {
		t.Fatal("duplicate attachment reorder unexpectedly succeeded")
	}
	listed, err := service.ListAttachments(context.Background(), entryID)
	if err != nil || listed[0].ID != imported[3].ID || listed[3].Caption != "At the lake" {
		t.Fatalf("reordered attachments = %+v, error = %v", listed, err)
	}
	removedPath := filepath.Join(paths.Root, filepath.FromSlash(imported[1].RelativePath))
	if err := service.RemoveAttachment(context.Background(), imported[1].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(removedPath); !os.IsNotExist(err) {
		t.Fatalf("removed file still exists: %v", err)
	}
	remainingPath := filepath.Join(paths.Root, filepath.FromSlash(imported[0].RelativePath))
	if err := diary.New(db).TrashEntry(context.Background(), entryID); err != nil {
		t.Fatal(err)
	}
	if err := service.PermanentlyDeleteEntry(context.Background(), entryID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(remainingPath); !os.IsNotExist(err) {
		t.Fatalf("permanent entry deletion left media behind: %v", err)
	}
}

func TestBase64PasteRestartMissingFileAndMediaHandler(t *testing.T) {
	service, db, paths, entryID := testAttachmentService(t)
	attachment, err := service.ImportBase64(context.Background(), entryID, "pasted image.png", "image/png", "iVBORw0KGgppbWFnZQ==")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = database.Open(context.Background(), paths.Database)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service = New(db, paths)
	listed, err := service.ListAttachments(context.Background(), entryID)
	if err != nil || len(listed) != 1 || !listed[0].Available {
		t.Fatalf("attachments after restart = %+v, error = %v", listed, err)
	}
	handler := Handler(service, http.NotFoundHandler())
	request := httptest.NewRequest(http.MethodGet, attachment.URL, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.HasPrefix(response.Header().Get("Content-Type"), "image/png") {
		t.Fatalf("media response = %d %q", response.Code, response.Header().Get("Content-Type"))
	}
	path := filepath.Join(paths.Root, filepath.FromSlash(attachment.RelativePath))
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	listed, err = service.ListAttachments(context.Background(), entryID)
	if err != nil || listed[0].Available {
		t.Fatalf("missing attachment state = %+v, error = %v", listed, err)
	}
}

func TestLargeAndInterruptedImportsLeaveNoReferences(t *testing.T) {
	service, db, paths, entryID := testAttachmentService(t)
	defer db.Close()
	large := filepath.Join(t.TempDir(), "large.bin")
	file, err := os.Create(large)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(MaxAttachmentSize + 1); err != nil {
		t.Fatal(err)
	}
	file.Close()
	if _, err := service.ImportFiles(context.Background(), entryID, []string{large}); err == nil {
		t.Fatal("expected large attachment rejection")
	}
	if _, err := service.importReader(context.Background(), entryID, "broken.png", "image/png", &failingReader{}); err == nil {
		t.Fatal("expected interrupted import failure")
	}
	listed, err := service.ListAttachments(context.Background(), entryID)
	if err != nil || len(listed) != 0 {
		t.Fatalf("broken imports left metadata: %+v, %v", listed, err)
	}
	matches, err := filepath.Glob(filepath.Join(paths.Root, ".attachment-import-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("staging files remain: %v, %v", matches, err)
	}
}

type failingReader struct{ sent bool }

func (reader *failingReader) Read(buffer []byte) (int, error) {
	if reader.sent {
		return 0, errors.New("simulated interruption")
	}
	reader.sent = true
	return copy(buffer, []byte("partial")), nil
}

var _ io.Reader = (*failingReader)(nil)
