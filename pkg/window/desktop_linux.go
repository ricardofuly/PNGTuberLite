//go:build linux

package window

import (
	"fmt"
	"os"
	"os/exec"
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

	// 1. Install Icons into standard XDG directories
	if len(assets.AppLogoPNG) > 0 {
		// Large & medium icons
		for _, s := range []string{"64x64", "128x128", "256x256", "scalable"} {
			sDir := filepath.Join(homeDir, ".local", "share", "icons", "hicolor", s, "apps")
			_ = os.MkdirAll(sDir, 0755)
			_ = os.WriteFile(filepath.Join(sDir, "pngtuber-lite.png"), assets.AppLogoPNG, 0644)
		}

		// Small / tray-sized icons
		trayBytes := assets.AppTrayPNG
		if len(trayBytes) == 0 {
			trayBytes = assets.AppLogoPNG
		}
		for _, s := range []string{"16x16", "24x24", "32x32", "48x48"} {
			sDir := filepath.Join(homeDir, ".local", "share", "icons", "hicolor", s, "apps")
			_ = os.MkdirAll(sDir, 0755)
			_ = os.WriteFile(filepath.Join(sDir, "pngtuber-lite.png"), trayBytes, 0644)
		}

		pixmapDir := filepath.Join(homeDir, ".local", "share", "pixmaps")
		_ = os.MkdirAll(pixmapDir, 0755)
		_ = os.WriteFile(filepath.Join(pixmapDir, "pngtuber-lite.png"), assets.AppLogoPNG, 0644)
	}

	appDir := filepath.Join(homeDir, ".local", "share", "applications")
	_ = os.MkdirAll(appDir, 0755)

	// 2. Install Primary .desktop file (Ubuntu Dock & Application Menu)
	primaryDesktop := filepath.Join(appDir, "pngtuber-lite.desktop")
	primaryContent := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=PNGTuber Lite
GenericName=PNGTuber Avatar
Comment=Engine leve para avatares 2D com suporte a .save
Exec=%s
Icon=pngtuber-lite
Terminal=false
Categories=AudioVideo;Graphics;Utility;
StartupWMClass=pngtuber-lite
StartupNotify=true
`, cleanExecPath)

	_ = os.WriteFile(primaryDesktop, []byte(primaryContent), 0755)
	_ = exec.Command("gio", "set", primaryDesktop, "metadata::trusted", "true").Run()

	// 3. Install Fallback WM_CLASS association entries for GNOME Shell / Ubuntu Dock
	// Different versions of Raylib/GLFW may identify as 'PNGTuber Lite', 'raylib', or 'main'
	wmAliases := map[string]string{
		"pngtuber-lite-wm-title.desktop":  "PNGTuber Lite",
		"pngtuber-lite-wm-raylib.desktop": "raylib",
		"pngtuber-lite-wm-main.desktop":   "main",
		"pngtuber-lite-wm-glfw.desktop":   "GLFW-Application",
	}

	for fileName, wmClass := range wmAliases {
		aliasPath := filepath.Join(appDir, fileName)
		aliasContent := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=PNGTuber Lite
GenericName=PNGTuber Avatar
Comment=Engine leve para avatares 2D com suporte a .save
Exec=%s
Icon=pngtuber-lite
Terminal=false
NoDisplay=true
Categories=AudioVideo;Graphics;Utility;
StartupWMClass=%s
StartupNotify=true
`, cleanExecPath, wmClass)
		_ = os.WriteFile(aliasPath, []byte(aliasContent), 0755)
		_ = exec.Command("gio", "set", aliasPath, "metadata::trusted", "true").Run()
	}

	// 4. Update local desktop and icon caches
	_ = exec.Command("update-desktop-database", appDir).Run()
	_ = exec.Command("gtk-update-icon-cache", "-f", "-t", filepath.Join(homeDir, ".local", "share", "icons", "hicolor")).Run()
}
