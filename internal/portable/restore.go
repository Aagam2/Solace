package portable

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"solace/internal/platform"
)

const maxArchiveUncompressedSize int64 = 200 << 30

type inspectedArchive struct {
	summary ArchiveSummary
	journal JournalExport
	files   map[string]*zip.File
}

func (s *Service) inspectArchive(ctx context.Context, archivePath string) (ArchiveSummary, error) {
	inspected, closer, err := s.openAndValidateArchive(ctx, archivePath)
	if closer != nil {
		defer closer.Close()
	}
	if err != nil {
		return ArchiveSummary{}, err
	}
	return inspected.summary, nil
}

func (s *Service) openAndValidateArchive(ctx context.Context, archivePath string) (inspectedArchive, *zip.ReadCloser, error) {
	result := inspectedArchive{files: map[string]*zip.File{}}
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return result, nil, fmt.Errorf("open Solace archive: %w", err)
	}
	var total int64
	for _, file := range reader.File {
		if err := safeArchiveName(file.Name); err != nil {
			reader.Close()
			return result, nil, err
		}
		if _, exists := result.files[file.Name]; exists {
			reader.Close()
			return result, nil, fmt.Errorf("archive contains duplicate path %q", file.Name)
		}
		total += int64(file.UncompressedSize64)
		if total > maxArchiveUncompressedSize {
			reader.Close()
			return result, nil, fmt.Errorf("archive is too large")
		}
		result.files[file.Name] = file
	}
	manifestFile := result.files["manifest.json"]
	journalFile := result.files["exports/journal.json"]
	if manifestFile == nil || journalFile == nil {
		reader.Close()
		return result, nil, fmt.Errorf("archive is missing manifest.json or exports/journal.json")
	}
	var manifest ArchiveManifest
	if err := decodeZipJSON(manifestFile, &manifest); err != nil {
		reader.Close()
		return result, nil, fmt.Errorf("read archive manifest: %w", err)
	}
	if manifest.ArchiveVersion != ArchiveVersion || manifest.ExportVersion < 1 || manifest.ExportVersion > ExportVersion {
		reader.Close()
		return result, nil, fmt.Errorf("unsupported archive version %d", manifest.ArchiveVersion)
	}
	recorded := make(map[string]struct{}, len(manifest.Files))
	for _, record := range manifest.Files {
		if err := s.checkCancelled(); err != nil {
			reader.Close()
			return result, nil, err
		}
		file := result.files[record.Path]
		if file == nil {
			reader.Close()
			return result, nil, fmt.Errorf("archive is missing %q", record.Path)
		}
		if _, exists := recorded[record.Path]; exists {
			reader.Close()
			return result, nil, fmt.Errorf("archive manifest contains duplicate path %q", record.Path)
		}
		recorded[record.Path] = struct{}{}
		size, checksum, err := zipChecksum(ctx, file)
		if err != nil {
			reader.Close()
			return result, nil, err
		}
		if size != record.Size || checksum != record.SHA256 {
			reader.Close()
			return result, nil, fmt.Errorf("archive checksum mismatch for %q", record.Path)
		}
	}
	for name := range result.files {
		if name == "manifest.json" {
			continue
		}
		if _, exists := recorded[name]; !exists {
			reader.Close()
			return result, nil, fmt.Errorf("archive payload %q is not recorded in the manifest", name)
		}
	}
	if err := decodeZipJSON(journalFile, &result.journal); err != nil {
		reader.Close()
		return result, nil, fmt.Errorf("read journal export: %w", err)
	}
	if result.journal.ExportVersion < 1 || result.journal.ExportVersion > ExportVersion {
		reader.Close()
		return result, nil, fmt.Errorf("unsupported journal export version %d", result.journal.ExportVersion)
	}
	if err := validateJournal(result.journal); err != nil {
		reader.Close()
		return result, nil, err
	}
	if manifest.EntryCount != len(result.journal.Entries) || manifest.AttachmentCount != len(result.journal.Attachments) {
		reader.Close()
		return result, nil, fmt.Errorf("archive manifest counts do not match journal data")
	}
	missing := []string{}
	for _, attachment := range result.journal.Attachments {
		if result.files[filepath.ToSlash(attachment.RelativePath)] == nil {
			missing = append(missing, attachment.OriginalFilename)
		}
	}
	result.summary = ArchiveSummary{ArchiveVersion: manifest.ArchiveVersion, CreatedAt: manifest.CreatedAt, EntryCount: len(result.journal.Entries), AttachmentCount: len(result.journal.Attachments), MissingAttachments: missing, Valid: len(missing) == 0}
	return result, reader, nil
}

func (s *Service) restoreArchive(ctx context.Context, archivePath string, replaceExisting bool) (OperationResult, error) {
	inspected, reader, err := s.openAndValidateArchive(ctx, archivePath)
	if reader != nil {
		defer reader.Close()
	}
	if err != nil {
		return OperationResult{}, err
	}
	if len(inspected.summary.MissingAttachments) > 0 {
		return OperationResult{}, fmt.Errorf("archive is missing %d attachment files", len(inspected.summary.MissingAttachments))
	}
	var existing int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM entries`).Scan(&existing); err != nil {
		return OperationResult{}, err
	}
	if existing > 0 && !replaceExisting {
		return OperationResult{}, fmt.Errorf("the journal is not empty; enable replacement to restore this archive")
	}

	recovery := ""
	if existing > 0 {
		settings, err := s.LoadBackupSettings(ctx)
		if err != nil {
			return OperationResult{}, err
		}
		settings.Destination = s.paths.Backups
		backup, err := s.createBackup(ctx, settings, "solace-before-restore")
		if err != nil {
			return OperationResult{}, fmt.Errorf("create recovery backup: %w", err)
		}
		recovery = backup.OutputPath
	}

	staging, err := os.MkdirTemp(s.paths.Root, ".restore-staging-*")
	if err != nil {
		return OperationResult{}, err
	}
	defer os.RemoveAll(staging)
	stagedMedia := filepath.Join(staging, "media")
	for _, directory := range []string{"images", "videos", "audio", "documents"} {
		if err := os.MkdirAll(filepath.Join(stagedMedia, directory), 0o700); err != nil {
			return OperationResult{}, err
		}
	}
	for _, attachment := range inspected.journal.Attachments {
		if err := s.checkCancelled(); err != nil {
			return OperationResult{}, err
		}
		archiveName := filepath.ToSlash(attachment.RelativePath)
		file := inspected.files[archiveName]
		target := filepath.Join(staging, filepath.FromSlash(archiveName))
		if err := extractZipFile(file, target); err != nil {
			return OperationResult{}, err
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return OperationResult{}, err
	}
	defer tx.Rollback()
	if err := replaceJournal(ctx, tx, inspected.journal); err != nil {
		return OperationResult{}, err
	}

	rollbackMedia := filepath.Join(s.paths.Root, fmt.Sprintf(".media-before-restore-%d", time.Now().UnixNano()))
	if err := os.Rename(s.paths.Media, rollbackMedia); err != nil {
		return OperationResult{}, fmt.Errorf("stage existing media: %w", err)
	}
	restoreMedia := func() {
		_ = os.RemoveAll(s.paths.Media)
		_ = os.Rename(rollbackMedia, s.paths.Media)
	}
	if err := os.Rename(stagedMedia, s.paths.Media); err != nil {
		restoreMedia()
		return OperationResult{}, fmt.Errorf("activate restored media: %w", err)
	}
	if err := tx.Commit(); err != nil {
		restoreMedia()
		return OperationResult{}, fmt.Errorf("commit restored journal: %w", err)
	}
	_ = os.RemoveAll(rollbackMedia)
	_ = os.RemoveAll(s.paths.Thumbnails)
	if err := platform.EnsureDirectories(s.paths); err != nil {
		return OperationResult{}, err
	}
	return OperationResult{EntryCount: len(inspected.journal.Entries), AttachmentCount: len(inspected.journal.Attachments), RecoveryBackup: recovery, Warnings: []string{}, OperationMessage: "Journal restored"}, nil
}

func replaceJournal(ctx context.Context, tx *sql.Tx, journal JournalExport) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM entries`); err != nil {
		return fmt.Errorf("clear existing entries: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM tags`); err != nil {
		return fmt.Errorf("clear existing tags: %w", err)
	}
	for _, entry := range journal.Entries {
		if _, err := tx.ExecContext(ctx, `INSERT INTO entries
			(id,title,content_json,content_text,start_date,end_date,created_at,updated_at,mood,is_favorite,is_pinned,is_draft,location_text,display_order,deleted_at,version)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,NULLIF(?,''),?)`, entry.ID, nullString(entry.Title), entry.ContentJSON, entry.ContentText, entry.StartDate, entry.EndDate,
			entry.CreatedAt, entry.UpdatedAt, nullString(entry.Mood), boolInt(entry.Favorite), boolInt(entry.Pinned), boolInt(entry.Draft), nullString(entry.LocationText), entry.DisplayOrder, entry.DeletedAt, entry.Version); err != nil {
			return fmt.Errorf("restore entry %s: %w", entry.ID, err)
		}
	}
	for _, tag := range journal.Tags {
		if _, err := tx.ExecContext(ctx, `INSERT INTO tags(id,name,created_at) VALUES(?,?,?)`, tag.ID, tag.Name, tag.CreatedAt); err != nil {
			return fmt.Errorf("restore tag: %w", err)
		}
	}
	for _, relation := range journal.EntryTags {
		if _, err := tx.ExecContext(ctx, `INSERT INTO entry_tags(entry_id,tag_id) VALUES(?,?)`, relation.EntryID, relation.TagID); err != nil {
			return fmt.Errorf("restore entry tag: %w", err)
		}
	}
	for _, revision := range journal.Revisions {
		tagsJSON, err := json.Marshal(revision.Tags)
		if err != nil {
			return fmt.Errorf("encode revision tags: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO entry_revisions
			(id,entry_id,title,content_json,content_text,start_date,end_date,mood,is_favorite,is_pinned,is_draft,location_text,display_order,tags_json,source_version,saved_at)
			VALUES(?,?,NULLIF(?,''),?,?,?,?,NULLIF(?,''),?,?,?,NULLIF(?,''),?,?,?,?)`,
			revision.ID, revision.EntryID, revision.Title, revision.ContentJSON, revision.ContentText, revision.StartDate, revision.EndDate,
			revision.Mood, boolInt(revision.Favorite), boolInt(revision.Pinned), boolInt(revision.Draft), revision.LocationText,
			revision.DisplayOrder, string(tagsJSON), revision.SourceVersion, revision.SavedAt); err != nil {
			return fmt.Errorf("restore revision %s: %w", revision.ID, err)
		}
	}
	for _, attachment := range journal.Attachments {
		if _, err := tx.ExecContext(ctx, `INSERT INTO attachments
			(id,entry_id,original_filename,stored_filename,relative_path,mime_type,file_size,created_at,caption,display_order,checksum,metadata_json)
			VALUES(?,?,?,?,?,?,?,?,NULLIF(?,''),?,?,?)`, attachment.ID, attachment.EntryID, attachment.OriginalFilename, attachment.StoredFilename, attachment.RelativePath,
			attachment.MIMEType, attachment.FileSize, attachment.CreatedAt, attachment.Caption, attachment.DisplayOrder, attachment.Checksum, attachment.MetadataJSON); err != nil {
			return fmt.Errorf("restore attachment %s: %w", attachment.ID, err)
		}
	}
	for key, value := range journal.Settings {
		if _, err := tx.ExecContext(ctx, `INSERT INTO settings(key,value,updated_at) VALUES(?,?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value,updated_at=excluded.updated_at`, key, value, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
	}
	return nil
}

func validateJournal(journal JournalExport) error {
	entryIDs, tagIDs, attachmentIDs, revisionIDs, paths := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, entry := range journal.Entries {
		if entry.ID == "" || entryIDs[entry.ID] {
			return fmt.Errorf("journal contains a missing or duplicate entry id")
		}
		if entry.StartDate == "" || entry.EndDate == "" || entry.StartDate > entry.EndDate {
			return fmt.Errorf("entry %s has an invalid date range", entry.ID)
		}
		entryIDs[entry.ID] = true
	}
	for _, tag := range journal.Tags {
		if tag.ID == "" || tag.Name == "" || tagIDs[tag.ID] {
			return fmt.Errorf("journal contains an invalid tag")
		}
		tagIDs[tag.ID] = true
	}
	for _, relation := range journal.EntryTags {
		if !entryIDs[relation.EntryID] || !tagIDs[relation.TagID] {
			return fmt.Errorf("journal contains an invalid tag relationship")
		}
	}
	for _, attachment := range journal.Attachments {
		if attachment.ID == "" || attachmentIDs[attachment.ID] || !entryIDs[attachment.EntryID] {
			return fmt.Errorf("journal contains an invalid attachment")
		}
		if _, err := safeOwnedPath("/portable-root", attachment.RelativePath); err != nil {
			return err
		}
		clean := filepath.ToSlash(filepath.Clean(attachment.RelativePath))
		if !strings.HasPrefix(clean, "media/") || paths[clean] {
			return fmt.Errorf("journal contains an invalid attachment path")
		}
		attachmentIDs[attachment.ID], paths[clean] = true, true
	}
	for _, revision := range journal.Revisions {
		if revision.ID == "" || revisionIDs[revision.ID] || !entryIDs[revision.EntryID] || revision.StartDate == "" || revision.EndDate == "" || revision.StartDate > revision.EndDate || revision.SourceVersion < 1 {
			return fmt.Errorf("journal contains an invalid entry revision")
		}
		revisionIDs[revision.ID] = true
	}
	return nil
}

func safeArchiveName(name string) error {
	if name == "" || strings.Contains(name, "\\") || strings.HasPrefix(name, "/") {
		return fmt.Errorf("archive contains unsafe path %q", name)
	}
	clean := path.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != name {
		return fmt.Errorf("archive contains unsafe path %q", name)
	}
	return nil
}

func decodeZipJSON(file *zip.File, target any) error {
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	decoder := json.NewDecoder(io.LimitReader(reader, 128<<20))
	return decoder.Decode(target)
}

func zipChecksum(ctx context.Context, file *zip.File) (int64, string, error) {
	reader, err := file.Open()
	if err != nil {
		return 0, "", err
	}
	defer reader.Close()
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(hash, contextWriter{ctx: ctx}), reader)
	if err != nil {
		return 0, "", err
	}
	return written, "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

type contextWriter struct{ ctx context.Context }

func (writer contextWriter) Write(value []byte) (int, error) {
	select {
	case <-writer.ctx.Done():
		return 0, writer.ctx.Err()
	default:
		return len(value), nil
	}
}

func extractZipFile(file *zip.File, target string) error {
	if file == nil {
		return fmt.Errorf("archive file is missing")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, reader)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func nullString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
