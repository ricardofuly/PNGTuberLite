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
	"pngtuber-lite/pkg/i18n"
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

	// Initialize user interface language
	if cfg.Language != "" {
		i18n.SetLanguage(cfg.Language)
	} else {
		i18n.SetLanguage(i18n.DetectSystemLanguage())
	}

	// 2. Load and Parse Avatar .save file with robust fallbacks and embedded assets
	ensureDefaultAvatars()

	if _, err := os.Stat(cfg.AvatarPath); os.IsNotExist(err) {
		fallbacks := []string{
			"assets/samples/defaultAvatar.save",
			"assets/samples/slugcat.save",
			"defaultAvatar.save",
			"slugcat.save",
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
		log.Printf("Warning: Could not parse avatar from disk %q: %v. Attempting embedded fallback...", cfg.AvatarPath, err)
		if strings.Contains(strings.ToLower(cfg.AvatarPath), "slugcat") && len(assets.SlugcatAvatarSave) > 0 {
			avatar, err = model.ParseSaveData(assets.SlugcatAvatarSave)
		} else if len(assets.DefaultAvatarSave) > 0 {
			avatar, err = model.ParseSaveData(assets.DefaultAvatarSave)
		}
		if err != nil {
			log.Fatalf("Fatal: Error parsing avatar file %q: %v", cfg.AvatarPath, err)
		}
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

	uiState.OnCreateNewAvatar = func() {
		editorState.NewBlankAvatar()
		editorState.IsOpen = true
		uiState.IsOpen = false
		avatar = editorState.Avatar
		cfg.AvatarPath = editorState.AvatarFilePath
	}

	uiState.OnImportFolderAvatar = func(dirPath string) {
		if err := editorState.CreateAvatarFromDirectory(dirPath); err == nil {
			avatar = editorState.Avatar
			cfg.AvatarPath = editorState.AvatarFilePath
			editorState.IsOpen = true
			uiState.IsOpen = false
			uiState.ScanAvatars()
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

			// Toggle Debug HUD (F3, H, or Configured Key)
			if rl.IsKeyPressed(cfg.Keybinds.ToggleHUD) || rl.IsKeyPressed(rl.KeyF3) || rl.IsKeyPressed(rl.KeyH) {
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
			FlipHorizontal: cfg.FlipHorizontal,
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

// drawHUDProgressBar renders a rounded progress bar with background track and dynamic fill.
func drawHUDProgressBar(rec rl.Rectangle, val float32, maxVal float32, fillColor, trackColor rl.Color) {
	rl.DrawRectangleRounded(rec, 0.5, 4, trackColor)
	if maxVal <= 0 {
		return
	}
	fillW := (val / maxVal) * rec.Width
	if fillW > rec.Width {
		fillW = rec.Width
	}
	if fillW > 2 {
		rl.DrawRectangleRounded(rl.NewRectangle(rec.X, rec.Y, fillW, rec.Height), 0.5, 4, fillColor)
	}
}

// drawHUDSparkline renders a real-time smoothly scrolling sparkline graph with area fill and reference guide line.
func drawHUDSparkline(rec rl.Rectangle, history []float32, minVal, maxVal float32, lineColor, fillColor rl.Color, targetLine float32, scrollProgress float32) {
	rl.DrawRectangleRounded(rec, 0.2, 4, rl.NewColor(14, 20, 32, 220))
	rl.DrawRectangleRoundedLines(rec, 0.2, 4, rl.NewColor(36, 52, 82, 255))

	if len(history) < 2 || maxVal <= minVal {
		return
	}

	// Target reference line (e.g. 16.6ms for 60fps)
	if targetLine > minVal && targetLine < maxVal {
		normTarget := (targetLine - minVal) / (maxVal - minVal)
		targetY := rec.Y + rec.Height - 3 - (normTarget * (rec.Height - 6))
		rl.DrawLineEx(rl.NewVector2(rec.X+3, targetY), rl.NewVector2(rec.X+rec.Width-3, targetY), 1.0, rl.NewColor(70, 105, 155, 140))
	}

	stepX := (rec.Width - 6) / float32(profiler.MaxHistoryPoints-1)
	subpixelOffset := (1.0 - scrollProgress) * stepX
	if subpixelOffset < 0 {
		subpixelOffset = 0
	}
	startX := rec.X + 3 + float32(profiler.MaxHistoryPoints-len(history))*stepX - subpixelOffset

	pts := make([]rl.Vector2, 0, len(history))
	for i, v := range history {
		norm := (v - minVal) / (maxVal - minVal)
		if norm < 0 {
			norm = 0
		}
		if norm > 1 {
			norm = 1
		}
		py := rec.Y + rec.Height - 3 - (norm * (rec.Height - 6))
		px := startX + float32(i)*stepX
		if px >= rec.X+2 && px <= rec.X+rec.Width+stepX {
			if px < rec.X+3 {
				px = rec.X + 3
			}
			if px > rec.X+rec.Width-3 {
				px = rec.X + rec.Width - 3
			}
			pts = append(pts, rl.NewVector2(px, py))
		}
	}

	if len(pts) < 2 {
		return
	}

	// Draw filled polygon underneath line
	bottomY := rec.Y + rec.Height - 3
	for i := 0; i < len(pts)-1; i++ {
		p1 := pts[i]
		p2 := pts[i+1]
		b1 := rl.NewVector2(p1.X, bottomY)
		b2 := rl.NewVector2(p2.X, bottomY)

		rl.DrawTriangle(p1, b1, b2, fillColor)
		rl.DrawTriangle(p1, b2, p2, fillColor)
		rl.DrawLineEx(p1, p2, 1.8, lineColor)
	}

	// Glowing indicator dot at the latest value
	lastPt := pts[len(pts)-1]
	glowCol := rl.NewColor(lineColor.R, lineColor.G, lineColor.B, 70)
	rl.DrawCircleV(lastPt, 4.0, glowCol)
	rl.DrawCircleV(lastPt, 2.5, lineColor)
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
	panelW := float32(420)
	panelH := float32(495)
	panelRec := rl.NewRectangle(12, 52, panelW, panelH)

	rl.DrawRectangleRounded(panelRec, 0.04, 6, rl.NewColor(12, 18, 30, 245))
	rl.DrawRectangleRoundedLines(panelRec, 0.04, 6, ui.ColPanelBorder)

	y := float32(62)
	contentW := panelW - 28
	contentX := float32(26)

	// 1. Title Header (No overlapping!)
	ui.GlobalIcons.DrawIcon(ui.IconPhysics, contentX, y, 18, ui.ColSkyBlue)
	uiState.DrawTextBold(i18n.T("telemetry_title"), int32(contentX)+24, int32(y)+1, 14, ui.ColTextTitle)
	uiState.DrawBadge(panelRec.X+panelW-85, y-1, "F3 [HUD]", ui.ColCardBg, ui.ColSkyBlue)
	y += 26

	// 2. CPU Usage & History Graph
	cpuColor := ui.ColLime
	if stats.CPUPercent > 40.0 {
		cpuColor = ui.ColYellow
	}
	if stats.CPUPercent > 70.0 {
		cpuColor = ui.ColRed
	}

	uiState.DrawTextBold(fmt.Sprintf("%s: %.1f%% (%s: %.1f%%)", i18n.T("label_hud_cpu"), stats.CPUPercent, i18n.T("label_total_system"), stats.CPUTotalPercent), int32(contentX), int32(y), 12.5, cpuColor)
	uiState.DrawText(fmt.Sprintf("%d Goroutines", stats.NumGoroutine), int32(contentX+contentW)-90, int32(y)+1, 11, ui.ColTextMuted)
	y += 18

	// CPU Progress Bar (Smooth)
	drawHUDProgressBar(rl.NewRectangle(contentX, y, contentW, 6), stats.SmoothCPU, 100.0, cpuColor, ui.ColScrollTrack)
	y += 10

	// CPU Sparkline Graph (Continuous sub-pixel scroll)
	cpuFill := rl.NewColor(cpuColor.R, cpuColor.G, cpuColor.B, 40)
	drawHUDSparkline(rl.NewRectangle(contentX, y, contentW, 30), stats.CPUHistory, 0, 50.0, cpuColor, cpuFill, 0, stats.SampleProgress)
	y += 36

	// 3. RAM (Physical RSS & Go Heap)
	uiState.DrawTextBold(fmt.Sprintf("%s: %.1f MB", i18n.T("label_hud_ram_rss"), stats.RamRSSMB), int32(contentX), int32(y), 12.5, ui.ColSkyBlue)
	uiState.DrawText(fmt.Sprintf("Heap: %.1f MB (Sys: %.1f MB | GC: %d)", stats.RamAllocMB, stats.RamSysMB, stats.NumGC), int32(contentX), int32(y)+16, 11, ui.ColTextMuted)
	y += 32

	// RAM Progress Bar (0 to 512 MB scale, Smooth)
	drawHUDProgressBar(rl.NewRectangle(contentX, y, contentW, 6), stats.SmoothRAM, 512.0, ui.ColSkyBlue, ui.ColScrollTrack)
	y += 14

	// 4. GPU & Frametime Graph
	frameColor := ui.ColLime
	if stats.FrameTimeMS > 18.0 {
		frameColor = ui.ColYellow
	}
	if stats.FrameTimeMS > 33.0 {
		frameColor = ui.ColRed
	}

	uiState.DrawTextBold(fmt.Sprintf("%s: %.1f ms | %s: %d", i18n.T("label_hud_frametime"), stats.FrameTimeMS, i18n.T("label_hud_fps"), stats.FPS), int32(contentX), int32(y), 12.5, frameColor)
	uiState.DrawText(fmt.Sprintf("%s: %.1f MB (%d tex)", i18n.T("label_hud_vram"), stats.VRAMMB, stats.TextureCount), int32(contentX+contentW)-135, int32(y)+1, 11, ui.ColTextBody)
	y += 18

	// Frametime Sparkline Graph (target line at 16.6ms for 60fps)
	frameFill := rl.NewColor(frameColor.R, frameColor.G, frameColor.B, 40)
	drawHUDSparkline(rl.NewRectangle(contentX, y, contentW, 30), stats.FrametimeHistory, 0, 33.3, frameColor, frameFill, 16.6, 1.0)
	y += 36

	// 5. Audio & VAD
	micName := micDevice
	if micName == "" {
		micName = i18n.T("label_system_default")
	}
	if len(micName) > 24 {
		micName = micName[:24] + "..."
	}

	uiState.DrawTextBold(fmt.Sprintf("Mic: %s", micName), int32(contentX), int32(y), 12.5, ui.ColYellow)

	vadText := i18n.T("label_status_silent")
	vadCol := ui.ColTextMuted
	vadBg := ui.ColCardBg
	if isSpeaking {
		vadText = i18n.T("label_status_speaking")
		vadCol = ui.ColLime
		vadBg = rl.NewColor(24, 85, 48, 255)
	}
	uiState.DrawBadge(contentX+contentW-175, y-1, vadText, vadBg, vadCol)

	uiState.DrawText(fmt.Sprintf("RMS: %.4f | %s: %.4f (+/-)", volume, i18n.T("label_vad_threshold"), threshold), int32(contentX), int32(y)+16, 11, ui.ColTextMuted)
	y += 32

	// Audio Level Bar with Threshold needle
	barRec := rl.NewRectangle(contentX, y, contentW, 8)
	volFill := ui.ColSkyBlue
	if isSpeaking {
		volFill = ui.ColLime
	}
	drawHUDProgressBar(barRec, volume, 0.15, volFill, ui.ColScrollTrack)

	// Threshold red line marker
	threshX := barRec.X + (threshold/0.15)*barRec.Width
	if threshX > barRec.X+barRec.Width {
		threshX = barRec.X + barRec.Width
	}
	rl.DrawLineEx(rl.NewVector2(threshX, barRec.Y-3), rl.NewVector2(threshX, barRec.Y+barRec.Height+3), 2.5, ui.ColRed)
	y += 18

	// 6. Avatar & Window Status
	uiState.DrawText(fmt.Sprintf("%s: %d/10 | Zoom: %.2fx | Pos: (%.0f, %.0f)", i18n.T("label_costume_slot"), costume, scale, avatarX, avatarY), int32(contentX), int32(y), 11.5, ui.ColTextBody)
	y += 18

	// Hotkey status badges
	borderlessBadge := "F10 Borderless"
	borderlessCol := ui.ColTextMuted
	if wm.IsBorderless() {
		borderlessCol = ui.ColLime
	}
	topmostBadge := "F11 Topmost"
	topmostCol := ui.ColTextMuted
	if wm.IsAlwaysOnTop() {
		topmostCol = ui.ColSkyBlue
	}
	clickThruBadge := "F9 Click-Thru"
	clickThruCol := ui.ColTextMuted
	if wm.IsClickThrough() {
		clickThruCol = ui.ColLavender
	}

	badgeY := y
	uiState.DrawBadge(contentX, badgeY, borderlessBadge, ui.ColCardBg, borderlessCol)
	uiState.DrawBadge(contentX+125, badgeY, topmostBadge, ui.ColCardBg, topmostCol)
	uiState.DrawBadge(contentX+238, badgeY, clickThruBadge, ui.ColCardBg, clickThruCol)
}

// ensureDefaultAvatars guarantees defaultAvatar.save and slugcat.save exist in assets/samples.
func ensureDefaultAvatars() {
	_ = os.MkdirAll("assets/samples", 0755)

	if len(assets.DefaultAvatarSave) > 0 {
		defaultPath := "assets/samples/defaultAvatar.save"
		if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
			_ = os.WriteFile(defaultPath, assets.DefaultAvatarSave, 0644)
		}
	}

	if len(assets.SlugcatAvatarSave) > 0 {
		slugcatPath := "assets/samples/slugcat.save"
		if _, err := os.Stat(slugcatPath); os.IsNotExist(err) {
			_ = os.WriteFile(slugcatPath, assets.SlugcatAvatarSave, 0644)
		}
	}
}
