package diary

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"solace/internal/database"
)

func testService(t *testing.T) *Service {
	t.Helper()
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "solace.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return New(db)
}

func TestCreateSingleMultiDayAndOverlappingEntries(t *testing.T) {
	service := testService(t)
	ctx := context.Background()
	inputs := []EntryInput{
		{Title: "A", ContentText: "single", StartDate: "2026-11-15", Draft: false},
		{Title: "B", ContentText: "weekend", StartDate: "2026-11-15", EndDate: "2026-11-16", Tags: []string{"Travel", "Friends", "travel"}},
		{Title: "C", StartDate: "2026-11-16", Draft: true},
		{Title: "D", StartDate: "2026-11-15", EndDate: "2026-11-20"},
	}
	ids := map[string]bool{}
	for _, input := range inputs {
		entry, err := service.CreateEntry(ctx, input)
		if err != nil {
			t.Fatal(err)
		}
		if ids[entry.ID] {
			t.Fatal("duplicate entry ID")
		}
		ids[entry.ID] = true
	}
	entries, err := service.ListEntries(ctx, ListOptions{StartDate: "2026-11-16", EndDate: "2026-11-16"})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("entries covering Nov 16 = %d, want 3", len(entries))
	}
	var tagged Entry
	for _, entry := range entries {
		if entry.Title == "B" {
			tagged = entry
		}
	}
	if len(tagged.Tags) != 2 {
		t.Fatalf("normalized tags = %v", tagged.Tags)
	}
}

func TestInvalidDateRange(t *testing.T) {
	_, err := testService(t).CreateEntry(context.Background(), EntryInput{StartDate: "2026-11-20", EndDate: "2026-11-15"})
	if !errors.Is(err, ErrInvalidDates) {
		t.Fatalf("error = %v, want ErrInvalidDates", err)
	}
}

func TestUpdateRejectsStaleVersion(t *testing.T) {
	service := testService(t)
	ctx := context.Background()
	entry, err := service.CreateEntry(ctx, EntryInput{Title: "Before", StartDate: "2026-11-15", Draft: true})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.UpdateEntry(ctx, entry.ID, EntryInput{Title: "After", StartDate: entry.StartDate, Draft: false, Favorite: true}, entry.Version)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "After" || updated.Draft || !updated.Favorite || updated.Version != 2 {
		t.Fatalf("unexpected update: %+v", updated)
	}
	_, err = service.UpdateEntry(ctx, entry.ID, EntryInput{Title: "Stale", StartDate: entry.StartDate}, entry.Version)
	if !errors.Is(err, ErrStaleVersion) {
		t.Fatalf("error = %v, want ErrStaleVersion", err)
	}
}

func TestBoundedRevisionHistoryAndRecovery(t *testing.T) {
	service := testService(t)
	ctx := context.Background()
	entry, err := service.CreateEntry(ctx, EntryInput{Title: "Original", ContentText: "First version", StartDate: "2026-11-18", Tags: []string{"Private"}})
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 24; index++ {
		entry, err = service.UpdateEntry(ctx, entry.ID, EntryInput{Title: fmt.Sprintf("Version %d", index+1), ContentText: fmt.Sprintf("Body %d", index+1), StartDate: entry.StartDate, Tags: []string{"Private"}}, entry.Version)
		if err != nil {
			t.Fatal(err)
		}
	}
	revisions, err := service.ListRevisions(ctx, entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(revisions) != revisionLimit {
		t.Fatalf("revision count = %d, want %d", len(revisions), revisionLimit)
	}
	oldest := revisions[len(revisions)-1]
	restored, err := service.RestoreRevision(ctx, entry.ID, oldest.ID, entry.Version)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Title != oldest.Title || restored.ContentText != oldest.ContentText || restored.Version != entry.Version+1 {
		t.Fatalf("restored entry = %+v, revision = %+v", restored, oldest)
	}
	revisions, err = service.ListRevisions(ctx, entry.ID)
	if err != nil || len(revisions) != revisionLimit || revisions[0].Title != entry.Title {
		t.Fatalf("history after restore = %+v, %v", revisions, err)
	}
}

func TestTrashRestoreAndPermanentDelete(t *testing.T) {
	service := testService(t)
	ctx := context.Background()
	entry, err := service.CreateEntry(ctx, EntryInput{StartDate: "2026-11-15"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.TrashEntry(ctx, entry.ID); err != nil {
		t.Fatal(err)
	}
	active, err := service.ListEntries(ctx, ListOptions{})
	if err != nil || len(active) != 0 {
		t.Fatalf("active entries = %v, error = %v", active, err)
	}
	trash, err := service.ListEntries(ctx, ListOptions{IncludeTrash: true})
	if err != nil || len(trash) != 1 {
		t.Fatalf("trash entries = %v, error = %v", trash, err)
	}
	if err := service.RestoreEntry(ctx, entry.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.TrashEntry(ctx, entry.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.PermanentlyDeleteEntry(ctx, entry.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetEntry(ctx, entry.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetEntry error = %v", err)
	}
}

func TestDuplicateIsIndependentDraft(t *testing.T) {
	service := testService(t)
	ctx := context.Background()
	original, err := service.CreateEntry(ctx, EntryInput{Title: "Trip", ContentText: "Memory", StartDate: "2026-11-15", EndDate: "2026-11-16", Favorite: true, Tags: []string{"Travel"}})
	if err != nil {
		t.Fatal(err)
	}
	copy, err := service.DuplicateEntry(ctx, original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if copy.ID == original.ID || copy.Title != "Trip (copy)" || !copy.Draft || copy.Favorite {
		t.Fatalf("unexpected duplicate: %+v", copy)
	}
	if _, err := service.UpdateEntry(ctx, copy.ID, EntryInput{Title: "Changed", StartDate: copy.StartDate, EndDate: copy.EndDate}, copy.Version); err != nil {
		t.Fatal(err)
	}
	unchanged, err := service.GetEntry(ctx, original.ID)
	if err != nil || unchanged.Title != "Trip" {
		t.Fatalf("original changed: %+v, %v", unchanged, err)
	}
}

func TestRetroactiveMonthAndYearBoundaryFiltering(t *testing.T) {
	service := testService(t)
	ctx := context.Background()
	inputs := []EntryInput{
		{Title: "Old memory", StartDate: "1999-12-31"},
		{Title: "New year trip", StartDate: "2025-12-30", EndDate: "2026-01-02"},
		{Title: "February", StartDate: "2026-02-01"},
	}
	for _, input := range inputs {
		if _, err := service.CreateEntry(ctx, input); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := service.ListEntries(ctx, ListOptions{StartDate: "2026-01-01", EndDate: "2026-01-31"})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Title != "New year trip" {
		t.Fatalf("January entries = %+v", entries)
	}
	entries, err = service.ListEntries(ctx, ListOptions{StartDate: "1999-12-31", EndDate: "1999-12-31"})
	if err != nil || len(entries) != 1 || entries[0].Title != "Old memory" {
		t.Fatalf("retroactive entries = %+v, error = %v", entries, err)
	}
}

func TestEntryPersistsAcrossDatabaseRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "solace.db")
	db, err := database.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	created, err := New(db).CreateEntry(ctx, EntryInput{
		Title: "Persistent page", ContentText: "Still here", StartDate: "2026-11-17", Tags: []string{"Personal"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = database.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	reloaded, err := New(db).GetEntry(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Title != created.Title || reloaded.ContentText != created.ContentText || len(reloaded.Tags) != 1 {
		t.Fatalf("reloaded entry = %+v", reloaded)
	}
}
