package main

import (
	"context"
	"embed"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime/debug"

	solaceapp "solace/internal/app"
	"solace/internal/attachments"
	"solace/internal/database"
	"solace/internal/diary"
	solacelog "solace/internal/logging"
	"solace/internal/platform"
	"solace/internal/portable"
	"solace/internal/privacy"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Solace failed to start: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) (runErr error) {
	platform.HardenProcessPermissions()

	paths, err := platform.ResolvePaths()
	if err != nil {
		return err
	}
	if err := platform.EnsureDirectories(paths); err != nil {
		return err
	}

	logger, logFile, err := newLogger(paths.Logs)
	if err != nil {
		return err
	}
	defer logFile.Close()
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.Error("Solace stopped after an unexpected internal error", "panic", fmt.Sprint(recovered), "stack", string(debug.Stack()))
			runErr = fmt.Errorf("unexpected internal error")
		} else if runErr != nil {
			logger.Error("Solace stopped with an error", "error", runErr)
		}
	}()

	db, err := database.Open(ctx, paths.Database)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	defer db.Close()

	service := solaceapp.NewService(db, paths, logger)
	diaryService := diary.New(db)
	attachmentService := attachments.New(db, paths)
	portableService := portable.New(db, paths)
	privacyService := privacy.New(db)
	assetHandler := attachments.Handler(attachmentService, application.AssetFileServerFS(assets))
	app := application.New(application.Options{
		Name:        "Solace",
		Description: "A private, local-first personal diary and memory archive",
		Logger:      logger,
		Services: []application.Service{
			application.NewService(service),
			application.NewService(diaryService),
			application.NewService(attachmentService),
			application.NewService(portableService),
			application.NewService(privacyService),
		},
		Assets: application.AssetOptions{
			Handler: assetHandler,
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Solace",
		Width:            1240,
		Height:           780,
		MinWidth:         800,
		MinHeight:        600,
		BackgroundColour: application.NewRGB(246, 243, 237),
		URL:              "/",
		EnableFileDrop:   true,
	})
	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		window.EmitEvent("solace:files-dropped", event.Context().DroppedFiles())
	})

	logger.Info("starting Solace", "version", "0.1.0", "dataRoot", paths.Root)
	go func() {
		if err := portableService.RunScheduledBackup(context.Background()); err != nil {
			logger.Error("scheduled backup failed", "error", err)
		}
	}()
	if err := app.Run(); err != nil {
		return fmt.Errorf("run application: %w", err)
	}
	return nil
}

func newLogger(logDir string) (*slog.Logger, io.WriteCloser, error) {
	file, err := solacelog.NewRotatingWriter(solacelog.LogPath(logDir), 2*1024*1024, 5)
	if err != nil {
		return nil, nil, fmt.Errorf("open application log: %w", err)
	}
	handler := slog.NewJSONHandler(io.MultiWriter(os.Stderr, file), &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(handler), file, nil
}
