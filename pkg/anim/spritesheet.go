package anim

import (
	"pngtuber-lite/pkg/model"
)

// SpriteSheetState tracks the active frame and timer for an animated layer.
type SpriteSheetState struct {
	CurrentFrame int
	Timer        float32
}

// SpriteSheetAnimator manages frame stepping for all spritesheet layers.
type SpriteSheetAnimator struct {
	states map[int64]*SpriteSheetState
}

// NewSpriteSheetAnimator creates a new spritesheet animator.
func NewSpriteSheetAnimator() *SpriteSheetAnimator {
	return &SpriteSheetAnimator{
		states: make(map[int64]*SpriteSheetState),
	}
}

// Update advances the animation timers for all animated layers.
func (sa *SpriteSheetAnimator) Update(avatar *model.Avatar, dt float32) {
	if avatar == nil {
		return
	}

	for id, layer := range avatar.Layers {
		if layer.Frames <= 1 || layer.AnimSpeed <= 0 {
			continue
		}

		st, exists := sa.states[id]
		if !exists {
			st = &SpriteSheetState{}
			sa.states[id] = st
		}

		// Frame advance rate: AnimSpeed frames per second
		st.Timer += dt * layer.AnimSpeed
		if st.Timer >= 1.0 {
			advances := int(st.Timer)
			st.CurrentFrame = (st.CurrentFrame + advances) % layer.Frames
			st.Timer -= float32(advances)
		}
	}
}

// GetCurrentFrame returns the active frame index for the specified layer.
func (sa *SpriteSheetAnimator) GetCurrentFrame(layerID int64) int {
	if st, exists := sa.states[layerID]; exists {
		return st.CurrentFrame
	}
	return 0
}
