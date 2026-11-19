package app

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"strings"
	"time"

	"solace/internal/database"
	"solace/internal/platform"
	"solace/internal/settings"
)

type Info struct {
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	DataPaths   platform.Paths `json:"dataPaths"`
	Initialized bool           `json:"initialized"`
}

type DataHealth struct {
	Healthy   bool   `json:"healthy"`
	CheckedAt string `json:"checkedAt"`
	Message   string `json:"message"`
}

// Service is the stable frontend/backend boundary. A Wails v3 adapter can
// expose these methods without coupling domain packages to the UI runtime.
type Service struct {
	db       *sql.DB
	paths    platform.Paths
	settings *settings.Service
	logger   *slog.Logger
}

func NewService(db *sql.DB, paths platform.Paths, loggers ...*slog.Logger) *Service {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	}
	return &Service{db: db, paths: paths, settings: settings.New(db), logger: logger}
}
func (s *Service) CheckDataHealth(ctx context.Context) DataHealth {
	checkedAt := time.Now().UTC().Format(time.RFC3339Nano)
	if err := database.CheckIntegrity(ctx, s.db); err != nil {
		s.logger.Error("journal data health check failed", "error", err)
		return DataHealth{Healthy: false, CheckedAt: checkedAt, Message: "Your journal needs attention. Nothing was changed; keep Solace open and use a recent backup before making further changes."}
	}
	return DataHealth{Healthy: true, CheckedAt: checkedAt, Message: "Your journal data passed every safety check."}
}
func (s *Service) GetInfo() Info {
	return Info{Name: "Solace", Version: "0.1.0", DataPaths: s.paths, Initialized: true}
}
func (s *Service) LoadSettings(ctx context.Context) (settings.Settings, error) {
	return s.settings.Load(ctx)
}
func (s *Service) SaveSettings(ctx context.Context, value settings.Settings) error {
	return s.settings.Save(ctx, value)
}

// ReportClientError records technical UI failures without requiring the
// interface to expose raw implementation details to the person using Solace.
// Callers must not include journal text, passwords, or other private content.
func (s *Service) ReportClientError(area, detail string) {
	area = singleLine(area, 80)
	detail = singleLine(detail, 1000)
	if area == "" {
		area = "interface"
	}
	if detail == "" {
		detail = "No technical detail was available"
	}
	s.logger.Error("frontend operation failed", "area", area, "error", detail)
}

func singleLine(value string, limit int) string {
	value = strings.TrimSpace(strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(value))
	if len(value) > limit {
		return value[:limit] + "…"
	}
	return value
}
