package anim

// BounceController simulates the avatar's global jump/bounce physics.
type BounceController struct {
	Y              float32 // current Y offset (negative = in the air)
	YVel           float32 // vertical velocity
	Gravity        float32 // gravity acceleration (default 1000.0)
	BounceStrength float32 // initial jump velocity (default 250.0)
	DeltaChange    float32 // displacement in current frame (used by physics inertia)
}

// NewBounceController creates a new bounce controller.
func NewBounceController() *BounceController {
	return &BounceController{
		Y:              0,
		YVel:           0,
		Gravity:        1000.0,
		BounceStrength: 250.0,
	}
}

// Trigger starts a jump/bounce impulse (e.g. when starting to speak or changing costume).
func (bc *BounceController) Trigger() {
	if bc.Y >= 0 {
		bc.YVel = -bc.BounceStrength
	}
}

// Update advances the bounce physics.
func (bc *BounceController) Update(dt float32) {
	prevY := bc.Y

	bc.Y += bc.YVel * dt
	if bc.Y > 0 {
		bc.Y = 0
		bc.YVel = 0
	} else {
		bc.YVel += bc.Gravity * dt
	}

	bc.DeltaChange = prevY - bc.Y
}
