package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
	"pngtuber-lite/assets"
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

// UIState manages the in-app interactive control drawer and UI overlays.
type UIState struct {
	IsOpen                  bool
	CurrentTab              Tab
	AvailableAvatars        []string
	AudioDevices            []string
	SelectedDeviceIdx       int
	Font                    rl.Font
	FontBold                rl.Font
	HasCustomFont           bool
	RebindingAction         string // Action currently waiting for a key press
	ShowCloseModal          bool

	// Scrollable container state
	ScrollOffset            float32
	IsDraggingScrollbar     bool
	ScrollbarDragStartY     float32
	ScrollbarStartOffset    float32

	// Callbacks
	OnAvatarSelected        func(filePath string)
	OnDeviceSelected        func(deviceName string)
	OnResetAvatar           func()
	OnOpenEditor            func()
	OnCreateNewAvatar       func()
	OnImportFolderAvatar    func(dirPath string)
	OnRequestClose          func()
	OnRequestMinimizeToTray func()
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
		'•', '▶', '◀', '▲', '▼', '✓', '✕', '⚙', '★', '☆', '→', '←', '↑', '↓', '—', '“', '”', '…', '🗣', '🤫', '●', '○', '└', '├', '│', '─', '⭐',
	}
	runes = append(runes, symbols...)

	return runes
}

// InitFont loads crisp anti-aliased TrueType fonts with full Portuguese UTF-8 accent support.
func (ui *UIState) InitFont() {
	runes := getSupportedRunes()

	// 1. Load Regular Font from embedded assets or local files
	if len(assets.RegularFontTTF) > 0 {
		ui.Font = rl.LoadFontFromMemory(".ttf", assets.RegularFontTTF, 38, runes)
		rl.SetTextureFilter(ui.Font.Texture, rl.FilterBilinear)
		ui.HasCustomFont = true
	} else if _, err := os.Stat("assets/fonts/font.ttf"); err == nil {
		ui.Font = rl.LoadFontEx("assets/fonts/font.ttf", 38, runes, int32(len(runes)))
		rl.SetTextureFilter(ui.Font.Texture, rl.FilterBilinear)
		ui.HasCustomFont = true
	} else {
		ui.Font = rl.GetFontDefault()
	}

	// 2. Load Bold Font
	if len(assets.BoldFontTTF) > 0 {
		ui.FontBold = rl.LoadFontFromMemory(".ttf", assets.BoldFontTTF, 38, runes)
		rl.SetTextureFilter(ui.FontBold.Texture, rl.FilterBilinear)
	} else if _, err := os.Stat("assets/fonts/font_bold.ttf"); err == nil {
		ui.FontBold = rl.LoadFontEx("assets/fonts/font_bold.ttf", 38, runes, int32(len(runes)))
		rl.SetTextureFilter(ui.FontBold.Texture, rl.FilterBilinear)
	} else {
		ui.FontBold = ui.Font
	}
}

// Unload releases GPU font textures.
func (ui *UIState) Unload() {
	if ui.HasCustomFont {
		rl.UnloadFont(ui.Font)
		if ui.FontBold.Texture.ID != ui.Font.Texture.ID && ui.FontBold.Texture.ID != 0 {
			rl.UnloadFont(ui.FontBold)
		}
	}
}

// DrawText draws text using the regular TrueType font.
func (ui *UIState) DrawText(text string, x, y int32, size float32, color rl.Color) {
	if ui.HasCustomFont {
		rl.DrawTextEx(ui.Font, text, rl.Vector2{X: float32(x), Y: float32(y)}, size, 1.0, color)
	} else {
		rl.DrawText(text, x, y, int32(size), color)
	}
}

// DrawTextBold draws text using the bold TrueType font.
func (ui *UIState) DrawTextBold(text string, x, y int32, size float32, color rl.Color) {
	if ui.HasCustomFont {
		rl.DrawTextEx(ui.FontBold, text, rl.Vector2{X: float32(x), Y: float32(y)}, size, 1.0, color)
	} else {
		rl.DrawText(text, x, y, int32(size), color)
	}
}

// MeasureText calculates the width of a string with the regular font.
func (ui *UIState) MeasureText(text string, size float32) float32 {
	if ui.HasCustomFont {
		vec := rl.MeasureTextEx(ui.Font, text, size, 1.0)
		return vec.X
	}
	return float32(rl.MeasureText(text, int32(size)))
}

// MeasureTextBold calculates the width of a string with the bold font.
func (ui *UIState) MeasureTextBold(text string, size float32) float32 {
	if ui.HasCustomFont {
		vec := rl.MeasureTextEx(ui.FontBold, text, size, 1.0)
		return vec.X
	}
	return float32(rl.MeasureText(text, int32(size)))
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

// BeginScrollView handles mouse wheel input and starts GPU scissor clipping for scrollable areas.
func (ui *UIState) BeginScrollView(containerRec rl.Rectangle, contentHeight float32) float32 {
	mousePos := rl.GetMousePosition()
	if rl.CheckCollisionPointRec(mousePos, containerRec) {
		wheel := rl.GetMouseWheelMove()
		if wheel != 0 {
			ui.ScrollOffset -= wheel * 36.0
		}
	}

	maxScroll := contentHeight - containerRec.Height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if ui.ScrollOffset < 0 {
		ui.ScrollOffset = 0
	}
	if ui.ScrollOffset > maxScroll {
		ui.ScrollOffset = maxScroll
	}

	rl.BeginScissorMode(
		int32(containerRec.X),
		int32(containerRec.Y),
		int32(containerRec.Width),
		int32(containerRec.Height),
	)

	return containerRec.Y - ui.ScrollOffset
}

// EndScrollView finishes scissor clipping and renders the custom cozy scrollbar.
func (ui *UIState) EndScrollView(containerRec rl.Rectangle, contentHeight float32) {
	rl.EndScissorMode()

	if contentHeight <= containerRec.Height {
		return // No scrollbar needed
	}

	maxScroll := contentHeight - containerRec.Height
	trackW := float32(6)
	trackX := containerRec.X + containerRec.Width - trackW - 3
	trackY := containerRec.Y + 4
	trackH := containerRec.Height - 8

	// Draw Track
	rl.DrawRectangleRounded(rl.NewRectangle(trackX, trackY, trackW, trackH), 0.5, 4, ColScrollTrack)

	// Calculate Thumb position
	thumbH := (containerRec.Height / contentHeight) * trackH
	if thumbH < 28 {
		thumbH = 28
	}
	scrollRatio := ui.ScrollOffset / maxScroll
	thumbY := trackY + scrollRatio*(trackH-thumbH)
	thumbRec := rl.NewRectangle(trackX-1, thumbY, trackW+2, thumbH)

	mousePos := rl.GetMousePosition()
	isHovered := rl.CheckCollisionPointRec(mousePos, thumbRec)

	thumbCol := ColScrollThumb
	if isHovered || ui.IsDraggingScrollbar {
		thumbCol = ColScrollThumbHov
	}

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) && isHovered {
		ui.IsDraggingScrollbar = true
		ui.ScrollbarDragStartY = mousePos.Y
		ui.ScrollbarStartOffset = ui.ScrollOffset
	}

	if ui.IsDraggingScrollbar {
		if rl.IsMouseButtonDown(rl.MouseLeftButton) {
			deltaY := mousePos.Y - ui.ScrollbarDragStartY
			scrollChange := (deltaY / (trackH - thumbH)) * maxScroll
			ui.ScrollOffset = ui.ScrollbarStartOffset + scrollChange
			if ui.ScrollOffset < 0 {
				ui.ScrollOffset = 0
			}
			if ui.ScrollOffset > maxScroll {
				ui.ScrollOffset = maxScroll
			}
		} else {
			ui.IsDraggingScrollbar = false
		}
	}

	rl.DrawRectangleRounded(thumbRec, 0.5, 4, thumbCol)
}

// Draw renders the navigation toolbar, drawer panel, and active modals.
func (ui *UIState) Draw(
	cfg *config.Config,
	wm *window.WindowManager,
	costumeMgr *costume.CostumeManager,
	audioEngine *audio.CaptureEngine,
	scale *float32,
) {
	screenW := float32(rl.GetScreenWidth())
	screenH := float32(rl.GetScreenHeight())
	mousePos := rl.GetMousePosition()

	// 1. Floating Menu Toggle Button (Pill shaped with generous breathing room)
	menuBtnRec := rl.NewRectangle(14, 12, 125, 36)
	menuHovered := rl.CheckCollisionPointRec(mousePos, menuBtnRec)

	menuBgColor := ColPillBg
	menuBorderColor := ColPanelBorder
	if ui.IsOpen {
		menuBgColor = ColCardActive
		menuBorderColor = ColSkyBlue
	} else if menuHovered {
		menuBgColor = ColCardHover
		menuBorderColor = ColSkyBlue
	}

	if menuHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		ui.IsOpen = !ui.IsOpen
		if ui.IsOpen {
			ui.ScrollOffset = 0
			ui.ScanAvatars()
			if devices, err := audioEngine.ListDevices(); err == nil {
				ui.AudioDevices = devices
			}
		}
	}

	rl.DrawRectangleRounded(menuBtnRec, 0.45, 6, menuBgColor)
	rl.DrawRectangleRoundedLines(menuBtnRec, 0.45, 6, menuBorderColor)

	if ui.IsOpen {
		GlobalIcons.DrawIcon(IconClose, menuBtnRec.X+14, menuBtnRec.Y+9, 18, ColRed)
		ui.DrawTextBold("FECHAR", int32(menuBtnRec.X)+40, int32(menuBtnRec.Y)+9, 13, ColTextTitle)
	} else {
		GlobalIcons.DrawIcon(IconSettings, menuBtnRec.X+14, menuBtnRec.Y+9, 18, ColSkyBlue)
		ui.DrawTextBold("CONFIG", int32(menuBtnRec.X)+40, int32(menuBtnRec.Y)+9, 13, ColTextTitle)
	}

	// 2. Floating Editor Toggle Button
	editorBtnRec := rl.NewRectangle(146, 12, 125, 36)
	editorHovered := rl.CheckCollisionPointRec(mousePos, editorBtnRec)
	editorBgColor := ColPillBg
	editorBorderColor := ColPanelBorder

	if editorHovered {
		editorBgColor = ColCardHover
		editorBorderColor = ColLavender
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) && ui.OnOpenEditor != nil {
			ui.OnOpenEditor()
		}
	}
	rl.DrawRectangleRounded(editorBtnRec, 0.45, 6, editorBgColor)
	rl.DrawRectangleRoundedLines(editorBtnRec, 0.45, 6, editorBorderColor)
	GlobalIcons.DrawIcon(IconEditor, editorBtnRec.X+14, editorBtnRec.Y+9, 18, ColLavender)
	ui.DrawTextBold("EDITOR", int32(editorBtnRec.X)+40, int32(editorBtnRec.Y)+9, 13, ColTextTitle)

	// 3. Floating Update Button (Shown when new release/hotfix is available)
	upState := updater.GetUpdateState()

	if upState.Success && !upState.RestartTriggered {
		upState.RestartCountdown -= rl.GetFrameTime()
		if upState.RestartCountdown <= 0 {
			upState.RestartTriggered = true
			go updater.RestartApp()
		}
	}

	if upState.Available {
		tag := "Nova Versão"
		if upState.Latest != nil {
			tag = upState.Latest.TagName
		}
		btnLabel := fmt.Sprintf("ATUALIZAR (%s)", tag)
		if upState.IsUpdating {
			btnLabel = fmt.Sprintf("Baixando... %d%%", int(upState.Progress*100))
		} else if upState.Success {
			btnLabel = fmt.Sprintf("Reiniciando (%.0fs)", upState.RestartCountdown)
		}

		upBtnRec := rl.NewRectangle(278, 12, 230, 36)
		upHovered := rl.CheckCollisionPointRec(mousePos, upBtnRec)
		upBgColor := rl.NewColor(24, 75, 45, 235)
		if upHovered {
			upBgColor = rl.NewColor(32, 105, 60, 255)
			if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
				if upState.Success {
					go updater.RestartApp()
				} else if !upState.IsUpdating {
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
							upState.RestartCountdown = 2.0
						}
					}()
				}
			}
		}
		rl.DrawRectangleRounded(upBtnRec, 0.45, 6, upBgColor)
		rl.DrawRectangleRoundedLines(upBtnRec, 0.45, 6, ColLime)
		GlobalIcons.DrawIcon(IconUpdate, upBtnRec.X+14, upBtnRec.Y+9, 18, ColLime)
		ui.DrawTextBold(btnLabel, int32(upBtnRec.X)+40, int32(upBtnRec.Y)+9, 12.5, ColWhite)
	}

	// 4. Main Control Drawer Panel
	if ui.IsOpen {
		panelW := float32(500)
		panelH := screenH - 68
		if panelH > 640 {
			panelH = 640
		}
		if panelH < 440 {
			panelH = 440
		}
		panelRec := rl.NewRectangle(14, 54, panelW, panelH)

		// Panel backdrop & soft border
		rl.DrawRectangleRounded(panelRec, 0.04, 6, ColPanelBg)
		rl.DrawRectangleRoundedLines(panelRec, 0.04, 6, ColPanelBorder)

		// Panel Header
		headerRec := rl.NewRectangle(panelRec.X+14, panelRec.Y+12, panelRec.Width-28, 44)
		ui.DrawIconBadge(headerRec.X+4, headerRec.Y+4, 36, IconSettings, ColSkyBlue, ColIconBoxBg)
		ui.DrawTextBold("PNGTuber Lite", int32(headerRec.X)+50, int32(headerRec.Y)+6, 16, ColTextTitle)
		ui.DrawText("Painel de Controle e Ajustes", int32(headerRec.X)+50, int32(headerRec.Y)+25, 12, ColTextMuted)

		// Close 'X' button
		closeHeaderRec := rl.NewRectangle(panelRec.X+panelRec.Width-42, panelRec.Y+18, 28, 28)
		if rl.CheckCollisionPointRec(mousePos, closeHeaderRec) {
			rl.DrawRectangleRounded(closeHeaderRec, 0.35, 4, ColCardHover)
			if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
				ui.IsOpen = false
			}
		}
		GlobalIcons.DrawIcon(IconClose, closeHeaderRec.X+6, closeHeaderRec.Y+6, 16, ColLightGray)

		// Pill Tabs Navigation Bar (Centered Icon + Text per tab)
		tabs := []struct {
			id   Tab
			name string
			icon int
		}{
			{TabAvatars, "Avatar", IconAvatar},
			{TabAudio, "Áudio", IconAudio},
			{TabCostumes, "Roupas", IconCostumes},
			{TabPhysics, "Física", IconPhysics},
			{TabKeybinds, "Teclas", IconKeys},
			{TabOBS, "OBS", IconOBS},
		}

		tabBarRec := rl.NewRectangle(panelRec.X+14, panelRec.Y+62, panelRec.Width-28, 36)
		rl.DrawRectangleRounded(tabBarRec, 0.45, 6, ColPillBg)

		tabW := tabBarRec.Width / float32(len(tabs))
		for i, t := range tabs {
			tabRec := rl.NewRectangle(tabBarRec.X+float32(i)*tabW+2, tabBarRec.Y+2, tabW-4, tabBarRec.Height-4)
			isActive := (ui.CurrentTab == t.id)
			isHovered := rl.CheckCollisionPointRec(mousePos, tabRec)

			tabBg := rl.NewColor(0, 0, 0, 0)
			textColor := ColTextMuted
			iconColor := ColTextMuted

			if isActive {
				tabBg = ColCardActive
				textColor = ColWhite
				iconColor = ColSkyBlue
			} else if isHovered {
				tabBg = ColCardHover
				textColor = ColWhite
				iconColor = ColLightGray
			}

			if isHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
				ui.CurrentTab = t.id
				ui.ScrollOffset = 0 // Reset scroll position when switching tabs
			}

			if isActive || isHovered {
				rl.DrawRectangleRounded(tabRec, 0.4, 4, tabBg)
			}

			iconW := float32(16)
			textW := ui.MeasureTextBold(t.name, 11)
			totalW := iconW + 4 + textW
			startX := tabRec.X + (tabRec.Width-totalW)/2

			GlobalIcons.DrawIcon(t.icon, startX, tabRec.Y+8, 16, iconColor)
			ui.DrawTextBold(t.name, int32(startX+iconW+4), int32(tabRec.Y)+9, 11, textColor)
		}

		// Scrollable Content Viewport
		viewRec := rl.NewRectangle(panelRec.X+14, panelRec.Y+106, panelRec.Width-28, panelRec.Height-118)

		switch ui.CurrentTab {
		case TabAvatars:
			ui.drawAvatarsTab(viewRec, cfg, mousePos)
		case TabAudio:
			ui.drawAudioTab(viewRec, cfg, audioEngine, mousePos)
		case TabCostumes:
			ui.drawCostumesTab(viewRec, cfg, costumeMgr, mousePos)
		case TabPhysics:
			ui.drawPhysicsTab(viewRec, cfg, mousePos)
		case TabKeybinds:
			ui.drawKeybindsTab(viewRec, cfg, mousePos)
		case TabOBS:
			ui.drawOBSTab(viewRec, cfg, wm, scale, mousePos)
		}
	}

	// 5. Update Popup Modal
	ui.drawUpdateModal(mousePos, screenW, screenH)

	// 6. Close Confirmation Modal
	ui.drawCloseConfirmModal(mousePos, screenW, screenH)
}

// -------------------------------------------------------------
// TAB 1: AVATARES
// -------------------------------------------------------------
func (ui *UIState) drawAvatarsTab(viewRec rl.Rectangle, cfg *config.Config, mousePos rl.Vector2) {
	itemH := float32(58)
	avatarCount := float32(len(ui.AvailableAvatars))
	contentH := float32(40) + avatarCount*(itemH+8) + 400.0

	startY := ui.BeginScrollView(viewRec, contentH)
	curY := startY

	// Section Title
	ui.DrawTextBold("Avatares Salvos (.save)", int32(viewRec.X)+4, int32(curY), 14, ColSkyBlue)
	ui.DrawBadge(viewRec.X+viewRec.Width-75, curY-1, fmt.Sprintf("%d arqs", len(ui.AvailableAvatars)), ColCardBg, ColTextMuted)
	curY += 28

	// Avatar Cards List
	for _, avatarPath := range ui.AvailableAvatars {
		isCurrent := (cfg.AvatarPath == avatarPath)
		cardRec := rl.NewRectangle(viewRec.X+2, curY, viewRec.Width-16, itemH)
		hovered := rl.CheckCollisionPointRec(mousePos, cardRec)

		DrawCard(cardRec, hovered, isCurrent)

		// Icon Badge
		iconTint := ColSkyBlue
		if isCurrent {
			iconTint = ColLime
		}
		ui.DrawIconBadge(cardRec.X+10, cardRec.Y+9, 40, IconAvatar, iconTint, ColIconBoxBg)

		// Title & Path
		baseName := filepath.Base(avatarPath)
		ui.DrawTextBold(baseName, int32(cardRec.X)+58, int32(cardRec.Y)+11, 13.5, ColTextTitle)

		dirInfo := filepath.Dir(avatarPath)
		if dirInfo == "." {
			dirInfo = "Pasta principal"
		}
		ui.DrawText(dirInfo, int32(cardRec.X)+58, int32(cardRec.Y)+32, 11.5, ColTextMuted)

		// Status Badge
		if isCurrent {
			ui.DrawBadge(cardRec.X+cardRec.Width-80, cardRec.Y+18, "✓ Ativo", rl.NewColor(24, 85, 48, 255), ColLime)
		} else if hovered {
			ui.DrawBadge(cardRec.X+cardRec.Width-92, cardRec.Y+18, "Carregar", ColCardHover, ColSkyBlue)
		}

		if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			if ui.OnAvatarSelected != nil {
				ui.OnAvatarSelected(avatarPath)
			}
		}

		curY += itemH + 8
	}

	curY += 12

	// Section 2: Criar e Importar Novos Personagens
	ui.DrawTextBold("Criar / Importar Novo Personagem", int32(viewRec.X)+4, int32(curY), 14, ColSkyBlue)
	curY += 26

	// Action Card 1: Criar Novo Avatar em Branco
	newRec := rl.NewRectangle(viewRec.X+2, curY, viewRec.Width-16, 50)
	hoveredNew := rl.CheckCollisionPointRec(mousePos, newRec)
	DrawCard(newRec, hoveredNew, false)
	ui.DrawIconBadge(newRec.X+8, newRec.Y+8, 34, IconAdd, ColLime, ColIconBoxBg)
	ui.DrawTextBold("Criar Novo Avatar (Tela em Branco)", int32(newRec.X)+52, int32(newRec.Y)+10, 13, ColTextTitle)
	ui.DrawText("Inicia um projeto limpo no editor visual", int32(newRec.X)+52, int32(newRec.Y)+28, 11, ColTextMuted)
	if hoveredNew && rl.IsMouseButtonPressed(rl.MouseLeftButton) && ui.OnCreateNewAvatar != nil {
		ui.OnCreateNewAvatar()
	}

	curY += 58

	// Action Card 2: Importar Pasta de PNGs (ex: SlugcatPNGs)
	importRec := rl.NewRectangle(viewRec.X+2, curY, viewRec.Width-16, 50)
	hoveredImport := rl.CheckCollisionPointRec(mousePos, importRec)
	DrawCard(importRec, hoveredImport, false)
	ui.DrawIconBadge(importRec.X+8, importRec.Y+8, 34, IconPNGFile, ColSkyBlue, ColIconBoxBg)
	ui.DrawTextBold("Importar Pasta com Sprites PNG", int32(importRec.X)+52, int32(importRec.Y)+10, 13, ColTextTitle)
	ui.DrawText("Cria avatares 4-estados automaticamente de uma pasta", int32(importRec.X)+52, int32(importRec.Y)+28, 11, ColTextMuted)
	if hoveredImport && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		slugcatDir := "assets/samples/SlugcatPNGs"
		if _, err := os.Stat(slugcatDir); err == nil && ui.OnImportFolderAvatar != nil {
			ui.OnImportFolderAvatar(slugcatDir)
		} else if ui.OnOpenEditor != nil {
			ui.OnOpenEditor()
		}
	}

	curY += 58

	// Action Card 3: Abrir no Editor Visual
	editRec := rl.NewRectangle(viewRec.X+2, curY, viewRec.Width-16, 46)
	hoveredEdit := rl.CheckCollisionPointRec(mousePos, editRec)
	DrawCard(editRec, hoveredEdit, false)
	ui.DrawIconBadge(editRec.X+8, editRec.Y+7, 32, IconEditor, ColLavender, ColIconBoxBg)
	ui.DrawTextBold("Abrir Avatar Ativo no Editor Visual", int32(editRec.X)+48, int32(editRec.Y)+14, 13, ColTextTitle)
	if hoveredEdit && rl.IsMouseButtonPressed(rl.MouseLeftButton) && ui.OnOpenEditor != nil {
		ui.OnOpenEditor()
	}

	curY += 54

	// Section 3: Orientação e Exibição do Avatar
	ui.DrawTextBold("Orientação e Ajustes de Exibição", int32(viewRec.X)+4, int32(curY), 14, ColSkyBlue)
	curY += 26

	// Flip Horizontal Toggle Card
	flipCard := rl.NewRectangle(viewRec.X+2, curY, viewRec.Width-16, 52)
	DrawCard(flipCard, false, false)
	toggleRec := rl.NewRectangle(flipCard.X+14, flipCard.Y+12, flipCard.Width-28, 28)
	cfg.FlipHorizontal = ui.DrawToggle(toggleRec, "Inverter Avatar Horizontalmente (Flip H)", cfg.FlipHorizontal, mousePos)

	curY += 60

	// Action Card 4: Recarregar Lista
	scanRec := rl.NewRectangle(viewRec.X+2, curY, viewRec.Width-16, 46)
	hoveredScan := rl.CheckCollisionPointRec(mousePos, scanRec)
	DrawCard(scanRec, hoveredScan, false)
	ui.DrawIconBadge(scanRec.X+8, scanRec.Y+7, 32, IconRestart, ColLavender, ColIconBoxBg)
	ui.DrawTextBold("Escanear Pasta por Novos .save", int32(scanRec.X)+48, int32(scanRec.Y)+14, 13, ColTextTitle)
	if hoveredScan && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		ui.ScanAvatars()
	}

	ui.EndScrollView(viewRec, contentH)
}

// -------------------------------------------------------------
// TAB 2: ÁUDIO & MICROFONE
// -------------------------------------------------------------
func (ui *UIState) drawAudioTab(viewRec rl.Rectangle, cfg *config.Config, audioEngine *audio.CaptureEngine, mousePos rl.Vector2) {
	vol := audioEngine.GetVolume()
	isTalking := audioEngine.IsTalking()

	devCount := len(ui.AudioDevices)
	contentH := float32(310) + float32(devCount)*54 + 50

	startY := ui.BeginScrollView(viewRec, contentH)
	curY := startY

	// Card 1: Live VU Meter & Status
	vuCard := rl.NewRectangle(viewRec.X+2, curY, viewRec.Width-16, 116)
	DrawCard(vuCard, false, false)

	ui.DrawTextBold("Monitor de Volume (VAD)", int32(vuCard.X)+14, int32(vuCard.Y)+12, 14, ColSkyBlue)

	statusText := "🤫 Silêncio (Boca Fechada)"
	statusCol := ColTextMuted
	statusBg := ColIconBoxBg
	if isTalking {
		statusText = "🗣 Falando (Boca Aberta)"
		statusCol = ColLime
		statusBg = rl.NewColor(24, 85, 48, 255)
	}
	ui.DrawBadge(vuCard.X+vuCard.Width-195, vuCard.Y+10, statusText, statusBg, statusCol)

	// Level Bar
	barW := vuCard.Width - 28
	barRec := rl.NewRectangle(vuCard.X+14, vuCard.Y+44, barW, 22)
	rl.DrawRectangleRounded(barRec, 0.5, 4, ColScrollTrack)

	fillW := (vol / 0.15) * barW
	if fillW > barW {
		fillW = barW
	}
	fillCol := ColSkyBlue
	if isTalking {
		fillCol = ColLime
	}
	if fillW > 0 {
		rl.DrawRectangleRounded(rl.NewRectangle(barRec.X, barRec.Y, fillW, barRec.Height), 0.5, 4, fillCol)
	}

	// Threshold red line marker
	threshX := barRec.X + (cfg.AudioThreshold/0.15)*barW
	if threshX > barRec.X+barW {
		threshX = barRec.X + barW
	}
	rl.DrawLineEx(rl.NewVector2(threshX, barRec.Y-3), rl.NewVector2(threshX, barRec.Y+barRec.Height+3), 2.5, ColRed)

	ui.DrawText(fmt.Sprintf("Volume RMS: %.4f", vol), int32(vuCard.X)+14, int32(vuCard.Y)+78, 12, ColTextBody)
	ui.DrawText(fmt.Sprintf("Corte: %.4f", cfg.AudioThreshold), int32(vuCard.X+vuCard.Width)-115, int32(vuCard.Y)+78, 12, ColYellow)

	curY += 128

	// Card 2: Sensibilidade Slider
	sensCard := rl.NewRectangle(viewRec.X+2, curY, viewRec.Width-16, 90)
	DrawCard(sensCard, false, false)

	ui.DrawTextBold("Sensibilidade do Microfone", int32(sensCard.X)+14, int32(sensCard.Y)+12, 14, ColSkyBlue)
	ui.DrawText("Arraste para que a barra passe da linha vermelha ao falar", int32(sensCard.X)+14, int32(sensCard.Y)+34, 11.5, ColTextMuted)

	sliderTrack := rl.NewRectangle(sensCard.X+14, sensCard.Y+58, sensCard.Width-28, 12)
	newSens := ui.DrawSliderControl(sliderTrack, cfg.AudioThreshold, 0.002, 0.12, mousePos, ColSkyBlue)
	if newSens != cfg.AudioThreshold {
		cfg.AudioThreshold = newSens
		audioEngine.SetThreshold(cfg.AudioThreshold)
	}

	curY += 102

	// Card 3: Dispositivos de Entrada
	ui.DrawTextBold("Dispositivos de Entrada (Microfones / DSP)", int32(viewRec.X)+4, int32(curY), 14, ColSkyBlue)
	ui.DrawBadge(viewRec.X+viewRec.Width-75, curY-1, fmt.Sprintf("%d devs", devCount), ColCardBg, ColTextMuted)
	curY += 28

	if devCount == 0 {
		devCard := rl.NewRectangle(viewRec.X+2, curY, viewRec.Width-16, 44)
		DrawCard(devCard, false, false)
		ui.DrawIconBadge(devCard.X+8, devCard.Y+6, 32, IconAudio, ColSkyBlue, ColIconBoxBg)
		ui.DrawText("Dispositivo Padrão do Sistema Operacional", int32(devCard.X)+48, int32(devCard.Y)+13, 12, ColTextBody)
	} else {
		for i, devName := range ui.AudioDevices {
			isCurrent := (cfg.AudioDevice == devName || (cfg.AudioDevice == "" && i == 0))
			devCard := rl.NewRectangle(viewRec.X+2, curY, viewRec.Width-16, 44)
			hovered := rl.CheckCollisionPointRec(mousePos, devCard)

			DrawCard(devCard, hovered, isCurrent)
			ui.DrawIconBadge(devCard.X+8, devCard.Y+6, 32, IconAudio, ColSkyBlue, ColIconBoxBg)

			shortName := devName
			if len(shortName) > 34 {
				shortName = shortName[:34] + "..."
			}
			ui.DrawTextBold(shortName, int32(devCard.X)+48, int32(devCard.Y)+13, 12, ColTextTitle)

			if isCurrent {
				ui.DrawBadge(devCard.X+devCard.Width-80, devCard.Y+11, "● Ativo", rl.NewColor(24, 85, 48, 255), ColLime)
			} else if hovered {
				ui.DrawBadge(devCard.X+devCard.Width-92, devCard.Y+11, "○ Escolher", ColCardHover, ColSkyBlue)
			}

			if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
				cfg.AudioDevice = devName
				_ = audioEngine.Start(devName)
			}

			curY += 52
		}
	}

	ui.EndScrollView(viewRec, contentH)
}

// -------------------------------------------------------------
// TAB 3: ROUPAS (FIGURINOS)
// -------------------------------------------------------------
func (ui *UIState) drawCostumesTab(viewRec rl.Rectangle, cfg *config.Config, costumeMgr *costume.CostumeManager, mousePos rl.Vector2) {
	contentH := float32(360)
	startY := ui.BeginScrollView(viewRec, contentH)
	curY := startY

	// Section Title
	ui.DrawTextBold("Figurinos do Avatar (Slots 1 a 10)", int32(viewRec.X)+4, int32(curY), 14, ColSkyBlue)
	ui.DrawText("Pressione as teclas 1 a 0 ou clique nos cards para trocar de roupa", int32(viewRec.X)+4, int32(curY)+22, 11.5, ColTextMuted)
	curY += 46

	activeCostume := costumeMgr.GetCostume()

	// Grid: 2 rows of 5 cards
	gridCols := 5
	cardGap := float32(8)
	cardW := (viewRec.Width - 16 - float32(gridCols-1)*cardGap) / float32(gridCols)
	cardH := float32(60)

	for i := 1; i <= 10; i++ {
		row := float32((i - 1) / gridCols)
		col := float32((i - 1) % gridCols)

		cardRec := rl.NewRectangle(viewRec.X+2+col*(cardW+cardGap), curY+row*(cardH+cardGap), cardW, cardH)
		isActive := (activeCostume == i)
		hovered := rl.CheckCollisionPointRec(mousePos, cardRec)

		DrawCard(cardRec, hovered, isActive)

		numStr := fmt.Sprintf("%d", i)
		if i == 10 {
			numStr = "0"
		}

		numCol := ColSkyBlue
		if isActive {
			numCol = ColLime
		}
		ui.DrawTextBold(numStr, int32(cardRec.X)+int32(cardW/2)-int32(ui.MeasureTextBold(numStr, 17)/2), int32(cardRec.Y)+10, 17, numCol)
		ui.DrawText("Slot", int32(cardRec.X)+int32(cardW/2)-int32(ui.MeasureText("Slot", 11)/2), int32(cardRec.Y)+34, 11, ColTextMuted)

		if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			costumeMgr.SetCostume(i)
		}
	}

	curY += (cardH+cardGap)*2 + 22

	// Bounce on Costume Change Toggle Card
	bounceCard := rl.NewRectangle(viewRec.X+2, curY, viewRec.Width-16, 52)
	DrawCard(bounceCard, false, false)

	toggleRec := rl.NewRectangle(bounceCard.X+14, bounceCard.Y+12, bounceCard.Width-28, 28)
	newBounce := ui.DrawToggle(toggleRec, "Efeito de Pulo (Bounce) ao trocar de figurino", cfg.BounceOnCostume, mousePos)
	if newBounce != cfg.BounceOnCostume {
		cfg.BounceOnCostume = newBounce
		costumeMgr.BounceOnChange = newBounce
	}

	ui.EndScrollView(viewRec, contentH)
}

// -------------------------------------------------------------
// TAB 4: FÍSICA & ANIMAÇÃO
// -------------------------------------------------------------
func (ui *UIState) drawPhysicsTab(viewRec rl.Rectangle, cfg *config.Config, mousePos rl.Vector2) {
	contentH := float32(460)
	startY := ui.BeginScrollView(viewRec, contentH)
	curY := startY

	// Card 1: Flutuação / Respiração (Bobbing)
	bobCard := rl.NewRectangle(viewRec.X+2, curY, viewRec.Width-16, 80)
	DrawCard(bobCard, false, false)
	ui.DrawTextBold("Flutuação / Respiração (Idle Bobbing)", int32(bobCard.X)+14, int32(bobCard.Y)+12, 13, ColSkyBlue)
	ui.DrawBadge(bobCard.X+bobCard.Width-70, bobCard.Y+10, fmt.Sprintf("%.2fx", cfg.BobbingIntensity), ColIconBoxBg, ColYellow)
	bobTrack := rl.NewRectangle(bobCard.X+14, bobCard.Y+46, bobCard.Width-28, 12)
	cfg.BobbingIntensity = ui.DrawSliderControl(bobTrack, cfg.BobbingIntensity, 0.0, 2.0, mousePos, ColSkyBlue)
	curY += 92

	// Card 2: Inércia / Wobble
	wobCard := rl.NewRectangle(viewRec.X+2, curY, viewRec.Width-16, 80)
	DrawCard(wobCard, false, false)
	ui.DrawTextBold("Inércia e Amortecimento (Wobble)", int32(wobCard.X)+14, int32(wobCard.Y)+12, 13, ColSkyBlue)
	ui.DrawBadge(wobCard.X+wobCard.Width-70, wobCard.Y+10, fmt.Sprintf("%.2fx", cfg.WobbleIntensity), ColIconBoxBg, ColYellow)
	wobTrack := rl.NewRectangle(wobCard.X+14, wobCard.Y+46, wobCard.Width-28, 12)
	cfg.WobbleIntensity = ui.DrawSliderControl(wobTrack, cfg.WobbleIntensity, 0.0, 2.0, mousePos, ColSkyBlue)
	curY += 92

	// Card 3: Salto ao Falar (Bounce & Gravidade)
	bounceCard := rl.NewRectangle(viewRec.X+2, curY, viewRec.Width-16, 135)
	DrawCard(bounceCard, false, false)
	ui.DrawTextBold("Salto ao Falar (Bounce)", int32(bounceCard.X)+14, int32(bounceCard.Y)+12, 13, ColSkyBlue)
	ui.DrawText(fmt.Sprintf("Força: %.0f", cfg.BounceStrength), int32(bounceCard.X)+14, int32(bounceCard.Y)+34, 11.5, ColTextMuted)
	bTrack := rl.NewRectangle(bounceCard.X+14, bounceCard.Y+52, bounceCard.Width-28, 10)
	cfg.BounceStrength = ui.DrawSliderControl(bTrack, cfg.BounceStrength, 50.0, 600.0, mousePos, ColSkyBlue)

	ui.DrawText(fmt.Sprintf("Gravidade: %.0f", cfg.BounceGravity), int32(bounceCard.X)+14, int32(bounceCard.Y)+80, 11.5, ColTextMuted)
	gTrack := rl.NewRectangle(bounceCard.X+14, bounceCard.Y+98, bounceCard.Width-28, 10)
	cfg.BounceGravity = ui.DrawSliderControl(gTrack, cfg.BounceGravity, 200.0, 2000.0, mousePos, ColSkyBlue)
	curY += 147

	// Card 4: Piscar dos Olhos (Blink)
	blinkCard := rl.NewRectangle(viewRec.X+2, curY, viewRec.Width-16, 80)
	DrawCard(blinkCard, false, false)
	ui.DrawTextBold("Duração do Piscar (Blink)", int32(blinkCard.X)+14, int32(blinkCard.Y)+12, 13, ColSkyBlue)
	ui.DrawBadge(blinkCard.X+blinkCard.Width-70, blinkCard.Y+10, fmt.Sprintf("%.2fx", cfg.BlinkSpeed), ColIconBoxBg, ColYellow)
	bsTrack := rl.NewRectangle(blinkCard.X+14, blinkCard.Y+46, blinkCard.Width-28, 12)
	cfg.BlinkSpeed = ui.DrawSliderControl(bsTrack, cfg.BlinkSpeed, 0.2, 3.0, mousePos, ColSkyBlue)

	ui.EndScrollView(viewRec, contentH)
}

// -------------------------------------------------------------
// TAB 5: TECLAS & ATALHOS
// -------------------------------------------------------------
func (ui *UIState) drawKeybindsTab(viewRec rl.Rectangle, cfg *config.Config, mousePos rl.Vector2) {
	// Process rebind keypress
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
				case "increaseSens":
					cfg.Keybinds.IncreaseSens = key
				case "decreaseSens":
					cfg.Keybinds.DecreaseSens = key
				case "testBounce":
					cfg.Keybinds.TestBounce = key
				case "resetAvatar":
					cfg.Keybinds.ResetAvatar = key
				}
				ui.RebindingAction = ""
			}
		}
	}

	items := []struct {
		label    string
		actionID string
		current  int32
	}{
		{"Menu / Configurações", "toggleMenu", cfg.Keybinds.ToggleMenu},
		{"Editor Visual de Avatar", "toggleEditor", cfg.Keybinds.ToggleEditor},
		{"Painel de Depuração (HUD)", "toggleHUD", cfg.Keybinds.ToggleHUD},
		{"Modo Click-Through", "toggleClickThrough", cfg.Keybinds.ToggleClickThrough},
		{"Janela Sem Bordas (OBS)", "toggleBorderless", cfg.Keybinds.ToggleBorderless},
		{"Sempre no Topo (Topmost)", "toggleAlwaysOnTop", cfg.Keybinds.ToggleAlwaysOnTop},
		{"Aumentar Sensibilidade", "increaseSens", cfg.Keybinds.IncreaseSens},
		{"Diminuir Sensibilidade", "decreaseSens", cfg.Keybinds.DecreaseSens},
		{"Pulo / Teste de Fala", "testBounce", cfg.Keybinds.TestBounce},
		{"Resetar Posição e Escala", "resetAvatar", cfg.Keybinds.ResetAvatar},
	}

	contentH := float32(44) + float32(len(items))*50 + 64
	startY := ui.BeginScrollView(viewRec, contentH)
	curY := startY

	ui.DrawTextBold("Atalhos de Teclado", int32(viewRec.X)+4, int32(curY), 14, ColSkyBlue)
	ui.DrawText("Clique em qualquer botão e pressione a nova tecla", int32(viewRec.X)+4, int32(curY)+22, 11.5, ColTextMuted)
	curY += 44

	for _, it := range items {
		cardRec := rl.NewRectangle(viewRec.X+2, curY, viewRec.Width-16, 42)
		hovered := rl.CheckCollisionPointRec(mousePos, cardRec)
		isListening := (ui.RebindingAction == it.actionID)

		DrawCard(cardRec, hovered, isListening)
		ui.DrawText(it.label, int32(cardRec.X)+14, int32(cardRec.Y)+13, 12.5, ColTextTitle)

		// Key Badge with exact width measurement so it never clips or feels tight
		keyName := config.GetKeyName(it.current)
		badgeBg := ColIconBoxBg
		badgeTextCol := ColWhite
		if isListening {
			keyName = "Pressione tecla..."
			badgeBg = ColWine
			badgeTextCol = ColYellow
		} else if hovered {
			badgeBg = ColCardHover
			badgeTextCol = ColSkyBlue
		}

		badgeW := ui.MeasureTextBold(keyName, 11) + 20
		badgeX := cardRec.X + cardRec.Width - badgeW - 12
		ui.DrawBadge(badgeX, cardRec.Y+9, keyName, badgeBg, badgeTextCol)

		if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			if isListening {
				ui.RebindingAction = ""
			} else {
				ui.RebindingAction = it.actionID
			}
		}

		curY += 48
	}

	curY += 8

	// Reset Defaults Card
	resetRec := rl.NewRectangle(viewRec.X+2, curY, viewRec.Width-16, 44)
	hoveredReset := rl.CheckCollisionPointRec(mousePos, resetRec)
	DrawCard(resetRec, hoveredReset, false)
	ui.DrawIconBadge(resetRec.X+8, resetRec.Y+6, 32, IconRestore, ColSkyBlue, ColIconBoxBg)
	ui.DrawTextBold("Restaurar Atalhos Padrão", int32(resetRec.X)+48, int32(resetRec.Y)+13, 12.5, ColTextTitle)

	if hoveredReset && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		cfg.Keybinds = config.DefaultKeybinds()
		ui.RebindingAction = ""
	}

	ui.EndScrollView(viewRec, contentH)
}

// -------------------------------------------------------------
// TAB 6: OBS STUDIO & OVERLAY
// -------------------------------------------------------------
func (ui *UIState) drawOBSTab(viewRec rl.Rectangle, cfg *config.Config, wm *window.WindowManager, scale *float32, mousePos rl.Vector2) {
	contentH := float32(520)
	startY := ui.BeginScrollView(viewRec, contentH)
	curY := startY

	// Card 1: OBS Quick Overlay Preset
	presetCard := rl.NewRectangle(viewRec.X+2, curY, viewRec.Width-16, 56)
	hoveredPreset := rl.CheckCollisionPointRec(mousePos, presetCard)
	DrawCard(presetCard, hoveredPreset, false)
	ui.DrawIconBadge(presetCard.X+8, presetCard.Y+8, 40, IconOBS, ColLime, ColIconBoxBg)
	ui.DrawTextBold("Ativar Modo Overlay para OBS", int32(presetCard.X)+56, int32(presetCard.Y)+12, 13.5, ColTextTitle)
	ui.DrawText("Fundo transparente + Sem bordas + Sempre no topo", int32(presetCard.X)+56, int32(presetCard.Y)+32, 11.5, ColLime)

	if hoveredPreset && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		if !wm.IsBorderless() {
			wm.ToggleBorderless()
		}
		if !wm.IsAlwaysOnTop() {
			wm.ToggleAlwaysOnTop()
		}
		cfg.BackgroundColor = [4]uint8{0, 0, 0, 0}
		ui.IsOpen = false
	}
	curY += 68

	// Card 2: Cor de Fundo (Chroma Key & Transparência) - 2x2 Clean Grid
	bgCard := rl.NewRectangle(viewRec.X+2, curY, viewRec.Width-16, 126)
	DrawCard(bgCard, false, false)
	ui.DrawTextBold("Cor de Fundo da Janela", int32(bgCard.X)+14, int32(bgCard.Y)+12, 13.5, ColSkyBlue)

	colorPresets := []struct {
		name string
		rgba [4]uint8
		col  rl.Color
	}{
		{"Transparente (Alpha)", [4]uint8{0, 0, 0, 0}, rl.NewColor(30, 42, 70, 255)},
		{"Verde Chroma Key", [4]uint8{0, 255, 0, 255}, rl.NewColor(0, 228, 54, 255)},
		{"Azul Chroma Key", [4]uint8{0, 0, 255, 255}, rl.NewColor(41, 173, 255, 255)},
		{"Magenta Key", [4]uint8{255, 0, 255, 255}, rl.NewColor(255, 0, 77, 255)},
	}

	chipW := (bgCard.Width - 28 - 10) / 2
	chipH := float32(36)
	for i, c := range colorPresets {
		row := float32(i / 2)
		col := float32(i % 2)
		chipRec := rl.NewRectangle(bgCard.X+14+col*(chipW+10), bgCard.Y+40+row*(chipH+8), chipW, chipH)
		isCur := (cfg.BackgroundColor == c.rgba)
		hoveredChip := rl.CheckCollisionPointRec(mousePos, chipRec)

		DrawCard(chipRec, hoveredChip, isCur)
		rl.DrawRectangleRounded(rl.NewRectangle(chipRec.X+8, chipRec.Y+9, 18, 18), 0.3, 4, c.col)
		rl.DrawRectangleRoundedLines(rl.NewRectangle(chipRec.X+8, chipRec.Y+9, 18, 18), 0.3, 4, ColPanelBorder)
		ui.DrawTextBold(c.name, int32(chipRec.X)+34, int32(chipRec.Y)+10, 11.5, ColTextBody)

		if hoveredChip && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			cfg.BackgroundColor = c.rgba
		}
	}
	curY += 138

	// Card 3: Zoom / Escala
	zoomCard := rl.NewRectangle(viewRec.X+2, curY, viewRec.Width-16, 80)
	DrawCard(zoomCard, false, false)
	ui.DrawTextBold("Zoom e Escala do Avatar", int32(zoomCard.X)+14, int32(zoomCard.Y)+12, 13, ColSkyBlue)
	ui.DrawBadge(zoomCard.X+zoomCard.Width-70, zoomCard.Y+10, fmt.Sprintf("%.2fx", *scale), ColIconBoxBg, ColYellow)
	zTrack := rl.NewRectangle(zoomCard.X+14, zoomCard.Y+46, zoomCard.Width-28, 12)
	*scale = ui.DrawSliderControl(zTrack, *scale, 0.2, 4.0, mousePos, ColSkyBlue)
	curY += 92

	// Card 4: OBS Step-by-Step Guide
	guideCard := rl.NewRectangle(viewRec.X+2, curY, viewRec.Width-16, 135)
	DrawCard(guideCard, false, false)
	ui.DrawTextBold("Passo a Passo para o OBS Studio:", int32(guideCard.X)+14, int32(guideCard.Y)+14, 13.5, ColSkyBlue)
	ui.DrawText("1. Adicione uma fonte 'Captura de Janela' no OBS", int32(guideCard.X)+14, int32(guideCard.Y)+38, 11.5, ColTextBody)
	ui.DrawText("2. Selecione a janela 'PNGTuber Lite'", int32(guideCard.X)+14, int32(guideCard.Y)+58, 11.5, ColTextBody)
	ui.DrawText("3. Marque 'Permitir Transparência' (Allow Transparency)", int32(guideCard.X)+14, int32(guideCard.Y)+78, 11.5, ColLime)
	ui.DrawText("✓ Pronto! O avatar fica recortado sem precisar de tela verde!", int32(guideCard.X)+14, int32(guideCard.Y)+102, 12, ColYellow)

	ui.EndScrollView(viewRec, contentH)
}

// -------------------------------------------------------------
// UPDATE MODAL
// -------------------------------------------------------------
func (ui *UIState) drawUpdateModal(mousePos rl.Vector2, screenW, screenH float32) {
	upState := updater.GetUpdateState()
	if !upState.ShowPopup || upState.Latest == nil {
		return
	}

	// Backdrop
	rl.DrawRectangle(0, 0, int32(screenW), int32(screenH), rl.NewColor(0, 0, 0, 195))

	// Modal Box
	modalW := float32(560)
	modalH := float32(380)
	modalX := (screenW - modalW) / 2
	modalY := (screenH - modalH) / 2
	modalRec := rl.NewRectangle(modalX, modalY, modalW, modalH)

	rl.DrawRectangleRounded(modalRec, 0.05, 6, ColPanelBg)
	rl.DrawRectangleRoundedLines(modalRec, 0.05, 6, ColPanelBorder)

	tag := upState.Latest.TagName
	titleText := "Nova Versão Disponível!"
	if upState.Latest.IsHotfix {
		titleText = "Hotfix Importante Disponível!"
	}

	// Header
	ui.DrawIconBadge(modalX+18, modalY+16, 38, IconUpdate, ColLime, ColIconBoxBg)
	ui.DrawTextBold(titleText, int32(modalX)+66, int32(modalY)+18, 16, ColTextTitle)
	ui.DrawText(fmt.Sprintf("Versão %s do PNGTuber Lite", tag), int32(modalX)+66, int32(modalY)+38, 12, ColTextMuted)

	// Close 'X' button
	closeBtnRec := rl.NewRectangle(modalX+modalW-40, modalY+18, 26, 26)
	if rl.CheckCollisionPointRec(mousePos, closeBtnRec) {
		rl.DrawRectangleRounded(closeBtnRec, 0.35, 4, ColCardHover)
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			upState.ShowPopup = false
			upState.Dismissed = true
		}
	}
	GlobalIcons.DrawIcon(IconClose, closeBtnRec.X+5, closeBtnRec.Y+5, 16, ColLightGray)

	// Changelog / Release Notes Box
	boxH := float32(148)
	boxRec := rl.NewRectangle(modalX+18, modalY+68, modalW-36, boxH)
	DrawCard(boxRec, false, false)

	ui.DrawTextBold("Destaques e Novidades:", int32(boxRec.X)+14, int32(boxRec.Y)+12, 13.5, ColSkyBlue)
	lines := upState.Latest.GetCleanSummary()
	lineY := int32(boxRec.Y) + 36
	for _, l := range lines {
		ui.DrawText(l, int32(boxRec.X)+14, lineY, 11.5, ColTextBody)
		lineY += 22
	}

	curY := modalY + 232

	if upState.IsUpdating {
		barW := modalW - 36
		barRec := rl.NewRectangle(modalX+18, curY+16, barW, 26)
		rl.DrawRectangleRounded(barRec, 0.5, 4, ColScrollTrack)
		fillW := barW * upState.Progress
		if fillW > barW {
			fillW = barW
		}
		rl.DrawRectangleRounded(rl.NewRectangle(barRec.X, barRec.Y, fillW, barRec.Height), 0.5, 4, ColLime)

		pctText := fmt.Sprintf("Baixando atualização... %d%%", int(upState.Progress*100))
		ui.DrawTextBold(pctText, int32(modalX)+28, int32(curY)+21, 12, ColWhite)
		return
	}

	if upState.Success {
		succCard := rl.NewRectangle(modalX+18, curY+10, modalW-36, 56)
		DrawCard(succCard, false, false)
		rl.DrawRectangleRoundedLines(succCard, 0.16, 4, ColLime)
		ui.DrawIconBadge(succCard.X+10, succCard.Y+9, 38, IconUpdate, ColLime, ColIconBoxBg)
		ui.DrawTextBold("✓ Atualização instalada com sucesso!", int32(succCard.X)+56, int32(succCard.Y)+11, 13.5, ColLime)
		ui.DrawText(fmt.Sprintf("Reiniciando automaticamente em %.1fs...", upState.RestartCountdown), int32(succCard.X)+56, int32(succCard.Y)+31, 12, ColYellow)
		return
	}

	if upState.ErrorMessage != "" {
		errCard := rl.NewRectangle(modalX+18, curY+10, modalW-36, 40)
		DrawCard(errCard, false, false)
		rl.DrawRectangleRoundedLines(errCard, 0.16, 4, ColRed)
		ui.DrawTextBold(fmt.Sprintf("Erro: %s", upState.ErrorMessage), int32(errCard.X)+14, int32(errCard.Y)+11, 11.5, ColRed)
		curY += 48
	}

	// Action buttons with generous breathing room
	btnW := (modalW - 48) / 2
	btnH := float32(44)

	// Button 1: Atualizar Agora
	nowRec := rl.NewRectangle(modalX+18, curY+38, btnW, btnH)
	hoverNow := rl.CheckCollisionPointRec(mousePos, nowRec)
	nowBg := rl.NewColor(28, 105, 58, 255)
	if hoverNow {
		nowBg = rl.NewColor(38, 140, 76, 255)
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
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
					upState.RestartCountdown = 2.0
				}
			}()
		}
	}
	rl.DrawRectangleRounded(nowRec, 0.45, 6, nowBg)
	rl.DrawRectangleRoundedLines(nowRec, 0.45, 6, ColLime)
	GlobalIcons.DrawIcon(IconUpdate, nowRec.X+24, nowRec.Y+13, 18, ColWhite)
	ui.DrawTextBold("Atualizar Agora", int32(nowRec.X)+52, int32(nowRec.Y)+13, 13, ColWhite)

	// Button 2: Lembrar Depois
	laterRec := rl.NewRectangle(modalX+30+btnW, curY+38, btnW, btnH)
	hoverLater := rl.CheckCollisionPointRec(mousePos, laterRec)
	laterBg := ColCardBg
	if hoverLater {
		laterBg = ColCardHover
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			upState.ShowPopup = false
			upState.Dismissed = true
		}
	}
	rl.DrawRectangleRounded(laterRec, 0.45, 6, laterBg)
	rl.DrawRectangleRoundedLines(laterRec, 0.45, 6, ColPanelBorder)
	GlobalIcons.DrawIcon(IconRestore, laterRec.X+24, laterRec.Y+13, 18, ColLightGray)
	ui.DrawTextBold("Lembrar Depois", int32(laterRec.X)+52, int32(laterRec.Y)+13, 13, ColTextBody)
}

// -------------------------------------------------------------
// CLOSE CONFIRMATION MODAL
// -------------------------------------------------------------
func (ui *UIState) drawCloseConfirmModal(mousePos rl.Vector2, screenW, screenH float32) {
	if !ui.ShowCloseModal {
		return
	}

	// Backdrop
	rl.DrawRectangle(0, 0, int32(screenW), int32(screenH), rl.NewColor(0, 0, 0, 195))

	// Modal Box
	modalW := float32(560)
	modalH := float32(290)
	modalX := (screenW - modalW) / 2
	modalY := (screenH - modalH) / 2
	modalRec := rl.NewRectangle(modalX, modalY, modalW, modalH)

	rl.DrawRectangleRounded(modalRec, 0.06, 6, ColPanelBg)
	rl.DrawRectangleRoundedLines(modalRec, 0.06, 6, ColPanelBorder)

	// Header
	ui.DrawIconBadge(modalX+18, modalY+16, 38, IconClose, ColSkyBlue, ColIconBoxBg)
	ui.DrawTextBold("Fechar ou Manter em Segundo Plano?", int32(modalX)+66, int32(modalY)+18, 16, ColTextTitle)
	ui.DrawText("Escolha como deseja prosseguir com o PNGTuber Lite", int32(modalX)+66, int32(modalY)+38, 12, ColTextMuted)

	// Close 'X' button
	closeBtnRec := rl.NewRectangle(modalX+modalW-40, modalY+18, 26, 26)
	if rl.CheckCollisionPointRec(mousePos, closeBtnRec) {
		rl.DrawRectangleRounded(closeBtnRec, 0.35, 4, ColCardHover)
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			ui.ShowCloseModal = false
		}
	}
	GlobalIcons.DrawIcon(IconClose, closeBtnRec.X+5, closeBtnRec.Y+5, 16, ColLightGray)

	// Info / Explanation Card
	infoBox := rl.NewRectangle(modalX+18, modalY+68, modalW-36, 120)
	DrawCard(infoBox, false, false)

	ui.DrawText("• Minimizar no Tray: o avatar continuará ativo capturando áudio.", int32(infoBox.X)+14, int32(infoBox.Y)+14, 12, ColTextBody)
	ui.DrawText("• Fechar Aplicativo: encerra totalmente o processo do programa.", int32(infoBox.X)+14, int32(infoBox.Y)+38, 12, ColTextBody)
	ui.DrawText("✓ Dica: Clique duas vezes no ícone perto do relógio para reabrir.", int32(infoBox.X)+14, int32(infoBox.Y)+64, 11.5, ColLime)
	ui.DrawText("A janela pode ser restaurada ou ocultada a qualquer momento.", int32(infoBox.X)+14, int32(infoBox.Y)+86, 11.5, ColTextMuted)

	// Action buttons in bottom bar with generous breathing room
	btnH := float32(44)
	curY := modalY + 226

	// 1. Minimizar no Tray
	minBtn := rl.NewRectangle(modalX+18, curY, 204, btnH)
	minHover := rl.CheckCollisionPointRec(mousePos, minBtn)
	minBg := rl.NewColor(28, 68, 125, 255)
	if minHover {
		minBg = rl.NewColor(38, 92, 165, 255)
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			ui.ShowCloseModal = false
			if ui.OnRequestMinimizeToTray != nil {
				ui.OnRequestMinimizeToTray()
			}
		}
	}
	rl.DrawRectangleRounded(minBtn, 0.45, 6, minBg)
	rl.DrawRectangleRoundedLines(minBtn, 0.45, 6, ColSkyBlue)
	GlobalIcons.DrawIcon(IconEnable, minBtn.X+16, minBtn.Y+13, 18, ColWhite)
	ui.DrawTextBold("Minimizar no Tray", int32(minBtn.X)+44, int32(minBtn.Y)+13, 12.5, ColWhite)

	// 2. Fechar Aplicativo
	quitBtn := rl.NewRectangle(modalX+232, curY, 178, btnH)
	quitHover := rl.CheckCollisionPointRec(mousePos, quitBtn)
	quitBg := rl.NewColor(135, 32, 45, 255)
	if quitHover {
		quitBg = rl.NewColor(175, 42, 58, 255)
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			ui.ShowCloseModal = false
			if ui.OnRequestClose != nil {
				ui.OnRequestClose()
			}
		}
	}
	rl.DrawRectangleRounded(quitBtn, 0.45, 6, quitBg)
	rl.DrawRectangleRoundedLines(quitBtn, 0.45, 6, ColRed)
	GlobalIcons.DrawIcon(IconClose, quitBtn.X+16, quitBtn.Y+13, 18, ColWhite)
	ui.DrawTextBold("Fechar Aplicativo", int32(quitBtn.X)+44, int32(quitBtn.Y)+13, 12.5, ColWhite)

	// 3. Cancelar
	canBtn := rl.NewRectangle(modalX+420, curY, 122, btnH)
	canHover := rl.CheckCollisionPointRec(mousePos, canBtn)
	canBg := ColCardBg
	if canHover {
		canBg = ColCardHover
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			ui.ShowCloseModal = false
		}
	}
	rl.DrawRectangleRounded(canBtn, 0.45, 6, canBg)
	rl.DrawRectangleRoundedLines(canBtn, 0.45, 6, ColPanelBorder)
	ui.DrawTextBold("Cancelar", int32(canBtn.X)+30, int32(canBtn.Y)+13, 12.5, ColTextBody)
}
