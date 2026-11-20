package diary

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	revisionLimit    = 5
	revisionInterval = 10 * time.Minute
)

func recordRevision(ctx context.Context, tx *sql.Tx, entryID string, incoming EntryInput, incomingTags []string, expectedVersion int64, savedAt string, force bool) error {
	var revision Revision
	var favorite, pinned, draft int
	var deletedAt string
	err := tx.QueryRowContext(ctx, `SELECT id, COALESCE(title,''), content_json, content_text, start_date, end_date,
		COALESCE(mood,''), is_favorite, is_pinned, is_draft, COALESCE(location_text,''), display_order,
		version, COALESCE(deleted_at,'') FROM entries WHERE id=?`, entryID).
		Scan(&revision.EntryID, &revision.Title, &revision.ContentJSON, &revision.ContentText, &revision.StartDate, &revision.EndDate,
			&revision.Mood, &favorite, &pinned, &draft, &revision.LocationText, &revision.DisplayOrder, &revision.SourceVersion, &deletedAt)
	if errors.Is(err, sql.ErrNoRows) || deletedAt != "" {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("load entry revision snapshot: %w", err)
	}
	if revision.SourceVersion != expectedVersion {
		return ErrStaleVersion
	}
	revision.ID, revision.SavedAt = uuid.NewString(), savedAt
	revision.Favorite, revision.Pinned, revision.Draft = favorite == 1, pinned == 1, draft == 1
	tags, err := loadTagsFromTx(ctx, tx, entryID)
	if err != nil {
		return err
	}
	if !meaningfulRevisionChange(revision, tags, incoming, incomingTags) {
		return nil
	}
	latest, err := latestRevision(ctx, tx, entryID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil {
		if sameMeaningfulRevision(revision, tags, latest, latest.Tags) {
			return nil
		}
		latestAt, parseErr := time.Parse(time.RFC3339Nano, latest.SavedAt)
		currentAt, currentParseErr := time.Parse(time.RFC3339Nano, savedAt)
		if parseErr != nil || currentParseErr != nil {
			return fmt.Errorf("read entry revision time: %w", errors.Join(parseErr, currentParseErr))
		}
		if !force && currentAt.Sub(latestAt) < revisionInterval {
			return nil
		}
	}
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return fmt.Errorf("encode revision tags: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO entry_revisions
		(id,entry_id,title,content_json,content_text,start_date,end_date,mood,is_favorite,is_pinned,is_draft,location_text,display_order,tags_json,source_version,saved_at)
		VALUES(?,?,NULLIF(?,''),?,?,?,?,NULLIF(?,''),?,?,?,NULLIF(?,''),?,?,?,?)`,
		revision.ID, entryID, revision.Title, revision.ContentJSON, revision.ContentText, revision.StartDate, revision.EndDate,
		revision.Mood, favorite, pinned, draft, revision.LocationText, revision.DisplayOrder, string(tagsJSON), revision.SourceVersion, revision.SavedAt)
	if err != nil {
		return fmt.Errorf("save entry revision: %w", err)
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM entry_revisions WHERE entry_id=? AND id NOT IN
		(SELECT id FROM entry_revisions WHERE entry_id=? ORDER BY saved_at DESC, id DESC LIMIT ?)`, entryID, entryID, revisionLimit)
	if err != nil {
		return fmt.Errorf("prune entry revisions: %w", err)
	}
	return nil
}

func (s *Service) ListRevisions(ctx context.Context, entryID string) ([]Revision, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, entry_id, COALESCE(title,''), content_json, content_text, start_date, end_date,
		COALESCE(mood,''), is_favorite, is_pinned, is_draft, COALESCE(location_text,''), display_order, tags_json, source_version, saved_at
		FROM entry_revisions WHERE entry_id=? ORDER BY saved_at DESC, id DESC LIMIT ?`, entryID, revisionLimit)
	if err != nil {
		return nil, fmt.Errorf("list entry revisions: %w", err)
	}
	defer rows.Close()
	result := []Revision{}
	for rows.Next() {
		revision, err := scanRevision(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, revision)
	}
	return result, rows.Err()
}

func (s *Service) RestoreRevision(ctx context.Context, entryID, revisionID string, expectedVersion int64) (Entry, error) {
	revision, err := s.getRevision(ctx, entryID, revisionID)
	if err != nil {
		return Entry{}, err
	}
	return s.updateEntry(ctx, entryID, EntryInput{
		Title: revision.Title, ContentJSON: revision.ContentJSON, ContentText: revision.ContentText,
		StartDate: revision.StartDate, EndDate: revision.EndDate, Mood: revision.Mood,
		Favorite: revision.Favorite, Pinned: revision.Pinned, Draft: revision.Draft,
		LocationText: revision.LocationText, DisplayOrder: revision.DisplayOrder, Tags: revision.Tags,
	}, expectedVersion, true)
}

func (s *Service) DeleteRevision(ctx context.Context, entryID, revisionID string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM entry_revisions WHERE id=? AND entry_id=?`, revisionID, entryID)
	return mutationResult("delete saved version", result, err)
}

func (s *Service) ClearRevisions(ctx context.Context, entryID string) error {
	if _, err := s.GetEntry(ctx, entryID); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM entry_revisions WHERE entry_id=?`, entryID); err != nil {
		return fmt.Errorf("clear saved versions: %w", err)
	}
	return nil
}

func (s *Service) getRevision(ctx context.Context, entryID, revisionID string) (Revision, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, entry_id, COALESCE(title,''), content_json, content_text, start_date, end_date,
		COALESCE(mood,''), is_favorite, is_pinned, is_draft, COALESCE(location_text,''), display_order, tags_json, source_version, saved_at
		FROM entry_revisions WHERE id=? AND entry_id=?`, revisionID, entryID)
	revision, err := scanRevision(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Revision{}, ErrNotFound
	}
	return revision, err
}

func scanRevision(row scanner) (Revision, error) {
	var result Revision
	var favorite, pinned, draft int
	var tagsJSON string
	if err := row.Scan(&result.ID, &result.EntryID, &result.Title, &result.ContentJSON, &result.ContentText, &result.StartDate, &result.EndDate,
		&result.Mood, &favorite, &pinned, &draft, &result.LocationText, &result.DisplayOrder, &tagsJSON, &result.SourceVersion, &result.SavedAt); err != nil {
		return result, err
	}
	result.Favorite, result.Pinned, result.Draft = favorite == 1, pinned == 1, draft == 1
	if err := json.Unmarshal([]byte(tagsJSON), &result.Tags); err != nil {
		return Revision{}, fmt.Errorf("decode revision tags: %w", err)
	}
	return result, nil
}

func latestRevision(ctx context.Context, tx *sql.Tx, entryID string) (Revision, error) {
	return scanRevision(tx.QueryRowContext(ctx, `SELECT id, entry_id, COALESCE(title,''), content_json, content_text, start_date, end_date,
		COALESCE(mood,''), is_favorite, is_pinned, is_draft, COALESCE(location_text,''), display_order, tags_json, source_version, saved_at
		FROM entry_revisions WHERE entry_id=? ORDER BY saved_at DESC, id DESC LIMIT 1`, entryID))
}

func meaningfulRevisionChange(current Revision, currentTags []string, incoming EntryInput, incomingTags []string) bool {
	return current.Title != strings.TrimSpace(incoming.Title) ||
		current.ContentJSON != incoming.ContentJSON || current.ContentText != incoming.ContentText ||
		current.StartDate != incoming.StartDate || current.EndDate != incoming.EndDate ||
		current.Mood != strings.TrimSpace(incoming.Mood) || current.LocationText != strings.TrimSpace(incoming.LocationText) ||
		!slices.Equal(currentTags, incomingTags)
}

func sameMeaningfulRevision(left Revision, leftTags []string, right Revision, rightTags []string) bool {
	return left.Title == right.Title && left.ContentJSON == right.ContentJSON && left.ContentText == right.ContentText &&
		left.StartDate == right.StartDate && left.EndDate == right.EndDate && left.Mood == right.Mood &&
		left.LocationText == right.LocationText && slices.Equal(leftTags, rightTags)
}

func loadTagsFromTx(ctx context.Context, tx *sql.Tx, entryID string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `SELECT tags.name FROM tags JOIN entry_tags ON entry_tags.tag_id=tags.id WHERE entry_tags.entry_id=? ORDER BY tags.name COLLATE NOCASE`, entryID)
	if err != nil {
		return nil, fmt.Errorf("load revision tags: %w", err)
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
