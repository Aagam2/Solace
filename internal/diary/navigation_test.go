package diary

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"solace/internal/database"
)

func TestCalendarAndTimelineExpandRangesAcrossMonthsAndYears(t *testing.T) {
	service := testService(t)
	ctx := context.Background()
	inputs := []EntryInput{
		{Title: "Month bridge", StartDate: "2026-01-31", EndDate: "2026-02-02", Mood: "Happy"},
		{Title: "Year bridge", StartDate: "2025-12-31", EndDate: "2026-01-01"},
		{Title: "Same day one", StartDate: "2026-02-01"},
		{Title: "Same day two", StartDate: "2026-02-01"},
	}
	created := map[string]Entry{}
	for _, input := range inputs {
		entry, err := service.CreateEntry(ctx, input)
		if err != nil {
			t.Fatal(err)
		}
		created[input.Title] = entry
	}
	days, err := service.CalendarMonth(ctx, "2026-02")
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, day := range days {
		counts[day.Date] = day.EntryCount
	}
	if counts["2026-02-01"] != 3 || counts["2026-02-02"] != 1 {
		t.Fatalf("calendar counts = %v", counts)
	}
	page, err := service.Timeline(ctx, TimelineOptions{StartDate: "2025-12-01", EndDate: "2026-02-28", Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	seenMonthBridge := map[string]string{}
	seenYearBridge := map[string]string{}
	for _, occurrence := range page.Occurrences {
		if occurrence.Entry.Title == "Month bridge" {
			seenMonthBridge[occurrence.Date] = occurrence.Entry.ID
		}
		if occurrence.Entry.Title == "Year bridge" {
			seenYearBridge[occurrence.Date] = occurrence.Entry.ID
		}
	}
	if len(seenMonthBridge) != 3 || seenMonthBridge["2026-01-31"] != created["Month bridge"].ID || len(seenYearBridge) != 2 {
		t.Fatalf("range occurrences: month=%v year=%v", seenMonthBridge, seenYearBridge)
	}
	empty, err := service.Timeline(ctx, TimelineOptions{StartDate: "1990-01-01", EndDate: "1990-12-31", Limit: 100})
	if err != nil || empty.Total != 0 {
		t.Fatalf("empty year = %+v, %v", empty, err)
	}
}

func TestTodayDashboardOnThisDayAndPinnedSummary(t *testing.T) {
	service := testService(t)
	ctx := context.Background()
	historical, err := service.CreateEntry(ctx, EntryInput{Title: "Old trip", StartDate: "2024-11-16", EndDate: "2024-11-18", Favorite: true})
	if err != nil {
		t.Fatal(err)
	}
	today, err := service.CreateEntry(ctx, EntryInput{Title: "Today", StartDate: "2026-11-17", Pinned: true})
	if err != nil {
		t.Fatal(err)
	}
	dashboard, err := service.TodayDashboard(ctx, "2026-11-17")
	if err != nil {
		t.Fatal(err)
	}
	if len(dashboard.TodayEntries) != 1 || dashboard.TodayEntries[0].ID != today.ID || !dashboard.TodayEntries[0].Pinned {
		t.Fatalf("today dashboard entries = %+v", dashboard.TodayEntries)
	}
	found := false
	for _, entry := range dashboard.OnThisDay {
		found = found || entry.ID == historical.ID
	}
	if !found {
		t.Fatalf("On This Day did not include spanning entry: %+v", dashboard.OnThisDay)
	}
}

func TestNavigationWithTenThousandEntries(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "solace.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	entryStatement, err := tx.PrepareContext(ctx, `INSERT INTO entries
		(id,title,content_json,content_text,start_date,end_date,created_at,updated_at,is_draft)
		VALUES(?,?,?,?,?,?,?,?,0)`)
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 10000; index++ {
		date := fmt.Sprintf("2026-01-%02d", index%28+1)
		id := fmt.Sprintf("entry-%05d", index)
		if _, err := entryStatement.ExecContext(ctx, id, "Synthetic", `{"type":"doc","content":[]}`, "memory", date, date, "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z"); err != nil {
			t.Fatal(err)
		}
	}
	entryStatement.Close()
	for index := 0; index < 2000; index++ {
		if _, err := tx.ExecContext(ctx, `INSERT INTO tags(id,name,created_at) VALUES(?,?,?)`, fmt.Sprintf("tag-%d", index), fmt.Sprintf("Tag %d", index), "2026-01-01T00:00:00Z"); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO attachments(id,entry_id,original_filename,stored_filename,relative_path,mime_type,file_size,created_at,checksum) VALUES(?,?,?,?,?,?,?,?,?)`,
			fmt.Sprintf("attachment-%d", index), fmt.Sprintf("entry-%05d", index), "photo.jpg", fmt.Sprintf("%d.jpg", index), fmt.Sprintf("media/images/%d.jpg", index), "image/jpeg", 10, "2026-01-01T00:00:00Z", "sha256:test"); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	started := time.Now()
	page, err := New(db).Timeline(ctx, TimelineOptions{StartDate: "2026-01-01", EndDate: "2026-01-31", Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 10000 || len(page.Occurrences) != 100 || !page.HasMore {
		t.Fatalf("large timeline page = total %d, rows %d, more %v", page.Total, len(page.Occurrences), page.HasMore)
	}
	days, err := New(db).CalendarMonth(ctx, "2026-01")
	if err != nil {
		t.Fatal(err)
	}
	if len(days) != 28 || days[0].EntryCount == 0 {
		t.Fatalf("large calendar = %d days, first = %+v", len(days), days[0])
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("large timeline and calendar took %v", elapsed)
	}
}
