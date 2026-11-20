package portable

import (
	"archive/zip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"solace/internal/attachments"
	"solace/internal/database"
	"solace/internal/diary"
	"solace/internal/platform"
)

type testJournal struct {
	service     *Service
	diary       *diary.Service
	attachments *attachments.Service
	paths       platform.Paths
}

func newTestJournal(t *testing.T) testJournal {
	t.Helper()
	paths := platform.PathsForRoot(filepath.Join(t.TempDir(), "solace"))
	if err := platform.EnsureDirectories(paths); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(context.Background(), paths.Database)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return testJournal{service: New(db, paths), diary: diary.New(db), attachments: attachments.New(db, paths), paths: paths}
}

func seedJournal(t *testing.T, journal testJournal, count int) []diary.Entry {
	t.Helper()
	ctx := context.Background()
	entries := make([]diary.Entry, 0, count)
	for index := 0; index < count; index++ {
		day := index%27 + 1
		start := fmt.Sprintf("2026-%02d-%02d", index%12+1, day)
		end := start
		if index%7 == 0 {
			end = fmt.Sprintf("2026-%02d-%02d", index%12+1, day+1)
		}
		title := fmt.Sprintf("Memory / duplicate %d", index%5)
		entry, err := journal.diary.CreateEntry(ctx, diary.EntryInput{Title: title, ContentJSON: `{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"A portable memory."}]}]}`, ContentText: "A portable memory.", StartDate: start, EndDate: end, Mood: "Happy", Favorite: index%4 == 0, Pinned: index%9 == 0, Draft: index%3 == 0, LocationText: "Somewhere", Tags: []string{"Travel", fmt.Sprintf("Group %d", index%3)}})
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, entry)
	}
	if len(entries) > 0 {
		files := []struct{ name, mime, data string }{
			{"photo.png", "image/png", base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\nportable"))},
			{"clip.mp4", "video/mp4", base64.StdEncoding.EncodeToString([]byte("portable video"))},
			{"notes.txt", "text/plain", base64.StdEncoding.EncodeToString([]byte("portable document"))},
		}
		for _, file := range files {
			if _, err := journal.attachments.ImportBase64(ctx, entries[0].ID, file.name, file.mime, file.data); err != nil {
				t.Fatal(err)
			}
		}
	}
	return entries
}

func TestReadableAndCompleteExports(t *testing.T) {
	journal := newTestJournal(t)
	entries := seedJournal(t, journal, 12)
	directory := t.TempDir()
	formats := []string{"markdown", "html", "json", "archive"}
	for _, format := range formats {
		extension := ".zip"
		if format == "json" {
			extension = ".json"
		}
		destination := filepath.Join(directory, format+extension)
		result, err := journal.service.Export(context.Background(), ExportRequest{Format: format, Scope: "entire", IncludeAttachments: true, Destination: destination})
		if err != nil {
			t.Fatalf("%s export: %v", format, err)
		}
		if result.EntryCount != 12 || result.AttachmentCount != 3 {
			t.Fatalf("%s result = %+v", format, result)
		}
		if info, err := os.Stat(destination); err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("%s file = %v, %v", format, info, err)
		}
	}
	var exported JournalExport
	data, err := os.ReadFile(filepath.Join(directory, "json.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &exported); err != nil {
		t.Fatal(err)
	}
	if len(exported.Entries) != 12 || len(exported.Attachments) != 3 || len(exported.SchemaVersions) != 8 {
		t.Fatalf("unexpected JSON export: %+v", exported)
	}

	markdownFiles := zipNames(t, filepath.Join(directory, "markdown.zip"))
	if !containsPrefix(markdownFiles, "entries/2026/") || !containsPrefix(markdownFiles, "media/images/") || !contains(markdownFiles, "README.md") {
		t.Fatalf("markdown files = %v", markdownFiles)
	}
	htmlFiles := zipNames(t, filepath.Join(directory, "html.zip"))
	if !contains(htmlFiles, "index.html") || !containsPrefix(htmlFiles, "entries/") {
		t.Fatalf("html files = %v", htmlFiles)
	}
	summary, err := journal.service.InspectArchive(context.Background(), filepath.Join(directory, "archive.zip"))
	if err != nil || !summary.Valid || summary.EntryCount != 12 || summary.AttachmentCount != 3 {
		t.Fatalf("archive summary = %+v, %v", summary, err)
	}

	dateExport := filepath.Join(directory, "date.json")
	if result, err := journal.service.Export(context.Background(), ExportRequest{Format: "json", Scope: "date", StartDate: "2026-01-02", EndDate: "2026-01-02", Destination: dateExport}); err != nil || result.EntryCount != 1 {
		t.Fatalf("overlap date export = %+v, %v", result, err)
	}
	selectedExport := filepath.Join(directory, "selected.json")
	if result, err := journal.service.Export(context.Background(), ExportRequest{Format: "json", Scope: "selected", EntryIDs: []string{entries[2].ID, entries[7].ID}, Destination: selectedExport}); err != nil || result.EntryCount != 2 {
		t.Fatalf("selected export = %+v, %v", result, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	cancelledDestination := filepath.Join(directory, "cancelled.zip")
	if _, err := journal.service.Export(cancelled, ExportRequest{Format: "archive", Scope: "entire", Destination: cancelledDestination}); err == nil {
		t.Fatal("cancelled export succeeded")
	}
	if _, err := os.Stat(cancelledDestination); !os.IsNotExist(err) {
		t.Fatalf("cancelled export left destination: %v", err)
	}
}

func TestCriticalRoundTripAndReplacementRecovery(t *testing.T) {
	source := newTestJournal(t)
	original := seedJournal(t, source, 100)
	updated, err := source.diary.UpdateEntry(context.Background(), original[0].ID, diary.EntryInput{Title: "Updated before export", ContentText: "Revision safety", StartDate: original[0].StartDate, EndDate: original[0].EndDate, Tags: original[0].Tags}, original[0].Version)
	if err != nil {
		t.Fatal(err)
	}
	original[0] = updated
	archive := filepath.Join(t.TempDir(), "round-trip.zip")
	if _, err := source.service.Export(context.Background(), ExportRequest{Format: "archive", Scope: "entire", IncludeAttachments: true, Destination: archive}); err != nil {
		t.Fatal(err)
	}

	target := newTestJournal(t)
	result, err := target.service.RestoreArchive(context.Background(), archive, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.EntryCount != 100 || result.AttachmentCount != 3 || result.RecoveryBackup != "" {
		t.Fatalf("restore result = %+v", result)
	}
	restored, err := target.diary.ListEntries(context.Background(), diary.ListOptions{Limit: 200})
	if err != nil {
		t.Fatal(err)
	}
	if len(restored) != len(original) {
		t.Fatalf("restored count = %d", len(restored))
	}
	originalIDs, restoredIDs := ids(original), ids(restored)
	if strings.Join(originalIDs, ",") != strings.Join(restoredIDs, ",") {
		t.Fatalf("entry IDs differ")
	}
	attachmentsAfter, err := target.attachments.ListAttachments(context.Background(), original[0].ID)
	if err != nil || len(attachmentsAfter) != 3 {
		t.Fatalf("restored attachments = %+v, %v", attachmentsAfter, err)
	}
	for _, attachment := range attachmentsAfter {
		data, err := os.ReadFile(filepath.Join(target.paths.Root, filepath.FromSlash(attachment.RelativePath)))
		if err != nil || !strings.HasPrefix(attachment.Checksum, "sha256:") || len(data) == 0 {
			t.Fatalf("restored media = %v, %v", len(data), err)
		}
	}
	revisionsAfter, err := target.diary.ListRevisions(context.Background(), original[0].ID)
	if err != nil || len(revisionsAfter) != 1 || revisionsAfter[0].Title == updated.Title {
		t.Fatalf("restored revisions = %+v, %v", revisionsAfter, err)
	}

	if _, err := target.diary.CreateEntry(context.Background(), diary.EntryInput{Title: "New local entry", StartDate: "2026-12-20"}); err != nil {
		t.Fatal(err)
	}
	if _, err := target.service.RestoreArchive(context.Background(), archive, false); err == nil {
		t.Fatal("non-empty restore without replacement succeeded")
	}
	replaced, err := target.service.RestoreArchive(context.Background(), archive, true)
	if err != nil {
		t.Fatal(err)
	}
	if replaced.RecoveryBackup == "" {
		t.Fatal("replacement did not create a recovery backup")
	}
	if summary, err := target.service.InspectArchive(context.Background(), replaced.RecoveryBackup); err != nil || !summary.Valid {
		t.Fatalf("recovery backup = %+v, %v", summary, err)
	}
}

func TestBackupRetentionEmptyExportAndInvalidArchives(t *testing.T) {
	journal := newTestJournal(t)
	empty := filepath.Join(t.TempDir(), "empty.zip")
	result, err := journal.service.Export(context.Background(), ExportRequest{Format: "markdown", Scope: "entire", Destination: empty})
	if err != nil || result.EntryCount != 0 {
		t.Fatalf("empty export = %+v, %v", result, err)
	}

	backupDir := t.TempDir()
	settings := BackupSettings{Schedule: "daily", Destination: backupDir, Retention: 2, IncludeAttachments: true}
	if err := journal.service.SaveBackupSettings(context.Background(), settings); err != nil {
		t.Fatal(err)
	}
	seedJournal(t, journal, 2)
	for range 3 {
		if _, err := journal.service.CreateBackup(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	matches, _ := filepath.Glob(filepath.Join(backupDir, "solace-backup-*.zip"))
	if len(matches) > 2 {
		t.Fatalf("retained backups = %d", len(matches))
	}

	unsafe := filepath.Join(t.TempDir(), "unsafe.zip")
	file, err := os.Create(unsafe)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entry, _ := writer.Create("../outside")
	entry.Write([]byte("unsafe"))
	writer.Close()
	file.Close()
	if _, err := journal.service.InspectArchive(context.Background(), unsafe); err == nil {
		t.Fatal("unsafe archive path accepted")
	}
	var count int
	if err := journal.service.db.QueryRow(`SELECT COUNT(*) FROM entries`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("invalid restore changed journal: %d, %v", count, err)
	}
}

func zipNames(t *testing.T, archive string) []string {
	t.Helper()
	reader, err := zip.OpenReader(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	result := make([]string, len(reader.File))
	for index, file := range reader.File {
		result[index] = file.Name
	}
	sort.Strings(result)
	return result
}
func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
func containsPrefix(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}
func ids(entries []diary.Entry) []string {
	result := make([]string, len(entries))
	for index, entry := range entries {
		result[index] = entry.ID
	}
	sort.Strings(result)
	return result
}
