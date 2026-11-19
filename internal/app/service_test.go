package app

import (
	"context"
	"path/filepath"
	"testing"

	"solace/internal/database"
	"solace/internal/platform"
	"solace/internal/settings"
)

func TestServiceFrontendContract(t *testing.T) {
	paths := platform.PathsForRoot(t.TempDir())
	db, err := database.Open(context.Background(), filepath.Join(paths.Root, "solace.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	service := NewService(db, paths)
	info := service.GetInfo()
	if !info.Initialized || info.Name != "Solace" || info.DataPaths.Root != paths.Root {
		t.Fatalf("unexpected application info: %+v", info)
	}
	want := settings.Defaults()
	want.Theme = settings.ThemeLight
	want.SidebarCollapsed = true
	if err := service.SaveSettings(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	got, err := service.LoadSettings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("LoadSettings() = %+v, want %+v", got, want)
	}
	health := service.CheckDataHealth(context.Background())
	if !health.Healthy || health.CheckedAt == "" || health.Message == "" {
		t.Fatalf("data health = %+v", health)
	}
}
