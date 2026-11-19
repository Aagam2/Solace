package diary

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound     = errors.New("entry not found")
	ErrStaleVersion = errors.New("entry was changed by another operation")
	ErrInvalidDates = errors.New("start date must be on or before end date")
)

type Service struct{ db *sql.DB }

func New(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) CreateEntry(ctx context.Context, input EntryInput) (Entry, error) {
	input, err := validateInput(input)
	if err != nil {
		return Entry{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	entry := Entry{
		ID: uuid.NewString(), Title: strings.TrimSpace(input.Title), ContentJSON: input.ContentJSON,
		ContentText: input.ContentText, StartDate: input.StartDate, EndDate: input.EndDate,
		CreatedAt: now, UpdatedAt: now, Mood: strings.TrimSpace(input.Mood), Favorite: input.Favorite,
		Pinned: input.Pinned, Draft: input.Draft, LocationText: strings.TrimSpace(input.LocationText), DisplayOrder: input.DisplayOrder,
		Version: 1, Tags: normalizeTags(input.Tags),
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Entry{}, fmt.Errorf("begin entry creation: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO entries
		(id, title, content_json, content_text, start_date, end_date, created_at, updated_at, mood, is_favorite, is_pinned, is_draft, location_text, display_order, version)
		VALUES (?, NULLIF(?, ''), ?, ?, ?, ?, ?, ?, NULLIF(?, ''), ?, ?, ?, NULLIF(?, ''), ?, 1)`,
		entry.ID, entry.Title, entry.ContentJSON, entry.ContentText, entry.StartDate, entry.EndDate,
		entry.CreatedAt, entry.UpdatedAt, entry.Mood, entry.Favorite, entry.Pinned, entry.Draft, entry.LocationText, entry.DisplayOrder); err != nil {
		return Entry{}, fmt.Errorf("create entry: %w", err)
	}
	if err := replaceTags(ctx, tx, entry.ID, entry.Tags, now); err != nil {
		return Entry{}, err
	}
	if err := tx.Commit(); err != nil {
		return Entry{}, fmt.Errorf("commit entry creation: %w", err)
	}
	return entry, nil
}

func (s *Service) GetEntry(ctx context.Context, id string) (Entry, error) {
	return getEntry(ctx, s.db, id)
}

func (s *Service) UpdateEntry(ctx context.Context, id string, input EntryInput, expectedVersion int64) (Entry, error) {
	input, err := validateInput(input)
	if err != nil {
		return Entry{}, err
	}
	if expectedVersion < 1 {
		return Entry{}, fmt.Errorf("expected version must be positive")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	tags := normalizeTags(input.Tags)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Entry{}, fmt.Errorf("begin entry update: %w", err)
	}
	defer tx.Rollback()
	if err := recordRevision(ctx, tx, id, expectedVersion, now); err != nil {
		return Entry{}, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE entries SET
		title=NULLIF(?, ''), content_json=?, content_text=?, start_date=?, end_date=?, updated_at=?, mood=NULLIF(?, ''),
		is_favorite=?, is_pinned=?, is_draft=?, location_text=NULLIF(?, ''), display_order=?, version=version+1
		WHERE id=? AND version=? AND deleted_at IS NULL`,
		strings.TrimSpace(input.Title), input.ContentJSON, input.ContentText, input.StartDate, input.EndDate,
		now, strings.TrimSpace(input.Mood), input.Favorite, input.Pinned, input.Draft, strings.TrimSpace(input.LocationText),
		input.DisplayOrder, id, expectedVersion)
	if err != nil {
		return Entry{}, fmt.Errorf("update entry: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return Entry{}, fmt.Errorf("inspect entry update: %w", err)
	}
	if changed == 0 {
		var exists int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM entries WHERE id=? AND deleted_at IS NULL`, id).Scan(&exists); err != nil {
			return Entry{}, err
		}
		if exists == 0 {
			return Entry{}, ErrNotFound
		}
		return Entry{}, ErrStaleVersion
	}
	if err := replaceTags(ctx, tx, id, tags, now); err != nil {
		return Entry{}, err
	}
	if err := tx.Commit(); err != nil {
		return Entry{}, fmt.Errorf("commit entry update: %w", err)
	}
	return s.GetEntry(ctx, id)
}

func (s *Service) ListEntries(ctx context.Context, options ListOptions) ([]Entry, error) {
	where := []string{"1=1"}
	args := make([]any, 0, 8)
	if options.IncludeTrash {
		where = append(where, "deleted_at IS NOT NULL")
	} else {
		where = append(where, "deleted_at IS NULL")
	}
	if options.StartDate != "" || options.EndDate != "" {
		start, end := options.StartDate, options.EndDate
		if start == "" {
			start = end
		}
		if end == "" {
			end = start
		}
		if err := validateDateRange(start, end); err != nil {
			return nil, err
		}
		where = append(where, "start_date <= ? AND end_date >= ?")
		args = append(args, end, start)
	}
	if options.FavoritesOnly {
		where = append(where, "is_favorite=1")
	}
	if options.DraftsOnly {
		where = append(where, "is_draft=1")
	}
	order := map[string]string{
		"oldest": "start_date ASC, created_at ASC", "updated": "updated_at DESC",
		"favorites": "is_favorite DESC, start_date DESC, created_at DESC",
	}[options.Sort]
	if order == "" {
		order = "start_date DESC, created_at DESC"
	}
	limit := options.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}
	if options.Offset < 0 {
		options.Offset = 0
	}
	args = append(args, limit, options.Offset)
	query := `SELECT id, COALESCE(title,''), content_json, content_text, start_date, end_date, created_at, updated_at,
		COALESCE(mood,''), is_favorite, is_pinned, is_draft, COALESCE(location_text,''), display_order, COALESCE(deleted_at,''), version
		FROM entries WHERE ` + strings.Join(where, " AND ") + ` ORDER BY ` + order + ` LIMIT ? OFFSET ?`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}
	defer rows.Close()
	entries := make([]Entry, 0)
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close entry list: %w", err)
	}
	for i := range entries {
		entries[i].Tags, err = loadTags(ctx, s.db, entries[i].ID)
		if err != nil {
			return nil, err
		}
	}
	return entries, nil
}

func (s *Service) TrashEntry(ctx context.Context, id string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := s.db.ExecContext(ctx, `UPDATE entries SET deleted_at=?, updated_at=?, version=version+1 WHERE id=? AND deleted_at IS NULL`, now, now, id)
	return mutationResult("trash entry", result, err)
}

func (s *Service) RestoreEntry(ctx context.Context, id string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := s.db.ExecContext(ctx, `UPDATE entries SET deleted_at=NULL, updated_at=?, version=version+1 WHERE id=? AND deleted_at IS NOT NULL`, now, id)
	return mutationResult("restore entry", result, err)
}

func (s *Service) PermanentlyDeleteEntry(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM entries WHERE id=? AND deleted_at IS NOT NULL`, id)
	return mutationResult("permanently delete entry", result, err)
}

func (s *Service) DuplicateEntry(ctx context.Context, id string) (Entry, error) {
	entry, err := s.GetEntry(ctx, id)
	if err != nil {
		return Entry{}, err
	}
	return s.CreateEntry(ctx, EntryInput{
		Title: duplicateTitle(entry.Title), ContentJSON: entry.ContentJSON, ContentText: entry.ContentText, StartDate: entry.StartDate, EndDate: entry.EndDate,
		Mood: entry.Mood, Favorite: false, Pinned: false, Draft: true, LocationText: entry.LocationText,
		DisplayOrder: entry.DisplayOrder, Tags: entry.Tags,
	})
}

type scanner interface{ Scan(dest ...any) error }

func scanEntry(row scanner) (Entry, error) {
	var entry Entry
	var favorite, pinned, draft int
	if err := row.Scan(&entry.ID, &entry.Title, &entry.ContentJSON, &entry.ContentText, &entry.StartDate, &entry.EndDate,
		&entry.CreatedAt, &entry.UpdatedAt, &entry.Mood, &favorite, &pinned, &draft, &entry.LocationText,
		&entry.DisplayOrder, &entry.DeletedAt, &entry.Version); err != nil {
		return Entry{}, fmt.Errorf("scan entry: %w", err)
	}
	entry.Favorite, entry.Pinned, entry.Draft = favorite == 1, pinned == 1, draft == 1
	entry.Tags = []string{}
	return entry, nil
}

func getEntry(ctx context.Context, db *sql.DB, id string) (Entry, error) {
	entry, err := scanEntry(db.QueryRowContext(ctx, `SELECT id, COALESCE(title,''), content_json, content_text, start_date, end_date,
		created_at, updated_at, COALESCE(mood,''), is_favorite, is_pinned, is_draft, COALESCE(location_text,''), display_order,
		COALESCE(deleted_at,''), version FROM entries WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Entry{}, ErrNotFound
	}
	if err != nil {
		return Entry{}, err
	}
	entry.Tags, err = loadTags(ctx, db, id)
	return entry, err
}

func validateInput(input EntryInput) (EntryInput, error) {
	if input.EndDate == "" {
		input.EndDate = input.StartDate
	}
	if err := validateDateRange(input.StartDate, input.EndDate); err != nil {
		return EntryInput{}, err
	}
	if strings.TrimSpace(input.ContentJSON) == "" {
		input.ContentJSON = contentDocument(input.ContentText)
	} else {
		var document struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(input.ContentJSON), &document); err != nil || document.Type != "doc" {
			return EntryInput{}, fmt.Errorf("invalid rich-text document")
		}
	}
	return input, nil
}

func validateDateRange(start, end string) error {
	startDate, err := time.Parse(time.DateOnly, start)
	if err != nil {
		return fmt.Errorf("invalid start date %q: %w", start, err)
	}
	endDate, err := time.Parse(time.DateOnly, end)
	if err != nil {
		return fmt.Errorf("invalid end date %q: %w", end, err)
	}
	if startDate.After(endDate) {
		return ErrInvalidDates
	}
	return nil
}

func contentDocument(text string) string {
	paragraphs := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	content := make([]map[string]any, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		node := map[string]any{"type": "paragraph"}
		if paragraph != "" {
			node["content"] = []map[string]string{{"type": "text", "text": paragraph}}
		}
		content = append(content, node)
	}
	document, _ := json.Marshal(map[string]any{"type": "doc", "content": content})
	return string(document)
}

func normalizeTags(tags []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		value := strings.TrimSpace(tag)
		key := strings.ToLower(value)
		if value != "" && !seen[key] {
			seen[key] = true
			result = append(result, value)
		}
	}
	return result
}

func replaceTags(ctx context.Context, tx *sql.Tx, entryID string, tags []string, now string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM entry_tags WHERE entry_id=?`, entryID); err != nil {
		return fmt.Errorf("clear entry tags: %w", err)
	}
	for _, name := range tags {
		if _, err := tx.ExecContext(ctx, `INSERT INTO tags(id, name, created_at) VALUES(?, ?, ?) ON CONFLICT(name) DO NOTHING`, uuid.NewString(), name, now); err != nil {
			return fmt.Errorf("create tag: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO entry_tags(entry_id, tag_id) SELECT ?, id FROM tags WHERE name=? COLLATE NOCASE`, entryID, name); err != nil {
			return fmt.Errorf("assign tag: %w", err)
		}
	}
	return nil
}

func loadTags(ctx context.Context, db *sql.DB, entryID string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT tags.name FROM tags JOIN entry_tags ON entry_tags.tag_id=tags.id WHERE entry_tags.entry_id=? ORDER BY tags.name COLLATE NOCASE`, entryID)
	if err != nil {
		return nil, fmt.Errorf("load entry tags: %w", err)
	}
	defer rows.Close()
	result := []string{}
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		result = append(result, tag)
	}
	return result, rows.Err()
}

func mutationResult(operation string, result sql.Result, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	if changed == 0 {
		return ErrNotFound
	}
	return nil
}

func duplicateTitle(title string) string {
	if strings.TrimSpace(title) == "" {
		return "Copy"
	}
	return title + " (copy)"
}
