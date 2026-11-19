package privacy

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"solace/internal/database"
)

func TestConfigureLockUnlockAndDisable(t *testing.T) {
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "solace.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := New(db)
	status, err := service.GetStatus(context.Background())
	if err != nil || status.Enabled || status.Locked {
		t.Fatalf("initial status = %+v, %v", status, err)
	}

	status, err = service.Configure(context.Background(), Configuration{NewPassword: "correct horse battery", LockOnStart: true, InactivityMinutes: 15})
	if err != nil || !status.Enabled || status.Locked {
		t.Fatalf("configured status = %+v, %v", status, err)
	}
	var stored string
	if err := db.QueryRow(`SELECT password_hash FROM privacy_settings WHERE id=1`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == "" || stored == "correct horse battery" {
		t.Fatal("password was not stored as a derived hash")
	}
	restarted, err := New(db).GetStatus(context.Background())
	if err != nil || !restarted.Locked {
		t.Fatalf("startup lock status = %+v, %v", restarted, err)
	}

	if status, err = service.Lock(context.Background()); err != nil || !status.Locked {
		t.Fatalf("locked status = %+v, %v", status, err)
	}
	if _, err := service.Unlock(context.Background(), "wrong password"); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("wrong password error = %v", err)
	}
	if status, err = service.Unlock(context.Background(), "correct horse battery"); err != nil || status.Locked {
		t.Fatalf("unlock status = %+v, %v", status, err)
	}
	if _, err := service.Disable(context.Background(), "wrong password"); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("wrong disable password error = %v", err)
	}
	if status, err = service.Disable(context.Background(), "correct horse battery"); err != nil || status.Enabled {
		t.Fatalf("disabled status = %+v, %v", status, err)
	}
}

func TestPasswordValidation(t *testing.T) {
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "solace.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := New(db)
	if _, err := service.Configure(context.Background(), Configuration{NewPassword: "short", InactivityMinutes: 5}); err == nil {
		t.Fatal("short password accepted")
	}
	if _, err := service.Configure(context.Background(), Configuration{NewPassword: "long enough", InactivityMinutes: 121}); err == nil {
		t.Fatal("invalid inactivity accepted")
	}
}
