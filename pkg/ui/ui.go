package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
	"pngtuber-lite/pkg/audio"
	"pngtuber-lite/pkg/config"
	"pngtuber-lite/pkg/costume"
	"pngtuber-lite/pkg/updater"
	"pngtuber-lite/pkg/window"
)

type Tab int

const (
	TabAvatars Tab = iota
	TabAudio
	TabCostumes
	TabPhysics
	TabKeybinds
	TabOBS
)

// UIState manages the in-app interactive control drawer.
type UIState struct {
	IsOpen            bool
	CurrentTab        Tab
	AvailableAvatars  []string
	AudioDevices      []string
	SelectedDeviceIdx int
	Font              rl.Font
	HasCustomFont     bool
	RebindingAction   string // Action currently waiting for a key press
	OnAvatarSelected  func(filePath string)
	OnDeviceSelected  func(deviceName string)
	OnResetAvatar     func()
	OnOpenEditor      func()
}

// NewUIState creates a new UI manager.
func NewUIState() *UIState {
	ui := &UIState{
		IsOpen:     false,
		CurrentTab: TabAvatars,
	}
	ui.ScanAvatars()
	return ui
}

// getSupportedRunes returns the complete set of UTF-8 runes (ASCII + Latin-1 Accents + Symbols).
func getSupportedRunes() []rune {
	runes := make([]rune, 0, 512)

	// 1. Basic ASCII printable (32 to 126)
	for r := rune(32); r <= 126; r++ {
		runes = append(runes, r)
	}

	// 2. Latin-1 Supplement (160 to 255: á, é, í, ó, ú, â, ê, ô, ã, õ, ç, à, ü, Á, É, Í, Ó, Ú, Â, Ê, Ô, Ã, Õ, Ç, etc.)
	for r := rune(160); r <= 255; r++ {
		runes = append(runes, r)
	}

	// 3. Latin Extended-A & B (256 to 383)
	for r := rune(256); r <= 383; r++ {
		runes = append(runes, r)
	}

	// 4. Clean UI Symbols
	symbols := []rune{
		'•', '▶', '◀', '▲', '▼', '✓', '✕', '⚙', '★', '☆', '→', '←', '↑', '↓', '—', '“', '”', '…',
	}
	runes = append(runes, symbols...)

	return runes
}

// InitFont loads a crisp anti-aliased TrueType font with full Portuguese UTF-8 accent support.
func (ui *UIState) InitFont() {
	fontPaths := []string{
		"assets/fonts/font.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
		"/usr/share/fonts/truetype/ubuntu/Ubuntu-R.ttf",
	}

	runes := getSupportedRunes()

	for _, p := range fontPaths {
		if _, err := os.Stat(p); err == nil {
			ui.Font = rl.LoadFontEx(p, 36, runes, int32(len(runes)))
			rl.SetTextureFilter(ui.Font.Texture, rl.FilterBilinear)
			ui.HasCustomFont = true
			return
		}
	}
	ui.Font = rl.GetFontDefault()
}

// Unload releases GPU font textures.
func (ui *UIState) Unload() {
	if ui.HasCustomFont {
		rl.UnloadFont(ui.Font)
	}
}

// DrawText draws text using the custom TrueType font if available.
func (ui *UIState) DrawText(text string, x, y int32, size float32, color rl.Color) {
	if ui.HasCustomFont {
		rl.DrawTextEx(ui.Font, text, rl.Vector2{X: float32(x), Y: float32(y)}, size, 1.0, color)
	} else {
		rl.DrawText(text, x, y, int32(size), color)
	}
}

// ScanAvatars searches for .save files in common directories.
func (ui *UIState) ScanAvatars() {
	avatars := make([]string, 0)

	dirs := []string{".", "assets", "assets/samples"}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".save") {
				fullPath := filepath.Join(dir, entry.Name())
				duplicate := false
				for _, a := range avatars {
					if a == fullPath {
						duplicate = true
						break
					}
				}
				if !duplicate {
					avatars = append(avatars, fullPath)
				}
			}
		}
	}

	ui.AvailableAvatars = avatars
}

// Draw renders the menu button and the settings drawer when open.
func (ui *UIState) Draw(
	cfg *config.Config,
	wm *window.WindowManager,
	costumeMgr *costume.CostumeManager,
	audioEngine *audio.CaptureEngine,
	scale *float32,
) {
	screenH := int32(rl.GetScreenHeight())
	mousePos := rl.GetMousePosition()

	// 1. Floating Menu Toggle Button (Top Left)
	menuBtnRec := rl.NewRectangle(12, 12, 110, 34)
	menuHovered := rl.CheckCollisionPointRec(mousePos, menuBtnRec)

	btnBgColor := rl.NewColor(30, 32, 45, 230)
	if menuHovered {
		btnBgColor = rl.NewColor(55, 65, 100, 250)
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			ui.IsOpen = !ui.IsOpen
			if ui.IsOpen {
				ui.ScanAvatars()
				if devices, err := audioEngine.ListDevices(); err == nil {
					ui.AudioDevices = devices
				}
			}
		}
	}

	btnText := "⚙ CONFIG"
	if ui.IsOpen {
		btnText = "✕ FECHAR"
	}
	rl.DrawRectangleRounded(menuBtnRec, 0.3, 4, btnBgColor)
	rl.DrawRectangleRoundedLines(menuBtnRec, 0.3, 4, rl.SkyBlue)
	ui.DrawText(btnText, int32(menuBtnRec.X)+12, int32(menuBtnRec.Y)+8, 15, rl.RayWhite)

	// Floating Editor Toggle Button
	editorBtnRec := rl.NewRectangle(130, 12, 110, 34)
	editorHovered := rl.CheckCollisionPointRec(mousePos, editorBtnRec)
	editorBgColor := rl.NewColor(30, 32, 45, 230)
	if editorHovered {
		editorBgColor = rl.NewColor(45, 75, 120, 250)
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) && ui.OnOpenEditor != nil {
			ui.OnOpenEditor()
		}
	}
	rl.DrawRectangleRounded(editorBtnRec, 0.3, 4, editorBgColor)
	rl.DrawRectangleRoundedLines(editorBtnRec, 0.3, 4, rl.NewColor(80, 140, 220, 255))
	ui.DrawText("✏ EDITOR", int32(editorBtnRec.X)+14, int32(editorBtnRec.Y)+8, 15, rl.RayWhite)

	// Floating Update Button (Shown when new release/hotfix is available)
	upState := updater.GetUpdateState()
	if upState.Available {
		tag := "Nova Versão"
		if upState.Latest != nil {
			tag = upState.Latest.TagName
		}
		btnLabel := fmt.Sprintf("🚀 ATUALIZAR (%s)", tag)
		if upState.IsUpdating {
			btnLabel = fmt.Sprintf("⬇ Baixando... %d%%", int(upState.Progress*100))
		} else if upState.Success {
			btnLabel = "✓ Atualizado! Reinicie"
		}

		upBtnRec := rl.NewRectangle(248, 12, 185, 34)
		upHovered := rl.CheckCollisionPointRec(mousePos, upBtnRec)
		upBgColor := rl.NewColor(30, 110, 60, 235)
		if upHovered {
			upBgColor = rl.NewColor(40, 145, 75, 255)
			if rl.IsMouseButtonPressed(rl.MouseLeftButton) && !upState.IsUpdating && !upState.Success {
				upState.IsUpdating = true
				go func() {
					err := updater.ApplyUpdate(upState.Latest, func(pct float32) {
						upState.Progress = pct
					})
					upState.IsUpdating = false
					if err != nil {
						upState.ErrorMessage = err.Error()
					} else {
						upState.Success = true
					}
				}()
			}
		}
		rl.DrawRectangleRounded(upBtnRec, 0.3, 4, upBgColor)
		rl.DrawRectangleRoundedLines(upBtnRec, 0.3, 4, rl.Lime)
		ui.DrawText(btnLabel, int32(upBtnRec.X)+10, int32(upBtnRec.Y)+8, 13, rl.RayWhite)
	}

	if !ui.IsOpen {
		return
	}

	// 2. Control Drawer Window (Left side panel)
	panelW := int32(410)
	panelH := screenH - 60
	if panelH > 580 {
		panelH = 580
	}
	panelRec := rl.NewRectangle(12, 52, float32(panelW), float32(panelH))

	// Dark semi-transparent background
	rl.DrawRectangleRounded(panelRec, 0.04, 6, rl.NewColor(16, 18, 26, 248))
	rl.DrawRectangleRoundedLines(panelRec, 0.04, 6, rl.NewColor(65, 80, 115, 255))

	// 3. Tab Buttons Header
	tabs := []struct {
		id   Tab
		name string
	}{
		{TabAvatars, "Avatar"},
		{TabAudio, "Áudio"},
		{TabCostumes, "Roupas"},
		{TabPhysics, "Física"},
		{TabKeybinds, "Teclas"},
		{TabOBS, "OBS"},
	}

	tabW := float32(panelW-20) / float32(len(tabs))
	for i, t := range tabs {
		tabRec := rl.NewRectangle(panelRec.X+10+float32(i)*tabW, panelRec.Y+10, tabW-2, 30)
		isActive := ui.CurrentTab == t.id
		isHovered := rl.CheckCollisionPointRec(mousePos, tabRec)

		tabBg := rl.NewColor(28, 32, 45, 255)
		tabTextColor := rl.LightGray
		if isActive {
			tabBg = rl.NewColor(45, 85, 150, 255)
			tabTextColor = rl.RayWhite
		} else if isHovered {
			tabBg = rl.NewColor(40, 50, 70, 255)
		}

		if isHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			ui.CurrentTab = t.id
		}

		rl.DrawRectangleRounded(tabRec, 0.2, 4, tabBg)
		ui.DrawText(t.name, int32(tabRec.X)+8, int32(tabRec.Y)+7, 13, tabTextColor)
	}

	// Content area starting Y
	contentY := int32(panelRec.Y) + 50

	// 4. Render Active Tab Content
	switch ui.CurrentTab {
	case TabAvatars:
		ui.drawAvatarsTab(panelRec, contentY, cfg, mousePos)
	case TabAudio:
		ui.drawAudioTab(panelRec, contentY, cfg, audioEngine, mousePos)
	case TabCostumes:
		ui.drawCostumesTab(panelRec, contentY, cfg, costumeMgr, mousePos)
	case TabPhysics:
		ui.drawPhysicsTab(panelRec, contentY, cfg, mousePos)
	case TabKeybinds:
		ui.drawKeybindsTab(panelRec, contentY, cfg, mousePos)
	case TabOBS:
		ui.drawOBSTab(panelRec, contentY, cfg, wm, scale, mousePos)
	}
}

func (ui *UIState) drawAvatarsTab(panelRec rl.Rectangle, startY int32, cfg *config.Config, mousePos rl.Vector2) {
	ui.DrawText("Avatares Disponíveis (.save):", int32(panelRec.X)+16, startY, 15, rl.SkyBlue)

	y := startY + 28
	for _, avatarPath := range ui.AvailableAvatars {
		isCurrent := (cfg.AvatarPath == avatarPath)
		btnRec := rl.NewRectangle(panelRec.X+16, float32(y), panelRec.Width-32, 36)
		hovered := rl.CheckCollisionPointRec(mousePos, btnRec)

		bgCol := rl.NewColor(32, 38, 52, 255)
		textCol := rl.LightGray
		if isCurrent {
			bgCol = rl.NewColor(35, 105, 65, 255)
			textCol = rl.Lime
		} else if hovered {
			bgCol = rl.NewColor(50, 60, 85, 255)
			textCol = rl.RayWhite
		}

		if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			if ui.OnAvatarSelected != nil {
				ui.OnAvatarSelected(avatarPath)
			}
		}

		rl.DrawRectangleRounded(btnRec, 0.2, 4, bgCol)
		baseName := filepath.Base(avatarPath)
		tag := ""
		if isCurrent {
			tag = " (Ativo)"
		}
		ui.DrawText(fmt.Sprintf("%s%s", baseName, tag), int32(btnRec.X)+12, int32(btnRec.Y)+9, 14, textCol)
		y += 42
	}

	// Open in Editor button
	editRec := rl.NewRectangle(panelRec.X+16, float32(y+10), panelRec.Width-32, 34)
	if rl.CheckCollisionPointRec(mousePos, editRec) {
		rl.DrawRectangleRounded(editRec, 0.2, 4, rl.NewColor(45, 80, 135, 255))
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) && ui.OnOpenEditor != nil {
			ui.OnOpenEditor()
		}
	} else {
		rl.DrawRectangleRounded(editRec, 0.2, 4, rl.NewColor(32, 55, 95, 255))
	}
	ui.DrawText("✏ Abrir no Editor Visual", int32(editRec.X)+60, int32(editRec.Y)+8, 14, rl.RayWhite)

	y += 44

	// Rescan button
	scanRec := rl.NewRectangle(panelRec.X+16, float32(y+10), panelRec.Width-32, 34)
	if rl.CheckCollisionPointRec(mousePos, scanRec) {
		rl.DrawRectangleRounded(scanRec, 0.2, 4, rl.NewColor(55, 65, 95, 255))
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			ui.ScanAvatars()
		}
	} else {
		rl.DrawRectangleRounded(scanRec, 0.2, 4, rl.NewColor(38, 45, 65, 255))
	}
	ui.DrawText("▶ Atualizar Lista de Avatares", int32(scanRec.X)+50, int32(scanRec.Y)+8, 14, rl.RayWhite)
}

func (ui *UIState) drawAudioTab(panelRec rl.Rectangle, startY int32, cfg *config.Config, audioEngine *audio.CaptureEngine, mousePos rl.Vector2) {
	ui.DrawText("Configuração de Microfone:", int32(panelRec.X)+16, startY, 15, rl.SkyBlue)

	// Live VU Meter
	vol := audioEngine.GetVolume()
	isTalking := audioEngine.IsTalking()

	ui.DrawText(fmt.Sprintf("Volume RMS Atual: %.4f", vol), int32(panelRec.X)+16, startY+28, 14, rl.LightGray)

	barW := panelRec.Width - 32
	barRec := rl.NewRectangle(panelRec.X+16, float32(startY+50), barW, 18)
	rl.DrawRectangleRounded(barRec, 0.3, 4, rl.NewColor(25, 25, 30, 255))

	fillW := (vol / 0.15) * barW
	if fillW > barW {
		fillW = barW
	}
	volColor := rl.Lime
	if isTalking {
		volColor = rl.Green
	}
	rl.DrawRectangleRounded(rl.NewRectangle(barRec.X, barRec.Y, fillW, barRec.Height), 0.3, 4, volColor)

	// Threshold red line
	threshX := barRec.X + (cfg.AudioThreshold/0.15)*barW
	if threshX > barRec.X+barW {
		threshX = barRec.X + barW
	}
	rl.DrawLine(int32(threshX), int32(barRec.Y)-3, int32(threshX), int32(barRec.Y)+21, rl.Red)

	// Status text
	statusStr := "Status: SILÊNCIO (Boca Fechada)"
	statusCol := rl.LightGray
	if isTalking {
		statusStr = "Status: FALANDO (Boca Aberta)"
		statusCol = rl.Lime
	}
	ui.DrawText(statusStr, int32(panelRec.X)+16, startY+78, 14, statusCol)

	// Sensitivity Slider
	ui.DrawText(fmt.Sprintf("Limiar de Sensibilidade: %.3f", cfg.AudioThreshold), int32(panelRec.X)+16, startY+108, 14, rl.Yellow)

	sliderRec := rl.NewRectangle(panelRec.X+16, float32(startY+132), barW, 14)
	rl.DrawRectangleRounded(sliderRec, 0.3, 4, rl.DarkGray)

	if rl.IsMouseButtonDown(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mousePos, rl.NewRectangle(sliderRec.X-10, sliderRec.Y-10, sliderRec.Width+20, sliderRec.Height+20)) {
		ratio := (mousePos.X - sliderRec.X) / sliderRec.Width
		if ratio < 0.005 {
			ratio = 0.005
		}
		if ratio > 0.15 {
			ratio = 0.15
		}
		cfg.AudioThreshold = ratio
		audioEngine.SetThreshold(cfg.AudioThreshold)
	}

	sliderHandleX := sliderRec.X + (cfg.AudioThreshold/0.15)*sliderRec.Width
	rl.DrawCircle(int32(sliderHandleX), int32(sliderRec.Y)+7, 9, rl.SkyBlue)

	// Device List
	ui.DrawText("Dispositivos de Entrada (Microfones / DSP):", int32(panelRec.X)+16, startY+160, 14, rl.SkyBlue)
	if len(ui.AudioDevices) == 0 {
		ui.DrawText("(Dispositivo Padrão do Sistema)", int32(panelRec.X)+16, startY+185, 13, rl.Gray)
	} else {
		y := startY + 182
		maxDevs := len(ui.AudioDevices)
		if maxDevs > 8 {
			maxDevs = 8
		}
		for i := 0; i < maxDevs; i++ {
			devName := ui.AudioDevices[i]
			isCurrent := (cfg.AudioDevice == devName || (cfg.AudioDevice == "" && i == 0))
			devBtn := rl.NewRectangle(panelRec.X+16, float32(y), barW, 28)
			hovered := rl.CheckCollisionPointRec(mousePos, devBtn)

			col := rl.NewColor(28, 34, 48, 255)
			textC := rl.LightGray
			prefix := "• "
			if isCurrent {
				col = rl.NewColor(30, 85, 135, 255)
				textC = rl.Lime
				prefix = "✓ "
			} else if hovered {
				col = rl.NewColor(45, 55, 78, 255)
				textC = rl.RayWhite
			}

			if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
				cfg.AudioDevice = devName
				_ = audioEngine.Start(devName)
			}

			rl.DrawRectangleRounded(devBtn, 0.2, 4, col)
			nameShort := devName
			if len(nameShort) > 36 {
				nameShort = nameShort[:36] + "..."
			}
			ui.DrawText(fmt.Sprintf("%s%s", prefix, nameShort), int32(devBtn.X)+8, int32(devBtn.Y)+6, 12, textC)
			y += 32
		}
	}
}

func (ui *UIState) drawCostumesTab(panelRec rl.Rectangle, startY int32, cfg *config.Config, costumeMgr *costume.CostumeManager, mousePos rl.Vector2) {
	ui.DrawText("Selecione o Figurino (Costume 1 a 10):", int32(panelRec.X)+16, startY, 15, rl.SkyBlue)

	// Grid of 10 buttons (2 rows of 5)
	activeCostume := costumeMgr.GetCostume()
	btnW := (panelRec.Width - 32 - 4*8) / 5
	btnH := float32(42)

	for i := 1; i <= 10; i++ {
		row := float32((i - 1) / 5)
		col := float32((i - 1) % 5)

		btnRec := rl.NewRectangle(panelRec.X+16+col*(btnW+8), float32(startY+28)+row*(btnH+8), btnW, btnH)
		isActive := (activeCostume == i)
		hovered := rl.CheckCollisionPointRec(mousePos, btnRec)

		bgCol := rl.NewColor(32, 40, 56, 255)
		txtCol := rl.LightGray
		if isActive {
			bgCol = rl.NewColor(40, 115, 75, 255)
			txtCol = rl.Lime
		} else if hovered {
			bgCol = rl.NewColor(52, 65, 92, 255)
			txtCol = rl.RayWhite
		}

		if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			costumeMgr.SetCostume(i)
		}

		rl.DrawRectangleRounded(btnRec, 0.2, 4, bgCol)
		ui.DrawText(fmt.Sprintf("%d", i), int32(btnRec.X)+int32(btnW/2)-5, int32(btnRec.Y)+11, 16, txtCol)
	}

	// Bounce on Costume Checkbox
	chkRec := rl.NewRectangle(panelRec.X+16, float32(startY+140), 22, 22)
	if rl.CheckCollisionPointRec(mousePos, chkRec) && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		cfg.BounceOnCostume = !cfg.BounceOnCostume
		costumeMgr.BounceOnChange = cfg.BounceOnCostume
	}

	rl.DrawRectangleRounded(chkRec, 0.2, 4, rl.DarkGray)
	if cfg.BounceOnCostume {
		rl.DrawRectangle(int32(chkRec.X)+4, int32(chkRec.Y)+4, 14, 14, rl.Lime)
	}
	ui.DrawText("Pular ao trocar figurino", int32(chkRec.X)+32, int32(chkRec.Y)+3, 14, rl.RayWhite)
}

func (ui *UIState) drawPhysicsTab(panelRec rl.Rectangle, startY int32, cfg *config.Config, mousePos rl.Vector2) {
	ui.DrawText("Ajustes de Animação e Física:", int32(panelRec.X)+16, startY, 15, rl.SkyBlue)

	barW := panelRec.Width - 32
	y := startY + 28

	// 1. Idle Floating / Breathing Intensity (Bobbing)
	ui.DrawText(fmt.Sprintf("Intensidade da Flutuação / Respiração: %.2fx", cfg.BobbingIntensity), int32(panelRec.X)+16, y, 13, rl.Yellow)
	bobRec := rl.NewRectangle(panelRec.X+16, float32(y+18), barW, 12)
	rl.DrawRectangleRounded(bobRec, 0.3, 4, rl.DarkGray)
	if rl.IsMouseButtonDown(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mousePos, rl.NewRectangle(bobRec.X-10, bobRec.Y-8, bobRec.Width+20, bobRec.Height+16)) {
		ratio := (mousePos.X - bobRec.X) / bobRec.Width
		if ratio < 0.0 {
			ratio = 0.0
		}
		if ratio > 1.0 {
			ratio = 1.0
		}
		cfg.BobbingIntensity = ratio * 2.0
	}
	hBob := bobRec.X + (cfg.BobbingIntensity/2.0)*bobRec.Width
	rl.DrawCircle(int32(hBob), int32(bobRec.Y)+6, 7, rl.SkyBlue)

	y += 40

	// 2. Wobble Inertia Intensity
	ui.DrawText(fmt.Sprintf("Intensidade da Inércia (Wobble): %.2fx", cfg.WobbleIntensity), int32(panelRec.X)+16, y, 13, rl.Yellow)
	wobRec := rl.NewRectangle(panelRec.X+16, float32(y+18), barW, 12)
	rl.DrawRectangleRounded(wobRec, 0.3, 4, rl.DarkGray)
	if rl.IsMouseButtonDown(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mousePos, rl.NewRectangle(wobRec.X-10, wobRec.Y-8, wobRec.Width+20, wobRec.Height+16)) {
		ratio := (mousePos.X - wobRec.X) / wobRec.Width
		if ratio < 0.0 {
			ratio = 0.0
		}
		if ratio > 1.0 {
			ratio = 1.0
		}
		cfg.WobbleIntensity = ratio * 2.0
	}
	hWob := wobRec.X + (cfg.WobbleIntensity/2.0)*wobRec.Width
	rl.DrawCircle(int32(hWob), int32(wobRec.Y)+6, 7, rl.SkyBlue)

	y += 40

	// 3. Bounce Strength
	ui.DrawText(fmt.Sprintf("Força do Pulo (Bounce): %.0f", cfg.BounceStrength), int32(panelRec.X)+16, y, 13, rl.LightGray)
	bRec := rl.NewRectangle(panelRec.X+16, float32(y+18), barW, 12)
	rl.DrawRectangleRounded(bRec, 0.3, 4, rl.DarkGray)
	if rl.IsMouseButtonDown(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mousePos, rl.NewRectangle(bRec.X-10, bRec.Y-8, bRec.Width+20, bRec.Height+16)) {
		ratio := (mousePos.X - bRec.X) / bRec.Width
		if ratio < 0.0 {
			ratio = 0.0
		}
		if ratio > 1.0 {
			ratio = 1.0
		}
		cfg.BounceStrength = 50.0 + ratio*550.0
	}
	hX := bRec.X + ((cfg.BounceStrength-50.0)/550.0)*bRec.Width
	rl.DrawCircle(int32(hX), int32(bRec.Y)+6, 7, rl.SkyBlue)

	y += 40

	// 4. Gravity
	ui.DrawText(fmt.Sprintf("Gravidade: %.0f", cfg.BounceGravity), int32(panelRec.X)+16, y, 13, rl.LightGray)
	gRec := rl.NewRectangle(panelRec.X+16, float32(y+18), barW, 12)
	rl.DrawRectangleRounded(gRec, 0.3, 4, rl.DarkGray)
	if rl.IsMouseButtonDown(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mousePos, rl.NewRectangle(gRec.X-10, gRec.Y-8, gRec.Width+20, gRec.Height+16)) {
		ratio := (mousePos.X - gRec.X) / gRec.Width
		if ratio < 0.0 {
			ratio = 0.0
		}
		if ratio > 1.0 {
			ratio = 1.0
		}
		cfg.BounceGravity = 200.0 + ratio*1800.0
	}
	hX = gRec.X + ((cfg.BounceGravity-200.0)/1800.0)*gRec.Width
	rl.DrawCircle(int32(hX), int32(gRec.Y)+6, 7, rl.SkyBlue)

	y += 40

	// 5. Blink Speed
	ui.DrawText(fmt.Sprintf("Duração do Piscar: %.2fx", cfg.BlinkSpeed), int32(panelRec.X)+16, y, 13, rl.LightGray)
	bsRec := rl.NewRectangle(panelRec.X+16, float32(y+18), barW, 12)
	rl.DrawRectangleRounded(bsRec, 0.3, 4, rl.DarkGray)
	if rl.IsMouseButtonDown(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mousePos, rl.NewRectangle(bsRec.X-10, bsRec.Y-8, bsRec.Width+20, bsRec.Height+16)) {
		ratio := (mousePos.X - bsRec.X) / bsRec.Width
		if ratio < 0.0 {
			ratio = 0.0
		}
		if ratio > 1.0 {
			ratio = 1.0
		}
		cfg.BlinkSpeed = 0.2 + ratio*2.8
	}
	hX = bsRec.X + ((cfg.BlinkSpeed-0.2)/2.8)*bsRec.Width
	rl.DrawCircle(int32(hX), int32(bsRec.Y)+6, 7, rl.SkyBlue)
}

func (ui *UIState) drawOBSTab(panelRec rl.Rectangle, startY int32, cfg *config.Config, wm *window.WindowManager, scale *float32, mousePos rl.Vector2) {
	ui.DrawText("Configurações para Transmissão / OBS:", int32(panelRec.X)+16, startY, 15, rl.SkyBlue)

	y := startY + 28

	// OBS Overlay Preset Button
	obsPresetRec := rl.NewRectangle(panelRec.X+16, float32(y), panelRec.Width-32, 36)
	if rl.CheckCollisionPointRec(mousePos, obsPresetRec) {
		rl.DrawRectangleRounded(obsPresetRec, 0.2, 4, rl.NewColor(45, 125, 75, 255))
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			if !wm.IsBorderless() {
				wm.ToggleBorderless()
			}
			if !wm.IsAlwaysOnTop() {
				wm.ToggleAlwaysOnTop()
			}
			cfg.BackgroundColor = [4]uint8{0, 0, 0, 0}
			ui.IsOpen = false
		}
	} else {
		rl.DrawRectangleRounded(obsPresetRec, 0.2, 4, rl.NewColor(32, 85, 55, 255))
	}
	ui.DrawText("▶ Ativar Modo Overlay OBS (Sem Bordas)", int32(obsPresetRec.X)+14, int32(obsPresetRec.Y)+9, 14, rl.RayWhite)

	y += 48

	// Background Presets
	ui.DrawText("Cor de Fundo (Chroma Key):", int32(panelRec.X)+16, y, 14, rl.LightGray)
	y += 22

	colorPresets := []struct {
		name string
		rgba [4]uint8
	}{
		{"Transparente", [4]uint8{0, 0, 0, 0}},
		{"Verde", [4]uint8{0, 255, 0, 255}},
		{"Magenta", [4]uint8{255, 0, 255, 255}},
		{"Azul", [4]uint8{0, 0, 255, 255}},
	}

	cBtnW := (panelRec.Width - 32 - 3*6) / 4
	for i, c := range colorPresets {
		cRec := rl.NewRectangle(panelRec.X+16+float32(i)*(cBtnW+6), float32(y), cBtnW, 28)
		hovered := rl.CheckCollisionPointRec(mousePos, cRec)
		isCur := (cfg.BackgroundColor == c.rgba)

		col := rl.NewColor(35, 42, 58, 255)
		if isCur {
			col = rl.NewColor(55, 95, 155, 255)
		} else if hovered {
			col = rl.NewColor(48, 58, 78, 255)
		}

		if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			cfg.BackgroundColor = c.rgba
		}

		rl.DrawRectangleRounded(cRec, 0.2, 4, col)
		ui.DrawText(c.name, int32(cRec.X)+4, int32(cRec.Y)+6, 12, rl.RayWhite)
	}

	y += 40

	// Zoom Slider
	ui.DrawText(fmt.Sprintf("Zoom / Escala: %.2fx", *scale), int32(panelRec.X)+16, y, 14, rl.Yellow)
	sRec := rl.NewRectangle(panelRec.X+16, float32(y+20), panelRec.Width-32, 14)
	rl.DrawRectangleRounded(sRec, 0.3, 4, rl.DarkGray)
	if rl.IsMouseButtonDown(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mousePos, rl.NewRectangle(sRec.X-10, sRec.Y-8, sRec.Width+20, sRec.Height+16)) {
		ratio := (mousePos.X - sRec.X) / sRec.Width
		*scale = 0.2 + ratio*3.8
	}
	hX := sRec.X + ((*scale-0.2)/3.8)*sRec.Width
	rl.DrawCircle(int32(hX), int32(sRec.Y)+7, 8, rl.SkyBlue)

	y += 45

	// Reset Position Button
	resetRec := rl.NewRectangle(panelRec.X+16, float32(y), panelRec.Width-32, 30)
	if rl.CheckCollisionPointRec(mousePos, resetRec) {
		rl.DrawRectangleRounded(resetRec, 0.2, 4, rl.NewColor(65, 65, 85, 255))
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) && ui.OnResetAvatar != nil {
			ui.OnResetAvatar()
		}
	} else {
		rl.DrawRectangleRounded(resetRec, 0.2, 4, rl.NewColor(40, 42, 55, 255))
	}
	ui.DrawText("▶ Resetar Posição e Escala (R)", int32(resetRec.X)+45, int32(resetRec.Y)+7, 13, rl.RayWhite)

	y += 42

	// OBS Step-by-Step Guide
	guideRec := rl.NewRectangle(panelRec.X+16, float32(y), panelRec.Width-32, 95)
	rl.DrawRectangleRounded(guideRec, 0.1, 4, rl.NewColor(14, 22, 36, 255))
	rl.DrawRectangleRoundedLines(guideRec, 0.1, 4, rl.NewColor(35, 65, 105, 255))

	ui.DrawText("Como adicionar no OBS Studio:", int32(guideRec.X)+10, int32(guideRec.Y)+8, 13, rl.SkyBlue)
	ui.DrawText("1. Adicionar Fonte -> Captura de Janela (Window Capture)", int32(guideRec.X)+10, int32(guideRec.Y)+26, 12, rl.LightGray)
	ui.DrawText("2. Selecione a janela 'PNGTuber Lite'", int32(guideRec.X)+10, int32(guideRec.Y)+44, 12, rl.LightGray)
	ui.DrawText("3. Marque 'Permitir Transparência' (Allow Transparency)", int32(guideRec.X)+10, int32(guideRec.Y)+62, 12, rl.Lime)
	ui.DrawText("Pronto! O avatar fica transparente sem tela verde!", int32(guideRec.X)+10, int32(guideRec.Y)+79, 12, rl.Yellow)

	y += 105

	// Version & Update Check section
	upRec := rl.NewRectangle(panelRec.X+16, float32(y), panelRec.Width-32, 30)
	hoveredUp := rl.CheckCollisionPointRec(mousePos, upRec)
	upCol := rl.NewColor(35, 45, 68, 255)
	if hoveredUp {
		upCol = rl.NewColor(48, 65, 95, 255)
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			updater.CheckForUpdateAsync()
		}
	}
	rl.DrawRectangleRounded(upRec, 0.2, 4, upCol)
	ui.DrawText(fmt.Sprintf("PNGTuber Lite %s | [ 🔍 Verificar Updates ]", updater.CurrentVersion), int32(upRec.X)+12, int32(upRec.Y)+7, 12, rl.SkyBlue)
}

func (ui *UIState) drawKeybindsTab(panelRec rl.Rectangle, startY int32, cfg *config.Config, mousePos rl.Vector2) {
	ui.DrawText("Atalhos de Teclado (Clique para remapear):", int32(panelRec.X)+16, startY, 15, rl.SkyBlue)

	// Check if user is currently rebinding a key
	if ui.RebindingAction != "" {
		key := rl.GetKeyPressed()
		if key != 0 {
			if key == int32(rl.KeyEscape) {
				ui.RebindingAction = ""
			} else {
				switch ui.RebindingAction {
				case "toggleMenu":
					cfg.Keybinds.ToggleMenu = key
				case "toggleEditor":
					cfg.Keybinds.ToggleEditor = key
				case "toggleHUD":
					cfg.Keybinds.ToggleHUD = key
				case "toggleClickThrough":
					cfg.Keybinds.ToggleClickThrough = key
				case "toggleBorderless":
					cfg.Keybinds.ToggleBorderless = key
				case "toggleAlwaysOnTop":
					cfg.Keybinds.ToggleAlwaysOnTop = key
				case "testBounce":
					cfg.Keybinds.TestBounce = key
				case "resetAvatar":
					cfg.Keybinds.ResetAvatar = key
				case "increaseSens":
					cfg.Keybinds.IncreaseSens = key
				case "decreaseSens":
					cfg.Keybinds.DecreaseSens = key
				}
				ui.RebindingAction = ""
			}
		}
	}

	items := []struct {
		actionID string
		label    string
		current  int32
	}{
		{"toggleMenu", "Menu / Configurações", cfg.Keybinds.ToggleMenu},
		{"toggleEditor", "Editor Visual de Avatar", cfg.Keybinds.ToggleEditor},
		{"toggleHUD", "Painel de Depuração (HUD)", cfg.Keybinds.ToggleHUD},
		{"testBounce", "Pulo / Teste de Fala", cfg.Keybinds.TestBounce},
		{"resetAvatar", "Resetar Posição / Escala", cfg.Keybinds.ResetAvatar},
		{"toggleClickThrough", "Modo Click-Through", cfg.Keybinds.ToggleClickThrough},
		{"toggleBorderless", "Janela Sem Bordas (Overlay)", cfg.Keybinds.ToggleBorderless},
		{"toggleAlwaysOnTop", "Sempre no Topo (Always on Top)", cfg.Keybinds.ToggleAlwaysOnTop},
		{"increaseSens", "Aumentar Sensibilidade Mic", cfg.Keybinds.IncreaseSens},
		{"decreaseSens", "Diminuir Sensibilidade Mic", cfg.Keybinds.DecreaseSens},
	}

	y := startY + 26
	for _, it := range items {
		// Action description label
		ui.DrawText(it.label, int32(panelRec.X)+16, y+4, 12, rl.LightGray)

		// Key button on the right
		btnW := float32(115)
		btnH := float32(23)
		btnRec := rl.NewRectangle(panelRec.X+panelRec.Width-btnW-16, float32(y), btnW, btnH)
		hovered := rl.CheckCollisionPointRec(mousePos, btnRec)
		isListening := (ui.RebindingAction == it.actionID)

		col := rl.NewColor(30, 38, 55, 255)
		txtCol := rl.RayWhite
		text := config.GetKeyName(it.current)

		if isListening {
			col = rl.NewColor(170, 55, 40, 255)
			txtCol = rl.Yellow
			text = "Pressione..."
		} else if hovered {
			col = rl.NewColor(45, 65, 95, 255)
			txtCol = rl.SkyBlue
		}

		if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			if isListening {
				ui.RebindingAction = ""
			} else {
				ui.RebindingAction = it.actionID
			}
		}

		rl.DrawRectangleRounded(btnRec, 0.25, 4, col)
		rl.DrawRectangleRoundedLines(btnRec, 0.25, 4, rl.NewColor(60, 80, 120, 255))
		ui.DrawText(text, int32(btnRec.X)+8, int32(btnRec.Y)+4, 12, txtCol)

		y += 28
	}

	// Restore Defaults Button
	resetBtnRec := rl.NewRectangle(panelRec.X+16, float32(y+6), panelRec.Width-32, 28)
	hoveredReset := rl.CheckCollisionPointRec(mousePos, resetBtnRec)
	resetBg := rl.NewColor(36, 42, 58, 255)
	if hoveredReset {
		resetBg = rl.NewColor(52, 64, 88, 255)
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			cfg.Keybinds = config.DefaultKeybinds()
			ui.RebindingAction = ""
		}
	}
	rl.DrawRectangleRounded(resetBtnRec, 0.2, 4, resetBg)
	ui.DrawText("↺ Restaurar Teclas Padrão", int32(resetBtnRec.X)+int32(resetBtnRec.Width/2)-75, int32(resetBtnRec.Y)+6, 12, rl.RayWhite)
}
