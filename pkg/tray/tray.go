package tray

import (
	"runtime"
	"sync"

	"fyne.io/systray"
	"pngtuber-lite/assets"
)

// TrayManager manages the system tray icon and menu.
type TrayManager struct {
	mu        sync.Mutex
	startFunc func()
	endFunc   func()
	onOpen    func()
	onQuit    func()
	running   bool
}

var globalTray = &TrayManager{}

// GetTrayManager returns the singleton tray manager.
func GetTrayManager() *TrayManager {
	return globalTray
}

// Setup initializes the tray with callbacks for opening the window and quitting.
func (tm *TrayManager) Setup(onOpen func(), onQuit func()) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.onOpen = onOpen
	tm.onQuit = onQuit

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
	// Set icon according to OS
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

	// Set direct click action on tray icon
	systray.SetOnTapped(func() {
		if tm.onOpen != nil {
			tm.onOpen()
		}
	})

	mOpen := systray.AddMenuItem("Abrir PNGTuber Lite", "Restaura e exibe a janela do aplicativo")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Sair", "Encerra o aplicativo")

	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				if tm.onOpen != nil {
					tm.onOpen()
				}
			case <-mQuit.ClickedCh:
				if tm.onQuit != nil {
					tm.onQuit()
				}
				return
			}
		}
	}()
}

func (tm *TrayManager) onExit() {
	// Cleanup if needed
}
