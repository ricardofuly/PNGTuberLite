package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"pngtuber-lite/pkg/i18n"
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

// TreeItem represents a node in the hierarchical layer tree.
type TreeItem struct {
	Layer       *model.Layer // nil when IsRootNode is true
	IsRootNode  bool
	Depth       int
	IsLastChild bool
}

// EditorState manages the avatar visual editor.
type EditorState struct {
	IsOpen                bool
	Avatar                *model.Avatar
	SelectedLayerID       int64 // 0 = Root / Pivot Central
	AvatarFilePath        string
	StatusMessage         string
	StatusTimer           float32
	ActiveTab             EditorTab
	TextureCache          *render.TextureCache
	UI                    *ui.UIState

	// Scroll state
	LeftScrollOffset      float32
	RightScrollOffset     float32
	IsDraggingLeftScroll  bool
	IsDraggingRightScroll bool
	LeftScrollDragStartY  float32
	RightScrollDragStartY float32
	LeftScrollStartOffset float32
	RightScrollStartOffset float32

	// Gizmo interaction
	IsDraggingPos         bool
	IsDraggingPivot       bool
	DragStartMouse        rl.Vector2
	DragStartLayerPos     model.Vector2
	DragStartPivot        model.Vector2

	// Callbacks
	OnAvatarModified      func()
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
		if _, exists := avatar.Layers[e.SelectedLayerID]; !exists && e.SelectedLayerID != 0 {
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
	e.SetStatus("Novo avatar criado. Clique no botão de imagem para importar .png!")
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
	layer.UpdateContentBounds()

	// Set parent to currently selected layer if any (and not Root)
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
	e.TextureCache.UnloadTexture(e.SelectedLayerID)

	// Default back to Root Node
	e.SelectedLayerID = 0
	if len(e.Avatar.DrawOrder) > 0 {
		e.SelectedLayerID = e.Avatar.DrawOrder[0].Identification
	}

	e.SetStatus(fmt.Sprintf("Camada %q excluída.", name))
	if e.OnAvatarModified != nil {
		e.OnAvatarModified()
	}
}

// DuplicateSelectedLayer clones the active layer.
func (e *EditorState) DuplicateSelectedLayer() {
	if e.Avatar == nil || e.SelectedLayerID == 0 {
		return
	}

	src := e.Avatar.GetLayer(e.SelectedLayerID)
	if src == nil {
		return
	}

	id := time.Now().UnixNano() / 1000000
	dup := model.NewDefaultLayer(id)
	dup.Path = src.Path + "_copia"
	dup.ImageData = src.ImageData
	dup.ImageWidth = src.ImageWidth
	dup.ImageHeight = src.ImageHeight
	dup.ContentMinX = src.ContentMinX
	dup.ContentMinY = src.ContentMinY
	dup.ContentMaxX = src.ContentMaxX
	dup.ContentMaxY = src.ContentMaxY
	dup.HasContentBounds = src.HasContentBounds
	dup.Pos = model.Vector2{X: src.Pos.X + 15, Y: src.Pos.Y + 15}
	dup.Offset = src.Offset
	dup.ZIndex = src.ZIndex + 1
	dup.ParentID = src.ParentID
	dup.ShowBlink = src.ShowBlink
	dup.ShowTalk = src.ShowTalk
	dup.CostumeLayers = src.CostumeLayers
	dup.RotDrag = src.RotDrag
	dup.RLimitMin = src.RLimitMin
	dup.RLimitMax = src.RLimitMax
	dup.StretchAmount = src.StretchAmount
	dup.Frames = src.Frames
	dup.AnimSpeed = src.AnimSpeed

	e.Avatar.AddLayer(dup)
	e.Avatar.BuildHierarchy()
	e.SelectedLayerID = id

	if err := e.TextureCache.LoadAvatarTextures(e.Avatar); err != nil {
		e.SetStatus(fmt.Sprintf("Erro ao carregar textura: %v", err))
		return
	}

	e.SetStatus(fmt.Sprintf("Camada %q duplicada com sucesso!", src.Path))
	if e.OnAvatarModified != nil {
		e.OnAvatarModified()
	}
}

// SaveCurrentAvatar exports the active avatar to the save file path.
func (e *EditorState) SaveCurrentAvatar() error {
	if e.Avatar == nil {
		return fmt.Errorf("nenhum avatar carregado")
	}

	targetPath := e.AvatarFilePath
	if targetPath == "" {
		targetPath = "assets/samples/meu_avatar.save"
	}

	dir := filepath.Dir(targetPath)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}

	if err := model.SaveAvatarToFile(e.Avatar, targetPath); err != nil {
		e.SetStatus(fmt.Sprintf("Erro ao salvar: %v", err))
		return err
	}

	e.SetStatus(fmt.Sprintf("Avatar salvo com sucesso em %q!", targetPath))
	if e.OnAvatarModified != nil {
		e.OnAvatarModified()
	}
	return nil
}

// CreateAvatarFromDirectory builds and loads a new avatar from a folder of PNG sprites.
func (e *EditorState) CreateAvatarFromDirectory(dirPath string) error {
	av, err := model.BuildAvatarFromDirectory(dirPath)
	if err != nil {
		e.SetStatus(fmt.Sprintf("Erro ao importar pasta: %v", err))
		return err
	}

	folderName := filepath.Base(dirPath)
	targetSave := filepath.Join("assets/samples", strings.ToLower(folderName)+".save")
	e.SetAvatar(av, targetSave)
	_ = e.TextureCache.LoadAvatarTextures(av)
	_ = e.SaveCurrentAvatar()

	e.SetStatus(fmt.Sprintf("Avatar %q criado com %d camadas e salvo com sucesso!", folderName, len(av.Layers)))
	if e.OnAvatarModified != nil {
		e.OnAvatarModified()
	}
	return nil
}

// HandleFileDrops detects drag-and-dropped PNG files, directories, and .save files onto the application window.
func (e *EditorState) HandleFileDrops() {
	if !rl.IsFileDropped() {
		return
	}

	files := rl.LoadDroppedFiles()
	defer rl.UnloadDroppedFiles()

	var pngList []string
	var dirList []string
	var saveList []string

	for _, f := range files {
		fi, err := os.Stat(f)
		if err != nil {
			continue
		}
		if fi.IsDir() {
			dirList = append(dirList, f)
		} else {
			ext := strings.ToLower(filepath.Ext(f))
			if ext == ".png" {
				pngList = append(pngList, f)
			} else if ext == ".save" {
				saveList = append(saveList, f)
			}
		}
	}

	// 1. If a folder was dropped: build complete avatar from folder!
	if len(dirList) > 0 {
		_ = e.CreateAvatarFromDirectory(dirList[0])
		return
	}

	// 2. If multiple PNGs were dropped and creating a new avatar:
	if len(pngList) >= 2 && (e.Avatar == nil || len(e.Avatar.Layers) == 0) {
		if av, err := model.BuildAvatarFromPNGFiles(pngList); err == nil {
			targetSave := "assets/samples/novo_avatar.save"
			e.SetAvatar(av, targetSave)
			_ = e.TextureCache.LoadAvatarTextures(av)
			_ = e.SaveCurrentAvatar()
			e.SetStatus(fmt.Sprintf("Avatar criado com %d camadas a partir das imagens!", len(av.Layers)))
			if e.OnAvatarModified != nil {
				e.OnAvatarModified()
			}
			return
		}
	}

	// 3. If individual PNGs were dropped: add as layers to existing avatar
	importedCount := 0
	for _, pngFile := range pngList {
		if err := e.AddLayerFromPNG(pngFile); err == nil {
			importedCount++
		}
	}

	// 4. If a .save file was dropped: load avatar
	for _, saveFile := range saveList {
		if av, err := model.ParseSaveFile(saveFile); err == nil {
			e.SetAvatar(av, saveFile)
			_ = e.TextureCache.LoadAvatarTextures(av)
			e.SetStatus(fmt.Sprintf("Avatar %q carregado do arquivo .save!", filepath.Base(saveFile)))
			if e.OnAvatarModified != nil {
				e.OnAvatarModified()
			}
		}
	}

	if importedCount > 0 {
		e.SetStatus(fmt.Sprintf("%d camada(s) PNG importada(s) com sucesso!", importedCount))
	}
}

// Update ticks timers and file drops.
func (e *EditorState) Update(dt float32) {
	if e.StatusTimer > 0 {
		e.StatusTimer -= dt
		if e.StatusTimer <= 0 {
			e.StatusMessage = ""
		}
	}

	e.HandleFileDrops()
}

// buildTreeItems traverses the avatar hierarchy and builds a flattened list of tree nodes.
func (e *EditorState) buildTreeItems() []*TreeItem {
	items := make([]*TreeItem, 0)

	// 1. Root Pivot Item (Index 0)
	items = append(items, &TreeItem{
		IsRootNode: true,
		Depth:      0,
	})

	if e.Avatar == nil || len(e.Avatar.Layers) == 0 {
		return items
	}

	// 2. Recursive branch traversal
	var addBranch func(layer *model.Layer, depth int, isLast bool)
	addBranch = func(layer *model.Layer, depth int, isLast bool) {
		items = append(items, &TreeItem{
			Layer:       layer,
			Depth:       depth,
			IsLastChild: isLast,
		})

		children := e.Avatar.GetChildren(layer.Identification)
		for i, child := range children {
			addBranch(child, depth+1, i == len(children)-1)
		}
	}

	for i, root := range e.Avatar.RootLayers {
		addBranch(root, 1, i == len(e.Avatar.RootLayers)-1)
	}

	// 3. Fallback for any orphan layers
	included := make(map[int64]bool)
	for _, it := range items {
		if it.Layer != nil {
			included[it.Layer.Identification] = true
		}
	}
	for _, l := range e.Avatar.DrawOrder {
		if !included[l.Identification] {
			items = append(items, &TreeItem{
				Layer: l,
				Depth: 1,
			})
		}
	}

	return items
}

// Draw renders the visual editor overlay (left layer tree, right properties panel, and central gizmo).
func (e *EditorState) Draw(scale float32, origin rl.Vector2) {
	if !e.IsOpen {
		return
	}

	screenW := float32(rl.GetScreenWidth())
	screenH := float32(rl.GetScreenHeight())
	mousePos := rl.GetMousePosition()

	// 1. Top Header Bar
	headerRec := rl.NewRectangle(0, 0, screenW, 48)
	rl.DrawRectangleRec(headerRec, ui.ColPanelBg)
	rl.DrawLine(0, 48, int32(screenW), 48, ui.ColPanelBorder)

	ui.GlobalIcons.DrawIcon(ui.IconEditor, 16, 14, 20, ui.ColSkyBlue)
	e.UI.DrawTextBold(i18n.T("label_editor_title"), 46, 15, 15, ui.ColTextTitle)

	// Save Button (Pill with generous breathing room)
	saveBtnRec := rl.NewRectangle(screenW-265, 7, 120, 34)
	saveHovered := rl.CheckCollisionPointRec(mousePos, saveBtnRec)
	saveBg := rl.NewColor(30, 95, 55, 255)
	if saveHovered {
		saveBg = rl.NewColor(40, 130, 75, 255)
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			_ = e.SaveCurrentAvatar()
		}
	}
	rl.DrawRectangleRounded(saveBtnRec, 0.45, 6, saveBg)
	rl.DrawRectangleRoundedLines(saveBtnRec, 0.45, 6, ui.ColLime)
	ui.GlobalIcons.DrawIcon(ui.IconSave, saveBtnRec.X+14, saveBtnRec.Y+8, 18, ui.ColWhite)
	e.UI.DrawTextBold(i18n.T("btn_save"), int32(saveBtnRec.X)+40, int32(saveBtnRec.Y)+9, 12.5, ui.ColWhite)

	// Close Editor Button (Pill with generous breathing room)
	closeBtnRec := rl.NewRectangle(screenW-135, 7, 120, 34)
	closeHovered := rl.CheckCollisionPointRec(mousePos, closeBtnRec)
	closeBg := ui.ColCardBg
	if closeHovered {
		closeBg = ui.ColCardHover
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			e.IsOpen = false
		}
	}
	rl.DrawRectangleRounded(closeBtnRec, 0.45, 6, closeBg)
	rl.DrawRectangleRoundedLines(closeBtnRec, 0.45, 6, ui.ColPanelBorder)
	ui.GlobalIcons.DrawIcon(ui.IconClose, closeBtnRec.X+14, closeBtnRec.Y+8, 18, ui.ColRed)
	e.UI.DrawTextBold(i18n.T("btn_close"), int32(closeBtnRec.X)+40, int32(closeBtnRec.Y)+9, 12.5, ui.ColTextTitle)

	// Status Notification Banner
	if e.StatusMessage != "" {
		bannerRec := rl.NewRectangle(screenW/2-210, 58, 420, 34)
		rl.DrawRectangleRounded(bannerRec, 0.4, 4, ui.ColPanelBg)
		rl.DrawRectangleRoundedLines(bannerRec, 0.4, 4, ui.ColSkyBlue)
		e.UI.DrawTextBold(e.StatusMessage, int32(bannerRec.X)+16, int32(bannerRec.Y)+9, 12, ui.ColYellow)
	}

	// 2. Left Sidebar: Layer Hierarchy Tree
	leftW := float32(310)
	leftRec := rl.NewRectangle(14, 58, leftW, screenH-72)
	rl.DrawRectangleRounded(leftRec, 0.04, 6, ui.ColPanelBg)
	rl.DrawRectangleRoundedLines(leftRec, 0.04, 6, ui.ColPanelBorder)

	e.UI.DrawTextBold(i18n.T("label_layers_list"), int32(leftRec.X)+14, int32(leftRec.Y)+12, 14, ui.ColSkyBlue)

	// Top Icon Toolbar (Import PNG, New Avatar, Duplicate, Delete) - Clean Icons Only!
	toolGap := float32(6)
	toolBtnW := (leftW - 24 - 3*toolGap) / 4
	toolBtnH := float32(32)
	toolY := leftRec.Y + 34

	// 1. Import PNG (Icon Only)
	pngBtn := rl.NewRectangle(leftRec.X+12, toolY, toolBtnW, toolBtnH)
	pngHovered := rl.CheckCollisionPointRec(mousePos, pngBtn)
	pngBg := ui.ColCardBg
	if pngHovered {
		pngBg = ui.ColCardHover
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			e.SetStatus("Arraste arquivos .png para a janela!")
		}
	}
	rl.DrawRectangleRounded(pngBtn, 0.35, 4, pngBg)
	rl.DrawRectangleRoundedLines(pngBtn, 0.35, 4, ui.ColPanelBorder)
	ui.GlobalIcons.DrawIcon(ui.IconPNGFile, pngBtn.X+(toolBtnW-18)/2, pngBtn.Y+7, 18, ui.ColSkyBlue)

	// 2. New Avatar (Icon Only)
	newBtn := rl.NewRectangle(leftRec.X+12+1*(toolBtnW+toolGap), toolY, toolBtnW, toolBtnH)
	newHovered := rl.CheckCollisionPointRec(mousePos, newBtn)
	newBg := ui.ColCardBg
	if newHovered {
		newBg = ui.ColCardHover
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			e.NewBlankAvatar()
		}
	}
	rl.DrawRectangleRounded(newBtn, 0.35, 4, newBg)
	rl.DrawRectangleRoundedLines(newBtn, 0.35, 4, ui.ColPanelBorder)
	ui.GlobalIcons.DrawIcon(ui.IconAdd, newBtn.X+(toolBtnW-18)/2, newBtn.Y+7, 18, ui.ColLime)

	// 3. Duplicate (Icon Only)
	dupBtn := rl.NewRectangle(leftRec.X+12+2*(toolBtnW+toolGap), toolY, toolBtnW, toolBtnH)
	dupHovered := rl.CheckCollisionPointRec(mousePos, dupBtn)
	dupBg := ui.ColCardBg
	if dupHovered {
		dupBg = ui.ColCardHover
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			e.DuplicateSelectedLayer()
		}
	}
	rl.DrawRectangleRounded(dupBtn, 0.35, 4, dupBg)
	rl.DrawRectangleRoundedLines(dupBtn, 0.35, 4, ui.ColPanelBorder)
	ui.GlobalIcons.DrawIcon(ui.IconDuplicate, dupBtn.X+(toolBtnW-18)/2, dupBtn.Y+7, 18, ui.ColLavender)

	// 4. Delete (Icon Only)
	delBtn := rl.NewRectangle(leftRec.X+12+3*(toolBtnW+toolGap), toolY, toolBtnW, toolBtnH)
	delHovered := rl.CheckCollisionPointRec(mousePos, delBtn)
	delBg := ui.ColCardBg
	if delHovered {
		delBg = rl.NewColor(135, 35, 45, 255)
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			e.RemoveSelectedLayer()
		}
	}
	rl.DrawRectangleRounded(delBtn, 0.35, 4, delBg)
	rl.DrawRectangleRoundedLines(delBtn, 0.35, 4, ui.ColPanelBorder)
	ui.GlobalIcons.DrawIcon(ui.IconDelete, delBtn.X+(toolBtnW-18)/2, delBtn.Y+7, 18, ui.ColRed)

	// Scrollable Layer Tree
	layerViewRec := rl.NewRectangle(leftRec.X+10, leftRec.Y+74, leftW-20, leftRec.Height-86)
	treeItems := e.buildTreeItems()
	contentH := float32(len(treeItems)) * 42.0

	// Handle left scroll
	if rl.CheckCollisionPointRec(mousePos, layerViewRec) {
		wheel := rl.GetMouseWheelMove()
		if wheel != 0 {
			e.LeftScrollOffset -= wheel * 36.0
		}
	}
	maxScroll := contentH - layerViewRec.Height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if e.LeftScrollOffset < 0 {
		e.LeftScrollOffset = 0
	}
	if e.LeftScrollOffset > maxScroll {
		e.LeftScrollOffset = maxScroll
	}

	rl.BeginScissorMode(int32(layerViewRec.X), int32(layerViewRec.Y), int32(layerViewRec.Width), int32(layerViewRec.Height))
	itemY := layerViewRec.Y - e.LeftScrollOffset

	for _, item := range treeItems {
		if item.IsRootNode {
			// Root / Pivot Central Node
			isSelected := (e.SelectedLayerID == 0)
			rootRec := rl.NewRectangle(layerViewRec.X+2, itemY, layerViewRec.Width-14, 38)
			hovered := rl.CheckCollisionPointRec(mousePos, rootRec)

			ui.DrawCard(rootRec, hovered, isSelected)
			if isSelected {
				rl.DrawRectangleRoundedLines(rootRec, 0.16, 4, ui.ColYellow)
			}

			ui.GlobalIcons.DrawIcon(ui.IconAvatar, rootRec.X+8, rootRec.Y+9, 18, ui.ColYellow)
			e.UI.DrawTextBold(i18n.T("label_root_pivot"), int32(rootRec.X)+32, int32(rootRec.Y)+10, 12.5, ui.ColYellow)
			e.UI.DrawBadge(rootRec.X+rootRec.Width-56, rootRec.Y+9, "Raiz", ui.ColIconBoxBg, ui.ColOrange)

			if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
				e.SelectedLayerID = 0
			}
		} else if item.Layer != nil {
			layer := item.Layer
			isSelected := (layer.Identification == e.SelectedLayerID)
			indent := float32(item.Depth) * 16.0
			itemRec := rl.NewRectangle(layerViewRec.X+4+indent, itemY, layerViewRec.Width-14-indent, 38)
			hovered := rl.CheckCollisionPointRec(mousePos, itemRec)

			// Draw Tree connector lines
			lineX := layerViewRec.X + indent - 4
			rl.DrawLineEx(rl.NewVector2(lineX, itemY-4), rl.NewVector2(lineX, itemY+19), 1.5, ui.ColPanelBorder)
			rl.DrawLineEx(rl.NewVector2(lineX, itemY+19), rl.NewVector2(itemRec.X, itemY+19), 1.5, ui.ColPanelBorder)

			ui.DrawCard(itemRec, hovered, isSelected)

			displayName := filepath.Base(layer.Path)
			maxLen := 17 - (item.Depth * 2)
			if maxLen < 8 {
				maxLen = 8
			}
			if len(displayName) > maxLen {
				displayName = displayName[:maxLen] + "..."
			}

			textCol := ui.ColTextBody
			if isSelected {
				textCol = ui.ColWhite
			}
			e.UI.DrawTextBold(displayName, int32(itemRec.X)+8, int32(itemRec.Y)+10, 12, textCol)
			e.UI.DrawBadge(itemRec.X+itemRec.Width-48, itemRec.Y+9, fmt.Sprintf("Z:%d", layer.ZIndex), ui.ColIconBoxBg, ui.ColTextMuted)

			if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
				e.SelectedLayerID = layer.Identification
			}
		}
		itemY += 42
	}
	rl.EndScissorMode()

	// Draw left scrollbar if content exceeds view
	if contentH > layerViewRec.Height {
		trackX := layerViewRec.X + layerViewRec.Width - 6
		trackY := layerViewRec.Y + 2
		trackH := layerViewRec.Height - 4
		rl.DrawRectangleRounded(rl.NewRectangle(trackX, trackY, 5, trackH), 0.5, 4, ui.ColScrollTrack)

		thumbH := (layerViewRec.Height / contentH) * trackH
		if thumbH < 26 {
			thumbH = 26
		}
		thumbY := trackY + (e.LeftScrollOffset/maxScroll)*(trackH-thumbH)
		rl.DrawRectangleRounded(rl.NewRectangle(trackX, thumbY, 5, thumbH), 0.5, 4, ui.ColScrollThumb)
	}

	// 3. Right Sidebar: Layer Properties & Settings (Wider for full visibility tab room)
	rightW := float32(420)
	rightRec := rl.NewRectangle(screenW-rightW-14, 58, rightW, screenH-72)
	rl.DrawRectangleRounded(rightRec, 0.04, 6, ui.ColPanelBg)
	rl.DrawRectangleRoundedLines(rightRec, 0.04, 6, ui.ColPanelBorder)

	// Direct Viewport Click Selection (Clicking on any layer's sprite in the viewport selects it)
	viewportRec := rl.NewRectangle(leftRec.X+leftRec.Width+10, headerRec.Height+10, rightRec.X-(leftRec.X+leftRec.Width+20), screenH-headerRec.Height-20)
	if e.Avatar != nil && rl.CheckCollisionPointRec(mousePos, viewportRec) && rl.IsMouseButtonPressed(rl.MouseLeftButton) && !e.IsDraggingPos {
		transforms := render.ComputeWorldTransforms(e.Avatar, origin, scale, 0, nil, nil)
		clickedLayerID := int64(-1)

		for i := len(e.Avatar.DrawOrder) - 1; i >= 0; i-- {
			l := e.Avatar.DrawOrder[i]
			tex, ok := e.TextureCache.GetTexture(l.Identification)
			if !ok || tex.Width == 0 {
				continue
			}
			t, ok := transforms[l.Identification]
			if !ok {
				continue
			}

			frames := l.Frames
			if frames < 1 {
				frames = 1
			}
			fW := float32(tex.Width) / float32(frames)
			fH := float32(tex.Height)
			topLeftX := t.WorldPos.X + (l.Offset.X * t.Scale)
			topLeftY := t.WorldPos.Y + (l.Offset.Y * t.Scale)

			minX := float32(0)
			minY := float32(0)
			maxX := fW
			maxY := fH
			if l.HasContentBounds {
				minX = l.ContentMinX
				minY = l.ContentMinY
				maxX = l.ContentMaxX
				maxY = l.ContentMaxY
			}
			lBox := rl.NewRectangle(topLeftX+minX*t.Scale, topLeftY+minY*t.Scale, (maxX-minX)*t.Scale, (maxY-minY)*t.Scale)
			if rl.CheckCollisionPointRec(mousePos, lBox) {
				clickedLayerID = l.Identification
				break
			}
		}

		if clickedLayerID != -1 {
			e.SelectedLayerID = clickedLayerID
		}
	}

	// If Root / Pivot Central is selected, show Master Avatar Properties
	if e.SelectedLayerID == 0 {
		e.drawRootPivotPanel(rightRec, mousePos, scale, origin)
		e.drawLayerGizmo(nil, scale, origin, mousePos)
		return
	}

	var curLayer *model.Layer
	if e.Avatar != nil {
		curLayer = e.Avatar.GetLayer(e.SelectedLayerID)
	}

	if curLayer == nil {
		e.UI.DrawText(i18n.T("label_no_layer_selected"), int32(rightRec.X)+20, int32(rightRec.Y)+40, 13, ui.ColTextMuted)
		return
	}

	// Properties Tabs Header (Wide enough with generous breathing room for "Visibilidade")
	tabs := []struct {
		tab  EditorTab
		name string
	}{
		{TabLayerGeneral, i18n.T("tab_layer_general")},
		{TabLayerVisibility, i18n.T("tab_layer_visibility")},
		{TabLayerPhysics, i18n.T("tab_layer_physics")},
		{TabLayerSprite, i18n.T("tab_layer_sprite")},
	}

	tW := (rightW - 24) / float32(len(tabs))
	for i, t := range tabs {
		tRec := rl.NewRectangle(rightRec.X+12+float32(i)*tW, rightRec.Y+12, tW-2, 32)
		isActive := e.ActiveTab == t.tab
		hovered := rl.CheckCollisionPointRec(mousePos, tRec)

		tBg := ui.ColCardBg
		tTextCol := ui.ColTextMuted
		if isActive {
			tBg = ui.ColCardActive
			tTextCol = ui.ColWhite
		} else if hovered {
			tBg = ui.ColCardHover
			tTextCol = ui.ColWhite
		}

		if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			e.ActiveTab = t.tab
			e.RightScrollOffset = 0
		}

		rl.DrawRectangleRounded(tRec, 0.4, 4, tBg)
		if isActive {
			rl.DrawRectangleRoundedLines(tRec, 0.4, 4, ui.ColSkyBlue)
		}

		fontSize := float32(11.5)
		textW := e.UI.MeasureTextBold(t.name, fontSize)
		if textW > tRec.Width-6 {
			fontSize = 10.0
			textW = e.UI.MeasureTextBold(t.name, fontSize)
		}
		e.UI.DrawTextBold(t.name, int32(tRec.X)+int32(tW/2)-int32(textW/2), int32(tRec.Y)+8, fontSize, tTextCol)
	}

	// Scrollable Properties Viewport
	propViewRec := rl.NewRectangle(rightRec.X+10, rightRec.Y+52, rightW-20, rightRec.Height-62)
	propContentH := float32(500)

	if rl.CheckCollisionPointRec(mousePos, propViewRec) {
		wheel := rl.GetMouseWheelMove()
		if wheel != 0 {
			e.RightScrollOffset -= wheel * 36.0
		}
	}
	maxRightScroll := propContentH - propViewRec.Height
	if maxRightScroll < 0 {
		maxRightScroll = 0
	}
	if e.RightScrollOffset < 0 {
		e.RightScrollOffset = 0
	}
	if e.RightScrollOffset > maxRightScroll {
		e.RightScrollOffset = maxRightScroll
	}

	rl.BeginScissorMode(int32(propViewRec.X), int32(propViewRec.Y), int32(propViewRec.Width), int32(propViewRec.Height))
	propY := int32(propViewRec.Y - e.RightScrollOffset)

	switch e.ActiveTab {
	case TabLayerGeneral:
		e.drawGeneralTab(propViewRec, propY, curLayer, mousePos)
	case TabLayerVisibility:
		e.drawVisibilityTab(propViewRec, propY, curLayer, mousePos)
	case TabLayerPhysics:
		e.drawPhysicsTab(propViewRec, propY, curLayer, mousePos)
	case TabLayerSprite:
		e.drawSpriteTab(propViewRec, propY, curLayer, mousePos)
	}
	rl.EndScissorMode()

	if propContentH > propViewRec.Height {
		trackX := propViewRec.X + propViewRec.Width - 6
		trackY := propViewRec.Y + 2
		trackH := propViewRec.Height - 4
		rl.DrawRectangleRounded(rl.NewRectangle(trackX, trackY, 5, trackH), 0.5, 4, ui.ColScrollTrack)

		thumbH := (propViewRec.Height / propContentH) * trackH
		if thumbH < 26 {
			thumbH = 26
		}
		thumbY := trackY + (e.RightScrollOffset/maxRightScroll)*(trackH-thumbH)
		rl.DrawRectangleRounded(rl.NewRectangle(trackX, thumbY, 5, thumbH), 0.5, 4, ui.ColScrollThumb)
	}

	// 4. Central Gizmo & Precise Sprite-Sized Bounding Box
	e.drawLayerGizmo(curLayer, scale, origin, mousePos)
}

// drawRootPivotPanel displays master character properties when the Root Node is selected.
func (e *EditorState) drawRootPivotPanel(rightRec rl.Rectangle, mousePos rl.Vector2, scale float32, origin rl.Vector2) {
	y := int32(rightRec.Y) + 14

	// Header Card
	hCard := rl.NewRectangle(rightRec.X+12, float32(y), rightRec.Width-24, 60)
	ui.DrawCard(hCard, false, false)
	rl.DrawRectangleRoundedLines(hCard, 0.16, 4, ui.ColYellow)
	e.UI.DrawIconBadge(hCard.X+10, hCard.Y+10, 40, ui.IconAvatar, ui.ColYellow, ui.ColIconBoxBg)
	e.UI.DrawTextBold(i18n.T("label_pivot_character"), int32(hCard.X)+58, int32(hCard.Y)+12, 14, ui.ColYellow)
	e.UI.DrawText(i18n.T("label_pivot_anchor_desc"), int32(hCard.X)+58, int32(hCard.Y)+34, 11.5, ui.ColTextMuted)

	y += 72

	// Info Card
	infoCard := rl.NewRectangle(rightRec.X+12, float32(y), rightRec.Width-24, 120)
	ui.DrawCard(infoCard, false, false)
	e.UI.DrawTextBold(i18n.T("label_avatar_stats"), int32(infoCard.X)+14, int32(infoCard.Y)+12, 13.5, ui.ColSkyBlue)

	layerCount := 0
	rootCount := 0
	if e.Avatar != nil {
		layerCount = len(e.Avatar.Layers)
		rootCount = len(e.Avatar.RootLayers)
	}
	e.UI.DrawText(fmt.Sprintf(i18n.T("label_total_layers"), layerCount), int32(infoCard.X)+14, int32(infoCard.Y)+36, 12, ui.ColTextBody)
	e.UI.DrawText(fmt.Sprintf(i18n.T("label_root_layers_connected"), rootCount), int32(infoCard.X)+14, int32(infoCard.Y)+58, 12, ui.ColTextBody)
	e.UI.DrawText(fmt.Sprintf(i18n.T("label_file_path"), filepath.Base(e.AvatarFilePath)), int32(infoCard.X)+14, int32(infoCard.Y)+80, 12, ui.ColTextBody)

	y += 134

	// Central Pivot Origin Display Card
	originCard := rl.NewRectangle(rightRec.X+12, float32(y), rightRec.Width-24, 100)
	ui.DrawCard(originCard, false, false)
	e.UI.DrawTextBold(i18n.T("label_canvas_origin"), int32(originCard.X)+14, int32(originCard.Y)+12, 13.5, ui.ColSkyBlue)
	e.UI.DrawText(fmt.Sprintf(i18n.T("label_origin_coords"), origin.X, origin.Y), int32(originCard.X)+14, int32(originCard.Y)+36, 12, ui.ColTextBody)
	e.UI.DrawText(fmt.Sprintf(i18n.T("label_global_scale"), scale), int32(originCard.X)+14, int32(originCard.Y)+58, 12, ui.ColTextBody)

	y += 114

	// Quick action: Save Avatar
	saveCard := rl.NewRectangle(rightRec.X+12, float32(y), rightRec.Width-24, 46)
	hoveredSave := rl.CheckCollisionPointRec(mousePos, saveCard)
	saveBg := rl.NewColor(30, 95, 55, 255)
	if hoveredSave {
		saveBg = rl.NewColor(40, 130, 75, 255)
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			_ = e.SaveCurrentAvatar()
		}
	}
	rl.DrawRectangleRounded(saveCard, 0.35, 4, saveBg)
	rl.DrawRectangleRoundedLines(saveCard, 0.35, 4, ui.ColLime)
	ui.GlobalIcons.DrawIcon(ui.IconSave, saveCard.X+14, saveCard.Y+12, 20, ui.ColWhite)
	e.UI.DrawTextBold(i18n.T("btn_save_apply"), int32(saveCard.X)+46, int32(saveCard.Y)+14, 13, ui.ColWhite)
}

func (e *EditorState) drawGeneralTab(rightRec rl.Rectangle, startY int32, layer *model.Layer, mousePos rl.Vector2) {
	y := startY + 6

	// Layer Info Card
	infoCard := rl.NewRectangle(rightRec.X+2, float32(y), rightRec.Width-14, 44)
	ui.DrawCard(infoCard, false, false)
	e.UI.DrawIconBadge(infoCard.X+8, infoCard.Y+6, 32, ui.IconAvatar, ui.ColSkyBlue, ui.ColIconBoxBg)
	e.UI.DrawTextBold(filepath.Base(layer.Path), int32(infoCard.X)+48, int32(infoCard.Y)+13, 13.5, ui.ColTextTitle)

	y += 52

	// Z-Index Card
	zCard := rl.NewRectangle(rightRec.X+2, float32(y), rightRec.Width-14, 52)
	ui.DrawCard(zCard, false, false)
	e.UI.DrawTextBold(fmt.Sprintf(i18n.T("label_zindex_depth"), layer.ZIndex), int32(zCard.X)+14, int32(zCard.Y)+16, 13.5, ui.ColSkyBlue)

	zMinus := rl.NewRectangle(zCard.X+zCard.Width-84, zCard.Y+10, 36, 32)
	zPlus := rl.NewRectangle(zCard.X+zCard.Width-44, zCard.Y+10, 36, 32)

	if rl.CheckCollisionPointRec(mousePos, zMinus) {
		rl.DrawRectangleRounded(zMinus, 0.35, 4, ui.ColCardHover)
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			layer.ZIndex--
			e.Avatar.BuildHierarchy()
		}
	} else {
		rl.DrawRectangleRounded(zMinus, 0.35, 4, ui.ColCardBg)
	}
	rl.DrawRectangleRoundedLines(zMinus, 0.35, 4, ui.ColPanelBorder)
	e.UI.DrawTextBold("-", int32(zMinus.X)+14, int32(zMinus.Y)+7, 16, ui.ColWhite)

	if rl.CheckCollisionPointRec(mousePos, zPlus) {
		rl.DrawRectangleRounded(zPlus, 0.35, 4, ui.ColCardHover)
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			layer.ZIndex++
			e.Avatar.BuildHierarchy()
		}
	} else {
		rl.DrawRectangleRounded(zPlus, 0.35, 4, ui.ColCardBg)
	}
	rl.DrawRectangleRoundedLines(zPlus, 0.35, 4, ui.ColPanelBorder)
	e.UI.DrawTextBold("+", int32(zPlus.X)+13, int32(zPlus.Y)+7, 16, ui.ColWhite)

	y += 60

	// Position X & Y Sliders Card
	posCard := rl.NewRectangle(rightRec.X+2, float32(y), rightRec.Width-14, 126)
	ui.DrawCard(posCard, false, false)
	e.UI.DrawTextBold(i18n.T("label_canvas_position"), int32(posCard.X)+14, int32(posCard.Y)+10, 13.5, ui.ColSkyBlue)

	e.UI.DrawText(fmt.Sprintf(i18n.T("label_pos_x"), layer.Pos.X), int32(posCard.X)+14, int32(posCard.Y)+34, 11.5, ui.ColTextBody)
	posXRec := rl.NewRectangle(posCard.X+14, posCard.Y+52, posCard.Width-28, 10)
	layer.Pos.X = e.UI.DrawSliderControl(posXRec, layer.Pos.X, -300.0, 300.0, mousePos, ui.ColSkyBlue)

	e.UI.DrawText(fmt.Sprintf(i18n.T("label_pos_y"), layer.Pos.Y), int32(posCard.X)+14, int32(posCard.Y)+78, 11.5, ui.ColTextBody)
	posYRec := rl.NewRectangle(posCard.X+14, posCard.Y+96, posCard.Width-28, 10)
	layer.Pos.Y = e.UI.DrawSliderControl(posYRec, layer.Pos.Y, -300.0, 300.0, mousePos, ui.ColSkyBlue)

	y += 134

	// Pivot Offset Card
	pivotCard := rl.NewRectangle(rightRec.X+2, float32(y), rightRec.Width-14, 126)
	ui.DrawCard(pivotCard, false, false)
	e.UI.DrawTextBold(i18n.T("label_pivot_offset"), int32(pivotCard.X)+14, int32(pivotCard.Y)+10, 13.5, ui.ColSkyBlue)

	e.UI.DrawText(fmt.Sprintf(i18n.T("label_pivot_x"), layer.Offset.X), int32(pivotCard.X)+14, int32(pivotCard.Y)+34, 11.5, ui.ColTextBody)
	offXRec := rl.NewRectangle(pivotCard.X+14, pivotCard.Y+52, pivotCard.Width-28, 10)
	layer.Offset.X = e.UI.DrawSliderControl(offXRec, layer.Offset.X, -200.0, 200.0, mousePos, ui.ColSkyBlue)

	e.UI.DrawText(fmt.Sprintf(i18n.T("label_pivot_y"), layer.Offset.Y), int32(pivotCard.X)+14, int32(pivotCard.Y)+78, 11.5, ui.ColTextBody)
	offYRec := rl.NewRectangle(pivotCard.X+14, pivotCard.Y+96, pivotCard.Width-28, 10)
	layer.Offset.Y = e.UI.DrawSliderControl(offYRec, layer.Offset.Y, -200.0, 200.0, mousePos, ui.ColSkyBlue)

	y += 134

	// Parent Node Selection Card
	pCard := rl.NewRectangle(rightRec.X+2, float32(y), rightRec.Width-14, 80)
	ui.DrawCard(pCard, false, false)
	parentName := i18n.T("label_root_pivot")
	if layer.ParentID != nil && *layer.ParentID != 0 {
		if p := e.Avatar.GetLayer(*layer.ParentID); p != nil {
			parentName = filepath.Base(p.Path)
		}
	}
	e.UI.DrawTextBold(fmt.Sprintf(i18n.T("label_current_parent"), parentName), int32(pCard.X)+14, int32(pCard.Y)+12, 13.5, ui.ColSkyBlue)

	pBtn := rl.NewRectangle(pCard.X+14, pCard.Y+40, pCard.Width-28, 30)
	if rl.CheckCollisionPointRec(mousePos, pBtn) {
		rl.DrawRectangleRounded(pBtn, 0.35, 4, ui.ColCardHover)
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			e.cycleParentLayer(layer)
		}
	} else {
		rl.DrawRectangleRounded(pBtn, 0.35, 4, ui.ColCardBg)
	}
	rl.DrawRectangleRoundedLines(pBtn, 0.35, 4, ui.ColPanelBorder)
	e.UI.DrawTextBold(i18n.T("btn_change_parent"), int32(pBtn.X)+int32(pBtn.Width/2)-60, int32(pBtn.Y)+7, 12, ui.ColWhite)
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
	y := startY + 6

	// Blink Card (Generous breathing room)
	blinkCard := rl.NewRectangle(rightRec.X+2, float32(y), rightRec.Width-14, 86)
	ui.DrawCard(blinkCard, false, false)
	e.UI.DrawTextBold(i18n.T("label_vis_blink"), int32(blinkCard.X)+14, int32(blinkCard.Y)+10, 13.5, ui.ColSkyBlue)

	blinkOpts := []struct {
		val  int
		name string
	}{
		{0, i18n.T("opt_vis_always")},
		{1, i18n.T("opt_vis_eyes_open")},
		{2, i18n.T("opt_vis_eyes_closed")},
	}

	bW := (blinkCard.Width - 28 - 12) / 3
	for i, opt := range blinkOpts {
		bRec := rl.NewRectangle(blinkCard.X+14+float32(i)*(bW+6), blinkCard.Y+40, bW, 34)
		isCur := (layer.ShowBlink == opt.val)
		hovered := rl.CheckCollisionPointRec(mousePos, bRec)

		ui.DrawCard(bRec, hovered, isCur)
		e.UI.DrawTextBold(opt.name, int32(bRec.X)+int32(bW/2)-int32(e.UI.MeasureTextBold(opt.name, 11.5)/2), int32(bRec.Y)+9, 11.5, ui.ColWhite)

		if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			layer.ShowBlink = opt.val
		}
	}

	y += 94

	// Talk Card (Generous breathing room)
	talkCard := rl.NewRectangle(rightRec.X+2, float32(y), rightRec.Width-14, 86)
	ui.DrawCard(talkCard, false, false)
	e.UI.DrawTextBold(i18n.T("label_vis_talk"), int32(talkCard.X)+14, int32(talkCard.Y)+10, 13.5, ui.ColSkyBlue)

	talkOpts := []struct {
		val  int
		name string
	}{
		{0, i18n.T("opt_vis_always")},
		{1, i18n.T("opt_vis_talk_quiet")},
		{2, i18n.T("opt_vis_talk_speaking")},
	}

	for i, opt := range talkOpts {
		tRec := rl.NewRectangle(talkCard.X+14+float32(i)*(bW+6), talkCard.Y+40, bW, 34)
		isCur := (layer.ShowTalk == opt.val)
		hovered := rl.CheckCollisionPointRec(mousePos, tRec)

		ui.DrawCard(tRec, hovered, isCur)
		e.UI.DrawTextBold(opt.name, int32(tRec.X)+int32(bW/2)-int32(e.UI.MeasureTextBold(opt.name, 11.5)/2), int32(tRec.Y)+9, 11.5, ui.ColWhite)

		if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			layer.ShowTalk = opt.val
		}
	}

	y += 94

	// Costumes Card
	costumeCard := rl.NewRectangle(rightRec.X+2, float32(y), rightRec.Width-14, 116)
	ui.DrawCard(costumeCard, false, false)
	e.UI.DrawTextBold(i18n.T("label_costumes_visibility"), int32(costumeCard.X)+14, int32(costumeCard.Y)+10, 13.5, ui.ColSkyBlue)

	cBtnW := (costumeCard.Width - 28 - 4*6) / 5
	for i := 1; i <= 10; i++ {
		row := float32((i - 1) / 5)
		col := float32((i - 1) % 5)

		cRec := rl.NewRectangle(costumeCard.X+14+col*(cBtnW+6), costumeCard.Y+38+row*34, cBtnW, 30)
		isActive := layer.CostumeLayers[i-1] == 1
		hovered := rl.CheckCollisionPointRec(mousePos, cRec)

		ui.DrawCard(cRec, hovered, isActive)

		numCol := ui.ColTextMuted
		if isActive {
			numCol = ui.ColLime
		}
		e.UI.DrawTextBold(fmt.Sprintf("%d", i), int32(cRec.X)+int32(cBtnW/2)-4, int32(cRec.Y)+7, 13.5, numCol)

		if hovered && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			if isActive {
				layer.CostumeLayers[i-1] = 0
			} else {
				layer.CostumeLayers[i-1] = 1
			}
		}
	}
}

func (e *EditorState) drawPhysicsTab(rightRec rl.Rectangle, startY int32, layer *model.Layer, mousePos rl.Vector2) {
	y := startY + 6

	// Wobble & Damping Card
	wobCard := rl.NewRectangle(rightRec.X+2, float32(y), rightRec.Width-14, 146)
	ui.DrawCard(wobCard, false, false)
	e.UI.DrawTextBold(i18n.T("label_wobble_inertia"), int32(wobCard.X)+14, int32(wobCard.Y)+10, 13.5, ui.ColSkyBlue)

	e.UI.DrawText(fmt.Sprintf(i18n.T("label_rot_drag_val"), layer.RotDrag), int32(wobCard.X)+14, int32(wobCard.Y)+34, 11.5, ui.ColTextBody)
	rdRec := rl.NewRectangle(wobCard.X+14, wobCard.Y+52, wobCard.Width-28, 10)
	layer.RotDrag = e.UI.DrawSliderControl(rdRec, layer.RotDrag, 0.0, 1.0, mousePos, ui.ColSkyBlue)

	e.UI.DrawText(fmt.Sprintf(i18n.T("label_rot_limit_val"), layer.RLimitMin, layer.RLimitMax), int32(wobCard.X)+14, int32(wobCard.Y)+80, 11.5, ui.ColTextBody)
	clampRec := rl.NewRectangle(wobCard.X+14, wobCard.Y+100, wobCard.Width-28, 10)
	angle := e.UI.DrawSliderControl(clampRec, layer.RLimitMax, 0.0, 90.0, mousePos, ui.ColSkyBlue)
	layer.RLimitMin = -angle
	layer.RLimitMax = angle

	y += 154

	// Breathing & Stretch Card
	breathCard := rl.NewRectangle(rightRec.X+2, float32(y), rightRec.Width-14, 186)
	ui.DrawCard(breathCard, false, false)
	e.UI.DrawTextBold(i18n.T("label_breathing_stretch"), int32(breathCard.X)+14, int32(breathCard.Y)+10, 13.5, ui.ColSkyBlue)

	e.UI.DrawText(fmt.Sprintf(i18n.T("label_idle_wave_val"), layer.YAmp), int32(breathCard.X)+14, int32(breathCard.Y)+34, 11.5, ui.ColTextBody)
	ampRec := rl.NewRectangle(breathCard.X+14, breathCard.Y+52, breathCard.Width-28, 10)
	layer.YAmp = e.UI.DrawSliderControl(ampRec, layer.YAmp, 0.0, 30.0, mousePos, ui.ColSkyBlue)
	layer.YFrq = 1.5

	e.UI.DrawText(fmt.Sprintf(i18n.T("label_stretch_val"), layer.StretchAmount), int32(breathCard.X)+14, int32(breathCard.Y)+80, 11.5, ui.ColTextBody)
	strRec := rl.NewRectangle(breathCard.X+14, breathCard.Y+100, breathCard.Width-28, 10)
	layer.StretchAmount = e.UI.DrawSliderControl(strRec, layer.StretchAmount, 0.0, 2.0, mousePos, ui.ColSkyBlue)

	toggleRec := rl.NewRectangle(breathCard.X+14, breathCard.Y+136, breathCard.Width-28, 28)
	layer.IgnoreBounce = e.UI.DrawToggle(toggleRec, i18n.T("label_ignore_bounce"), layer.IgnoreBounce, mousePos)
}

func (e *EditorState) drawSpriteTab(rightRec rl.Rectangle, startY int32, layer *model.Layer, mousePos rl.Vector2) {
	y := startY + 6

	spriteCard := rl.NewRectangle(rightRec.X+2, float32(y), rightRec.Width-14, 166)
	ui.DrawCard(spriteCard, false, false)
	e.UI.DrawTextBold(i18n.T("label_spritesheet_config"), int32(spriteCard.X)+14, int32(spriteCard.Y)+10, 13.5, ui.ColSkyBlue)

	e.UI.DrawTextBold(fmt.Sprintf(i18n.T("label_frames_val"), layer.Frames), int32(spriteCard.X)+14, int32(spriteCard.Y)+38, 12.5, ui.ColTextBody)
	fMinus := rl.NewRectangle(spriteCard.X+spriteCard.Width-84, spriteCard.Y+30, 36, 32)
	fPlus := rl.NewRectangle(spriteCard.X+spriteCard.Width-44, spriteCard.Y+30, 36, 32)

	if rl.CheckCollisionPointRec(mousePos, fMinus) {
		rl.DrawRectangleRounded(fMinus, 0.35, 4, ui.ColCardHover)
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) && layer.Frames > 1 {
			layer.Frames--
		}
	} else {
		rl.DrawRectangleRounded(fMinus, 0.35, 4, ui.ColCardBg)
	}
	rl.DrawRectangleRoundedLines(fMinus, 0.35, 4, ui.ColPanelBorder)
	e.UI.DrawTextBold("-", int32(fMinus.X)+14, int32(fMinus.Y)+7, 16, ui.ColWhite)

	if rl.CheckCollisionPointRec(mousePos, fPlus) {
		rl.DrawRectangleRounded(fPlus, 0.35, 4, ui.ColCardHover)
		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			layer.Frames++
		}
	} else {
		rl.DrawRectangleRounded(fPlus, 0.35, 4, ui.ColCardBg)
	}
	rl.DrawRectangleRoundedLines(fPlus, 0.35, 4, ui.ColPanelBorder)
	e.UI.DrawTextBold("+", int32(fPlus.X)+13, int32(fPlus.Y)+7, 16, ui.ColWhite)

	e.UI.DrawText(fmt.Sprintf(i18n.T("label_anim_speed_val"), layer.AnimSpeed), int32(spriteCard.X)+14, int32(spriteCard.Y)+84, 11.5, ui.ColTextBody)
	fpsRec := rl.NewRectangle(spriteCard.X+14, spriteCard.Y+108, spriteCard.Width-28, 10)
	layer.AnimSpeed = e.UI.DrawSliderControl(fpsRec, layer.AnimSpeed, 0.0, 30.0, mousePos, ui.ColSkyBlue)
}

// drawLayerGizmo renders a bounding box and interactive position handle over the selected layer or Root node.
func (e *EditorState) drawLayerGizmo(layer *model.Layer, scale float32, origin rl.Vector2, mousePos rl.Vector2) {
	// If Root / Pivot Central is selected, show global origin crosshair
	if e.SelectedLayerID == 0 || layer == nil {
		rl.DrawCircleLines(int32(origin.X), int32(origin.Y), 16, ui.ColOrange)
		rl.DrawLine(int32(origin.X)-24, int32(origin.Y), int32(origin.X)+24, int32(origin.Y), ui.ColYellow)
		rl.DrawLine(int32(origin.X), int32(origin.Y)-24, int32(origin.X), int32(origin.Y)+24, ui.ColYellow)
		rl.DrawCircle(int32(origin.X), int32(origin.Y), 6, ui.ColYellow)
		e.UI.DrawBadge(origin.X-55, origin.Y+22, "⭐ Pivot Central (0,0)", ui.ColPanelBg, ui.ColYellow)
		return
	}

	if e.TextureCache == nil || e.Avatar == nil {
		return
	}

	tex, ok := e.TextureCache.GetTexture(layer.Identification)
	if !ok || tex.Width == 0 || tex.Height == 0 {
		return
	}

	// Compute world transform matching renderer.go exactly
	transforms := render.ComputeWorldTransforms(
		e.Avatar,
		origin,
		scale,
		0,
		nil,
		nil,
	)

	t, ok := transforms[layer.Identification]
	if !ok {
		t = render.LayerTransform{
			WorldPos: rl.Vector2{
				X: origin.X + (layer.Pos.X * scale),
				Y: origin.Y + (layer.Pos.Y * scale),
			},
			Scale:    scale,
			Rotation: 0,
		}
	}

	frames := layer.Frames
	if frames < 1 {
		frames = 1
	}
	fW := float32(tex.Width) / float32(frames)
	fH := float32(tex.Height)

	// Exact top-left screen position matching render/renderer.go
	topLeftX := t.WorldPos.X + (layer.Offset.X * t.Scale)
	topLeftY := t.WorldPos.Y + (layer.Offset.Y * t.Scale)

	// Compute selection box based on non-transparent sprite pixel boundaries
	minX := float32(0)
	minY := float32(0)
	maxX := fW
	maxY := fH
	if layer.HasContentBounds {
		minX = layer.ContentMinX
		minY = layer.ContentMinY
		maxX = layer.ContentMaxX
		maxY = layer.ContentMaxY
	} else if len(layer.ImageData) > 0 {
		layer.UpdateContentBounds()
		if layer.HasContentBounds {
			minX = layer.ContentMinX
			minY = layer.ContentMinY
			maxX = layer.ContentMaxX
			maxY = layer.ContentMaxY
		}
	}

	boxX := topLeftX + (minX * t.Scale)
	boxY := topLeftY + (minY * t.Scale)
	boxW := (maxX - minX) * t.Scale
	boxH := (maxY - minY) * t.Scale
	boxRec := rl.NewRectangle(boxX, boxY, boxW, boxH)

	// 1. Semi-transparent fill highlight
	rl.DrawRectangleRec(boxRec, rl.NewColor(41, 173, 255, 30))

	// 2. Crisp Bounding Box around actual sprite pixels
	rl.DrawRectangleLinesEx(boxRec, 1.5, ui.ColSkyBlue)

	// 3. 4 Corner indicator squares
	cSize := float32(6)
	rl.DrawRectangleRec(rl.NewRectangle(boxRec.X-3, boxRec.Y-3, cSize, cSize), ui.ColWhite)
	rl.DrawRectangleRec(rl.NewRectangle(boxRec.X+boxRec.Width-3, boxRec.Y-3, cSize, cSize), ui.ColWhite)
	rl.DrawRectangleRec(rl.NewRectangle(boxRec.X-3, boxRec.Y+boxRec.Height-3, cSize, cSize), ui.ColWhite)
	rl.DrawRectangleRec(rl.NewRectangle(boxRec.X+boxRec.Width-3, boxRec.Y+boxRec.Height-3, cSize, cSize), ui.ColWhite)

	// 4. Center / Drag Anchor Handle
	scrX := t.WorldPos.X
	scrY := t.WorldPos.Y
	pivotRec := rl.NewRectangle(scrX-12, scrY-12, 24, 24)
	pivotHovered := rl.CheckCollisionPointRec(mousePos, pivotRec) || rl.CheckCollisionPointRec(mousePos, boxRec)

	if (pivotHovered || rl.CheckCollisionPointRec(mousePos, boxRec)) && rl.IsMouseButtonPressed(rl.MouseLeftButton) && !e.IsDraggingLeftScroll && !e.IsDraggingRightScroll {
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

	handleCol := ui.ColSkyBlue
	if e.IsDraggingPos || pivotHovered {
		handleCol = ui.ColLime
	}

	rl.DrawCircle(int32(scrX), int32(scrY), 8, handleCol)
	rl.DrawLine(int32(scrX)-14, int32(scrY), int32(scrX)+14, int32(scrY), rl.White)
	rl.DrawLine(int32(scrX), int32(scrY)-14, int32(scrX), int32(scrY)+14, rl.White)
}
