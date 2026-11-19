package diary

import (
	"context"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

func (s *Service) Search(ctx context.Context, options SearchOptions) (SearchPage, error) {
	where := []string{"e.deleted_at IS NULL"}
	args := []any{}
	query := safeFTSQuery(options.Query)
	from := "entries e"
	if query != "" {
		from = "entry_search JOIN entries e ON e.id=entry_search.entry_id"
		where = append(where, "entry_search MATCH ?")
		args = append(args, query)
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
			return SearchPage{}, err
		}
		where = append(where, "e.start_date <= ? AND e.end_date >= ?")
		args = append(args, end, start)
	}
	for _, tag := range normalizeTags(options.Tags) {
		where = append(where, `EXISTS (SELECT 1 FROM entry_tags filter_et JOIN tags filter_t ON filter_t.id=filter_et.tag_id WHERE filter_et.entry_id=e.id AND filter_t.name=? COLLATE NOCASE)`)
		args = append(args, tag)
	}
	if mood := strings.TrimSpace(options.Mood); mood != "" {
		where = append(where, "e.mood=?")
		args = append(args, mood)
	}
	if options.FavoritesOnly {
		where = append(where, "e.is_favorite=1")
	}
	if options.HasImages {
		where = append(where, `EXISTS (SELECT 1 FROM attachments image_a WHERE image_a.entry_id=e.id AND image_a.mime_type LIKE 'image/%')`)
	}
	if options.HasVideos {
		where = append(where, `EXISTS (SELECT 1 FROM attachments video_a WHERE video_a.entry_id=e.id AND video_a.mime_type LIKE 'video/%')`)
	}
	switch options.DraftStatus {
	case "draft":
		where = append(where, "e.is_draft=1")
	case "published":
		where = append(where, "e.is_draft=0")
	}

	condition := strings.Join(where, " AND ")
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+from+" WHERE "+condition, args...).Scan(&total); err != nil {
		return SearchPage{}, fmt.Errorf("count search results: %w", err)
	}
	limit := options.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset := options.Offset
	if offset < 0 {
		offset = 0
	}
	titleExpression, snippetExpression := "COALESCE(e.title,'')", "SUBSTR(e.content_text,1,220)"
	order := "e.start_date DESC, e.created_at DESC"
	if query != "" {
		titleExpression = `highlight(entry_search, 1, '<mark>', '</mark>')`
		snippetExpression = `snippet(entry_search, -1, '<mark>', '</mark>', '…', 28)`
		order = "bm25(entry_search, 0.0, 8.0, 3.0, 5.0, 4.0) ASC, e.start_date DESC"
	}
	if options.Sort == "newest" {
		order = "e.start_date DESC, e.created_at DESC"
	}
	if options.Sort == "oldest" {
		order = "e.start_date ASC, e.created_at ASC"
	}
	selectArgs := append(append([]any{}, args...), limit, offset)
	statement := `SELECT e.id, COALESCE(e.title,''), SUBSTR(e.content_text,1,180), e.start_date, e.end_date, e.updated_at,
		COALESCE(e.mood,''), e.is_favorite, e.is_pinned, e.is_draft,
		(SELECT COUNT(*) FROM attachments count_a WHERE count_a.entry_id=e.id), ` + titleExpression + `, ` + snippetExpression + `,
		COALESCE((SELECT GROUP_CONCAT(tag_t.name, char(31)) FROM tags tag_t JOIN entry_tags tag_et ON tag_et.tag_id=tag_t.id WHERE tag_et.entry_id=e.id), '')
		FROM ` + from + ` WHERE ` + condition + ` ORDER BY ` + order + ` LIMIT ? OFFSET ?`
	rows, err := s.db.QueryContext(ctx, statement, selectArgs...)
	if err != nil {
		return SearchPage{}, fmt.Errorf("search entries: %w", err)
	}
	defer rows.Close()
	results := []SearchResult{}
	for rows.Next() {
		var result SearchResult
		var favorite, pinned, draft int
		var joinedTags string
		if err := rows.Scan(&result.Entry.ID, &result.Entry.Title, &result.Entry.Preview, &result.Entry.StartDate, &result.Entry.EndDate, &result.Entry.UpdatedAt,
			&result.Entry.Mood, &favorite, &pinned, &draft, &result.Entry.AttachmentCount, &result.HighlightedTitle, &result.Snippet, &joinedTags); err != nil {
			return SearchPage{}, fmt.Errorf("scan search result: %w", err)
		}
		result.Entry.Favorite, result.Entry.Pinned, result.Entry.Draft = favorite == 1, pinned == 1, draft == 1
		if joinedTags != "" {
			result.Tags = strings.Split(joinedTags, string(rune(31)))
		} else {
			result.Tags = []string{}
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return SearchPage{}, fmt.Errorf("search entries: %w", err)
	}
	return SearchPage{Results: results, Total: total, HasMore: offset+len(results) < total}, nil
}

func (s *Service) RebuildSearchIndex(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin search rebuild: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM entry_search`); err != nil {
		return fmt.Errorf("clear search index: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO entry_search(entry_id,title,content,tags,attachment_captions)
		SELECT e.id, COALESCE(e.title,''), e.content_text,
		COALESCE((SELECT GROUP_CONCAT(t.name,' ') FROM tags t JOIN entry_tags et ON et.tag_id=t.id WHERE et.entry_id=e.id),''),
		COALESCE((SELECT GROUP_CONCAT(a.caption,' ') FROM attachments a WHERE a.entry_id=e.id AND a.caption IS NOT NULL),'') FROM entries e`)
	if err != nil {
		return fmt.Errorf("rebuild search index: %w", err)
	}
	return tx.Commit()
}

func (s *Service) ListTags(ctx context.Context) ([]Tag, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT t.id,t.name,COUNT(et.entry_id) FROM tags t LEFT JOIN entry_tags et ON et.tag_id=t.id GROUP BY t.id,t.name ORDER BY t.name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()
	result := []Tag{}
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.EntryCount); err != nil {
			return nil, err
		}
		result = append(result, tag)
	}
	return result, rows.Err()
}

func (s *Service) CreateTag(ctx context.Context, name string) (Tag, error) {
	name, err := validateTagName(name)
	if err != nil {
		return Tag{}, err
	}
	tag := Tag{ID: uuid.NewString(), Name: name}
	_, err = s.db.ExecContext(ctx, `INSERT INTO tags(id,name,created_at) VALUES(?,?,strftime('%Y-%m-%dT%H:%M:%fZ','now'))`, tag.ID, tag.Name)
	if err != nil {
		return Tag{}, fmt.Errorf("create tag: %w", err)
	}
	return tag, nil
}

func (s *Service) RenameTag(ctx context.Context, id, name string) error {
	name, err := validateTagName(name)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE tags SET name=? WHERE id=?`, name, id)
	return mutationResult("rename tag", result, err)
}

func (s *Service) DeleteTag(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM tags WHERE id=?`, id)
	return mutationResult("delete tag", result, err)
}

func validateTagName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("tag name is required")
	}
	if utf8.RuneCountInString(name) > 80 {
		return "", fmt.Errorf("tag name must be at most 80 characters")
	}
	return name, nil
}

func safeFTSQuery(value string) string {
	words := []string{}
	var word strings.Builder
	flush := func() {
		if word.Len() > 0 {
			words = append(words, `"`+strings.ReplaceAll(word.String(), `"`, `""`)+`"*`)
			word.Reset()
		}
	}
	for _, character := range value {
		if unicode.IsLetter(character) || unicode.IsNumber(character) || character == '_' {
			word.WriteRune(character)
		} else {
			flush()
		}
	}
	flush()
	return strings.Join(words, " AND ")
}
