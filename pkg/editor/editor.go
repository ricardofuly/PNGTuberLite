package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"pngtuber-lite/pkg/model"
	"pngtuber-lite/pkg/render"
	"pngtuber-lite/pkg/ui"
)

type EditorTab int

const (
	TabLayerGeneral EditorTab = iota
	TabLayerVisibility
	TabLayerPhysics
	TabLayerSprite
)

// EditorState manages the avatar visual editor.
type EditorState struct {
	IsOpen            bool
	Avatar            *model.Avatar
	SelectedLayerID   int64
	AvatarFilePath    string
	StatusMessage     string
	StatusTimer       float32
	ActiveTab         EditorTab
	TextureCache      *render.TextureCache
	UI                *ui.UIState

	// Gizmo interaction
	IsDraggingPos     bool
	IsDraggingPivot   bool
	DragStartMouse    rl.Vector2
	DragStartLayerPos model.Vector2
	DragStartPivot    model.Vector2

	// Callbacks
	OnAvatarModified  func()
}

// NewEditorState creates an editor instance.
func NewEditorState(texCache *render.TextureCache, uiState *ui.UIState) *EditorState {
	return &EditorState{
		IsOpen:          false,
		SelectedLayerID: 0,
		AvatarFilePath:  "assets/samples/novo_avatar.save",
		ActiveTab:       TabLayerGeneral,
		TextureCache:    texCache,
		UI:              uiState,
	}
}

// SetAvatar binds an avatar to the editor.
func (e *EditorState) SetAvatar(avatar *model.Avatar, filePath string) {
	e.Avatar = avatar
	if filePath != "" {
		e.AvatarFilePath = filePath
	}
	if avatar != nil && len(avatar.Layers) > 0 {
		if _, exists := avatar.Layers[e.SelectedLayerID]; !exists {
			for id := range avatar.Layers {
				e.SelectedLayerID = id
				break
			}
		}
	}
}

// NewBlankAvatar initializes an empty avatar with standard body and head template layers.
func (e *EditorState) NewBlankAvatar() {
	e.Avatar = model.NewAvatar()
	e.SelectedLayerID = 0
	e.AvatarFilePath = "assets/samples/novo_avatar.save"
	e.SetStatus("Novo avatar criado. Arraste arquivos .png para adicionar camadas!")
	if e.OnAvatarModified != nil {
		e.OnAvatarModified()
	}
}

// SetStatus displays a brief status message on the editor.
func (e *EditorState) SetStatus(msg string) {
	e.StatusMessage = msg
	e.StatusTimer = 4.0
}

// AddLayerFromPNG imports a PNG file and creates a new layer.
func (e *EditorState) AddLayerFromPNG(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("falha ao ler imagem: %w", err)
	}

	w, h := model.ExtractPNGDimensions(data)
	if w == 0 || h == 0 {
		return fmt.Errorf("arquivo PNG inválido ou sem cabeçalho IHDR")
	}

	if e.Avatar == nil {
		e.Avatar = model.NewAvatar()
	}

	// Generate unique ID
	id := time.Now().UnixNano() / 1000000

	layer := model.NewDefaultLayer(id)
	layer.Path = filepath.Base(filePath)
	layer.ImageData = data
	layer.ImageWidth = w
	layer.ImageHeight = h
	layer.ZIndex = 0

	// Set parent to currently selected layer if any
	if e.SelectedLayerID != 0 && e.Avatar.GetLayer(e.SelectedLayerID) != nil {
		pID := e.SelectedLayerID
		layer.ParentID = &pID
	}

	e.Avatar.AddLayer(layer)
	e.Avatar.BuildHierarchy()
	e.SelectedLayerID = id

	// Reload textures in cache
	if err := e.TextureCache.LoadAvatarTextures(e.Avatar); err != nil {
		return err
	}

	e.SetStatus(fmt.Sprintf("Camada %q adicionada com sucesso!", layer.Path))
	if e.OnAvatarModified != nil {
		e.OnAvatarModified()
	}
	return nil
}

// RemoveSelectedLayer deletes the active layer.
func (e *EditorState) RemoveSelectedLayer() {
	if e.Avatar == nil || e.SelectedLayerID == 0 {
		return
	}

	layer := e.Avatar.GetLayer(e.SelectedLayerID)
	if layer == nil {
		return
	}

	name := layer.Path
	e.Avatar.RemoveLayer(e.SelectedLayerID)
	e.Avatar.BuildHierarchy()

	// Select next available layer
	e.SelectedLayerID = 0
	for id := range e.Avatar.Layers {
		e.SelectedLayerID = id
		break
	}

	_ = e.TextureCache.LoadAvatarTextures(e.Avatar)
	e.SetStatus(fmt.Sprintf("Camada %q removida.", name))
	if e.OnAvatarModified != nil {
		e.OnAvatarModified()
	}
}

// DuplicateSelectedLayer clones the active layer.
func (e *EditorState) DuplicateSelectedLayer() {
	if e.Avatar == nil || e.SelectedLayerID == 0 {
		return
	}
	orig := e.Avatar.GetLayer(e.SelectedLayerID)
	if orig == nil {
		return
	}

	id := time.Now().UnixNano() / 1000000
	clone := *orig
	clone.Identification = id
	clone.Path = fmt.Sprintf("copia_%s", orig.Path)
	clone.Pos.X += 20
	clone.Pos.Y += 20

	e.Avatar.AddLayer(&clone)
	e.Avatar.BuildHierarchy()
	e.SelectedLayerID = id

	_ = e.TextureCache.LoadAvatarTextures(e.Avatar)
	e.SetStatus("Camada duplicada!")
	if e.OnAvatarModified != nil {
		e.OnAvatarModified()
	}
}

// SaveCurrentAvatar writes the avatar to disk.
func (e *EditorState) SaveCurrentAvatar() error {
	if e.Avatar == nil || len(e.Avatar.Layers) == 0 {
		return fmt.Errorf("avatar vazio")
	}

	if err := model.SaveAvatarToFile(e.Avatar, e.AvatarFilePath); err != nil {
		e.SetStatus(fmt.Sprintf("Erro ao salvar: %v", err))
		return err
	}

	e.SetStatus(fmt.Sprintf("Avatar salvo com sucesso em: %s", e.AvatarFilePath))
	if e.UI != nil {
		e.UI.ScanAvatars()
	}
	return nil
}

// HandleFileDrops processes dropped files (.png for new layers, .save to open avatar).
func (e *EditorState) HandleFileDrops() {
	if !rl.IsFileDropped() {
		return
	}

	droppedFiles := rl.LoadDroppedFiles()
	defer rl.UnloadDroppedFiles()

	for _, file := range droppedFiles {
		ext := strings.ToLower(filepath.Ext(file))
		if ext == ".png" {
			_ = e.AddLayerFromPNG(file)
		} else if ext == ".save" {
			av, err := model.ParseSaveFile(file)
			if err == nil {
				e.SetAvatar(av, file)
				_ = e.TextureCache.LoadAvatarTextures(av)
				e.SetStatus(fmt.Sprintf("Avatar carregado: %s", filepath.Base(file)))
				if e.OnAvatarModified != nil {
					e.OnAvatarModified()
				}
			}
		}
	}
}

// Update advances the editor status timer and handles interactions.
func (e *EditorState) Update(dt float32) {
	if e.StatusTimer > 0 {
		e.StatusTimer -= dt
		if e.StatusTimer <= 0 {
			e.StatusMessage = ""
		}
	}

	e.HandleFileDrops()
}

// Draw renders the visual editor overlay (left layer tree, right properties panel, and central gizmo).
func (e *EditorState) Draw(scale float32, origin rl.Vector2) {
	if !e.IsOpen {
		return
	}

	screenW := int32(rl.GetScreenWidth())
	screenH := int32(rl.GetScreenHeight())
	mousePos := rl.GetMousePosition()

	// 1. Top Header Bar
	headerRec := rl.NewRectangle(0, 0, float32(screenW), 44)
	rl.DrawRectangleRec(headerRec, rl.NewColor(15, 18, 28, 250))
	rl.DrawLine(0, 44, screenW, 44, rl.NewColor(45, 60, 95, 255))

	e.UI.DrawText("✏️ EDITOR DE AVATAR (PNGTuber Lite)", 16, 12, 16, rl.SkyBlue)

	// Save Button
	saveBtnRec := rl.NewRectangle(float32(screenW)-250, 7, 110, 30)
	saveHovered := rl.CheckCollisionPointRec(mousePos, saveBtnRec)
	saveBg := rl.NewColor(35, 110, 65, 255)
	if saveHovered {
		saveBg = rl.NewColor(45, 140, 80, 255)
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			_ = e.SaveCurrentAvatar()
		}
	}
	rl.DrawRectangleRounded(saveBtnRec, 0.25, 4, saveBg)
	e.UI.DrawText("💾 SALVAR", int32(saveBtnRec.X)+14, int32(saveBtnRec.Y)+7, 14, rl.RayWhite)

	// Close Editor Button
	closeBtnRec := rl.NewRectangle(float32(screenW)-125, 7, 110, 30)
	closeHovered := rl.CheckCollisionPointRec(mousePos, closeBtnRec)
	closeBg := rl.NewColor(120, 35, 45, 255)
	if closeHovered {
		closeBg = rl.NewColor(150, 45, 60, 255)
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			e.IsOpen = false
		}
	}
	rl.DrawRectangleRounded(closeBtnRec, 0.25, 4, closeBg)
	e.UI.DrawText("✕ FECHAR", int32(closeBtnRec.X)+18, int32(closeBtnRec.Y)+7, 14, rl.RayWhite)

	// Status Notification Banner
	if e.StatusMessage != "" {
		bannerRec := rl.NewRectangle(float32(screenW/2)-200, 52, 400, 32)
		rl.DrawRectangleRounded(bannerRec, 0.3, 4, rl.NewColor(20, 30, 48, 235))
		rl.DrawRectangleRoundedLines(bannerRec, 0.3, 4, rl.SkyBlue)
		e.UI.DrawText(e.StatusMessage, int32(bannerRec.X)+15, int32(bannerRec.Y)+8, 13, rl.Yellow)
	}

	// 2. Left Sidebar: Layer Hierarchy Tree
	leftW := float32(280)
	leftRec := rl.NewRectangle(12, 54, leftW, float32(screenH-66))
	rl.DrawRectangleRounded(leftRec, 0.03, 6, rl.NewColor(16, 20, 30, 245))
	rl.DrawRectangleRoundedLines(leftRec, 0.03, 6, rl.NewColor(45, 60, 90, 255))

	e.UI.DrawText("Camadas do Avatar:", int32(leftRec.X)+14, int32(leftRec.Y)+12, 15, rl.SkyBlue)

	// Layer Action Buttons (+ PNG, + Novo, Duplicar, Deletar)
	btnW := (leftW - 36) / 2
	addPNGBtn := rl.NewRectangle(leftRec.X+12, leftRec.Y+36, btnW, 28)
	if rl.CheckCollisionPointRec(mousePos, addPNGBtn) {
		rl.DrawRectangleRounded(addPNGBtn, 0.2, 4, rl.NewColor(40, 85, 140, 255))
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			e.SetStatus("Arraste qualquer arquivo .png para a janela!")
		}
	} else {
		rl.DrawRectangleRounded(addPNGBtn, 0.2, 4, rl.NewColor(28, 55, 95, 255))
	}
	e.UI.DrawText("+ PNG", int32(addPNGBtn.X)+28, int32(addPNGBtn.Y)+6, 13, rl.RayWhite)

	newAvBtn := rl.NewRectangle(leftRec.X+12+btnW+12, leftRec.Y+36, btnW, 28)
	if rl.CheckCollisionPointRec(mousePos, newAvBtn) {
		rl.DrawRectangleRounded(newAvBtn, 0.2, 4, rl.NewColor(50, 65, 95, 255))
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			e.NewBlankAvatar()
		}
	} else {
		rl.DrawRectangleRounded(newAvBtn, 0.2, 4, rl.NewColor(32, 42, 65, 255))
	}
	e.UI.DrawText("+ Novo", int32(newAvBtn.X)+25, int32(newAvBtn.Y)+6, 13, rl.RayWhite)

	// Layers List Scroll / Stack
	layerY := int32(leftRec.Y) + 74
	if e.Avatar != nil {
		for _, layer := range e.Avatar.DrawOrder {
			isSelected := (layer.Identification == e.SelectedLayerID)
			itemRec := rl.NewRectangle(leftRec.X+12, float32(layerY), leftW-24, 34)
			hovered := rl.CheckCollisionPointRec(mousePos, itemRec)

			itemBg := rl.NewColor(26, 32, 46, 255)
			itemTextCol := rl.LightGray
			if isSelected {
				itemBg = rl.NewColor(38, 90, 160, 255)
				itemTextCol = rl.RayWhite
			} else if hovered {
				itemBg = rl.NewColor(38, 48, 70, 255)
			}

			if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
				e.SelectedLayerID = layer.Identification
			}

			rl.DrawRectangleRounded(itemRec, 0.2, 4, itemBg)

			prefix := "• "
			if layer.ParentID != nil && *layer.ParentID != 0 {
				prefix = "  └ "
			}

			displayName := filepath.Base(layer.Path)
			if len(displayName) > 18 {
				displayName = displayName[:18] + "..."
			}

			e.UI.DrawText(fmt.Sprintf("%s%s [Z:%d]", prefix, displayName, layer.ZIndex), int32(itemRec.X)+8, int32(itemRec.Y)+8, 13, itemTextCol)

			layerY += 38
			if layerY > int32(leftRec.Y+leftRec.Height)-80 {
				break
			}
		}
	}

	// Bottom action bar of left sidebar (Duplicate, Delete)
	dupBtn := rl.NewRectangle(leftRec.X+12, leftRec.Y+leftRec.Height-44, btnW, 32)
	if rl.CheckCollisionPointRec(mousePos, dupBtn) {
		rl.DrawRectangleRounded(dupBtn, 0.2, 4, rl.NewColor(45, 65, 95, 255))
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			e.DuplicateSelectedLayer()
		}
	} else {
		rl.DrawRectangleRounded(dupBtn, 0.2, 4, rl.NewColor(30, 42, 62, 255))
	}
	e.UI.DrawText("Duplicar", int32(dupBtn.X)+20, int32(dupBtn.Y)+8, 13, rl.RayWhite)

	delBtn := rl.NewRectangle(leftRec.X+12+btnW+12, leftRec.Y+leftRec.Height-44, btnW, 32)
	if rl.CheckCollisionPointRec(mousePos, delBtn) {
		rl.DrawRectangleRounded(delBtn, 0.2, 4, rl.NewColor(120, 35, 45, 255))
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			e.RemoveSelectedLayer()
		}
	} else {
		rl.DrawRectangleRounded(delBtn, 0.2, 4, rl.NewColor(80, 25, 35, 255))
	}
	e.UI.DrawText("Excluir", int32(delBtn.X)+26, int32(delBtn.Y)+8, 13, rl.RayWhite)

	// 3. Right Sidebar: Layer Properties & Settings
	rightW := float32(360)
	rightRec := rl.NewRectangle(float32(screenW)-rightW-12, 54, rightW, float32(screenH-66))
	rl.DrawRectangleRounded(rightRec, 0.03, 6, rl.NewColor(16, 20, 30, 245))
	rl.DrawRectangleRoundedLines(rightRec, 0.03, 6, rl.NewColor(45, 60, 90, 255))

	var curLayer *model.Layer
	if e.Avatar != nil {
		curLayer = e.Avatar.GetLayer(e.SelectedLayerID)
	}

	if curLayer == nil {
		e.UI.DrawText("Selecione ou adicione uma camada para editar", int32(rightRec.X)+20, int32(rightRec.Y)+40, 14, rl.Gray)
		return
	}

	// Tabs Header for Layer Properties
	tabs := []struct {
		tab  EditorTab
		name string
	}{
		{TabLayerGeneral, "Geral"},
		{TabLayerVisibility, "Visibilidade"},
		{TabLayerPhysics, "Física"},
		{TabLayerSprite, "Sprite"},
	}

	tW := (rightW - 24) / float32(len(tabs))
	for i, t := range tabs {
		tRec := rl.NewRectangle(rightRec.X+12+float32(i)*tW, rightRec.Y+12, tW-2, 28)
		isActive := e.ActiveTab == t.tab
		hovered := rl.CheckCollisionPointRec(mousePos, tRec)

		tBg := rl.NewColor(28, 34, 48, 255)
		tTextCol := rl.LightGray
		if isActive {
			tBg = rl.NewColor(45, 95, 165, 255)
			tTextCol = rl.RayWhite
		} else if hovered {
			tBg = rl.NewColor(38, 48, 70, 255)
		}

		if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			e.ActiveTab = t.tab
		}

		rl.DrawRectangleRounded(tRec, 0.2, 4, tBg)
		e.UI.DrawText(t.name, int32(tRec.X)+10, int32(tRec.Y)+6, 13, tTextCol)
	}

	propY := int32(rightRec.Y) + 52

	// Render Selected Tab Content
	switch e.ActiveTab {
	case TabLayerGeneral:
		e.drawGeneralTab(rightRec, propY, curLayer, mousePos)
	case TabLayerVisibility:
		e.drawVisibilityTab(rightRec, propY, curLayer, mousePos)
	case TabLayerPhysics:
		e.drawPhysicsTab(rightRec, propY, curLayer, mousePos)
	case TabLayerSprite:
		e.drawSpriteTab(rightRec, propY, curLayer, mousePos)
	}

	// 4. Central Gizmo & Bounds on Active Layer
	e.drawLayerGizmo(curLayer, scale, origin, mousePos)
}

func (e *EditorState) drawGeneralTab(rightRec rl.Rectangle, startY int32, layer *model.Layer, mousePos rl.Vector2) {
	e.UI.DrawText(fmt.Sprintf("Camada: %s", filepath.Base(layer.Path)), int32(rightRec.X)+16, startY, 14, rl.Yellow)

	y := startY + 28

	// Z-Index Controls
	e.UI.DrawText(fmt.Sprintf("Profundidade Z-Index: %d", layer.ZIndex), int32(rightRec.X)+16, y, 14, rl.SkyBlue)
	zMinus := rl.NewRectangle(rightRec.X+240, float32(y-2), 34, 24)
	zPlus := rl.NewRectangle(rightRec.X+280, float32(y-2), 34, 24)

	if rl.CheckCollisionPointRec(mousePos, zMinus) {
		rl.DrawRectangleRounded(zMinus, 0.2, 4, rl.NewColor(45, 60, 90, 255))
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			layer.ZIndex--
			e.Avatar.BuildHierarchy()
		}
	} else {
		rl.DrawRectangleRounded(zMinus, 0.2, 4, rl.NewColor(32, 40, 60, 255))
	}
	e.UI.DrawText("-", int32(zMinus.X)+12, int32(zMinus.Y)+3, 16, rl.RayWhite)

	if rl.CheckCollisionPointRec(mousePos, zPlus) {
		rl.DrawRectangleRounded(zPlus, 0.2, 4, rl.NewColor(45, 60, 90, 255))
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			layer.ZIndex++
			e.Avatar.BuildHierarchy()
		}
	} else {
		rl.DrawRectangleRounded(zPlus, 0.2, 4, rl.NewColor(32, 40, 60, 255))
	}
	e.UI.DrawText("+", int32(zPlus.X)+10, int32(zPlus.Y)+3, 16, rl.RayWhite)

	y += 36

	// Position X Slider
	e.UI.DrawText(fmt.Sprintf("Posição X: %.1f", layer.Pos.X), int32(rightRec.X)+16, y, 13, rl.LightGray)
	posXRec := rl.NewRectangle(rightRec.X+16, float32(y+18), rightRec.Width-32, 12)
	rl.DrawRectangleRounded(posXRec, 0.3, 4, rl.DarkGray)
	if rl.IsMouseButtonDown(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mousePos, rl.NewRectangle(posXRec.X-10, posXRec.Y-8, posXRec.Width+20, posXRec.Height+16)) {
		ratio := (mousePos.X - posXRec.X) / posXRec.Width
		layer.Pos.X = -300.0 + ratio*600.0
	}
	hX := posXRec.X + ((layer.Pos.X+300.0)/600.0)*posXRec.Width
	rl.DrawCircle(int32(hX), int32(posXRec.Y)+6, 7, rl.SkyBlue)

	y += 42

	// Position Y Slider
	e.UI.DrawText(fmt.Sprintf("Posição Y: %.1f", layer.Pos.Y), int32(rightRec.X)+16, y, 13, rl.LightGray)
	posYRec := rl.NewRectangle(rightRec.X+16, float32(y+18), rightRec.Width-32, 12)
	rl.DrawRectangleRounded(posYRec, 0.3, 4, rl.DarkGray)
	if rl.IsMouseButtonDown(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mousePos, rl.NewRectangle(posYRec.X-10, posYRec.Y-8, posYRec.Width+20, posYRec.Height+16)) {
		ratio := (mousePos.X - posYRec.X) / posYRec.Width
		layer.Pos.Y = -300.0 + ratio*600.0
	}
	hY := posYRec.X + ((layer.Pos.Y+300.0)/600.0)*posYRec.Width
	rl.DrawCircle(int32(hY), int32(posYRec.Y)+6, 7, rl.SkyBlue)

	y += 42

	// Pivot Offset X Slider
	e.UI.DrawText(fmt.Sprintf("Pivô / Offset X: %.1f", layer.Offset.X), int32(rightRec.X)+16, y, 13, rl.LightGray)
	offXRec := rl.NewRectangle(rightRec.X+16, float32(y+18), rightRec.Width-32, 12)
	rl.DrawRectangleRounded(offXRec, 0.3, 4, rl.DarkGray)
	if rl.IsMouseButtonDown(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mousePos, rl.NewRectangle(offXRec.X-10, offXRec.Y-8, offXRec.Width+20, offXRec.Height+16)) {
		ratio := (mousePos.X - offXRec.X) / offXRec.Width
		layer.Offset.X = -200.0 + ratio*400.0
	}
	hOffX := offXRec.X + ((layer.Offset.X+200.0)/400.0)*offXRec.Width
	rl.DrawCircle(int32(hOffX), int32(offXRec.Y)+6, 7, rl.SkyBlue)

	y += 42

	// Pivot Offset Y Slider
	e.UI.DrawText(fmt.Sprintf("Pivô / Offset Y: %.1f", layer.Offset.Y), int32(rightRec.X)+16, y, 13, rl.LightGray)
	offYRec := rl.NewRectangle(rightRec.X+16, float32(y+18), rightRec.Width-32, 12)
	rl.DrawRectangleRounded(offYRec, 0.3, 4, rl.DarkGray)
	if rl.IsMouseButtonDown(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mousePos, rl.NewRectangle(offYRec.X-10, offYRec.Y-8, offYRec.Width+20, offYRec.Height+16)) {
		ratio := (mousePos.X - offYRec.X) / offYRec.Width
		layer.Offset.Y = -200.0 + ratio*400.0
	}
	hOffY := offYRec.X + ((layer.Offset.Y+200.0)/400.0)*offYRec.Width
	rl.DrawCircle(int32(hOffY), int32(offYRec.Y)+6, 7, rl.SkyBlue)

	y += 45

	// Parent Node Selection
	e.UI.DrawText("Camada Pai (Parent):", int32(rightRec.X)+16, y, 14, rl.SkyBlue)
	parentName := "Nenhum (Raiz)"
	if layer.ParentID != nil && *layer.ParentID != 0 {
		if p := e.Avatar.GetLayer(*layer.ParentID); p != nil {
			parentName = filepath.Base(p.Path)
		}
	}
	e.UI.DrawText(fmt.Sprintf("Pai Atual: %s", parentName), int32(rightRec.X)+16, y+20, 13, rl.Yellow)

	// Cycle Parent Button
	pBtn := rl.NewRectangle(rightRec.X+16, float32(y+42), rightRec.Width-32, 28)
	if rl.CheckCollisionPointRec(mousePos, pBtn) {
		rl.DrawRectangleRounded(pBtn, 0.2, 4, rl.NewColor(45, 65, 95, 255))
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			e.cycleParentLayer(layer)
		}
	} else {
		rl.DrawRectangleRounded(pBtn, 0.2, 4, rl.NewColor(32, 44, 66, 255))
	}
	e.UI.DrawText("▶ Trocar Camada Pai", int32(pBtn.X)+90, int32(pBtn.Y)+6, 13, rl.RayWhite)
}

func (e *EditorState) cycleParentLayer(layer *model.Layer) {
	layerIDs := []int64{0} // 0 = root
	for _, l := range e.Avatar.DrawOrder {
		if l.Identification != layer.Identification {
			layerIDs = append(layerIDs, l.Identification)
		}
	}

	curIdx := 0
	if layer.ParentID != nil && *layer.ParentID != 0 {
		for i, id := range layerIDs {
			if id == *layer.ParentID {
				curIdx = i
				break
			}
		}
	}

	nextIdx := (curIdx + 1) % len(layerIDs)
	nextID := layerIDs[nextIdx]
	if nextID == 0 {
		layer.ParentID = nil
	} else {
		layer.ParentID = &nextID
	}
	e.Avatar.BuildHierarchy()
}

func (e *EditorState) drawVisibilityTab(rightRec rl.Rectangle, startY int32, layer *model.Layer, mousePos rl.Vector2) {
	e.UI.DrawText("Condições de Exibição da Camada:", int32(rightRec.X)+16, startY, 14, rl.SkyBlue)

	y := startY + 28

	// Blink Visibility (0=Sempre, 1=Olhos Abertos, 2=Piscando)
	e.UI.DrawText("Modo de Piscar (Blink):", int32(rightRec.X)+16, y, 13, rl.LightGray)
	y += 20

	blinkOpts := []struct {
		val  int
		name string
	}{
		{0, "Sempre"},
		{1, "Olhos Abertos"},
		{2, "Piscando"},
	}

	bW := (rightRec.Width - 32 - 12) / 3
	for i, opt := range blinkOpts {
		bRec := rl.NewRectangle(rightRec.X+16+float32(i)*(bW+6), float32(y), bW, 28)
		isCur := (layer.ShowBlink == opt.val)
		hovered := rl.CheckCollisionPointRec(mousePos, bRec)

		col := rl.NewColor(32, 40, 58, 255)
		if isCur {
			col = rl.NewColor(45, 115, 75, 255)
		} else if hovered {
			col = rl.NewColor(48, 58, 80, 255)
		}

		if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			layer.ShowBlink = opt.val
		}

		rl.DrawRectangleRounded(bRec, 0.2, 4, col)
		e.UI.DrawText(opt.name, int32(bRec.X)+6, int32(bRec.Y)+6, 12, rl.RayWhite)
	}

	y += 42

	// Talk Visibility (0=Sempre, 1=Silêncio/Fechada, 2=Falando/Aberta)
	e.UI.DrawText("Modo de Fala (Talk):", int32(rightRec.X)+16, y, 13, rl.LightGray)
	y += 20

	talkOpts := []struct {
		val  int
		name string
	}{
		{0, "Sempre"},
		{1, "Silêncio"},
		{2, "Falando"},
	}

	for i, opt := range talkOpts {
		tRec := rl.NewRectangle(rightRec.X+16+float32(i)*(bW+6), float32(y), bW, 28)
		isCur := (layer.ShowTalk == opt.val)
		hovered := rl.CheckCollisionPointRec(mousePos, tRec)

		col := rl.NewColor(32, 40, 58, 255)
		if isCur {
			col = rl.NewColor(45, 115, 75, 255)
		} else if hovered {
			col = rl.NewColor(48, 58, 80, 255)
		}

		if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			layer.ShowTalk = opt.val
		}

		rl.DrawRectangleRounded(tRec, 0.2, 4, col)
		e.UI.DrawText(opt.name, int32(tRec.X)+12, int32(tRec.Y)+6, 12, rl.RayWhite)
	}

	y += 46

	// Costume Layers (1 to 10)
	e.UI.DrawText("Ativo nos Figurinos (1 a 10):", int32(rightRec.X)+16, y, 13, rl.LightGray)
	y += 22

	cBtnW := (rightRec.Width - 32 - 4*6) / 5
	for i := 1; i <= 10; i++ {
		row := float32((i - 1) / 5)
		col := float32((i - 1) % 5)

		cRec := rl.NewRectangle(rightRec.X+16+col*(cBtnW+6), float32(y)+row*34, cBtnW, 28)
		isActive := layer.CostumeLayers[i-1] == 1
		hovered := rl.CheckCollisionPointRec(mousePos, cRec)

		bgCol := rl.NewColor(32, 38, 54, 255)
		txtCol := rl.Gray
		if isActive {
			bgCol = rl.NewColor(35, 110, 65, 255)
			txtCol = rl.Lime
		} else if hovered {
			bgCol = rl.NewColor(45, 55, 75, 255)
		}

		if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			if isActive {
				layer.CostumeLayers[i-1] = 0
			} else {
				layer.CostumeLayers[i-1] = 1
			}
		}

		rl.DrawRectangleRounded(cRec, 0.2, 4, bgCol)
		e.UI.DrawText(fmt.Sprintf("%d", i), int32(cRec.X)+int32(cBtnW/2)-4, int32(cRec.Y)+6, 14, txtCol)
	}
}

func (e *EditorState) drawPhysicsTab(rightRec rl.Rectangle, startY int32, layer *model.Layer, mousePos rl.Vector2) {
	e.UI.DrawText("Física e Oscilação da Camada:", int32(rightRec.X)+16, startY, 14, rl.SkyBlue)

	barW := rightRec.Width - 32
	y := startY + 28

	// 1. Angular Damping (RotDrag)
	e.UI.DrawText(fmt.Sprintf("Arrasto Angular (Inércia): %.2f", layer.RotDrag), int32(rightRec.X)+16, y, 13, rl.Yellow)
	rdRec := rl.NewRectangle(rightRec.X+16, float32(y+18), barW, 12)
	rl.DrawRectangleRounded(rdRec, 0.3, 4, rl.DarkGray)
	if rl.IsMouseButtonDown(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mousePos, rl.NewRectangle(rdRec.X-10, rdRec.Y-8, rdRec.Width+20, rdRec.Height+16)) {
		ratio := (mousePos.X - rdRec.X) / rdRec.Width
		layer.RotDrag = ratio * 1.0
	}
	rl.DrawCircle(int32(rdRec.X+layer.RotDrag*rdRec.Width), int32(rdRec.Y)+6, 7, rl.SkyBlue)

	y += 38

	// 2. Rotation Clamp (RLimitMin, RLimitMax)
	e.UI.DrawText(fmt.Sprintf("Limite de Rotação: [%.0f° a %.0f°]", layer.RLimitMin, layer.RLimitMax), int32(rightRec.X)+16, y, 13, rl.LightGray)
	clampRec := rl.NewRectangle(rightRec.X+16, float32(y+18), barW, 12)
	rl.DrawRectangleRounded(clampRec, 0.3, 4, rl.DarkGray)
	if rl.IsMouseButtonDown(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mousePos, rl.NewRectangle(clampRec.X-10, clampRec.Y-8, clampRec.Width+20, clampRec.Height+16)) {
		ratio := (mousePos.X - clampRec.X) / clampRec.Width
		angle := ratio * 90.0
		layer.RLimitMin = -angle
		layer.RLimitMax = angle
	}
	rl.DrawCircle(int32(clampRec.X+(layer.RLimitMax/90.0)*clampRec.Width), int32(clampRec.Y)+6, 7, rl.SkyBlue)

	y += 38

	// 3. Idle Breathing (XAmp, YAmp)
	e.UI.DrawText(fmt.Sprintf("Respiração Idle (Oscilação): %.1f px", layer.YAmp), int32(rightRec.X)+16, y, 13, rl.LightGray)
	ampRec := rl.NewRectangle(rightRec.X+16, float32(y+18), barW, 12)
	rl.DrawRectangleRounded(ampRec, 0.3, 4, rl.DarkGray)
	if rl.IsMouseButtonDown(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mousePos, rl.NewRectangle(ampRec.X-10, ampRec.Y-8, ampRec.Width+20, ampRec.Height+16)) {
		ratio := (mousePos.X - ampRec.X) / ampRec.Width
		layer.YAmp = ratio * 30.0
		layer.YFrq = 1.5
	}
	rl.DrawCircle(int32(ampRec.X+(layer.YAmp/30.0)*ampRec.Width), int32(ampRec.Y)+6, 7, rl.SkyBlue)

	y += 38

	// 4. Elasticity / Stretch (StretchAmount)
	e.UI.DrawText(fmt.Sprintf("Elasticidade (Stretch): %.2f", layer.StretchAmount), int32(rightRec.X)+16, y, 13, rl.LightGray)
	strRec := rl.NewRectangle(rightRec.X+16, float32(y+18), barW, 12)
	rl.DrawRectangleRounded(strRec, 0.3, 4, rl.DarkGray)
	if rl.IsMouseButtonDown(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mousePos, rl.NewRectangle(strRec.X-10, strRec.Y-8, strRec.Width+20, strRec.Height+16)) {
		ratio := (mousePos.X - strRec.X) / strRec.Width
		layer.StretchAmount = ratio * 2.0
	}
	rl.DrawCircle(int32(strRec.X+(layer.StretchAmount/2.0)*strRec.Width), int32(strRec.Y)+6, 7, rl.SkyBlue)

	y += 42

	// Ignore Bounce Checkbox
	chkRec := rl.NewRectangle(rightRec.X+16, float32(y), 20, 20)
	if rl.CheckCollisionPointRec(mousePos, chkRec) && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		layer.IgnoreBounce = !layer.IgnoreBounce
	}
	rl.DrawRectangleRounded(chkRec, 0.2, 4, rl.DarkGray)
	if layer.IgnoreBounce {
		rl.DrawRectangle(int32(chkRec.X)+4, int32(chkRec.Y)+4, 12, 12, rl.Lime)
	}
	e.UI.DrawText("Ignorar Pulo Global (Ignore Bounce)", int32(chkRec.X)+28, int32(chkRec.Y)+2, 13, rl.RayWhite)
}

func (e *EditorState) drawSpriteTab(rightRec rl.Rectangle, startY int32, layer *model.Layer, mousePos rl.Vector2) {
	e.UI.DrawText("Configuração de SpriteSheet:", int32(rightRec.X)+16, startY, 14, rl.SkyBlue)

	y := startY + 28

	// Frames Count
	e.UI.DrawText(fmt.Sprintf("Quantidade de Quadros (Frames): %d", layer.Frames), int32(rightRec.X)+16, y, 13, rl.Yellow)
	fMinus := rl.NewRectangle(rightRec.X+240, float32(y-2), 34, 24)
	fPlus := rl.NewRectangle(rightRec.X+280, float32(y-2), 34, 24)

	if rl.CheckCollisionPointRec(mousePos, fMinus) {
		rl.DrawRectangleRounded(fMinus, 0.2, 4, rl.NewColor(45, 60, 90, 255))
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) && layer.Frames > 1 {
			layer.Frames--
		}
	} else {
		rl.DrawRectangleRounded(fMinus, 0.2, 4, rl.NewColor(32, 40, 60, 255))
	}
	e.UI.DrawText("-", int32(fMinus.X)+12, int32(fMinus.Y)+3, 16, rl.RayWhite)

	if rl.CheckCollisionPointRec(mousePos, fPlus) {
		rl.DrawRectangleRounded(fPlus, 0.2, 4, rl.NewColor(45, 60, 90, 255))
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			layer.Frames++
		}
	} else {
		rl.DrawRectangleRounded(fPlus, 0.2, 4, rl.NewColor(32, 40, 60, 255))
	}
	e.UI.DrawText("+", int32(fPlus.X)+10, int32(fPlus.Y)+3, 16, rl.RayWhite)

	y += 40

	// Animation Speed
	e.UI.DrawText(fmt.Sprintf("Velocidade da Animação: %.1f FPS", layer.AnimSpeed), int32(rightRec.X)+16, y, 13, rl.LightGray)
	fpsRec := rl.NewRectangle(rightRec.X+16, float32(y+18), rightRec.Width-32, 12)
	rl.DrawRectangleRounded(fpsRec, 0.3, 4, rl.DarkGray)
	if rl.IsMouseButtonDown(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mousePos, rl.NewRectangle(fpsRec.X-10, fpsRec.Y-8, fpsRec.Width+20, fpsRec.Height+16)) {
		ratio := (mousePos.X - fpsRec.X) / fpsRec.Width
		layer.AnimSpeed = ratio * 30.0
	}
	rl.DrawCircle(int32(fpsRec.X+(layer.AnimSpeed/30.0)*fpsRec.Width), int32(fpsRec.Y)+6, 7, rl.SkyBlue)
}

// drawLayerGizmo renders a bounding box and interactive position handle over the selected layer in the viewport.
func (e *EditorState) drawLayerGizmo(layer *model.Layer, scale float32, origin rl.Vector2, mousePos rl.Vector2) {
	if layer == nil || e.TextureCache == nil {
		return
	}

	tex, ok := e.TextureCache.GetTexture(layer.Identification)
	if !ok || tex.Width == 0 || tex.Height == 0 {
		return
	}

	frames := layer.Frames
	if frames < 1 {
		frames = 1
	}
	fW := float32(tex.Width) / float32(frames)
	fH := float32(tex.Height)

	// Compute screen position of layer center
	scrX := origin.X + (layer.Pos.X * scale)
	scrY := origin.Y + (layer.Pos.Y * scale)

	boxW := fW * scale
	boxH := fH * scale
	boxRec := rl.NewRectangle(
		scrX-(boxW*0.5)+(layer.Offset.X*scale),
		scrY-(boxH*0.5)+(layer.Offset.Y*scale),
		boxW,
		boxH,
	)

	// Draw bounding box
	rl.DrawRectangleLinesEx(boxRec, 1.5, rl.NewColor(50, 180, 255, 180))

	// Pivot / Center Anchor Handle (Circle)
	pivotRec := rl.NewRectangle(scrX-12, scrY-12, 24, 24)
	pivotHovered := rl.CheckCollisionPointRec(mousePos, pivotRec)

	if pivotHovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		e.IsDraggingPos = true
		e.DragStartMouse = mousePos
		e.DragStartLayerPos = layer.Pos
	}

	if e.IsDraggingPos {
		if rl.IsMouseButtonDown(rl.MouseLeftButton) {
			dx := (mousePos.X - e.DragStartMouse.X) / scale
			dy := (mousePos.Y - e.DragStartMouse.Y) / scale
			layer.Pos.X = e.DragStartLayerPos.X + dx
			layer.Pos.Y = e.DragStartLayerPos.Y + dy
		} else {
			e.IsDraggingPos = false
		}
	}

	handleCol := rl.SkyBlue
	if e.IsDraggingPos || pivotHovered {
		handleCol = rl.Lime
	}

	rl.DrawCircle(int32(scrX), int32(scrY), 8, handleCol)
	rl.DrawLine(int32(scrX)-14, int32(scrY), int32(scrX)+14, int32(scrY), rl.White)
	rl.DrawLine(int32(scrX), int32(scrY)-14, int32(scrX), int32(scrY)+14, rl.White)
}
