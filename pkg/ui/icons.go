package ui

import (
	"encoding/json"
	"fmt"
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Icon IDs mapped from the sprite atlas
const (
	IconPlay          = 1
	IconVolume        = 10
	IconMute          = 12
	IconPlus          = 21
	IconGrid          = 28
	IconLayers        = 30
	IconDuplicate     = 34
	IconInfo          = 35
	IconReset         = 36
	IconClock         = 37
	IconConfig        = 40
	IconMic           = 52
	IconMicMute       = 53
	IconHeart         = 54
	IconAvatar        = 55
	IconSearch        = 59
	IconTarget        = 66
	IconSelection     = 69
	IconMusic         = 73
	IconDownload      = 76
	IconPhysics       = 82
	IconFileText      = 84
	IconFileImage     = 87
	IconFolder        = 89
	IconSave          = 96
	IconGamepad       = 99
	IconClose         = 23 // Question/Cross alternative or drawn
)

type atlasJSON struct {
	Frames map[string]struct {
		Frame struct {
			X int `json:"x"`
			Y int `json:"y"`
			W int `json:"w"`
			H int `json:"h"`
		} `json:"frame"`
	} `json:"frames"`
}

// IconManager handles loading and drawing atlas icons and application logo.
type IconManager struct {
	AtlasTexture rl.Texture2D
	LogoTexture  rl.Texture2D
	Frames       map[int]rl.Rectangle
	Loaded       bool
}

var GlobalIcons = &IconManager{
	Frames: make(map[int]rl.Rectangle),
}

// Load loads the icon atlas and logo textures into GPU memory.
func (im *IconManager) Load() error {
	if !rl.IsWindowReady() {
		return nil
	}

	// 1. Load Sprite Atlas JSON
	jsonPaths := []string{
		"assets/icons/IconsFlat-32.json",
		"../assets/icons/IconsFlat-32.json",
	}
	var jsonData []byte
	for _, p := range jsonPaths {
		if d, err := os.ReadFile(p); err == nil {
			jsonData = d
			break
		}
	}

	if len(jsonData) > 0 {
		var a atlasJSON
		if err := json.Unmarshal(jsonData, &a); err == nil {
			for i := 0; i < 100; i++ {
				key := fmt.Sprintf("IconsFlat-32 %d.ase", i)
				if f, ok := a.Frames[key]; ok {
					im.Frames[i] = rl.NewRectangle(float32(f.Frame.X), float32(f.Frame.Y), float32(f.Frame.W), float32(f.Frame.H))
				}
			}
		}
	}

	// Fallback to mathematical grid if JSON parsing had missing frames
	if len(im.Frames) == 0 {
		for i := 0; i < 100; i++ {
			row := float32(i / 10)
			col := float32(i % 10)
			im.Frames[i] = rl.NewRectangle(col*32, row*32, 32, 32)
		}
	}

	// 2. Load Sprite Atlas Texture
	pngPaths := []string{
		"assets/icons/IconsFlat-32.png",
		"../assets/icons/IconsFlat-32.png",
	}
	for _, p := range pngPaths {
		if _, err := os.Stat(p); err == nil {
			im.AtlasTexture = rl.LoadTexture(p)
			rl.SetTextureFilter(im.AtlasTexture, rl.FilterPoint) // Crisp pixel art icons
			break
		}
	}

	// 3. Load Application Logo
	logoPaths := []string{
		"assets/icons/logo.png",
		"../assets/icons/logo.png",
	}
	for _, p := range logoPaths {
		if _, err := os.Stat(p); err == nil {
			logoImg := rl.LoadImage(p)
			if logoImg.Width > 0 {
				rl.SetWindowIcon(*logoImg)
				im.LogoTexture = rl.LoadTextureFromImage(logoImg)
				rl.SetTextureFilter(im.LogoTexture, rl.FilterBilinear)
				rl.UnloadImage(logoImg)
				break
			}
		}
	}

	im.Loaded = true
	return nil
}

// Unload releases GPU textures.
func (im *IconManager) Unload() {
	if !rl.IsWindowReady() {
		return
	}
	if im.AtlasTexture.ID > 0 {
		rl.UnloadTexture(im.AtlasTexture)
	}
	if im.LogoTexture.ID > 0 {
		rl.UnloadTexture(im.LogoTexture)
	}
	im.Loaded = false
}

// DrawIcon renders an icon from the atlas at the specified position, scale and tint color.
func (im *IconManager) DrawIcon(iconID int, x, y float32, size float32, tint rl.Color) {
	if !im.Loaded || im.AtlasTexture.ID == 0 {
		return
	}
	src, ok := im.Frames[iconID]
	if !ok {
		src = rl.NewRectangle(0, 0, 32, 32)
	}
	dest := rl.NewRectangle(x, y, size, size)
	rl.DrawTexturePro(im.AtlasTexture, src, dest, rl.Vector2{X: 0, Y: 0}, 0, tint)
}

// DrawLogo renders the application logo with aspect ratio preserved.
func (im *IconManager) DrawLogo(x, y float32, width, height float32, tint rl.Color) {
	if !im.Loaded || im.LogoTexture.ID == 0 {
		return
	}
	src := rl.NewRectangle(0, 0, float32(im.LogoTexture.Width), float32(im.LogoTexture.Height))
	dest := rl.NewRectangle(x, y, width, height)
	rl.DrawTexturePro(im.LogoTexture, src, dest, rl.Vector2{X: 0, Y: 0}, 0, tint)
}
