//go:build linux

package window

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"pngtuber-lite/assets"
)

// EnsureDesktopEntry registers the application icon and .desktop launcher on Linux (Ubuntu/GNOME Dock).
func EnsureDesktopEntry() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	execPath, err := os.Executable()
	if err != nil {
		return
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return
	}

	// Clean any .old suffixes if running
	dir := filepath.Dir(execPath)
	base := filepath.Base(execPath)
	for strings.HasSuffix(base, ".old") {
		base = strings.TrimSuffix(base, ".old")
	}
	cleanExecPath := filepath.Join(dir, base)

	// 1. Install Icon to ~/.local/share/icons/hicolor/256x256/apps/pngtuber-lite.png
	iconDir := filepath.Join(homeDir, ".local", "share", "icons", "hicolor", "256x256", "apps")
	_ = os.MkdirAll(iconDir, 0755)
	iconPath := filepath.Join(iconDir, "pngtuber-lite.png")

	if len(assets.AppLogoPNG) > 0 {
		_ = os.WriteFile(iconPath, assets.AppLogoPNG, 0644)
	}

	// 2. Install .desktop file to ~/.local/share/applications/pngtuber-lite.desktop
	appDir := filepath.Join(homeDir, ".local", "share", "applications")
	_ = os.MkdirAll(appDir, 0755)
	desktopPath := filepath.Join(appDir, "pngtuber-lite.desktop")

	desktopContent := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=PNGTuber Lite
Comment=Engine leve para avatares 2D com suporte a .save
Exec=%s
Icon=pngtuber-lite
Terminal=false
Categories=AudioVideo;Graphics;Utility;
StartupWMClass=pngtuber-lite
`, cleanExecPath)

	_ = os.WriteFile(desktopPath, []byte(desktopContent), 0644)
}
