package settings

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

type Theme string
type EditorWidth string
type DateFormat string

const (
	ThemeSystem Theme = "system"
	ThemeLight  Theme = "light"
	ThemeDark   Theme = "dark"

	EditorWidthNarrow      EditorWidth = "narrow"
	EditorWidthComfortable EditorWidth = "comfortable"
	EditorWidthWide        EditorWidth = "wide"

	DateFormatSystem DateFormat = "system"
	DateFormatLong   DateFormat = "long"
	DateFormatISO    DateFormat = "iso"
)

type Settings struct {
	Theme                    Theme       `json:"theme"`
	SidebarCollapsed         bool        `json:"sidebarCollapsed"`
	EditorFontSize           int         `json:"editorFontSize"`
	EditorWidth              EditorWidth `json:"editorWidth"`
	EditorLineSpacing        float64     `json:"editorLineSpacing"`
	AutosaveDelayMS          int         `json:"autosaveDelayMs"`
	DefaultExportFormat      string      `json:"defaultExportFormat"`
	ExportIncludeAttachments bool        `json:"exportIncludeAttachments"`
	DateFormat               DateFormat  `json:"dateFormat"`
}

type Service struct{ db *sql.DB }

func New(db *sql.DB) *Service { return &Service{db: db} }

func Defaults() Settings {
	return Settings{
		Theme: ThemeSystem, EditorFontSize: 17, EditorWidth: EditorWidthComfortable,
		EditorLineSpacing: 1.75, AutosaveDelayMS: 700, DefaultExportFormat: "archive",
		ExportIncludeAttachments: true, DateFormat: DateFormatSystem,
	}
}

func (s *Service) Load(ctx context.Context) (Settings, error) {
	result := Defaults()
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return result, fmt.Errorf("load settings: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return result, fmt.Errorf("scan setting: %w", err)
		}
		switch key {
		case "theme":
			result.Theme = Theme(value)
		case "sidebar_collapsed":
			result.SidebarCollapsed = value == "true"
		case "editor_font_size":
			if parsed, parseErr := strconv.Atoi(value); parseErr == nil {
				result.EditorFontSize = parsed
			}
		case "editor_width":
			result.EditorWidth = EditorWidth(value)
		case "editor_line_spacing":
			if parsed, parseErr := strconv.ParseFloat(value, 64); parseErr == nil {
				result.EditorLineSpacing = parsed
			}
		case "autosave_delay_ms":
			if parsed, parseErr := strconv.Atoi(value); parseErr == nil {
				result.AutosaveDelayMS = parsed
			}
		case "default_export_format":
			result.DefaultExportFormat = value
		case "export_include_attachments":
			result.ExportIncludeAttachments = value == "true"
		case "date_format":
			result.DateFormat = DateFormat(value)
		}
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	if err := validate(result); err != nil {
		return Defaults(), fmt.Errorf("stored settings are invalid: %w", err)
	}
	return result, nil
}

func (s *Service) Save(ctx context.Context, value Settings) error {
	if err := validate(value); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin settings update: %w", err)
	}
	defer tx.Rollback()
	updatedAt := time.Now().UTC().Format(time.RFC3339Nano)
	values := map[string]string{
		"theme": string(value.Theme), "sidebar_collapsed": strconv.FormatBool(value.SidebarCollapsed),
		"editor_font_size": strconv.Itoa(value.EditorFontSize), "editor_width": string(value.EditorWidth),
		"editor_line_spacing": strconv.FormatFloat(value.EditorLineSpacing, 'f', 2, 64),
		"autosave_delay_ms":   strconv.Itoa(value.AutosaveDelayMS), "default_export_format": value.DefaultExportFormat,
		"export_include_attachments": strconv.FormatBool(value.ExportIncludeAttachments), "date_format": string(value.DateFormat),
	}
	for key, setting := range values {
		if _, err := tx.ExecContext(ctx, `INSERT INTO settings(key, value, updated_at) VALUES(?, ?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at`, key, setting, updatedAt); err != nil {
			return fmt.Errorf("save setting %s: %w", key, err)
		}
	}
	return tx.Commit()
}

func validate(value Settings) error {
	if value.Theme != ThemeSystem && value.Theme != ThemeLight && value.Theme != ThemeDark {
		return fmt.Errorf("invalid theme %q", value.Theme)
	}
	if value.EditorWidth != EditorWidthNarrow && value.EditorWidth != EditorWidthComfortable && value.EditorWidth != EditorWidthWide {
		return fmt.Errorf("invalid editor width %q", value.EditorWidth)
	}
	if value.EditorFontSize < 14 || value.EditorFontSize > 24 {
		return fmt.Errorf("editor font size must be between 14 and 24")
	}
	if value.EditorLineSpacing < 1.4 || value.EditorLineSpacing > 2.2 {
		return fmt.Errorf("editor line spacing must be between 1.4 and 2.2")
	}
	if value.AutosaveDelayMS < 300 || value.AutosaveDelayMS > 5000 {
		return fmt.Errorf("autosave delay must be between 300 and 5000 milliseconds")
	}
	switch value.DefaultExportFormat {
	case "markdown", "html", "json", "archive":
	default:
		return fmt.Errorf("invalid default export format %q", value.DefaultExportFormat)
	}
	if value.DateFormat != DateFormatSystem && value.DateFormat != DateFormatLong && value.DateFormat != DateFormatISO {
		return fmt.Errorf("invalid date format %q", value.DateFormat)
	}
	return nil
}
