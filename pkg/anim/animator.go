package anim

import (
	"pngtuber-lite/pkg/model"
)

// Animator coordinates all avatar animations (wobble, blink, bounce, spritesheets).
type Animator struct {
	Wobble      *WobbleSystem
	Blink       *BlinkController
	Bounce      *BounceController
	SpriteSheet *SpriteSheetAnimator
}

// NewAnimator creates a new animation manager.
func NewAnimator() *Animator {
	return &Animator{
		Wobble:      NewWobbleSystem(),
		Blink:       NewBlinkController(),
		Bounce:      NewBounceController(),
		SpriteSheet: NewSpriteSheetAnimator(),
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

// BuildLayerAnimationMaps generates the map lookups consumed by the renderer.
func (a *Animator) BuildLayerAnimationMaps(avatar *model.Avatar) (
	offsets map[int64]model.Vector2,
	rotations map[int64]float32,
	frames map[int64]int,
	stretches map[int64]model.Vector2,
) {
	if avatar == nil {
		return nil, nil, nil, nil
	}

	offsets = make(map[int64]model.Vector2, len(avatar.Layers))
	rotations = make(map[int64]float32, len(avatar.Layers))
	frames = make(map[int64]int, len(avatar.Layers))
	stretches = make(map[int64]model.Vector2, len(avatar.Layers))

	for id, layer := range avatar.Layers {
		ox, oy := a.Wobble.GetCalculatedOffset(layer)
		offsets[id] = model.Vector2{X: ox, Y: oy}

		rotations[id] = a.Wobble.GetCalculatedRotation(layer)
		frames[id] = a.SpriteSheet.GetCurrentFrame(id)

		sx, sy := a.Wobble.GetCalculatedStretch(layer)
		stretches[id] = model.Vector2{X: sx, Y: sy}
	}

	return offsets, rotations, frames, stretches
}

