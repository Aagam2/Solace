//go:build !windows

package platform

import "syscall"

// HardenProcessPermissions ensures files created by SQLite, WebKit, and future
// media imports are private to the current user by default.
func HardenProcessPermissions() {
	syscall.Umask(0o077)
}
