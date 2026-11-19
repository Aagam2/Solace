package portable

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"solace/internal/diary"
)

func (s *Service) loadJournal(ctx context.Context, request ExportRequest, includeDeleted bool) (JournalExport, error) {
	result := JournalExport{ExportVersion: ExportVersion, AppVersion: appVersion, ExportedAt: time.Now().UTC().Format(time.RFC3339Nano), Entries: []diary.Entry{}, Tags: []TagRecord{}, EntryTags: []EntryTagRecord{}, Attachments: []AttachmentRecord{}, Revisions: []diary.Revision{}, Settings: map[string]string{}}
	query := `SELECT id, COALESCE(title,''), content_json, content_text, start_date, end_date, created_at, updated_at,
		COALESCE(mood,''), is_favorite, is_pinned, is_draft, COALESCE(location_text,''), display_order,
		COALESCE(deleted_at,''), version FROM entries`
	conditions, args := []string{}, []any{}
	if !includeDeleted {
		conditions = append(conditions, "deleted_at IS NULL")
	}
	switch request.Scope {
	case "date":
		conditions = append(conditions, "start_date <= ? AND end_date >= ?")
		args = append(args, request.EndDate, request.StartDate)
	case "selected":
		placeholders := make([]string, len(request.EntryIDs))
		for index, id := range request.EntryIDs {
			placeholders[index] = "?"
			args = append(args, id)
		}
		conditions = append(conditions, "id IN ("+strings.Join(placeholders, ",")+")")
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY start_date, created_at, id"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return result, fmt.Errorf("load entries for export: %w", err)
	}
	for rows.Next() {
		var entry diary.Entry
		var favorite, pinned, draft int
		if err := rows.Scan(&entry.ID, &entry.Title, &entry.ContentJSON, &entry.ContentText, &entry.StartDate, &entry.EndDate, &entry.CreatedAt, &entry.UpdatedAt,
			&entry.Mood, &favorite, &pinned, &draft, &entry.LocationText, &entry.DisplayOrder, &entry.DeletedAt, &entry.Version); err != nil {
			rows.Close()
			return result, fmt.Errorf("scan export entry: %w", err)
		}
		entry.Favorite, entry.Pinned, entry.Draft = favorite == 1, pinned == 1, draft == 1
		entry.Tags = []string{}
		result.Entries = append(result.Entries, entry)
	}
	if err := rows.Close(); err != nil {
		return result, err
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	scopedEntryIDs := make(map[string]bool, len(result.Entries))
	for _, entry := range result.Entries {
		scopedEntryIDs[entry.ID] = true
	}
	revisionRows, err := s.db.QueryContext(ctx, `SELECT id, entry_id, COALESCE(title,''), content_json, content_text, start_date, end_date,
		COALESCE(mood,''), is_favorite, is_pinned, is_draft, COALESCE(location_text,''), display_order, tags_json, source_version, saved_at
		FROM entry_revisions ORDER BY saved_at DESC, id DESC`)
	if err != nil {
		return result, fmt.Errorf("load revisions for export: %w", err)
	}
	for revisionRows.Next() {
		var revision diary.Revision
		var favorite, pinned, draft int
		var tagsJSON string
		if err := revisionRows.Scan(&revision.ID, &revision.EntryID, &revision.Title, &revision.ContentJSON, &revision.ContentText, &revision.StartDate, &revision.EndDate,
			&revision.Mood, &favorite, &pinned, &draft, &revision.LocationText, &revision.DisplayOrder, &tagsJSON, &revision.SourceVersion, &revision.SavedAt); err != nil {
			revisionRows.Close()
			return result, fmt.Errorf("scan export revision: %w", err)
		}
		if !scopedEntryIDs[revision.EntryID] {
			continue
		}
		revision.Favorite, revision.Pinned, revision.Draft = favorite == 1, pinned == 1, draft == 1
		if err := json.Unmarshal([]byte(tagsJSON), &revision.Tags); err != nil {
			revisionRows.Close()
			return result, fmt.Errorf("decode export revision tags: %w", err)
		}
		result.Revisions = append(result.Revisions, revision)
	}
	if err := revisionRows.Close(); err != nil {
		return result, err
	}
	if err := revisionRows.Err(); err != nil {
		return result, err
	}
	entryIDs := make([]string, len(result.Entries))
	entryIndex := make(map[string]int, len(result.Entries))
	for index, entry := range result.Entries {
		entryIDs[index], entryIndex[entry.ID] = entry.ID, index
	}
	if len(entryIDs) > 0 {
		if err := s.loadTags(ctx, &result, entryIDs, entryIndex); err != nil {
			return result, err
		}
		if request.IncludeAttachments {
			if err := s.loadAttachments(ctx, &result, entryIDs); err != nil {
				return result, err
			}
		}
	}
	settingsRows, err := s.db.QueryContext(ctx, `SELECT key, value FROM settings ORDER BY key`)
	if err != nil {
		return result, err
	}
	for settingsRows.Next() {
		var key, value string
		if err := settingsRows.Scan(&key, &value); err != nil {
			settingsRows.Close()
			return result, err
		}
		result.Settings[key] = value
	}
	settingsRows.Close()
	migrationRows, err := s.db.QueryContext(ctx, `SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		return result, err
	}
	for migrationRows.Next() {
		var version string
		if err := migrationRows.Scan(&version); err != nil {
			migrationRows.Close()
			return result, err
		}
		result.SchemaVersions = append(result.SchemaVersions, version)
	}
	migrationRows.Close()
	return result, nil
}

func (s *Service) loadTags(ctx context.Context, journal *JournalExport, entryIDs []string, entryIndex map[string]int) error {
	query, args := inQuery(`SELECT t.id, t.name, t.created_at, et.entry_id FROM tags t JOIN entry_tags et ON et.tag_id=t.id WHERE et.entry_id IN (%s) ORDER BY t.name`, entryIDs)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	tagSeen := map[string]bool{}
	for rows.Next() {
		var tag TagRecord
		var entryID string
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.CreatedAt, &entryID); err != nil {
			return err
		}
		if !tagSeen[tag.ID] {
			journal.Tags = append(journal.Tags, tag)
			tagSeen[tag.ID] = true
		}
		journal.EntryTags = append(journal.EntryTags, EntryTagRecord{EntryID: entryID, TagID: tag.ID})
		if index, ok := entryIndex[entryID]; ok {
			journal.Entries[index].Tags = append(journal.Entries[index].Tags, tag.Name)
		}
	}
	return rows.Err()
}

func (s *Service) loadAttachments(ctx context.Context, journal *JournalExport, entryIDs []string) error {
	query, args := inQuery(`SELECT id, entry_id, original_filename, stored_filename, relative_path, mime_type, file_size, created_at,
		COALESCE(caption,''), display_order, checksum, metadata_json FROM attachments WHERE entry_id IN (%s) ORDER BY entry_id, display_order`, entryIDs)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var attachment AttachmentRecord
		if err := rows.Scan(&attachment.ID, &attachment.EntryID, &attachment.OriginalFilename, &attachment.StoredFilename, &attachment.RelativePath,
			&attachment.MIMEType, &attachment.FileSize, &attachment.CreatedAt, &attachment.Caption, &attachment.DisplayOrder, &attachment.Checksum, &attachment.MetadataJSON); err != nil {
			return err
		}
		journal.Attachments = append(journal.Attachments, attachment)
	}
	return rows.Err()
}

func inQuery(template string, ids []string) (string, []any) {
	placeholders, args := make([]string, len(ids)), make([]any, len(ids))
	for index, id := range ids {
		placeholders[index], args[index] = "?", id
	}
	return fmt.Sprintf(template, strings.Join(placeholders, ",")), args
}

func (s *Service) exportJSON(destination string, journal JournalExport) error {
	return atomicFile(destination, func(file *os.File) error {
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		return encoder.Encode(journal)
	})
}

func (s *Service) exportMarkdown(ctx context.Context, destination string, journal JournalExport, includeAttachments bool) error {
	return s.atomicZip(destination, func(writer *zip.Writer, records *[]FileRecord) error {
		readme := fmt.Sprintf("# Solace journal export\n\nExported %s. Contains %d entries.\n\nOpen the files under `entries/` with any text editor.\n", journal.ExportedAt, len(journal.Entries))
		if err := addZipBytes(writer, records, "README.md", []byte(readme)); err != nil {
			return err
		}
		mediaPaths := readableMediaPaths(journal.Attachments)
		for _, entry := range journal.Entries {
			if err := s.checkCancelled(); err != nil {
				return err
			}
			name := markdownEntryPath(entry)
			body := markdownEntry(entry, attachmentsFor(entry.ID, journal.Attachments), mediaPaths, includeAttachments)
			if err := addZipBytes(writer, records, name, []byte(body)); err != nil {
				return err
			}
		}
		if includeAttachments {
			return s.addReadableMedia(ctx, writer, records, journal.Attachments, mediaPaths)
		}
		return nil
	})
}

func (s *Service) exportHTML(ctx context.Context, destination string, journal JournalExport, includeAttachments bool) error {
	return s.atomicZip(destination, func(writer *zip.Writer, records *[]FileRecord) error {
		mediaPaths := readableMediaPaths(journal.Attachments)
		var index strings.Builder
		index.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width\"><title>Solace journal</title><style>")
		index.WriteString(exportCSS())
		index.WriteString("</style></head><body><main><h1>Solace journal</h1><p class=\"meta\">Offline export · " + html.EscapeString(journal.ExportedAt) + "</p><ol>")
		for _, entry := range journal.Entries {
			if err := s.checkCancelled(); err != nil {
				return err
			}
			entryPath := "entries/" + entry.ID + ".html"
			index.WriteString("<li><a href=\"" + entryPath + "\">" + html.EscapeString(title(entry)) + "</a><span>" + html.EscapeString(dateLabel(entry)) + "</span></li>")
			body := htmlEntry(entry, attachmentsFor(entry.ID, journal.Attachments), mediaPaths, includeAttachments)
			if err := addZipBytes(writer, records, entryPath, []byte(body)); err != nil {
				return err
			}
		}
		index.WriteString("</ol></main></body></html>")
		if err := addZipBytes(writer, records, "index.html", []byte(index.String())); err != nil {
			return err
		}
		if includeAttachments {
			return s.addReadableMedia(ctx, writer, records, journal.Attachments, mediaPaths)
		}
		return nil
	})
}

func (s *Service) exportArchive(ctx context.Context, destination string, journal JournalExport, includeAttachments bool) error {
	return s.atomicZip(destination, func(writer *zip.Writer, records *[]FileRecord) error {
		readme := []byte("# Solace portable backup\n\nThis archive can be restored from Solace Settings → Restore. It also includes readable JSON and Markdown copies.\n")
		if err := addZipBytes(writer, records, "README.md", readme); err != nil {
			return err
		}
		encoded, err := json.MarshalIndent(journal, "", "  ")
		if err != nil {
			return err
		}
		if err := addZipBytes(writer, records, "exports/journal.json", append(encoded, '\n')); err != nil {
			return err
		}
		for _, entry := range journal.Entries {
			if err := s.checkCancelled(); err != nil {
				return err
			}
			body := markdownEntry(entry, attachmentsFor(entry.ID, journal.Attachments), archiveMediaPaths(journal.Attachments), includeAttachments)
			if err := addZipBytes(writer, records, markdownEntryPath(entry), []byte(body)); err != nil {
				return err
			}
		}
		tempDir, err := os.MkdirTemp(s.paths.Root, ".backup-snapshot-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tempDir)
		snapshot := filepath.Join(tempDir, "solace.db")
		if err := s.snapshotDatabase(ctx, snapshot); err != nil {
			return err
		}
		if err := addZipFile(writer, records, "database/solace.db", snapshot); err != nil {
			return err
		}
		if includeAttachments {
			for _, attachment := range journal.Attachments {
				if err := s.checkCancelled(); err != nil {
					return err
				}
				absolute, err := safeOwnedPath(s.paths.Root, attachment.RelativePath)
				if err != nil {
					return err
				}
				if err := addZipFile(writer, records, filepath.ToSlash(attachment.RelativePath), absolute); err != nil {
					return fmt.Errorf("include attachment %q: %w", attachment.OriginalFilename, err)
				}
			}
		}
		sortFileRecords(*records)
		manifest := ArchiveManifest{ArchiveVersion: ArchiveVersion, ExportVersion: ExportVersion, AppVersion: appVersion, CreatedAt: journal.ExportedAt, EntryCount: len(journal.Entries), AttachmentCount: len(journal.Attachments), Files: append([]FileRecord(nil), (*records)...)}
		manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return err
		}
		return addZipBytes(writer, nil, "manifest.json", append(manifestBytes, '\n'))
	})
}

func (s *Service) snapshotDatabase(ctx context.Context, destination string) error {
	if err := os.Remove(destination); err != nil && !os.IsNotExist(err) {
		return err
	}
	quoted := strings.ReplaceAll(destination, "'", "''")
	if _, err := s.db.ExecContext(ctx, "VACUUM INTO '"+quoted+"'"); err != nil {
		return fmt.Errorf("create consistent SQLite snapshot: %w", err)
	}
	return os.Chmod(destination, 0o600)
}

func (s *Service) atomicZip(destination string, populate func(*zip.Writer, *[]FileRecord) error) error {
	return atomicFile(destination, func(file *os.File) error {
		writer := zip.NewWriter(file)
		records := []FileRecord{}
		if err := populate(writer, &records); err != nil {
			writer.Close()
			return err
		}
		return writer.Close()
	})
}

func atomicFile(destination string, write func(*os.File) error) error {
	directory := filepath.Dir(destination)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(directory, ".solace-export-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return err
	}
	if err := write(temp); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, destination); err != nil {
		return fmt.Errorf("finalize export: %w", err)
	}
	return os.Chmod(destination, 0o600)
}

func addZipBytes(writer *zip.Writer, records *[]FileRecord, name string, data []byte) error {
	header := &zip.FileHeader{Name: filepath.ToSlash(name), Method: zip.Deflate}
	header.SetMode(0o600)
	header.SetModTime(time.Now().UTC())
	target, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	if _, err := target.Write(data); err != nil {
		return err
	}
	if records != nil {
		sum := sha256.Sum256(data)
		*records = append(*records, FileRecord{Path: header.Name, Size: int64(len(data)), SHA256: "sha256:" + hex.EncodeToString(sum[:])})
	}
	return nil
}

func addZipFile(writer *zip.Writer, records *[]FileRecord, name, source string) error {
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	header := &zip.FileHeader{Name: filepath.ToSlash(name), Method: zip.Deflate}
	header.SetMode(0o600)
	header.SetModTime(info.ModTime())
	target, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(target, hash), file)
	if err != nil {
		return err
	}
	if records != nil {
		*records = append(*records, FileRecord{Path: header.Name, Size: size, SHA256: "sha256:" + hex.EncodeToString(hash.Sum(nil))})
	}
	return nil
}

func (s *Service) addReadableMedia(ctx context.Context, writer *zip.Writer, records *[]FileRecord, attachments []AttachmentRecord, names map[string]string) error {
	for _, attachment := range attachments {
		if err := s.checkCancelled(); err != nil {
			return err
		}
		absolute, err := safeOwnedPath(s.paths.Root, attachment.RelativePath)
		if err != nil {
			return err
		}
		if err := addZipFile(writer, records, names[attachment.ID], absolute); err != nil {
			return fmt.Errorf("include attachment %q: %w", attachment.OriginalFilename, err)
		}
	}
	return ctx.Err()
}

func safeOwnedPath(root, relative string) (string, error) {
	if filepath.IsAbs(relative) {
		return "", fmt.Errorf("absolute attachment path is not portable")
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("attachment path escapes data directory")
	}
	absolute := filepath.Join(root, clean)
	rel, err := filepath.Rel(root, absolute)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("attachment path escapes data directory")
	}
	return absolute, nil
}

func readableMediaPaths(attachments []AttachmentRecord) map[string]string {
	result := map[string]string{}
	for _, attachment := range attachments {
		category := "documents"
		if strings.HasPrefix(attachment.MIMEType, "image/") {
			category = "images"
		} else if strings.HasPrefix(attachment.MIMEType, "video/") {
			category = "videos"
		} else if strings.HasPrefix(attachment.MIMEType, "audio/") {
			category = "audio"
		}
		extension := strings.ToLower(filepath.Ext(attachment.OriginalFilename))
		result[attachment.ID] = "media/" + category + "/" + attachment.ID + extension
	}
	return result
}

func archiveMediaPaths(attachments []AttachmentRecord) map[string]string {
	result := map[string]string{}
	for _, attachment := range attachments {
		result[attachment.ID] = filepath.ToSlash(attachment.RelativePath)
	}
	return result
}

func attachmentsFor(entryID string, attachments []AttachmentRecord) []AttachmentRecord {
	result := []AttachmentRecord{}
	for _, attachment := range attachments {
		if attachment.EntryID == entryID {
			result = append(result, attachment)
		}
	}
	return result
}

func markdownEntryPath(entry diary.Entry) string {
	date := entry.StartDate
	if entry.EndDate != entry.StartDate {
		date += "_to_" + entry.EndDate
	}
	return fmt.Sprintf("entries/%s/%s/%s-%s-%s.md", entry.StartDate[:4], entry.StartDate[5:7], date, slug(title(entry)), shortID(entry.ID))
}

func markdownEntry(entry diary.Entry, attachments []AttachmentRecord, media map[string]string, includeAttachments bool) string {
	var body strings.Builder
	body.WriteString("---\n")
	body.WriteString("id: " + strconv.Quote(entry.ID) + "\n")
	body.WriteString("title: " + strconv.Quote(title(entry)) + "\n")
	body.WriteString("start_date: " + entry.StartDate + "\nend_date: " + entry.EndDate + "\n")
	body.WriteString("created_at: " + strconv.Quote(entry.CreatedAt) + "\nupdated_at: " + strconv.Quote(entry.UpdatedAt) + "\n")
	body.WriteString("mood: " + strconv.Quote(entry.Mood) + "\nlocation: " + strconv.Quote(entry.LocationText) + "\n")
	body.WriteString("favorite: " + strconv.FormatBool(entry.Favorite) + "\npinned: " + strconv.FormatBool(entry.Pinned) + "\ndraft: " + strconv.FormatBool(entry.Draft) + "\n")
	body.WriteString("tags:\n")
	for _, tag := range entry.Tags {
		body.WriteString("  - " + strconv.Quote(tag) + "\n")
	}
	body.WriteString("---\n\n# " + title(entry) + "\n\n")
	if strings.TrimSpace(entry.ContentText) != "" {
		body.WriteString(entry.ContentText + "\n")
	}
	if includeAttachments && len(attachments) > 0 {
		body.WriteString("\n## Attachments\n\n")
		for _, attachment := range attachments {
			relative, _ := filepath.Rel(filepath.Dir(markdownEntryPath(entry)), media[attachment.ID])
			relative = filepath.ToSlash(relative)
			label := attachment.Caption
			if label == "" {
				label = attachment.OriginalFilename
			}
			if strings.HasPrefix(attachment.MIMEType, "image/") {
				body.WriteString("![" + label + "](" + relative + ")\n\n")
			} else {
				body.WriteString("[" + label + "](" + relative + ")\n\n")
			}
		}
	}
	return body.String()
}

func htmlEntry(entry diary.Entry, attachments []AttachmentRecord, media map[string]string, includeAttachments bool) string {
	var body strings.Builder
	body.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width\"><title>" + html.EscapeString(title(entry)) + "</title><style>" + exportCSS() + "</style></head><body><main><p><a href=\"../index.html\">← Journal index</a></p><article><h1>" + html.EscapeString(title(entry)) + "</h1>")
	body.WriteString("<p class=\"meta\">" + html.EscapeString(dateLabel(entry)) + formatMeta(entry) + "</p>")
	for _, paragraph := range strings.Split(entry.ContentText, "\n") {
		if strings.TrimSpace(paragraph) != "" {
			body.WriteString("<p>" + html.EscapeString(paragraph) + "</p>")
		}
	}
	if includeAttachments && len(attachments) > 0 {
		body.WriteString("<h2>Attachments</h2><div class=\"media\">")
		for _, attachment := range attachments {
			path := "../" + media[attachment.ID]
			label := attachment.Caption
			if label == "" {
				label = attachment.OriginalFilename
			}
			if strings.HasPrefix(attachment.MIMEType, "image/") {
				body.WriteString("<figure><img src=\"" + html.EscapeString(path) + "\" alt=\"" + html.EscapeString(label) + "\"><figcaption>" + html.EscapeString(label) + "</figcaption></figure>")
			} else {
				body.WriteString("<p><a href=\"" + html.EscapeString(path) + "\">" + html.EscapeString(label) + "</a></p>")
			}
		}
		body.WriteString("</div>")
	}
	body.WriteString("</article></main></body></html>")
	return body.String()
}

func exportCSS() string {
	return `:root{color-scheme:light dark}body{margin:0;background:#f6f3ed;color:#2b2825;font:17px/1.75 Georgia,serif}main{max-width:760px;margin:60px auto;padding:0 28px}h1,h2{line-height:1.2}a{color:#805532}.meta{color:#716a61;font:13px/1.5 system-ui,sans-serif}ol{padding:0;list-style:none;border-top:1px solid #ded7cc}li{padding:14px 0;border-bottom:1px solid #ded7cc;display:flex;justify-content:space-between;gap:20px}li span{color:#716a61;font:12px system-ui,sans-serif}img{display:block;max-width:100%;max-height:700px;margin:auto}.media{display:grid;gap:18px}figcaption{text-align:center;color:#716a61;font:12px system-ui,sans-serif}@media(prefers-color-scheme:dark){body{background:#1b1a18;color:#eee9df}a{color:#d39a72}.meta,li span,figcaption{color:#b8b0a5}ol,li{border-color:#3b3832}}`
}

func formatMeta(entry diary.Entry) string {
	parts := []string{}
	if entry.Mood != "" {
		parts = append(parts, entry.Mood)
	}
	if entry.LocationText != "" {
		parts = append(parts, entry.LocationText)
	}
	if len(entry.Tags) > 0 {
		parts = append(parts, strings.Join(entry.Tags, ", "))
	}
	if len(parts) == 0 {
		return ""
	}
	return " · " + html.EscapeString(strings.Join(parts, " · "))
}

func dateLabel(entry diary.Entry) string {
	if entry.StartDate == entry.EndDate {
		return entry.StartDate
	}
	return entry.StartDate + " – " + entry.EndDate
}
func title(entry diary.Entry) string {
	if strings.TrimSpace(entry.Title) == "" {
		return "Untitled entry"
	}
	return entry.Title
}
func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}
func slug(value string) string {
	var result strings.Builder
	lastDash := false
	for _, character := range strings.ToLower(value) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			result.WriteRune(character)
			lastDash = false
		} else if !lastDash {
			result.WriteByte('-')
			lastDash = true
		}
	}
	clean := strings.Trim(result.String(), "-")
	if clean == "" {
		return "untitled"
	}
	if len(clean) > 70 {
		clean = strings.Trim(clean[:70], "-")
	}
	return clean
}

func sortFileRecords(records []FileRecord) {
	sort.Slice(records, func(i, j int) bool { return records[i].Path < records[j].Path })
}
