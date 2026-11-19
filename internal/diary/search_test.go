package diary

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"solace/internal/database"
)

func TestSearchLifecycleMultipleWordsAndSafeSpecialCharacters(t *testing.T) {
	service := testService(t)
	ctx := context.Background()
	bookstore, err := service.CreateEntry(ctx, EntryInput{Title: "Bookstore", ContentText: "Went to the bookstore.", StartDate: "2026-11-14", Tags: []string{"Books"}})
	if err != nil {
		t.Fatal(err)
	}
	interview, err := service.CreateEntry(ctx, EntryInput{Title: "Career", ContentText: "Had an interview today.", StartDate: "2026-11-15"})
	if err != nil {
		t.Fatal(err)
	}
	technical, err := service.CreateEntry(ctx, EntryInput{Title: "Preparation", ContentText: "Prepared for a technical interview.", StartDate: "2026-11-16"})
	if err != nil {
		t.Fatal(err)
	}

	page, err := service.Search(ctx, SearchOptions{Query: "interv", Limit: 20})
	if err != nil || page.Total != 2 {
		t.Fatalf("partial search = %+v, %v", page, err)
	}
	page, err = service.Search(ctx, SearchOptions{Query: "technical interview", Limit: 20})
	if err != nil || page.Total != 1 || page.Results[0].Entry.ID != technical.ID {
		t.Fatalf("multiple words = %+v, %v", page, err)
	}
	if _, err := service.Search(ctx, SearchOptions{Query: `" OR * : ( ) -`, Limit: 20}); err != nil {
		t.Fatalf("special query: %v", err)
	}
	page, err = service.Search(ctx, SearchOptions{Query: "missingword", Limit: 20})
	if err != nil || page.Total != 0 {
		t.Fatalf("no result = %+v, %v", page, err)
	}

	updatedInput := EntryInput{Title: "Career", ContentText: "Accepted the position.", StartDate: interview.StartDate, EndDate: interview.EndDate, Draft: interview.Draft}
	if _, err := service.UpdateEntry(ctx, interview.ID, updatedInput, interview.Version); err != nil {
		t.Fatal(err)
	}
	page, _ = service.Search(ctx, SearchOptions{Query: "interview", Limit: 20})
	if page.Total != 1 || page.Results[0].Entry.ID != technical.ID {
		t.Fatalf("search after edit = %+v", page)
	}
	if err := service.TrashEntry(ctx, technical.ID); err != nil {
		t.Fatal(err)
	}
	page, _ = service.Search(ctx, SearchOptions{Query: "interview", Limit: 20})
	if page.Total != 0 {
		t.Fatalf("search after trash = %+v", page)
	}
	if err := service.TrashEntry(ctx, bookstore.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.PermanentlyDeleteEntry(ctx, bookstore.ID); err != nil {
		t.Fatal(err)
	}
}

func TestSearchFiltersTagsCaptionsAndOverlappingRanges(t *testing.T) {
	service := testService(t)
	ctx := context.Background()
	entry, err := service.CreateEntry(ctx, EntryInput{Title: "Weekend", ContentText: "Mountain train", StartDate: "2026-11-15", EndDate: "2026-11-20", Mood: "Happy", Favorite: true, Draft: false, Tags: []string{"Travel", "Friends"}})
	if err != nil {
		t.Fatal(err)
	}
	other, err := service.CreateEntry(ctx, EntryInput{Title: "Work note", ContentText: "Mountain planning", StartDate: "2026-10-01", Draft: true, Tags: []string{"Work"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.db.ExecContext(ctx, `INSERT INTO attachments(id,entry_id,original_filename,stored_filename,relative_path,mime_type,file_size,created_at,caption,checksum) VALUES('image-search',?,'photo.jpg','photo.jpg','media/images/photo.jpg','image/jpeg',1,?,'Sunset platform','sha256:test')`, entry.ID, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.db.ExecContext(ctx, `INSERT INTO attachments(id,entry_id,original_filename,stored_filename,relative_path,mime_type,file_size,created_at,caption,checksum) VALUES('video-search',?,'clip.mp4','clip.mp4','media/videos/clip.mp4','video/mp4',1,?,'Train arriving','sha256:test')`, entry.ID, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		t.Fatal(err)
	}

	page, err := service.Search(ctx, SearchOptions{Query: "sunset", StartDate: "2026-11-16", EndDate: "2026-11-18", Tags: []string{"Travel", "Friends"}, Mood: "Happy", FavoritesOnly: true, HasImages: true, HasVideos: true, DraftStatus: "published", Limit: 20})
	if err != nil || page.Total != 1 || page.Results[0].Entry.ID != entry.ID {
		t.Fatalf("combined filters = %+v, %v", page, err)
	}
	page, _ = service.Search(ctx, SearchOptions{Query: "mountain", Tags: []string{"Work"}, DraftStatus: "draft", Limit: 20})
	if page.Total != 1 || page.Results[0].Entry.ID != other.ID {
		t.Fatalf("tag draft filter = %+v", page)
	}
	page, _ = service.Search(ctx, SearchOptions{StartDate: "2026-11-18", EndDate: "2026-11-22", Limit: 20})
	if page.Total != 1 || page.Results[0].Entry.ID != entry.ID {
		t.Fatalf("overlap filter = %+v", page)
	}
}

func TestTagManagementRebuildAndRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "solace.db")
	db, err := database.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	service := New(db)
	entry, err := service.CreateEntry(ctx, EntryInput{Title: "Shelf", ContentText: "A new book", StartDate: "2024-01-02", Tags: []string{"Books"}})
	if err != nil {
		t.Fatal(err)
	}
	tags, err := service.ListTags(ctx)
	if err != nil || len(tags) != 1 || tags[0].EntryCount != 1 {
		t.Fatalf("tags = %+v, %v", tags, err)
	}
	if err := service.RenameTag(ctx, tags[0].ID, "Reading"); err != nil {
		t.Fatal(err)
	}
	page, _ := service.Search(ctx, SearchOptions{Query: "reading", Limit: 20})
	if page.Total != 1 || page.Results[0].Entry.ID != entry.ID {
		t.Fatalf("search renamed tag = %+v", page)
	}
	if _, err := service.db.ExecContext(ctx, `DELETE FROM entry_search`); err != nil {
		t.Fatal(err)
	}
	if err := service.RebuildSearchIndex(ctx); err != nil {
		t.Fatal(err)
	}
	page, _ = service.Search(ctx, SearchOptions{Query: "book", Limit: 20})
	if page.Total != 1 {
		t.Fatalf("rebuilt search = %+v", page)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = database.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service = New(db)
	page, err = service.Search(ctx, SearchOptions{Query: "book", Limit: 20})
	if err != nil || page.Total != 1 {
		t.Fatalf("search after restart = %+v, %v", page, err)
	}
	tags, _ = service.ListTags(ctx)
	if err := service.DeleteTag(ctx, tags[0].ID); err != nil {
		t.Fatal(err)
	}
	page, _ = service.Search(ctx, SearchOptions{Query: "reading", Limit: 20})
	if page.Total != 0 {
		t.Fatalf("search deleted tag = %+v", page)
	}
	created, err := service.CreateTag(ctx, "Health")
	if err != nil || created.Name != "Health" {
		t.Fatalf("create tag = %+v, %v", created, err)
	}
}

func TestSearchThousandsOfEntries(t *testing.T) {
	service := testService(t)
	ctx := context.Background()
	tx, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	statement, err := tx.PrepareContext(ctx, `INSERT INTO entries(id,title,content_json,content_text,start_date,end_date,created_at,updated_at,is_draft) VALUES(?,?,?,?,?,?,?,?,0)`)
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 5000; index++ {
		content := "ordinary memory"
		if index%500 == 0 {
			content = "needle interview memory"
		}
		id := fmt.Sprintf("search-entry-%05d", index)
		if _, err := statement.ExecContext(ctx, id, "Synthetic", `{"type":"doc","content":[]}`, content, "2026-01-01", "2026-01-01", "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z"); err != nil {
			t.Fatal(err)
		}
	}
	statement.Close()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	page, err := service.Search(ctx, SearchOptions{Query: "interv", Limit: 5})
	if err != nil || page.Total != 10 || len(page.Results) != 5 || !page.HasMore {
		t.Fatalf("large search = %+v, %v", page, err)
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("large search took %v", elapsed)
	}
}
