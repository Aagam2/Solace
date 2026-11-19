package settings_test

import (
	"context"
	"path/filepath"
	"testing"

	"solace/internal/database"
	"solace/internal/settings"
)

func TestLoadAndSaveSettings(t *testing.T) {
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "solace.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := settings.New(db)
	initial, err := service.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if initial.Theme != settings.ThemeSystem || initial.SidebarCollapsed || initial.EditorFontSize != 17 || initial.EditorWidth != settings.EditorWidthComfortable || initial.AutosaveDelayMS != 700 || !initial.ExportIncludeAttachments {
		t.Fatalf("unexpected defaults: %+v", initial)
	}

	want := settings.Settings{
		Theme: settings.ThemeDark, SidebarCollapsed: true, EditorFontSize: 19,
		EditorWidth: settings.EditorWidthWide, EditorLineSpacing: 1.9, AutosaveDelayMS: 1500,
		DefaultExportFormat: "html", ExportIncludeAttachments: false, DateFormat: settings.DateFormatISO,
	}
	if err := service.Save(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	got, err := service.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("Load() = %+v, want %+v", got, want)
	}
}

func TestSaveSettingsRejectsInvalidTheme(t *testing.T) {
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "solace.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	value := settings.Defaults()
	value.Theme = "sepia"
	err = settings.New(db).Save(context.Background(), value)
	if err == nil {
		t.Fatal("Save() expected validation error")
	}
}

func TestSaveSettingsRejectsUnsafeEditorValues(t *testing.T) {
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "solace.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	value := settings.Defaults()
	value.EditorFontSize = 42
	if err := settings.New(db).Save(context.Background(), value); err == nil {
		t.Fatal("Save() expected editor validation error")
	}
}
