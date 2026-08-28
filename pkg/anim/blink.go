package anim

import (
	"math/rand"
	"time"
)

// BlinkController controls natural, human-like blinking cycles of the avatar.
type BlinkController struct {
	IsBlinking   bool
	BlinkSpeed   float32 // Duration scaling multiplier (default 1.0)
	BlinkChance  int     // Compatibility factor
	timer        float32
	cooldown     float32
	rng          *rand.Rand
}

// NewBlinkController creates a new blink controller matching human rhythm (~14s intervals).
func NewBlinkController() *BlinkController {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &BlinkController{
		IsBlinking:  false,
		BlinkSpeed:  1.0,
		BlinkChance: 200,
		timer:       0,
		// Initial random interval between 4s and 12s
		cooldown: 4.0 + rng.Float32()*8.0,
		rng:      rng,
	}
}

// Update advances the blink timer and triggers human-like blinks (~10s to 18s intervals).
func (bc *BlinkController) Update(dt float32) {
	if bc.IsBlinking {
		bc.timer -= dt
		if bc.timer <= 0 {
			bc.IsBlinking = false
			// Human blink interval: 10s to 18s (average ~14 seconds)
			bc.cooldown = 10.0 + bc.rng.Float32()*8.0
		}
		return
	}

	if bc.cooldown > 0 {
		bc.cooldown -= dt
		return
	}

	// Trigger quick human-like blink (around 0.09s duration)
	bc.IsBlinking = true
	duration := float32(0.09)
	if bc.BlinkSpeed > 0 {
		duration = duration * bc.BlinkSpeed
	}
	bc.timer = duration
}

// ForceBlink immediately triggers a blink animation.
func (bc *BlinkController) ForceBlink() {
	bc.IsBlinking = true
	bc.timer = float32(0.09) * bc.BlinkSpeed
}
