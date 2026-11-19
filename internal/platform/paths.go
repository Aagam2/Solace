package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

// Paths contains every location Solace owns. These paths never point into
// the installation directory, which keeps packaged applications read-only.
type Paths struct {
	Root       string `json:"root"`
	Database   string `json:"database"`
	Media      string `json:"media"`
	Images     string `json:"images"`
	Videos     string `json:"videos"`
	Audio      string `json:"audio"`
	Documents  string `json:"documents"`
	Thumbnails string `json:"thumbnails"`
	Backups    string `json:"backups"`
	Logs       string `json:"logs"`
}

func ResolvePaths() (Paths, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, fmt.Errorf("resolve user data directory: %w", err)
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		root = xdg
	} else if home, err := os.UserHomeDir(); err == nil && home != "" && os.PathSeparator == '/' {
		root = filepath.Join(home, ".local", "share")
	}
	return PathsForRoot(filepath.Join(root, "solace")), nil
}

func PathsForRoot(root string) Paths {
	media := filepath.Join(root, "media")
	return Paths{
		Root: root, Database: filepath.Join(root, "solace.db"), Media: media,
		Images: filepath.Join(media, "images"), Videos: filepath.Join(media, "videos"),
		Audio: filepath.Join(media, "audio"), Documents: filepath.Join(media, "documents"),
		Thumbnails: filepath.Join(root, "thumbnails"), Backups: filepath.Join(root, "backups"),
		Logs: filepath.Join(root, "logs"),
	}
}

func EnsureDirectories(paths Paths) error {
	dirs := []string{paths.Root, paths.Images, paths.Videos, paths.Audio, paths.Documents, paths.Thumbnails, paths.Backups, paths.Logs}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create data directory %q: %w", dir, err)
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			return fmt.Errorf("secure data directory %q: %w", dir, err)
		}
	}
	return nil
}
