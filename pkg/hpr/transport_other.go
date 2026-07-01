//go:build !windows

package hpr

// On non-Windows builds, scanDevicesImpl stays at the default panic
// value (see manager.go). This file is intentionally empty so the
// build succeeds on every platform; only Windows has an actual
// platform implementation today.