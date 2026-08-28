package anim

import (
	"pngtuber-lite/pkg/model"
)

// Animator coordinates all avatar animations (wobble, blink, bounce, spritesheets).
type Animator struct {
	Wobble          *WobbleSystem
	Blink           *BlinkController
	Bounce          *BounceController
	SpriteSheet     *SpriteSheetAnimator
	cachedOffsets   map[int64]model.Vector2
	cachedRotations map[int64]float32
	cachedFrames    map[int64]int
	cachedStretches map[int64]model.Vector2
}

// NewAnimator creates a new animation manager.
func NewAnimator() *Animator {
	return &Animator{
		Wobble:          NewWobbleSystem(),
		Blink:           NewBlinkController(),
		Bounce:          NewBounceController(),
		SpriteSheet:     NewSpriteSheetAnimator(),
		cachedOffsets:   make(map[int64]model.Vector2, 32),
		cachedRotations: make(map[int64]float32, 32),
		cachedFrames:    make(map[int64]int, 32),
		cachedStretches: make(map[int64]model.Vector2, 32),
	}
}

// Update advances all animations by fixed timestep dt.
func (a *Animator) Update(avatar *model.Avatar, dt float32) {
	// 1. Advance global bounce
	a.Bounce.Update(dt)

	// 2. Advance blink timer
	a.Blink.Update(dt)

	// 3. Advance spritesheet frames
	a.SpriteSheet.Update(avatar, dt)

	// 4. Advance physics wobble with hierarchical motion propagation and inertia
	a.Wobble.Update(avatar, dt, a.Bounce.DeltaChange)
}

// BuildLayerAnimationMaps generates the map lookups consumed by the renderer with zero heap allocations.
func (a *Animator) BuildLayerAnimationMaps(avatar *model.Avatar) (
	offsets map[int64]model.Vector2,
	rotations map[int64]float32,
	frames map[int64]int,
	stretches map[int64]model.Vector2,
) {
	if avatar == nil {
		return nil, nil, nil, nil
	}

	// Ensure pre-allocated buffer maps exist
	if a.cachedOffsets == nil {
		a.cachedOffsets = make(map[int64]model.Vector2, len(avatar.Layers))
		a.cachedRotations = make(map[int64]float32, len(avatar.Layers))
		a.cachedFrames = make(map[int64]int, len(avatar.Layers))
		a.cachedStretches = make(map[int64]model.Vector2, len(avatar.Layers))
	}

	// Clear leftover keys if layer count changed
	if len(a.cachedOffsets) != len(avatar.Layers) {
		clear(a.cachedOffsets)
		clear(a.cachedRotations)
		clear(a.cachedFrames)
		clear(a.cachedStretches)
	}

	for id, layer := range avatar.Layers {
		ox, oy := a.Wobble.GetCalculatedOffset(layer)
		a.cachedOffsets[id] = model.Vector2{X: ox, Y: oy}

		a.cachedRotations[id] = a.Wobble.GetCalculatedRotation(layer)
		a.cachedFrames[id] = a.SpriteSheet.GetCurrentFrame(id)

		sx, sy := a.Wobble.GetCalculatedStretch(layer)
		a.cachedStretches[id] = model.Vector2{X: sx, Y: sy}
	}

	return a.cachedOffsets, a.cachedRotations, a.cachedFrames, a.cachedStretches
}
