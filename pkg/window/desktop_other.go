//go:build !linux

package window

// EnsureDesktopEntry is a no-op on non-Linux platforms.
func EnsureDesktopEntry() {
}
