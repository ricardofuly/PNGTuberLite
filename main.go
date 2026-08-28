package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"pngtuber-lite/assets"
	"pngtuber-lite/pkg/anim"
	"pngtuber-lite/pkg/audio"
	"pngtuber-lite/pkg/config"
	"pngtuber-lite/pkg/costume"
	"pngtuber-lite/pkg/editor"
	"pngtuber-lite/pkg/model"
	"pngtuber-lite/pkg/profiler"
	"pngtuber-lite/pkg/render"
	"pngtuber-lite/pkg/tray"
	"pngtuber-lite/pkg/ui"
	"pngtuber-lite/pkg/updater"
	"pngtuber-lite/pkg/window"
)

func main() {
	avatarFlag := flag.String("avatar", "", "Path to .save avatar file")
	configFlag := flag.String("config", "config.json", "Path to config file")
	thresholdFlag := flag.Float64("threshold", 0, "Microphone volume sensitivity threshold (0.0 to 1.0)")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	checkUpdateFlag := flag.Bool("check-update", false, "Check for new releases on GitHub and exit")
	updateFlag := flag.Bool("update", false, "Download and apply latest update/hotfix immediately")
	flag.Parse()

	// Cleanup lingering .old backup executables from previous updates
	updater.CleanupOldExecutables()

	// Register Linux Desktop Entry and App Icon for Ubuntu Dock
	window.EnsureDesktopEntry()

	if *versionFlag {
		fmt.Printf("PNGTuber Lite %s\n", updater.CurrentVersion)
		os.Exit(0)
	}

	if *checkUpdateFlag {
		fmt.Printf("Verificando atualizações para o PNGTuber Lite (versão atual: %s)...\n", updater.CurrentVersion)
		rel, hasUpdate, err := updater.CheckForUpdate()
		if err != nil {
			log.Fatalf("Erro ao verificar atualizações: %v\n", err)
		}
		if hasUpdate {
			fmt.Printf("✓ Nova versão disponível: %s\nURL: %s\n", rel.TagName, rel.HTMLURL)
		} else {
			fmt.Printf("✓ O aplicativo está na versão mais recente (%s).\n", updater.CurrentVersion)
		}
		os.Exit(0)
	}

	if *updateFlag {
		fmt.Printf("Buscando última versão no GitHub...\n")
		rel, hasUpdate, err := updater.CheckForUpdate()
		if err != nil {
			log.Fatalf("Erro: %v\n", err)
		}
		if !hasUpdate {
			fmt.Printf("O aplicativo já está atualizado na versão %s.\n", updater.CurrentVersion)
			os.Exit(0)
		}
		fmt.Printf("Baixando e aplicando versão %s...\n", rel.TagName)
		err = updater.ApplyUpdate(rel, func(pct float32) {
			fmt.Printf("\rProgresso: %3.0f%%", pct*100)
		})
		fmt.Println()
		if err != nil {
			log.Fatalf("Falha na atualização: %v\n", err)
		}
		fmt.Printf("✓ Atualizado com sucesso para %s! Reiniciando aplicativo...\n", rel.TagName)
		time.Sleep(600 * time.Millisecond)
		_ = updater.RestartApp()
		os.Exit(0)
	}

	// Start asynchronous background update check
	updater.CheckForUpdateAsync()

	// 1. Load User Configuration
	cfg, err := config.LoadConfig(*configFlag)
	if err != nil {
		log.Printf("Warning: Could not load config %q, using defaults: %v", *configFlag, err)
		cfg = config.DefaultConfig()
	}

	if *avatarFlag != "" {
		cfg.AvatarPath = *avatarFlag
	}
	if *thresholdFlag > 0 {
		cfg.AudioThreshold = float32(*thresholdFlag)
	}

	// 2. Load and Parse Avatar .save file with robust fallbacks
	if _, err := os.Stat(cfg.AvatarPath); os.IsNotExist(err) {
		fallbacks := []string{
			"assets/samples/defaultAvatar.save",
			"defaultAvatar.save",
			"../assets/samples/defaultAvatar.save",
		}
		for _, fb := range fallbacks {
			if _, err2 := os.Stat(fb); err2 == nil {
				cfg.AvatarPath = fb
				break
			}
		}
	}

	avatar, err := model.ParseSaveFile(cfg.AvatarPath)
	if err != nil {
		log.Fatalf("Error parsing avatar file %q: %v", cfg.AvatarPath, err)
	}
	log.Printf("Loaded avatar %q with %d layers (%d root layers)", cfg.AvatarPath, len(avatar.Layers), len(avatar.RootLayers))

	// 3. Initialize Window
	wm := window.NewWindowManager(cfg)
	wm.InitWindow()
	defer wm.CloseWindow()

	// Set window icon in Raylib / GLFW for Taskbar / Dock
	if len(assets.AppLogoPNG) > 0 {
		iconImg := rl.LoadImageFromMemory(".png", assets.AppLogoPNG, int32(len(assets.AppLogoPNG)))
		if iconImg.Width > 0 && iconImg.Height > 0 {
			rl.SetWindowIcon(*iconImg)
		}
		rl.UnloadImage(iconImg)
	}

	// Disable default exit key (ESC) to allow custom overlay handling
	rl.SetExitKey(0)

	// Load Icon Atlas and Official Logo
	if err := ui.GlobalIcons.Load(); err != nil {
		log.Printf("Warning: Could not load icons: %v", err)
	}
	defer ui.GlobalIcons.Unload()

	// 4. Initialize GPU Texture Cache and Renderer
	texCache := render.NewTextureCache()
	if err := texCache.LoadAvatarTextures(avatar); err != nil {
		log.Printf("Warning during texture loading: %v", err)
	}
	defer texCache.UnloadAll()

	renderer := render.NewRenderer(texCache)

	// 5. Initialize Audio Engine (Microphone & VAD)
	audioEngine := audio.NewCaptureEngine(cfg.AudioThreshold)
	if err := audioEngine.Start(cfg.AudioDevice); err != nil {
		log.Printf("Warning: Could not start audio capture: %v (talk reactivity disabled)", err)
	}
	defer audioEngine.Stop()

	// 6. Initialize Animation and Costume Subsystems
	animator := anim.NewAnimator()
	animator.Bounce.BounceStrength = cfg.BounceStrength
	animator.Bounce.Gravity = cfg.BounceGravity
	animator.Blink.BlinkSpeed = cfg.BlinkSpeed
	animator.Blink.BlinkChance = cfg.BlinkChance

	costumeMgr := costume.NewCostumeManager(cfg.BounceOnCostume)

	// 7. Initialize Interactive UI Control Drawer & Visual Editor
	uiState := ui.NewUIState()
	uiState.InitFont()
	defer uiState.Unload()

	// Setup System Tray
	trayMgr := tray.GetTrayManager()
	trayMgr.Setup()
	trayMgr.Start()
	defer trayMgr.Stop()

	isWindowHidden := false
	shouldExit := false

	uiState.OnRequestMinimizeToTray = func() {
		isWindowHidden = true
		rl.SetWindowState(rl.FlagWindowHidden)
	}
	uiState.OnRequestClose = func() {
		shouldExit = true
	}

	editorState := editor.NewEditorState(texCache, uiState)
	editorState.SetAvatar(avatar, cfg.AvatarPath)

	editorState.OnAvatarModified = func() {
		avatar = editorState.Avatar
		cfg.AvatarPath = editorState.AvatarFilePath
	}

	uiState.OnOpenEditor = func() {
		editorState.IsOpen = !editorState.IsOpen
		if editorState.IsOpen {
			uiState.IsOpen = false
			editorState.SetAvatar(avatar, cfg.AvatarPath)
		}
	}

	uiState.OnAvatarSelected = func(filePath string) {
		newAvatar, err := model.ParseSaveFile(filePath)
		if err != nil {
			log.Printf("Error loading selected avatar %q: %v", filePath, err)
			return
		}
		if err := texCache.LoadAvatarTextures(newAvatar); err != nil {
			log.Printf("Warning loading textures for avatar %q: %v", filePath, err)
		}
		avatar = newAvatar
		cfg.AvatarPath = filePath
		editorState.SetAvatar(avatar, filePath)
		log.Printf("Switched to avatar %q with %d layers", filePath, len(avatar.Layers))
	}

	// State variables for viewport and window manipulation
	if cfg.AvatarRelX <= 0 || cfg.AvatarRelX > 1.0 {
		cfg.AvatarRelX = 0.5
	}
	if cfg.AvatarRelY <= 0 || cfg.AvatarRelY > 1.0 {
		cfg.AvatarRelY = 0.5
	}

	avatarOriginX := float32(rl.GetScreenWidth()) * cfg.AvatarRelX
	avatarOriginY := float32(rl.GetScreenHeight()) * cfg.AvatarRelY
	avatarScale := cfg.Scale
	if avatarScale <= 0 {
		avatarScale = 1.0
	}

	prevWindowW := float32(rl.GetScreenWidth())
	prevWindowH := float32(rl.GetScreenHeight())

	uiState.OnResetAvatar = func() {
		cfg.AvatarRelX = 0.5
		cfg.AvatarRelY = 0.5
		avatarOriginX = float32(rl.GetScreenWidth()) * 0.5
		avatarOriginY = float32(rl.GetScreenHeight()) * 0.5
		avatarScale = 1.0
	}

	showHUD := false
	wasSpeaking := false

	// Avatar dragging inside viewport
	isDraggingAvatar := false
	var dragAvatarStartMouse rl.Vector2
	var dragAvatarStartOrigin rl.Vector2

	const fixedDeltaTime = float32(1.0 / 60.0)
	var timeAccumulator float32 = 0.0

	// System resource profiler
	sysProfiler := profiler.NewSystemProfiler()

	// 8. Main Loop
	for !shouldExit {
		// Process thread-safe Tray Signals on the main thread
		if trayMgr.CheckAndClearQuit() {
			shouldExit = true
			break
		}
		if trayMgr.CheckAndClearRestore() {
			if isWindowHidden {
				isWindowHidden = false
				rl.ClearWindowState(rl.FlagWindowHidden)
				rl.ClearWindowState(rl.FlagWindowMinimized)
				rl.RestoreWindow()
				rl.SetWindowFocused()
			} else {
				rl.SetWindowFocused()
			}
		}

		if isWindowHidden {
			time.Sleep(30 * time.Millisecond)
			continue
		}

		if rl.WindowShouldClose() {
			uiState.ShowCloseModal = true
		}
		dt := rl.GetFrameTime()
		if dt > 0.1 {
			dt = 0.1
		}
		timeAccumulator += dt

		curW := float32(rl.GetScreenWidth())
		curH := float32(rl.GetScreenHeight())

		// Dynamic Re-anchoring on Window Resize / Maximize / Restore
		if (curW != prevWindowW || curH != prevWindowH) && prevWindowW > 0 && prevWindowH > 0 {
			ratioX := avatarOriginX / prevWindowW
			ratioY := avatarOriginY / prevWindowH

			avatarOriginX = ratioX * curW
			avatarOriginY = ratioY * curH

			prevWindowW = curW
			prevWindowH = curH
		}

		// Keep entire avatar sprite safely within viewport boundaries (AABB Extents)
		extLeft, extRight, extTop, extBottom := render.ComputeAvatarExtents(avatar, avatarScale)
		padding := float32(10.0)

		minX := extLeft + padding
		maxX := curW - extRight - padding
		minY := extTop + padding
		maxY := curH - extBottom - padding

		if maxX < minX {
			avatarOriginX = curW * 0.5
		} else {
			if avatarOriginX < minX {
				avatarOriginX = minX
			}
			if avatarOriginX > maxX {
				avatarOriginX = maxX
			}
		}

		if maxY < minY {
			avatarOriginY = curH * 0.5
		} else {
			if avatarOriginY < minY {
				avatarOriginY = minY
			}
			if avatarOriginY > maxY {
				avatarOriginY = maxY
			}
		}

		mousePos := rl.GetMousePosition()
		isRebindingKey := (uiState.IsOpen && uiState.RebindingAction != "")

		// --- Toggle Control Drawer Menu ---
		if !isRebindingKey && (rl.IsKeyPressed(cfg.Keybinds.ToggleMenu) || rl.IsKeyPressed(rl.KeyTab) || rl.IsKeyPressed(rl.KeyC)) {
			if !editorState.IsOpen {
				uiState.IsOpen = !uiState.IsOpen
				if uiState.IsOpen {
					uiState.ScanAvatars()
					if devices, err := audioEngine.ListDevices(); err == nil {
						uiState.AudioDevices = devices
					}
				}
			}
		}

		// --- Toggle Avatar Visual Editor ---
		if !isRebindingKey && (rl.IsKeyPressed(cfg.Keybinds.ToggleEditor) || rl.IsKeyPressed(rl.KeyF2)) {
			editorState.IsOpen = !editorState.IsOpen
			if editorState.IsOpen {
				uiState.IsOpen = false
				editorState.SetAvatar(avatar, cfg.AvatarPath)
			}
		}

		if rl.IsKeyPressed(rl.KeyEscape) {
			if uiState.ShowCloseModal {
				uiState.ShowCloseModal = false
			} else if uiState.RebindingAction != "" {
				uiState.RebindingAction = ""
			} else if editorState.IsOpen {
				editorState.IsOpen = false
			} else if uiState.IsOpen {
				uiState.IsOpen = false
			}
		}

		// Update Editor state
		if editorState.IsOpen {
			editorState.Update(dt)
		} else {
			// Also check file drops outside editor to load .save directly
			if rl.IsFileDropped() {
				droppedFiles := rl.LoadDroppedFiles()
				for _, f := range droppedFiles {
					if strings.HasSuffix(strings.ToLower(f), ".save") {
						uiState.OnAvatarSelected(f)
					}
				}
				rl.UnloadDroppedFiles()
			}
		}

		// --- Window & Overlay Controls ---
		if !isRebindingKey {
			// Toggle Click-Through / Mouse Passthrough
			if rl.IsKeyPressed(cfg.Keybinds.ToggleClickThrough) {
				wm.ToggleClickThrough()
			}

			// Toggle Borderless / Decorated Window
			if rl.IsKeyPressed(cfg.Keybinds.ToggleBorderless) {
				wm.ToggleBorderless()
			}

			// Toggle Always-On-Top
			if rl.IsKeyPressed(cfg.Keybinds.ToggleAlwaysOnTop) {
				wm.ToggleAlwaysOnTop()
			}

			// Reset avatar position & scale
			if rl.IsKeyPressed(cfg.Keybinds.ResetAvatar) && !uiState.IsOpen && !editorState.IsOpen {
				uiState.OnResetAvatar()
			}

			// Costume Switching: Keys 1..9 and 0 (slot 10)
			for key := int32(rl.KeyOne); key <= int32(rl.KeyNine); key++ {
				if rl.IsKeyPressed(key) {
					slot := int(key - rl.KeyOne + 1)
					if costumeMgr.SetCostume(slot) && cfg.BounceOnCostume {
						animator.Bounce.Trigger()
					}
				}
			}
			if rl.IsKeyPressed(rl.KeyZero) {
				if costumeMgr.SetCostume(10) && cfg.BounceOnCostume {
					animator.Bounce.Trigger()
				}
			}

			// Force Blink & Bounce test
			if rl.IsKeyPressed(cfg.Keybinds.TestBounce) && !editorState.IsOpen {
				animator.Blink.ForceBlink()
				animator.Bounce.Trigger()
			}

			// Toggle Debug HUD
			if rl.IsKeyPressed(cfg.Keybinds.ToggleHUD) || rl.IsKeyPressed(rl.KeyF1) {
				showHUD = !showHUD
			}

			// Adjust Sensitivity Threshold (+ / - or PageUp / PageDown)
			if rl.IsKeyPressed(cfg.Keybinds.IncreaseSens) || rl.IsKeyPressed(rl.KeyPageUp) || rl.IsKeyPressed(rl.KeyKpAdd) {
				cfg.AudioThreshold += 0.005
				audioEngine.SetThreshold(cfg.AudioThreshold)
			}
			if rl.IsKeyPressed(cfg.Keybinds.DecreaseSens) || rl.IsKeyPressed(rl.KeyPageDown) || rl.IsKeyPressed(rl.KeyKpSubtract) {
				if cfg.AudioThreshold > 0.005 {
					cfg.AudioThreshold -= 0.005
				}
				audioEngine.SetThreshold(cfg.AudioThreshold)
			}
		}

		// --- Mouse Navigation (Compatível com Wayland e X11) ---

		// Don't drag avatar if clicking on UI menu button or open drawer panel/modals
		isOverUI := false
		if editorState.IsOpen || uiState.ShowCloseModal || (updater.GetUpdateState().ShowPopup && updater.GetUpdateState().Latest != nil) {
			isOverUI = true
		}
		menuBtnRec := rl.NewRectangle(12, 12, 450, 34)
		if rl.CheckCollisionPointRec(mousePos, menuBtnRec) {
			isOverUI = true
		}
		if uiState.IsOpen {
			panelRec := rl.NewRectangle(12, 52, 410, float32(rl.GetScreenHeight()-60))
			if rl.CheckCollisionPointRec(mousePos, panelRec) {
				isOverUI = true
			}
		}

		// 1. Arrastar Avatar na Tela (Botão Esquerdo ou Direito fora da UI)
		if !isOverUI && (rl.IsMouseButtonPressed(rl.MouseLeftButton) || rl.IsMouseButtonPressed(rl.MouseRightButton)) {
			isDraggingAvatar = true
			dragAvatarStartMouse = rl.GetMousePosition()
			dragAvatarStartOrigin = rl.Vector2{X: avatarOriginX, Y: avatarOriginY}
		}
		if isDraggingAvatar {
			if rl.IsMouseButtonDown(rl.MouseLeftButton) || rl.IsMouseButtonDown(rl.MouseRightButton) {
				currMouse := rl.GetMousePosition()
				newX := dragAvatarStartOrigin.X + (currMouse.X - dragAvatarStartMouse.X)
				newY := dragAvatarStartOrigin.Y + (currMouse.Y - dragAvatarStartMouse.Y)

				if maxX >= minX {
					if newX < minX {
						newX = minX
					}
					if newX > maxX {
						newX = maxX
					}
				} else {
					newX = curW * 0.5
				}

				if maxY >= minY {
					if newY < minY {
						newY = minY
					}
					if newY > maxY {
						newY = maxY
					}
				} else {
					newY = curH * 0.5
				}

				avatarOriginX = newX
				avatarOriginY = newY
			} else {
				isDraggingAvatar = false
			}
		}

		// 2. Zoom com a Roda do Mouse (Scroll) quando fora da UI
		if !isOverUI {
			wheel := rl.GetMouseWheelMove()
			if wheel != 0 {
				avatarScale += wheel * 0.05
				if avatarScale < 0.1 {
					avatarScale = 0.1
				} else if avatarScale > 5.0 {
					avatarScale = 5.0
				}
			}
		}

		// --- Audio Reaction ---
		isSpeaking := audioEngine.IsTalking()
		if isSpeaking && !wasSpeaking {
			animator.Bounce.Trigger()
		}
		wasSpeaking = isSpeaking

		// Sync animation settings from config
		animator.Bounce.BounceStrength = cfg.BounceStrength
		animator.Bounce.Gravity = cfg.BounceGravity
		animator.Blink.BlinkSpeed = cfg.BlinkSpeed
		animator.Wobble.BobbingIntensity = cfg.BobbingIntensity
		animator.Wobble.WobbleIntensity = cfg.WobbleIntensity

		// --- Fixed Timestep Animation Update ---
		for timeAccumulator >= fixedDeltaTime {
			animator.Update(avatar, fixedDeltaTime)
			timeAccumulator -= fixedDeltaTime
		}

		// --- Build Frame Render State ---
		offsets, rotations, frames, stretches := animator.BuildLayerAnimationMaps(avatar)

		renderState := render.RenderState{
			Avatar:         avatar,
			Origin:         rl.Vector2{X: avatarOriginX, Y: avatarOriginY},
			Scale:          avatarScale,
			GlobalBounceY:  animator.Bounce.Y,
			Costume:        costumeMgr.GetCostume(),
			IsBlinking:     animator.Blink.IsBlinking,
			IsTalking:      isSpeaking,
			LayerOffsets:   offsets,
			LayerRotations: rotations,
			LayerFrames:    frames,
			LayerStretches: stretches,
		}

		// --- Draw Frame ---
		rl.BeginDrawing()

		// Background clear (Transparente ou Chroma Key)
		bg := cfg.BackgroundColor
		rl.ClearBackground(rl.NewColor(bg[0], bg[1], bg[2], bg[3]))

		// Update Profiler metrics
		profStats := sysProfiler.Update(dt, rl.GetFPS(), texCache.GetTextureCount(), texCache.GetEstimatedVRAM())

		// 1. Render Avatar Layers
		renderer.Draw(&renderState)

		// 2. Optional Debug HUD Overlay (when menu is closed)
		if showHUD && !uiState.IsOpen && !editorState.IsOpen {
			drawDebugHUD(uiState, wm, costumeMgr.GetCostume(), isSpeaking, audioEngine.GetVolume(), cfg.AudioThreshold, avatarScale, profStats, avatarOriginX, avatarOriginY, cfg.AudioDevice)
		}

		// 3. Render Interactive UI Menu Drawer (on top)
		if !editorState.IsOpen {
			uiState.Draw(cfg, wm, costumeMgr, audioEngine, &avatarScale)
		} else {
			// 4. Render Visual Avatar Editor
			editorState.Draw(avatarScale, rl.Vector2{X: avatarOriginX, Y: avatarOriginY})
		}

		rl.EndDrawing()
	}

	// 9. Save persistent geometry and configuration on exit
	cfg.Scale = avatarScale
	finalW := float32(rl.GetScreenWidth())
	finalH := float32(rl.GetScreenHeight())
	if finalW > 0 {
		cfg.AvatarRelX = avatarOriginX / finalW
	}
	if finalH > 0 {
		cfg.AvatarRelY = avatarOriginY / finalH
	}
	pos := rl.GetWindowPosition()
	cfg.WindowX = int32(pos.X)
	cfg.WindowY = int32(pos.Y)
	cfg.WindowWidth = int32(finalW)
	cfg.WindowHeight = int32(finalH)
	_ = config.SaveConfig(*configFlag, cfg)
}

func drawDebugHUD(
	uiState *ui.UIState,
	wm *window.WindowManager,
	costume int,
	isSpeaking bool,
	volume float32,
	threshold float32,
	scale float32,
	stats profiler.ProfilerStats,
	avatarX, avatarY float32,
	micDevice string,
) {
	panelW := int32(370)
	panelH := int32(335)
	rl.DrawRectangleRounded(rl.NewRectangle(12, 52, float32(panelW), float32(panelH)), 0.05, 4, rl.NewColor(12, 16, 24, 230))
	rl.DrawRectangleRoundedLines(rl.NewRectangle(12, 52, float32(panelW), float32(panelH)), 0.05, 4, rl.NewColor(55, 80, 125, 255))

	y := int32(60)

	// 1. Title Header
	ui.GlobalIcons.DrawIcon(ui.IconPhysics, 24, float32(y), 18, rl.SkyBlue)
	uiState.DrawText("PNGTuber Lite - Profiler & HUD", 48, y, 15, rl.RayWhite)
	y += 24

	// 2. CPU Usage
	cpuColor := rl.Lime
	if stats.CPUPercent > 40.0 {
		cpuColor = rl.Orange
	} else if stats.CPUPercent > 70.0 {
		cpuColor = rl.Red
	}
	uiState.DrawText(fmt.Sprintf("CPU Processo: %.1f%% (%d Goroutines)", stats.CPUPercent, stats.NumGoroutine), 24, y, 13, cpuColor)
	y += 20

	// 3. RAM (Physical RSS & Go Heap)
	uiState.DrawText(fmt.Sprintf("RAM Física (RSS): %.1f MB", stats.RamRSSMB), 24, y, 13, rl.SkyBlue)
	uiState.DrawText(fmt.Sprintf("Go Heap: %.1f MB (Sys: %.1f MB | GC: %d)", stats.RamAllocMB, stats.RamSysMB, stats.NumGC), 24, y+16, 12, rl.LightGray)
	y += 36

	// 4. GPU & Render Time
	uiState.DrawText(fmt.Sprintf("GPU VRAM: %.1f MB (%d texturas carregadas)", stats.VRAMMB, stats.TextureCount), 24, y, 13, rl.Lime)
	uiState.DrawText(fmt.Sprintf("Frame: %.1f ms | FPS: %d (Meta: 60)", stats.FrameTimeMS, stats.FPS), 24, y+16, 12, rl.Yellow)
	y += 36

	// 5. Avatar Status
	uiState.DrawText(fmt.Sprintf("Figurino: %d/10 | Zoom: %.2fx | Pos: (%.0f, %.0f)", costume, scale, avatarX, avatarY), 24, y, 12, rl.LightGray)
	y += 20

	// 6. Window Overlay Status
	borderlessStr := "OFF [F10]"
	if wm.IsBorderless() {
		borderlessStr = "ON [F10]"
	}
	topmostStr := "OFF [F11]"
	if wm.IsAlwaysOnTop() {
		topmostStr = "ON [F11]"
	}
	passthroughStr := "OFF [F9]"
	if wm.IsClickThrough() {
		passthroughStr = "ON [F9]"
	}
	uiState.DrawText(fmt.Sprintf("Borderless: %s | Top: %s | Click-Thru: %s", borderlessStr, topmostStr, passthroughStr), 24, y, 12, rl.SkyBlue)
	y += 24

	// 7. Audio & VAD
	micName := micDevice
	if micName == "" {
		micName = "Padrão do Sistema"
	}
	if len(micName) > 28 {
		micName = micName[:28] + "..."
	}
	uiState.DrawText(fmt.Sprintf("Mic: %s", micName), 24, y, 12, rl.Yellow)
	uiState.DrawText(fmt.Sprintf("RMS: %.3f | Limiar: %.3f (+/-)", volume, threshold), 24, y+16, 12, rl.LightGray)
	y += 34

	// Audio Level Bar
	barWidth := float32(340.0)
	rl.DrawRectangle(24, y, int32(barWidth), 10, rl.DarkGray)

	volFill := (volume / 0.2) * barWidth
	if volFill > barWidth {
		volFill = barWidth
	}
	volColor := rl.Lime
	if isSpeaking {
		volColor = rl.Green
	}
	rl.DrawRectangle(24, y, int32(volFill), 10, volColor)

	// Threshold line
	threshX := 24 + int32((threshold/0.2)*barWidth)
	if threshX > 24+int32(barWidth) {
		threshX = 24 + int32(barWidth)
	}
	rl.DrawLine(threshX, y-3, threshX, y+13, rl.Red)
	y += 18

	statusText := "Status: SILÊNCIO (Boca Fechada)"
	statusColor := rl.LightGray
	if isSpeaking {
		statusText = "Status: FALANDO (Boca Aberta)"
		statusColor = rl.Lime
	}
	uiState.DrawText(statusText, 24, y, 13, statusColor)
}
