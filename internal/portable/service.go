package portable

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"solace/internal/platform"
)

const appVersion = "0.1.0-dev"

var ErrCancelled = errors.New("operation cancelled")

type Service struct {
	db        *sql.DB
	paths     platform.Paths
	operation sync.Mutex
	cancelled atomic.Bool
}

func New(db *sql.DB, paths platform.Paths) *Service { return &Service{db: db, paths: paths} }

func (s *Service) CancelCurrentOperation() { s.cancelled.Store(true) }

func (s *Service) Export(ctx context.Context, request ExportRequest) (OperationResult, error) {
	s.operation.Lock()
	defer s.operation.Unlock()
	s.cancelled.Store(false)
	if err := validateExportRequest(request); err != nil {
		return OperationResult{}, err
	}
	journal, err := s.loadJournal(ctx, request, false)
	if err != nil {
		return OperationResult{}, err
	}
	if !request.IncludeAttachments {
		journal.Attachments = []AttachmentRecord{}
	}
	result := OperationResult{OutputPath: request.Destination, EntryCount: len(journal.Entries), AttachmentCount: len(journal.Attachments), Warnings: []string{}}
	switch request.Format {
	case "markdown":
		err = s.exportMarkdown(ctx, request.Destination, journal, request.IncludeAttachments)
	case "html":
		err = s.exportHTML(ctx, request.Destination, journal, request.IncludeAttachments)
	case "json":
		err = s.exportJSON(request.Destination, journal)
	case "archive":
		err = s.exportArchive(ctx, request.Destination, journal, request.IncludeAttachments)
	default:
		err = fmt.Errorf("unsupported export format %q", request.Format)
	}
	if err != nil {
		return OperationResult{}, err
	}
	result.OperationMessage = fmt.Sprintf("Exported %d entries", result.EntryCount)
	return result, nil
}

func (s *Service) InspectArchive(ctx context.Context, archivePath string) (ArchiveSummary, error) {
	return s.inspectArchive(ctx, archivePath)
}

func (s *Service) RestoreArchive(ctx context.Context, archivePath string, replaceExisting bool) (OperationResult, error) {
	s.operation.Lock()
	defer s.operation.Unlock()
	s.cancelled.Store(false)
	return s.restoreArchive(ctx, archivePath, replaceExisting)
}

func (s *Service) LoadBackupSettings(ctx context.Context) (BackupSettings, error) {
	var result BackupSettings
	var include int
	err := s.db.QueryRowContext(ctx, `SELECT schedule, destination, retention, include_attachments, last_backup_at FROM backup_settings WHERE id=1`).
		Scan(&result.Schedule, &result.Destination, &result.Retention, &include, &result.LastBackupAt)
	if err != nil {
		return result, fmt.Errorf("load backup settings: %w", err)
	}
	result.IncludeAttachments = include == 1
	if result.Destination == "" {
		result.Destination = s.paths.Backups
	}
	return result, nil
}

func (s *Service) SaveBackupSettings(ctx context.Context, value BackupSettings) error {
	if value.Schedule != "manual" && value.Schedule != "daily" && value.Schedule != "weekly" {
		return fmt.Errorf("invalid backup schedule %q", value.Schedule)
	}
	if value.Retention < 1 || value.Retention > 100 {
		return fmt.Errorf("backup retention must be between 1 and 100")
	}
	destination := strings.TrimSpace(value.Destination)
	if destination == "" {
		destination = s.paths.Backups
	}
	if !filepath.IsAbs(destination) {
		return fmt.Errorf("backup destination must be an absolute path")
	}
	if err := os.MkdirAll(destination, 0o700); err != nil {
		return fmt.Errorf("create backup destination: %w", err)
	}
	_, err := s.db.ExecContext(ctx, `UPDATE backup_settings SET schedule=?, destination=?, retention=?, include_attachments=? WHERE id=1`, value.Schedule, destination, value.Retention, boolInt(value.IncludeAttachments))
	if err != nil {
		return fmt.Errorf("save backup settings: %w", err)
	}
	return nil
}

func (s *Service) CreateBackup(ctx context.Context) (OperationResult, error) {
	s.operation.Lock()
	defer s.operation.Unlock()
	s.cancelled.Store(false)
	settings, err := s.LoadBackupSettings(ctx)
	if err != nil {
		return OperationResult{}, err
	}
	return s.createBackup(ctx, settings, "solace-backup")
}

func (s *Service) RunScheduledBackup(ctx context.Context) error {
	settings, err := s.LoadBackupSettings(ctx)
	if err != nil || settings.Schedule == "manual" {
		return err
	}
	last, _ := time.Parse(time.RFC3339Nano, settings.LastBackupAt)
	interval := 24 * time.Hour
	if settings.Schedule == "weekly" {
		interval = 7 * 24 * time.Hour
	}
	if !last.IsZero() && time.Since(last) < interval {
		return nil
	}
	_, err = s.CreateBackup(ctx)
	return err
}

func (s *Service) createBackup(ctx context.Context, settings BackupSettings, prefix string) (OperationResult, error) {
	if err := os.MkdirAll(settings.Destination, 0o700); err != nil {
		return OperationResult{}, fmt.Errorf("create backup destination: %w", err)
	}
	destination := filepath.Join(settings.Destination, fmt.Sprintf("%s-%s.zip", prefix, time.Now().UTC().Format("20060102-150405.000000000")))
	journal, err := s.loadJournal(ctx, ExportRequest{Scope: "entire", IncludeAttachments: settings.IncludeAttachments}, true)
	if err != nil {
		return OperationResult{}, err
	}
	if !settings.IncludeAttachments {
		journal.Attachments = []AttachmentRecord{}
	}
	if err := s.exportArchive(ctx, destination, journal, settings.IncludeAttachments); err != nil {
		return OperationResult{}, err
	}
	if prefix == "solace-backup" {
		completed := time.Now().UTC().Format(time.RFC3339Nano)
		if _, err := s.db.ExecContext(ctx, `UPDATE backup_settings SET last_backup_at=? WHERE id=1`, completed); err != nil {
			return OperationResult{}, fmt.Errorf("record backup completion: %w", err)
		}
		if err := pruneBackups(settings.Destination, settings.Retention); err != nil {
			return OperationResult{}, err
		}
	}
	return OperationResult{OutputPath: destination, EntryCount: len(journal.Entries), AttachmentCount: len(journal.Attachments), Warnings: []string{}, OperationMessage: "Backup completed"}, nil
}

func (s *Service) checkCancelled() error {
	if s.cancelled.Load() {
		return ErrCancelled
	}
	return nil
}

func validateExportRequest(request ExportRequest) error {
	if request.Format != "markdown" && request.Format != "html" && request.Format != "json" && request.Format != "archive" {
		return fmt.Errorf("invalid export format")
	}
	if request.Scope != "entire" && request.Scope != "date" && request.Scope != "selected" {
		return fmt.Errorf("invalid export scope")
	}
	if request.Scope == "date" && (request.StartDate == "" || request.EndDate == "" || request.StartDate > request.EndDate) {
		return fmt.Errorf("valid start and end dates are required")
	}
	if request.Scope == "selected" && len(request.EntryIDs) == 0 {
		return fmt.Errorf("select at least one entry")
	}
	if strings.TrimSpace(request.Destination) == "" || !filepath.IsAbs(request.Destination) {
		return fmt.Errorf("an absolute destination path is required")
	}
	return nil
}

func pruneBackups(directory string, retention int) error {
	matches, err := filepath.Glob(filepath.Join(directory, "solace-backup-*.zip"))
	if err != nil {
		return err
	}
	sort.Strings(matches)
	for len(matches) > retention {
		if err := os.Remove(matches[0]); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove expired backup: %w", err)
		}
		matches = matches[1:]
	}
	return nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
