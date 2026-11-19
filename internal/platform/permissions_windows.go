//go:build windows

package platform

// Windows access control is inherited from the user's profile directory.
func HardenProcessPermissions() {}
