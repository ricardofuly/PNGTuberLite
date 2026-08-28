package render

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"pngtuber-lite/pkg/model"
)

// RenderState contains all dynamic inputs required to draw a frame.
type RenderState struct {
	Avatar         *model.Avatar
	Origin         rl.Vector2
	Scale          float32
	GlobalBounceY  float32
	Costume        int
	IsBlinking     bool
	IsTalking      bool
	LayerOffsets   map[int64]model.Vector2
	LayerRotations map[int64]float32
	LayerFrames    map[int64]int
	LayerStretches map[int64]model.Vector2
}

// Renderer handles drawing layers with Raylib.
type Renderer struct {
	TextureCache *TextureCache
}

// NewRenderer creates a new avatar renderer.
func NewRenderer(cache *TextureCache) *Renderer {
	return &Renderer{
		TextureCache: cache,
	}
}

// Draw renders the entire avatar in proper Z-index order with hierachical transforms applied.
func (r *Renderer) Draw(state *RenderState) {
	if state == nil || state.Avatar == nil || len(state.Avatar.Layers) == 0 {
		return
	}

	transforms := ComputeWorldTransforms(
		state.Avatar,
		state.Origin,
		state.Scale,
		state.GlobalBounceY,
		state.LayerOffsets,
		state.LayerRotations,
	)

	// Draw layers in sorted ZIndex order
	for _, layer := range state.Avatar.DrawOrder {
		// 1. Check visibility
		if !layer.IsVisible(state.Costume, state.IsBlinking, state.IsTalking) {
			continue
		}

		// 2. Fetch GPU texture
		tex, ok := r.TextureCache.GetTexture(layer.Identification)
		if !ok || tex.Width == 0 || tex.Height == 0 {
			continue
		}

		transform, ok := transforms[layer.Identification]
		if !ok {
			continue
		}

		// 3. Determine spritesheet frame
		frames := layer.Frames
		if frames < 1 {
			frames = 1
		}

		currentFrame := 0
		if state.LayerFrames != nil {
			if f, exists := state.LayerFrames[layer.Identification]; exists {
				currentFrame = f % frames
			}
		}

		frameWidth := float32(tex.Width) / float32(frames)
		frameHeight := float32(tex.Height)

		// Source rectangle inside texture
		srcRec := rl.NewRectangle(
			float32(currentFrame)*frameWidth,
			0,
			frameWidth,
			frameHeight,
		)

		// Stretch factors
		stretchX := float32(1.0)
		stretchY := float32(1.0)
		if state.LayerStretches != nil {
			if s, exists := state.LayerStretches[layer.Identification]; exists {
				stretchX += s.X
				stretchY += s.Y
			}
		}

		destWidth := frameWidth * transform.Scale * stretchX
		destHeight := frameHeight * transform.Scale * stretchY

		// Destination rectangle on screen
		destRec := rl.NewRectangle(
			transform.WorldPos.X,
			transform.WorldPos.Y,
			destWidth,
			destHeight,
		)

		// Pivot (origin relative to destination top-left).
		// By default Godot centers sprites at (width/2, height/2) adjusted by layer.Offset
		pivot := rl.Vector2{
			X: (destWidth * 0.5) - (layer.Offset.X * transform.Scale),
			Y: (destHeight * 0.5) - (layer.Offset.Y * transform.Scale),
		}

		// 4. Draw texture with rotation around pivot
		rl.DrawTexturePro(
			tex,
			srcRec,
			destRec,
			pivot,
			transform.Rotation,
			rl.White,
		)
	}
}
