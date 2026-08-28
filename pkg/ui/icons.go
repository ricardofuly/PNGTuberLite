package ui

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"pngtuber-lite/assets"
)

// Icon IDs mapped to individual embedded icon textures
const (
	IconAdd = iota + 1
	IconAudio
	IconAvatar
	IconChange
	IconClose
	IconDelete
	IconDuplicate
	IconEditor
	IconEnable
	IconFavorite
	IconPhysics
	IconKeys
	IconLogo
	IconOBS
	IconOpenEditor
	IconPNGFile
	IconRemove
	IconRestart
	IconRestore
	IconCostumes
	IconSave
	IconSelected
	IconSettings
	IconUpdate
	IconLanguage

	// Semantic & Compatibility Aliases
	IconLang      = IconLanguage
	IconDownload  = IconUpdate
	IconConfig    = IconSettings
	IconFileText  = IconOpenEditor
	IconFileImage = IconPNGFile
	IconReset     = IconRestore
	IconPlus      = IconAdd
	IconGrid      = IconCostumes
	IconGamepad   = IconKeys
	IconMic       = IconAudio
	IconSelection = IconOBS
	IconClock     = IconRestore
	IconSearch    = IconChange
)

// IconManager handles loading, caching, and drawing individual icons and application logo.
type IconManager struct {
	Textures    map[int]rl.Texture2D
	LogoTexture rl.Texture2D
	Loaded      bool
}

var GlobalIcons = &IconManager{
	Textures: make(map[int]rl.Texture2D),
}

// Load loads all individual icon textures and application logo into GPU memory.
func (im *IconManager) Load() error {
	if !rl.IsWindowReady() {
		return nil
	}

	rawIcons := map[int][]byte{
		IconAdd:        assets.IconAddPNG,
		IconAudio:      assets.IconAudioPNG,
		IconAvatar:     assets.IconAvatarPNG,
		IconChange:     assets.IconChangePNG,
		IconClose:      assets.IconClosePNG,
		IconDelete:     assets.IconDeletePNG,
		IconDuplicate:  assets.IconDuplicatePNG,
		IconEditor:     assets.IconEditorPNG,
		IconEnable:     assets.IconEnablePNG,
		IconFavorite:   assets.IconFavoritePNG,
		IconPhysics:    assets.IconPhysicsPNG,
		IconKeys:       assets.IconKeysPNG,
		IconLogo:       assets.AppLogoPNG,
		IconOBS:        assets.IconOBSPNG,
		IconOpenEditor: assets.IconOpenEditorPNG,
		IconPNGFile:    assets.IconPNGFilePNG,
		IconRemove:     assets.IconRemovePNG,
		IconRestart:    assets.IconRestartPNG,
		IconRestore:    assets.IconRestorePNG,
		IconCostumes:   assets.IconCostumesPNG,
		IconSave:       assets.IconSavePNG,
		IconSelected:   assets.IconSelectedPNG,
		IconSettings:   assets.IconSettingsPNG,
		IconUpdate:     assets.IconUpdatePNG,
		IconLanguage:   assets.IconLanguagePNG,
	}

	for id, data := range rawIcons {
		if len(data) == 0 {
			continue
		}
		img := rl.LoadImageFromMemory(".png", data, int32(len(data)))
		if img.Width > 0 {
			tex := rl.LoadTextureFromImage(img)
			rl.SetTextureFilter(tex, rl.FilterBilinear)
			im.Textures[id] = tex
			rl.UnloadImage(img)
		}
	}

	// Load Logo and Set Window Icon
	if len(assets.AppLogoPNG) > 0 {
		logoImg := rl.LoadImageFromMemory(".png", assets.AppLogoPNG, int32(len(assets.AppLogoPNG)))
		if logoImg.Width > 0 {
			rl.SetWindowIcon(*logoImg)
			im.LogoTexture = rl.LoadTextureFromImage(logoImg)
			rl.SetTextureFilter(im.LogoTexture, rl.FilterBilinear)
			rl.UnloadImage(logoImg)
		}
	}

	im.Loaded = true
	return nil
}

// Unload releases all GPU textures.
func (im *IconManager) Unload() {
	if !rl.IsWindowReady() {
		return
	}
	for id, tex := range im.Textures {
		if tex.ID > 0 {
			rl.UnloadTexture(tex)
		}
		delete(im.Textures, id)
	}
	if im.LogoTexture.ID > 0 {
		rl.UnloadTexture(im.LogoTexture)
	}
	im.Loaded = false
}

// DrawIcon renders an icon at the specified position, scale and tint color.
func (im *IconManager) DrawIcon(iconID int, x, y float32, size float32, tint rl.Color) {
	if !im.Loaded {
		return
	}
	tex, ok := im.Textures[iconID]
	if !ok || tex.ID == 0 {
		return
	}
	src := rl.NewRectangle(0, 0, float32(tex.Width), float32(tex.Height))
	dest := rl.NewRectangle(x, y, size, size)
	rl.DrawTexturePro(tex, src, dest, rl.Vector2{X: 0, Y: 0}, 0, tint)
}

// DrawLogo renders the application logo with aspect ratio preserved.
func (im *IconManager) DrawLogo(x, y float32, width, height float32, tint rl.Color) {
	if !im.Loaded {
		return
	}
	tex := im.LogoTexture
	if tex.ID == 0 {
		if t, ok := im.Textures[IconLogo]; ok && t.ID > 0 {
			tex = t
		}
	}
	if tex.ID == 0 {
		return
	}
	src := rl.NewRectangle(0, 0, float32(tex.Width), float32(tex.Height))
	dest := rl.NewRectangle(x, y, width, height)
	rl.DrawTexturePro(tex, src, dest, rl.Vector2{X: 0, Y: 0}, 0, tint)
}
