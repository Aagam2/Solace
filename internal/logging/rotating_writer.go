package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// RotatingWriter keeps a bounded set of private log files. Rotation is based
// on size so it works without a background process and remains predictable on
// every desktop platform.
type RotatingWriter struct {
	mu        sync.Mutex
	path      string
	maxBytes  int64
	backups   int
	file      *os.File
	size      int64
}

func NewRotatingWriter(path string, maxBytes int64, backups int) (*RotatingWriter, error) {
	if maxBytes <= 0 || backups < 1 {
		return nil, fmt.Errorf("invalid log rotation limits")
	}
	writer := &RotatingWriter{path: path, maxBytes: maxBytes, backups: backups}
	if err := writer.open(); err != nil {
		return nil, err
	}
	if writer.size >= writer.maxBytes {
		if err := writer.rotate(); err != nil {
			_ = writer.file.Close()
			return nil, err
		}
	}
	return writer, nil
}

func (w *RotatingWriter) Write(value []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return 0, os.ErrClosed
	}
	if w.size > 0 && w.size+int64(len(value)) > w.maxBytes {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}
	written, err := w.file.Write(value)
	w.size += int64(written)
	return written, err
}

func (w *RotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func (w *RotatingWriter) open() error {
	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return fmt.Errorf("secure log file: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("inspect log file: %w", err)
	}
	w.file = file
	w.size = info.Size()
	return nil
}

func (w *RotatingWriter) rotate() error {
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return fmt.Errorf("close log before rotation: %w", err)
		}
		w.file = nil
	}
	oldest := fmt.Sprintf("%s.%d", w.path, w.backups)
	if err := os.Remove(oldest); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove oldest log: %w", err)
	}
	for index := w.backups - 1; index >= 1; index-- {
		from := fmt.Sprintf("%s.%d", w.path, index)
		to := fmt.Sprintf("%s.%d", w.path, index+1)
		if err := os.Rename(from, to); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("rotate log backup: %w", err)
		}
	}
	if err := os.Rename(w.path, w.path+".1"); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("rotate active log: %w", err)
	}
	return w.open()
}

func LogPath(directory string) string {
	return filepath.Join(directory, "solace.log")
}
