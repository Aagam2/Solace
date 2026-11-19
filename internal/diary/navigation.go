package diary

import (
	"context"
	"fmt"
	"sort"
	"time"
)

func (s *Service) CalendarMonth(ctx context.Context, month string) ([]CalendarDay, error) {
	start, end, err := monthBounds(month)
	if err != nil {
		return nil, err
	}
	entries, err := s.summaryRange(ctx, start, end, 0, "e.start_date, e.created_at")
	if err != nil {
		return nil, err
	}
	days := map[string]*CalendarDay{}
	for _, entry := range entries {
		entryStart := maxDate(entry.StartDate, start)
		entryEnd := minDate(entry.EndDate, end)
		for date := entryStart; date <= entryEnd; date = addDays(date, 1) {
			day := days[date]
			if day == nil {
				day = &CalendarDay{Date: date}
				days[date] = day
			}
			day.EntryCount++
			if day.Mood == "" && entry.Mood != "" {
				day.Mood = entry.Mood
			}
			day.HasAttachments = day.HasAttachments || entry.AttachmentCount > 0
		}
	}
	result := make([]CalendarDay, 0, len(days))
	for _, day := range days {
		result = append(result, *day)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Date < result[j].Date })
	return result, nil
}

func (s *Service) Timeline(ctx context.Context, options TimelineOptions) (TimelinePage, error) {
	if err := validateDateRange(options.StartDate, options.EndDate); err != nil {
		return TimelinePage{}, err
	}
	startTime, _ := time.Parse(time.DateOnly, options.StartDate)
	endTime, _ := time.Parse(time.DateOnly, options.EndDate)
	if endTime.Sub(startTime) > 370*24*time.Hour {
		return TimelinePage{}, fmt.Errorf("timeline range cannot exceed one year")
	}
	entries, err := s.summaryRange(ctx, options.StartDate, options.EndDate, 0, "e.start_date DESC, e.created_at DESC")
	if err != nil {
		return TimelinePage{}, err
	}
	occurrences := make([]TimelineOccurrence, 0, len(entries))
	for _, entry := range entries {
		start := maxDate(entry.StartDate, options.StartDate)
		end := minDate(entry.EndDate, options.EndDate)
		for date := start; date <= end; date = addDays(date, 1) {
			occurrences = append(occurrences, TimelineOccurrence{Date: date, Entry: entry})
		}
	}
	sort.SliceStable(occurrences, func(i, j int) bool {
		if occurrences[i].Date != occurrences[j].Date {
			return occurrences[i].Date > occurrences[j].Date
		}
		if occurrences[i].Entry.Pinned != occurrences[j].Entry.Pinned {
			return occurrences[i].Entry.Pinned
		}
		if occurrences[i].Entry.UpdatedAt != occurrences[j].Entry.UpdatedAt {
			return occurrences[i].Entry.UpdatedAt > occurrences[j].Entry.UpdatedAt
		}
		return occurrences[i].Entry.ID < occurrences[j].Entry.ID
	})
	limit := options.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	offset := options.Offset
	if offset < 0 {
		offset = 0
	}
	total := len(occurrences)
	if offset > total {
		offset = total
	}
	last := offset + limit
	if last > total {
		last = total
	}
	return TimelinePage{Occurrences: occurrences[offset:last], Total: total, HasMore: last < total}, nil
}

func (s *Service) TodayDashboard(ctx context.Context, today string) (TodayDashboard, error) {
	if _, err := time.Parse(time.DateOnly, today); err != nil {
		return TodayDashboard{}, fmt.Errorf("invalid dashboard date: %w", err)
	}
	todayEntries, err := s.summaryRange(ctx, today, today, 20, "is_pinned DESC, start_date DESC, updated_at DESC")
	if err != nil {
		return TodayDashboard{}, err
	}
	recentlyEdited, err := s.summaryQuery(ctx, `e.deleted_at IS NULL`, nil, 6, "e.updated_at DESC")
	if err != nil {
		return TodayDashboard{}, err
	}
	recentMemories, err := s.summaryQuery(ctx, `e.deleted_at IS NULL AND e.end_date < ?`, []any{today}, 6, "e.end_date DESC, e.updated_at DESC")
	if err != nil {
		return TodayDashboard{}, err
	}
	currentYear := today[:4]
	candidates, err := s.summaryQuery(ctx, `e.deleted_at IS NULL AND e.start_date < ?`, []any{currentYear + "-01-01"}, 0, "e.start_date DESC")
	if err != nil {
		return TodayDashboard{}, err
	}
	monthDay := today[5:]
	onThisDay := make([]EntrySummary, 0, 8)
	for _, entry := range candidates {
		startYear := entry.StartDate[:4]
		endYear := minDate(entry.EndDate[:4], fmt.Sprintf("%04d", mustYear(currentYear)-1))
		matched := false
		for year := mustYear(startYear); year <= mustYear(endYear); year++ {
			date := fmt.Sprintf("%04d-%s", year, monthDay)
			if validDate(date) && date >= entry.StartDate && date <= entry.EndDate {
				matched = true
				break
			}
		}
		if matched {
			onThisDay = append(onThisDay, entry)
		}
		if len(onThisDay) == 8 {
			break
		}
	}
	return TodayDashboard{Today: today, TodayEntries: todayEntries, RecentlyEdited: recentlyEdited, RecentMemories: recentMemories, OnThisDay: onThisDay}, nil
}

func (s *Service) summaryRange(ctx context.Context, start, end string, limit int, order string) ([]EntrySummary, error) {
	if err := validateDateRange(start, end); err != nil {
		return nil, err
	}
	return s.summaryQuery(ctx, `e.deleted_at IS NULL AND e.start_date <= ? AND e.end_date >= ?`, []any{end, start}, limit, order)
}

func (s *Service) summaryQuery(ctx context.Context, where string, args []any, limit int, order string) ([]EntrySummary, error) {
	query := `SELECT e.id, COALESCE(e.title,''), SUBSTR(e.content_text,1,180), e.start_date, e.end_date, e.updated_at,
		COALESCE(e.mood,''), e.is_favorite, e.is_pinned, e.is_draft,
		(SELECT COUNT(*) FROM attachments a WHERE a.entry_id=e.id)
		FROM entries e WHERE ` + where + ` ORDER BY ` + order
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list entry summaries: %w", err)
	}
	defer rows.Close()
	result := []EntrySummary{}
	for rows.Next() {
		var entry EntrySummary
		var favorite, pinned, draft int
		if err := rows.Scan(&entry.ID, &entry.Title, &entry.Preview, &entry.StartDate, &entry.EndDate, &entry.UpdatedAt,
			&entry.Mood, &favorite, &pinned, &draft, &entry.AttachmentCount); err != nil {
			return nil, err
		}
		entry.Favorite, entry.Pinned, entry.Draft = favorite == 1, pinned == 1, draft == 1
		result = append(result, entry)
	}
	return result, rows.Err()
}

func monthBounds(month string) (string, string, error) {
	parsed, err := time.Parse("2006-01", month)
	if err != nil {
		return "", "", fmt.Errorf("invalid month %q: %w", month, err)
	}
	start := parsed.Format(time.DateOnly)
	end := parsed.AddDate(0, 1, -1).Format(time.DateOnly)
	return start, end, nil
}

func addDays(date string, amount int) string {
	parsed, _ := time.Parse(time.DateOnly, date)
	return parsed.AddDate(0, 0, amount).Format(time.DateOnly)
}
func maxDate(left, right string) string {
	if left > right {
		return left
	}
	return right
}
func minDate(left, right string) string {
	if left < right {
		return left
	}
	return right
}
func validDate(value string) bool { _, err := time.Parse(time.DateOnly, value); return err == nil }
func mustYear(value string) int   { parsed, _ := time.Parse("2006", value); return parsed.Year() }
