package tray

import (
	"runtime"
	"sync"
	"sync/atomic"

	"fyne.io/systray"
	"pngtuber-lite/assets"
)

// TrayManager manages the system tray icon and menu in a thread-safe manner.
type TrayManager struct {
	mu             sync.Mutex
	startFunc      func()
	endFunc        func()
	pendingRestore atomic.Bool
	pendingQuit    atomic.Bool
	running        bool
}

var globalTray = &TrayManager{}

// GetTrayManager returns the singleton tray manager.
func GetTrayManager() *TrayManager {
	return globalTray
}

// RequestRestore signals the main thread to restore and focus the window.
func (tm *TrayManager) RequestRestore() {
	tm.pendingRestore.Store(true)
}

// CheckAndClearRestore checks if a window restore was requested from the tray.
func (tm *TrayManager) CheckAndClearRestore() bool {
	return tm.pendingRestore.Swap(false)
}

// RequestQuit signals the main thread to exit the application.
func (tm *TrayManager) RequestQuit() {
	tm.pendingQuit.Store(true)
}

// CheckAndClearQuit checks if quit was requested from the tray menu.
func (tm *TrayManager) CheckAndClearQuit() bool {
	return tm.pendingQuit.Swap(false)
}

// Setup initializes the tray with callbacks.
func (tm *TrayManager) Setup() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	start, end := systray.RunWithExternalLoop(tm.onReady, tm.onExit)
	tm.startFunc = start
	tm.endFunc = end
}

// Start launches the background tray loop.
func (tm *TrayManager) Start() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.startFunc != nil && !tm.running {
		tm.running = true
		tm.startFunc()
	}
}

// Stop shuts down the tray icon and menu.
func (tm *TrayManager) Stop() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.running {
		tm.running = false
		if tm.endFunc != nil {
			tm.endFunc()
		}
	}
}

func (tm *TrayManager) onReady() {
	if runtime.GOOS == "windows" {
		if len(assets.AppIconICO) > 0 {
			systray.SetIcon(assets.AppIconICO)
		}
	} else {
		if len(assets.AppTrayPNG) > 0 {
			systray.SetIcon(assets.AppTrayPNG)
		} else if len(assets.AppLogoPNG) > 0 {
			systray.SetIcon(assets.AppLogoPNG)
		}
	}

	systray.SetTitle("PNGTuber Lite")
	systray.SetTooltip("PNGTuber Lite — Avatar 2D")

	systray.SetOnTapped(func() {
		tm.RequestRestore()
	})

	mOpen := systray.AddMenuItem("Abrir PNGTuber Lite", "Restaura e exibe a janela do aplicativo")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Sair", "Encerra o aplicativo")

	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				tm.RequestRestore()
			case <-mQuit.ClickedCh:
				tm.RequestQuit()
				return
			}
		}
	}()
}

func (tm *TrayManager) onExit() {
}
