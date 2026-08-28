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
	FlipHorizontal bool
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
	TextureCache     *TextureCache
	cachedTransforms map[int64]LayerTransform
}

// NewRenderer creates a new avatar renderer.
func NewRenderer(cache *TextureCache) *Renderer {
	return &Renderer{
		TextureCache:     cache,
		cachedTransforms: make(map[int64]LayerTransform, 32),
	}
}

// Draw renders the entire avatar in proper Z-index order with hierachical transforms applied.
func (r *Renderer) Draw(state *RenderState) {
	if state == nil || state.Avatar == nil || len(state.Avatar.Layers) == 0 {
		return
	}

	if r.cachedTransforms == nil {
		r.cachedTransforms = make(map[int64]LayerTransform, len(state.Avatar.Layers))
	}

	// Clear leftover keys if layer count changed
	if len(r.cachedTransforms) != len(state.Avatar.Layers) {
		clear(r.cachedTransforms)
	}

	ComputeWorldTransformsBuffer(
		state.Avatar,
		state.Origin,
		state.Scale,
		state.GlobalBounceY,
		state.LayerOffsets,
		state.LayerRotations,
		state.FlipHorizontal,
		r.cachedTransforms,
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

		transform, ok := r.cachedTransforms[layer.Identification]
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

		srcW := frameWidth
		if state.FlipHorizontal {
			srcW = -frameWidth
		}

		// Source rectangle inside texture (negative width flips horizontally in Raylib)
		srcRec := rl.NewRectangle(
			float32(currentFrame)*frameWidth,
			0,
			srcW,
			frameHeight,
		)

		// Stretch factors (1.0 baseline + dynamic deformation delta)
		stretchX := float32(1.0)
		stretchY := float32(1.0)
		if state.LayerStretches != nil {
			if s, exists := state.LayerStretches[layer.Identification]; exists {
				stretchX += s.X
				stretchY += s.Y
				if stretchX < 0.05 {
					stretchX = 0.05
				}
				if stretchY < 0.05 {
					stretchY = 0.05
				}
			}
		}

		destWidth := frameWidth * transform.Scale * stretchX
		destHeight := frameHeight * transform.Scale * stretchY

		destRec := rl.NewRectangle(
			transform.WorldPos.X,
			transform.WorldPos.Y,
			destWidth,
			destHeight,
		)

		// Pivot (Offset) point around which rotation and scaling occurs (Raylib origin is Pivot - TopLeft)
		pivotX := -layer.Offset.X * transform.Scale * stretchX
		pivotY := -layer.Offset.Y * transform.Scale * stretchY
		if state.FlipHorizontal {
			pivotX = -pivotX
		}
		origin := rl.Vector2{
			X: pivotX,
			Y: pivotY,
		}

		// Render layer with sub-pixel precision and bilinear filtering
		rl.DrawTexturePro(
			tex,
			srcRec,
			destRec,
			origin,
			transform.Rotation,
			rl.White,
		)
	}
}
