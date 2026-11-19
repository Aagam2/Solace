package attachments

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"

	"solace/internal/platform"

	"github.com/google/uuid"
)

const MaxAttachmentSize int64 = 1 << 30 // 1 GiB

var ErrNotFound = errors.New("attachment not found")

type Service struct {
	db    *sql.DB
	paths platform.Paths
}

func New(db *sql.DB, paths platform.Paths) *Service { return &Service{db: db, paths: paths} }

func (s *Service) ImportFiles(ctx context.Context, entryID string, sourcePaths []string) ([]Attachment, error) {
	result := make([]Attachment, 0, len(sourcePaths))
	for _, sourcePath := range sourcePaths {
		attachment, err := s.importFile(ctx, entryID, sourcePath)
		if err != nil {
			return result, err
		}
		result = append(result, attachment)
	}
	return result, nil
}

func (s *Service) ImportBase64(ctx context.Context, entryID, originalFilename, declaredMIME, encoded string) (Attachment, error) {
	if int64(base64.StdEncoding.DecodedLen(len(encoded))) > MaxAttachmentSize {
		return Attachment{}, fmt.Errorf("attachment exceeds %d bytes", MaxAttachmentSize)
	}
	reader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(encoded))
	return s.importReader(ctx, entryID, originalFilename, declaredMIME, reader)
}

func (s *Service) importFile(ctx context.Context, entryID, sourcePath string) (Attachment, error) {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return Attachment{}, fmt.Errorf("inspect attachment %q: %w", filepath.Base(sourcePath), err)
	}
	if !info.Mode().IsRegular() {
		return Attachment{}, fmt.Errorf("attachment must be a regular file")
	}
	if info.Size() > MaxAttachmentSize {
		return Attachment{}, fmt.Errorf("attachment exceeds %d bytes", MaxAttachmentSize)
	}
	file, err := os.Open(sourcePath)
	if err != nil {
		return Attachment{}, fmt.Errorf("open attachment %q: %w", filepath.Base(sourcePath), err)
	}
	defer file.Close()
	return s.importReader(ctx, entryID, filepath.Base(sourcePath), "", file)
}

func (s *Service) importReader(ctx context.Context, entryID, originalFilename, declaredMIME string, source io.Reader) (Attachment, error) {
	if err := s.requireActiveEntry(ctx, entryID); err != nil {
		return Attachment{}, err
	}
	originalFilename = cleanDisplayName(originalFilename)
	if originalFilename == "" {
		originalFilename = "attachment"
	}
	id := uuid.NewString()
	extension := safeExtension(originalFilename)
	storedFilename := id + extension
	targetDir, relativeDir := s.paths.Documents, "media/documents"

	temp, err := os.CreateTemp(s.paths.Root, ".attachment-import-*")
	if err != nil {
		return Attachment{}, fmt.Errorf("create attachment staging file: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return Attachment{}, err
	}
	hash := sha256.New()
	limited := io.LimitReader(source, MaxAttachmentSize+1)
	size, copyErr := io.Copy(io.MultiWriter(temp, hash), limited)
	if copyErr == nil && size > MaxAttachmentSize {
		copyErr = fmt.Errorf("attachment exceeds %d bytes", MaxAttachmentSize)
	}
	if copyErr == nil {
		copyErr = temp.Sync()
	}
	closeErr := temp.Close()
	if copyErr != nil {
		return Attachment{}, fmt.Errorf("copy attachment: %w", copyErr)
	}
	if closeErr != nil {
		return Attachment{}, fmt.Errorf("close attachment staging file: %w", closeErr)
	}

	detectedMIME, err := detectMIME(tempPath, originalFilename, declaredMIME)
	if err != nil {
		return Attachment{}, err
	}
	if strings.HasPrefix(detectedMIME, "image/") {
		targetDir, relativeDir = s.paths.Images, "media/images"
	} else if strings.HasPrefix(detectedMIME, "video/") {
		targetDir, relativeDir = s.paths.Videos, "media/videos"
	} else if strings.HasPrefix(detectedMIME, "audio/") {
		targetDir, relativeDir = s.paths.Audio, "media/audio"
	}
	finalPath := filepath.Join(targetDir, storedFilename)
	if err := os.Rename(tempPath, finalPath); err != nil {
		return Attachment{}, fmt.Errorf("finalize attachment: %w", err)
	}
	if err := os.Chmod(finalPath, 0o600); err != nil {
		os.Remove(finalPath)
		return Attachment{}, err
	}
	if strings.HasPrefix(detectedMIME, "image/") {
		_ = s.createThumbnail(finalPath, id)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	relativePath := filepath.ToSlash(filepath.Join(relativeDir, storedFilename))
	attachment := Attachment{
		ID: id, EntryID: entryID, OriginalFilename: originalFilename, StoredFilename: storedFilename,
		RelativePath: relativePath, MIMEType: detectedMIME, FileSize: size, CreatedAt: now,
		Checksum: fmt.Sprintf("sha256:%x", hash.Sum(nil)), MetadataJSON: "{}", Available: true, URL: "/media/" + id,
	}
	attachment.ThumbnailURL = s.thumbnailURL(attachment)
	err = s.db.QueryRowContext(ctx, `INSERT INTO attachments
		(id, entry_id, original_filename, stored_filename, relative_path, mime_type, file_size, created_at, display_order, checksum, metadata_json)
		SELECT ?, ?, ?, ?, ?, ?, ?, ?, COALESCE(MAX(display_order), -1) + 1, ?, '{}' FROM attachments WHERE entry_id=?
		RETURNING display_order`, attachment.ID, attachment.EntryID, attachment.OriginalFilename, attachment.StoredFilename,
		attachment.RelativePath, attachment.MIMEType, attachment.FileSize, attachment.CreatedAt, attachment.Checksum, attachment.EntryID).Scan(&attachment.DisplayOrder)
	if err != nil {
		os.Remove(finalPath)
		return Attachment{}, fmt.Errorf("record attachment: %w", err)
	}
	return attachment, nil
}

func (s *Service) ListAttachments(ctx context.Context, entryID string) ([]Attachment, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, entry_id, original_filename, stored_filename, relative_path, mime_type,
		file_size, created_at, COALESCE(caption,''), display_order, checksum, metadata_json
		FROM attachments WHERE entry_id=? ORDER BY display_order, created_at`, entryID)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}
	defer rows.Close()
	result := []Attachment{}
	for rows.Next() {
		var attachment Attachment
		if err := rows.Scan(&attachment.ID, &attachment.EntryID, &attachment.OriginalFilename, &attachment.StoredFilename,
			&attachment.RelativePath, &attachment.MIMEType, &attachment.FileSize, &attachment.CreatedAt, &attachment.Caption,
			&attachment.DisplayOrder, &attachment.Checksum, &attachment.MetadataJSON); err != nil {
			return nil, err
		}
		attachment.URL = "/media/" + attachment.ID
		attachment.ThumbnailURL = s.thumbnailURL(attachment)
		if path, err := s.absolutePath(attachment.RelativePath); err == nil {
			info, statErr := os.Stat(path)
			attachment.Available = statErr == nil && info.Mode().IsRegular()
		}
		result = append(result, attachment)
	}
	return result, rows.Err()
}

func (s *Service) UpdateCaption(ctx context.Context, id, caption string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE attachments SET caption=NULLIF(?, '') WHERE id=?`, strings.TrimSpace(caption), id)
	return mutationResult(result, err)
}

func (s *Service) ReorderAttachments(ctx context.Context, entryID string, ids []string) error {
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, exists := seen[id]; exists {
			return fmt.Errorf("reorder contains duplicate attachment id")
		}
		seen[id] = struct{}{}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM attachments WHERE entry_id=?`, entryID).Scan(&count); err != nil {
		return err
	}
	if count != len(ids) {
		return fmt.Errorf("reorder must include every attachment")
	}
	for order, id := range ids {
		result, err := tx.ExecContext(ctx, `UPDATE attachments SET display_order=? WHERE id=? AND entry_id=?`, order, id, entryID)
		if err != nil {
			return err
		}
		changed, _ := result.RowsAffected()
		if changed != 1 {
			return ErrNotFound
		}
	}
	return tx.Commit()
}

func (s *Service) RemoveAttachment(ctx context.Context, id string) error {
	attachment, err := s.get(ctx, id)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM attachments WHERE id=?`, id)
	if err := mutationResult(result, err); err != nil {
		return err
	}
	if path, err := s.absolutePath(attachment.RelativePath); err == nil {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove attachment file: %w", err)
		}
	}
	_ = os.Remove(filepath.Join(s.paths.Thumbnails, attachment.ID+".jpg"))
	return nil
}

func (s *Service) PermanentlyDeleteEntry(ctx context.Context, entryID string) error {
	attachments, err := s.ListAttachments(ctx, entryID)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM entries WHERE id=? AND deleted_at IS NOT NULL`, entryID)
	if err := mutationResult(result, err); err != nil {
		return err
	}
	var cleanupErrors []error
	for _, attachment := range attachments {
		path, pathErr := s.absolutePath(attachment.RelativePath)
		if pathErr != nil {
			cleanupErrors = append(cleanupErrors, pathErr)
			continue
		}
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			cleanupErrors = append(cleanupErrors, removeErr)
		}
		if removeErr := os.Remove(filepath.Join(s.paths.Thumbnails, attachment.ID+".jpg")); removeErr != nil && !os.IsNotExist(removeErr) {
			cleanupErrors = append(cleanupErrors, removeErr)
		}
	}
	return errors.Join(cleanupErrors...)
}

func (s *Service) OpenAttachment(ctx context.Context, id string) error {
	attachment, err := s.get(ctx, id)
	if err != nil {
		return err
	}
	path, err := s.absolutePath(attachment.RelativePath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("attachment file unavailable: %w", err)
	}
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", path)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	default:
		command = exec.Command("xdg-open", path)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("open attachment externally: %w", err)
	}
	return nil
}

func Handler(s *Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		isMedia := strings.HasPrefix(request.URL.Path, "/media/")
		isThumbnail := strings.HasPrefix(request.URL.Path, "/thumbnail/")
		if !isMedia && !isThumbnail {
			next.ServeHTTP(response, request)
			return
		}
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		prefix := "/media/"
		if isThumbnail {
			prefix = "/thumbnail/"
		}
		id := strings.TrimPrefix(request.URL.Path, prefix)
		if id == "" || strings.Contains(id, "/") {
			http.NotFound(response, request)
			return
		}
		attachment, err := s.get(request.Context(), id)
		if err != nil {
			http.NotFound(response, request)
			return
		}
		path, err := s.absolutePath(attachment.RelativePath)
		if err != nil {
			http.NotFound(response, request)
			return
		}
		contentType := attachment.MIMEType
		if isThumbnail {
			thumbnail := filepath.Join(s.paths.Thumbnails, id+".jpg")
			if _, statErr := os.Stat(thumbnail); statErr == nil {
				path, contentType = thumbnail, "image/jpeg"
			}
		}
		response.Header().Set("Content-Type", contentType)
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": attachment.OriginalFilename}))
		http.ServeFile(response, request, path)
	})
}

func (s *Service) createThumbnail(sourcePath, id string) error {
	file, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer file.Close()
	source, _, err := image.Decode(file)
	if err != nil {
		return err
	}
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return fmt.Errorf("invalid image dimensions")
	}
	const maximum = 360
	if width > maximum || height > maximum {
		if width >= height {
			height = height * maximum / width
			width = maximum
		} else {
			width = width * maximum / height
			height = maximum
		}
	}
	target := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sourceX := bounds.Min.X + x*bounds.Dx()/width
			sourceY := bounds.Min.Y + y*bounds.Dy()/height
			target.Set(x, y, source.At(sourceX, sourceY))
		}
	}
	path := filepath.Join(s.paths.Thumbnails, id+".jpg")
	output, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	encodeErr := jpeg.Encode(output, target, &jpeg.Options{Quality: 82})
	closeErr := output.Close()
	if encodeErr != nil {
		os.Remove(path)
		return encodeErr
	}
	return closeErr
}

func (s *Service) thumbnailURL(attachment Attachment) string {
	if !strings.HasPrefix(attachment.MIMEType, "image/") {
		return ""
	}
	if _, err := os.Stat(filepath.Join(s.paths.Thumbnails, attachment.ID+".jpg")); err == nil {
		return "/thumbnail/" + attachment.ID
	}
	return attachment.URL
}

func (s *Service) get(ctx context.Context, id string) (Attachment, error) {
	var attachment Attachment
	err := s.db.QueryRowContext(ctx, `SELECT id, entry_id, original_filename, stored_filename, relative_path, mime_type,
		file_size, created_at, COALESCE(caption,''), display_order, checksum, metadata_json FROM attachments WHERE id=?`, id).
		Scan(&attachment.ID, &attachment.EntryID, &attachment.OriginalFilename, &attachment.StoredFilename,
			&attachment.RelativePath, &attachment.MIMEType, &attachment.FileSize, &attachment.CreatedAt, &attachment.Caption,
			&attachment.DisplayOrder, &attachment.Checksum, &attachment.MetadataJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return Attachment{}, ErrNotFound
	}
	return attachment, err
}

func (s *Service) requireActiveEntry(ctx context.Context, id string) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM entries WHERE id=? AND deleted_at IS NULL`, id).Scan(&count); err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("entry unavailable")
	}
	return nil
}

func (s *Service) absolutePath(relative string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(relative))
	if filepath.IsAbs(clean) || clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", fmt.Errorf("invalid attachment path")
	}
	path := filepath.Join(s.paths.Root, clean)
	relativeToRoot, err := filepath.Rel(s.paths.Root, path)
	if err != nil || relativeToRoot == ".." || strings.HasPrefix(relativeToRoot, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("attachment path escapes data directory")
	}
	return path, nil
}

func detectMIME(path, filename, declared string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	header := make([]byte, 512)
	read, err := file.Read(header)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	detected := http.DetectContentType(header[:read])
	extensionMIME := mime.TypeByExtension(strings.ToLower(filepath.Ext(filename)))
	if extensionMIME != "" && (detected == "application/octet-stream" || strings.HasPrefix(extensionMIME, "video/") || strings.HasPrefix(extensionMIME, "audio/")) {
		detected = extensionMIME
	}
	if declared != "" && detected == "application/octet-stream" {
		if parsed, _, err := mime.ParseMediaType(declared); err == nil {
			detected = parsed
		}
	}
	return detected, nil
}

func cleanDisplayName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
}

func safeExtension(name string) string {
	extension := strings.ToLower(filepath.Ext(name))
	if len(extension) > 12 {
		return ""
	}
	for _, character := range strings.TrimPrefix(extension, ".") {
		if !unicode.IsLetter(character) && !unicode.IsDigit(character) {
			return ""
		}
	}
	return extension
}

func mutationResult(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return ErrNotFound
	}
	return nil
}
