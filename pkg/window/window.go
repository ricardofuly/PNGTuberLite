package window

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"pngtuber-lite/pkg/config"
)

// WindowManager handles Raylib window configuration, transparent overlay settings, and state toggles.
type WindowManager struct {
	Config          *config.Config
	isClickThrough  bool
}

// NewWindowManager creates a new window manager.
func NewWindowManager(cfg *config.Config) *WindowManager {
	return &WindowManager{
		Config:         cfg,
		isClickThrough: false,
	}
}

// InitWindow configures and creates the Raylib transparent window.
func (wm *WindowManager) InitWindow() {
	var flags uint32 = rl.FlagWindowResizable

	if wm.Config.WindowTransparent {
		flags |= rl.FlagWindowTransparent
	}
	if wm.Config.WindowAlwaysOnTop {
		flags |= rl.FlagWindowTopmost
	}
	if wm.Config.WindowBorderless {
		flags |= rl.FlagWindowUndecorated
	}

	rl.SetConfigFlags(flags)
	rl.InitWindow(wm.Config.WindowWidth, wm.Config.WindowHeight, "PNGTuber Lite")

	if wm.Config.TargetFPS > 0 {
		rl.SetTargetFPS(wm.Config.TargetFPS)
	} else {
		rl.SetTargetFPS(60)
	}

	if wm.Config.WindowX >= 0 && wm.Config.WindowY >= 0 {
		rl.SetWindowPosition(int(wm.Config.WindowX), int(wm.Config.WindowY))
	}
}

// ToggleBorderless toggles between decorated window (with titlebar/borders) and clean borderless overlay.
func (wm *WindowManager) ToggleBorderless() bool {
	if rl.IsWindowState(rl.FlagWindowUndecorated) {
		rl.ClearWindowState(rl.FlagWindowUndecorated)
		wm.Config.WindowBorderless = false
		return false
	}
	rl.SetWindowState(rl.FlagWindowUndecorated)
	wm.Config.WindowBorderless = true
	return true
}

// ToggleAlwaysOnTop toggles whether the window stays on top of other windows.
func (wm *WindowManager) ToggleAlwaysOnTop() bool {
	if rl.IsWindowState(rl.FlagWindowTopmost) {
		rl.ClearWindowState(rl.FlagWindowTopmost)
		wm.Config.WindowAlwaysOnTop = false
		return false
	}
	rl.SetWindowState(rl.FlagWindowTopmost)
	wm.Config.WindowAlwaysOnTop = true
	return true
}

// ToggleClickThrough toggles mouse passthrough (allowing clicks to pass through transparent window).
func (wm *WindowManager) ToggleClickThrough() bool {
	if wm.isClickThrough {
		rl.ClearWindowState(rl.FlagWindowMousePassthrough)
		wm.isClickThrough = false
		return false
	}
	rl.SetWindowState(rl.FlagWindowMousePassthrough)
	wm.isClickThrough = true
	return true
}

// IsBorderless returns true if window is currently undecorated.
func (wm *WindowManager) IsBorderless() bool {
	return rl.IsWindowState(rl.FlagWindowUndecorated)
}

// IsAlwaysOnTop returns true if window is currently topmost.
func (wm *WindowManager) IsAlwaysOnTop() bool {
	return rl.IsWindowState(rl.FlagWindowTopmost)
}

// IsClickThrough returns true if mouse passthrough is active.
func (wm *WindowManager) IsClickThrough() bool {
	return wm.isClickThrough
}

// MoveWindowBy moves the OS window position by delta (dx, dy).
func (wm *WindowManager) MoveWindowBy(dx, dy int) {
	pos := rl.GetWindowPosition()
	rl.SetWindowPosition(int(pos.X)+dx, int(pos.Y)+dy)
}

// CloseWindow closes the Raylib window.
func (wm *WindowManager) CloseWindow() {
	rl.CloseWindow()
}
