package privacy

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
)

const (
	passwordIterations   = 310_000
	derivedKeyBytes      = 32
	minimumPasswordRunes = 8
)

var ErrInvalidPassword = errors.New("incorrect password")

type Status struct {
	Enabled           bool `json:"enabled"`
	Locked            bool `json:"locked"`
	LockOnStart       bool `json:"lockOnStart"`
	InactivityMinutes int  `json:"inactivityMinutes"`
}

type Configuration struct {
	CurrentPassword   string `json:"currentPassword"`
	NewPassword       string `json:"newPassword"`
	LockOnStart       bool   `json:"lockOnStart"`
	InactivityMinutes int    `json:"inactivityMinutes"`
}

type storedSettings struct {
	enabled           bool
	salt              string
	hash              string
	iterations        int
	lockOnStart       bool
	inactivityMinutes int
}

type Service struct {
	db     *sql.DB
	mu     sync.Mutex
	loaded bool
	locked bool
}

func New(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) GetStatus(ctx context.Context) (Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	settings, err := s.load(ctx)
	if err != nil {
		return Status{}, err
	}
	if !s.loaded {
		s.locked = settings.enabled && settings.lockOnStart
		s.loaded = true
	}
	return status(settings, s.locked), nil
}

func (s *Service) Configure(ctx context.Context, value Configuration) (Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	settings, err := s.load(ctx)
	if err != nil {
		return Status{}, err
	}
	if value.InactivityMinutes < 0 || value.InactivityMinutes > 120 {
		return Status{}, fmt.Errorf("inactivity timeout must be between 0 and 120 minutes")
	}
	if settings.enabled {
		if err := verifyPassword(value.CurrentPassword, settings); err != nil {
			return Status{}, err
		}
	}
	password := value.NewPassword
	if settings.enabled && strings.TrimSpace(password) == "" {
		password = value.CurrentPassword
	}
	if len([]rune(password)) < minimumPasswordRunes {
		return Status{}, fmt.Errorf("password must contain at least %d characters", minimumPasswordRunes)
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return Status{}, fmt.Errorf("create password salt: %w", err)
	}
	hash, err := pbkdf2.Key(sha256.New, password, salt, passwordIterations, derivedKeyBytes)
	if err != nil {
		return Status{}, fmt.Errorf("derive password hash: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `UPDATE privacy_settings SET enabled=1, password_salt=?, password_hash=?, password_iterations=?, lock_on_start=?, inactivity_minutes=? WHERE id=1`,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash), passwordIterations, boolInt(value.LockOnStart), value.InactivityMinutes)
	if err != nil {
		return Status{}, fmt.Errorf("save privacy settings: %w", err)
	}
	s.loaded, s.locked = true, false
	return Status{Enabled: true, Locked: false, LockOnStart: value.LockOnStart, InactivityMinutes: value.InactivityMinutes}, nil
}

func (s *Service) Disable(ctx context.Context, currentPassword string) (Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	settings, err := s.load(ctx)
	if err != nil {
		return Status{}, err
	}
	if !settings.enabled {
		return Status{}, nil
	}
	if err := verifyPassword(currentPassword, settings); err != nil {
		return Status{}, err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE privacy_settings SET enabled=0, password_salt='', password_hash='', password_iterations=0, lock_on_start=0, inactivity_minutes=0 WHERE id=1`); err != nil {
		return Status{}, fmt.Errorf("disable privacy lock: %w", err)
	}
	s.loaded, s.locked = true, false
	return Status{}, nil
}

func (s *Service) Unlock(ctx context.Context, password string) (Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	settings, err := s.load(ctx)
	if err != nil {
		return Status{}, err
	}
	if settings.enabled {
		if err := verifyPassword(password, settings); err != nil {
			return status(settings, true), err
		}
	}
	s.loaded, s.locked = true, false
	return status(settings, false), nil
}

func (s *Service) Lock(ctx context.Context) (Status, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	settings, err := s.load(ctx)
	if err != nil {
		return Status{}, err
	}
	s.loaded = true
	s.locked = settings.enabled
	return status(settings, s.locked), nil
}

func (s *Service) load(ctx context.Context) (storedSettings, error) {
	var value storedSettings
	var enabled, lockOnStart int
	err := s.db.QueryRowContext(ctx, `SELECT enabled, password_salt, password_hash, password_iterations, lock_on_start, inactivity_minutes FROM privacy_settings WHERE id=1`).
		Scan(&enabled, &value.salt, &value.hash, &value.iterations, &lockOnStart, &value.inactivityMinutes)
	if err != nil {
		return value, fmt.Errorf("load privacy settings: %w", err)
	}
	value.enabled, value.lockOnStart = enabled == 1, lockOnStart == 1
	return value, nil
}

func verifyPassword(password string, settings storedSettings) error {
	salt, saltErr := base64.RawStdEncoding.DecodeString(settings.salt)
	want, hashErr := base64.RawStdEncoding.DecodeString(settings.hash)
	if saltErr != nil || hashErr != nil || len(salt) < 16 || len(want) != derivedKeyBytes || settings.iterations < 100_000 {
		return fmt.Errorf("stored privacy lock is invalid")
	}
	got, err := pbkdf2.Key(sha256.New, password, salt, settings.iterations, len(want))
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return ErrInvalidPassword
	}
	return nil
}

func status(settings storedSettings, locked bool) Status {
	return Status{Enabled: settings.enabled, Locked: settings.enabled && locked, LockOnStart: settings.lockOnStart, InactivityMinutes: settings.inactivityMinutes}
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
