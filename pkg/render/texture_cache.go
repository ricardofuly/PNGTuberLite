package render

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"
	"pngtuber-lite/pkg/model"
)

// TextureCache manages loading, storing, and releasing Raylib GPU textures for avatar layers.
type TextureCache struct {
	textures map[int64]rl.Texture2D
}

// NewTextureCache creates a new texture cache.
func NewTextureCache() *TextureCache {
	return &TextureCache{
		textures: make(map[int64]rl.Texture2D),
	}
}

// LoadAvatarTextures loads GPU textures for all layers in the given avatar.
// Note: Must be called on the main thread after Raylib window initialization.
func (tc *TextureCache) LoadAvatarTextures(avatar *model.Avatar) error {
	tc.UnloadAll()

	if avatar == nil || !rl.IsWindowReady() {
		return nil
	}

	for id, layer := range avatar.Layers {
		if len(layer.ImageData) == 0 {
			continue
		}

		img := rl.LoadImageFromMemory(".png", layer.ImageData, int32(len(layer.ImageData)))
		if img.Width == 0 || img.Height == 0 {
			rl.UnloadImage(img)
			return fmt.Errorf("failed to load image for layer %d", id)
		}

		tex := rl.LoadTextureFromImage(img)
		rl.UnloadImage(img)

		// Set bilinear texture filtering and clamp-to-edge wrapping to avoid texture bleeding artifacts
		rl.SetTextureFilter(tex, rl.FilterBilinear)
		rl.SetTextureWrap(tex, rl.WrapClamp)
		tc.textures[id] = tex
	}

	return nil
}

// GetTexture returns the GPU texture for the specified layer ID.
func (tc *TextureCache) GetTexture(layerID int64) (rl.Texture2D, bool) {
	tex, ok := tc.textures[layerID]
	return tex, ok
}

// UnloadAll frees all allocated GPU textures from VRAM.
func (tc *TextureCache) UnloadAll() {
	if !rl.IsWindowReady() {
		tc.textures = make(map[int64]rl.Texture2D)
		return
	}
	for id, tex := range tc.textures {
		rl.UnloadTexture(tex)
		delete(tc.textures, id)
	}
}

// GetTextureCount returns the number of active GPU textures in cache.
func (tc *TextureCache) GetTextureCount() int {
	return len(tc.textures)
}

// GetEstimatedVRAM calculates the total uncompressed GPU VRAM memory used by textures in bytes.
func (tc *TextureCache) GetEstimatedVRAM() int64 {
	var total int64
	for _, tex := range tc.textures {
		total += int64(tex.Width) * int64(tex.Height) * 4
	}
	return total
}
