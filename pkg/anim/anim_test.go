package anim

import (
	"testing"

	"pngtuber-lite/pkg/model"
)

func TestBlinkController(t *testing.T) {
	bc := NewBlinkController()
	bc.BlinkChance = 1 // 100% chance to trigger
	bc.cooldown = 0

	bc.Update(0.016)
	if !bc.IsBlinking {
		t.Errorf("expected blink to be triggered")
	}

	// Advance time past blink duration
	bc.Update(0.3)
	if bc.IsBlinking {
		t.Errorf("expected blink to end")
	}
}

func TestBounceController(t *testing.T) {
	bounce := NewBounceController()
	bounce.Trigger()

	if bounce.YVel >= 0 {
		t.Errorf("expected negative upward velocity after trigger, got %f", bounce.YVel)
	}

	bounce.Update(0.016)
	if bounce.Y >= 0 {
		t.Errorf("expected Y to be negative (in air), got %f", bounce.Y)
	}

	// Advance until landing
	for i := 0; i < 60; i++ {
		bounce.Update(0.016)
	}

	if bounce.Y != 0 {
		t.Errorf("expected avatar to land on ground (Y=0), got %f", bounce.Y)
	}
}

func TestSpriteSheetAnimator(t *testing.T) {
	avatar := model.NewAvatar()
	layer := model.NewDefaultLayer(1)
	layer.Frames = 4
	layer.AnimSpeed = 2.0 // 2 frames per second
	avatar.AddLayer(layer)

	anim := NewSpriteSheetAnimator()

	if frame := anim.GetCurrentFrame(1); frame != 0 {
		t.Errorf("initial frame should be 0, got %d", frame)
	}

	// Advance 0.5s (should advance 1 frame)
	anim.Update(avatar, 0.55)
	if frame := anim.GetCurrentFrame(1); frame != 1 {
		t.Errorf("frame after 0.5s should be 1, got %d", frame)
	}
}

func TestWobbleAngleLimits(t *testing.T) {
	avatar := model.NewAvatar()
	layer := model.NewDefaultLayer(1)
	layer.RLimitMin = -45
	layer.RLimitMax = 45
	layer.RotDrag = 0.1
	avatar.AddLayer(layer)
	avatar.BuildHierarchy()

	wobble := NewWobbleSystem()
	st := wobble.GetState(1)
	st.Angle = 90 // out of bounds

	wobble.Update(avatar, 0.016, 0)
	if st.Angle > 45 {
		t.Errorf("angle %f exceeded RLimitMax 45", st.Angle)
	}
}
