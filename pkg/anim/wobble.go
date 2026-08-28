package anim

import (
	"math"

	"pngtuber-lite/pkg/model"
)

// LayerPhysicsState tracks the dynamic physical state of a single layer.
type LayerPhysicsState struct {
	Angle        float32 // current angle in degrees
	AngVel       float32 // angular velocity
	OffsetX      float32 // linear spring displacement X
	OffsetY      float32 // linear spring displacement Y
	VelX         float32 // linear velocity X
	VelY         float32 // linear velocity Y
	StretchX     float32 // stretch factor X
	StretchY     float32 // stretch factor Y
	TickCount    float64 // elapsed tick counter for sinusoidal oscillation
	PrevPosX     float32 // previous frame world displacement X
	PrevPosY     float32 // previous frame world displacement Y
	PrevRotation float32 // previous frame rotation
}

// WobbleSystem calculates the spring-damper wobble and sinusoidal oscillations for all avatar layers.
type WobbleSystem struct {
	states           map[int64]*LayerPhysicsState
	BobbingIntensity float32 // Multiplier for idle floating / sinusoidal breathing (default 1.0)
	WobbleIntensity  float32 // Multiplier for spring inertia and rotation (default 1.0)
}

// NewWobbleSystem creates a new wobble and physics animation system.
func NewWobbleSystem() *WobbleSystem {
	return &WobbleSystem{
		states:           make(map[int64]*LayerPhysicsState),
		BobbingIntensity: 1.0,
		WobbleIntensity:  1.0,
	}
}

// GetState returns the physical state for a layer, initializing if necessary.
func (ws *WobbleSystem) GetState(layerID int64) *LayerPhysicsState {
	st, exists := ws.states[layerID]
	if !exists {
		st = &LayerPhysicsState{}
		ws.states[layerID] = st
	}
	return st
}

// Update advances the physics simulation hierarchically from parent layers to children.
func (ws *WobbleSystem) Update(avatar *model.Avatar, dt float32, globalBounceDelta float32) {
	if avatar == nil {
		return
	}

	const (
		springStiffness = float32(180.0) // spring return strength
		linearStiffness = float32(140.0)
	)

	// Recursive function to update physics node-by-node down the hierarchy
	var updateNode func(layer *model.Layer, parentDeltaX, parentDeltaY float32, parentRotDelta float32)
	updateNode = func(layer *model.Layer, parentDeltaX, parentDeltaY float32, parentRotDelta float32) {
		st := ws.GetState(layer.Identification)
		st.TickCount += float64(dt * 60.0)

		// 1. Angular spring-damper physics
		rotDamping := float32(12.0)
		if layer.RotDrag > 0 {
			rotDamping = layer.RotDrag * 10.0
		}

		// Inertia torque: reacted to parent horizontal translation and rotational acceleration
		inertiaTorque := -parentDeltaX*1.2 - parentRotDelta*0.8
		angAccel := (0.0-st.Angle)*springStiffness - st.AngVel*rotDamping + inertiaTorque
		st.AngVel += angAccel * dt
		st.Angle += st.AngVel * dt

		// Clamp rotation to layer limits [RLimitMin, RLimitMax]
		if st.Angle < layer.RLimitMin {
			st.Angle = layer.RLimitMin
			st.AngVel = 0
		} else if st.Angle > layer.RLimitMax {
			st.Angle = layer.RLimitMax
			st.AngVel = 0
		}

		// 2. Linear spring-damper physics (drag / follow-through)
		linearDamping := float32(10.0)
		dragFactor := layer.Drag
		if dragFactor > 0 {
			linearDamping = dragFactor * 8.0
		}

		accelX := (0.0-st.OffsetX)*linearStiffness - st.VelX*linearDamping - parentDeltaX*6.0
		accelY := (0.0-st.OffsetY)*linearStiffness - st.VelY*linearDamping - parentDeltaY*6.0

		st.VelX += accelX * dt
		st.VelY += accelY * dt
		st.OffsetX += st.VelX * dt
		st.OffsetY += st.VelY * dt

		// 3. Stretch physics (jiggle deformation)
		if layer.StretchAmount > 0 {
			stretchRatio := (st.VelY * 0.003) * layer.StretchAmount
			st.StretchY = stretchRatio
			st.StretchX = -stretchRatio * 0.5
		} else {
			st.StretchX = 0
			st.StretchY = 0
		}

		// 4. Calculate total dynamic motion of this layer to pass as parentDelta to its children
		totalDispX, totalDispY := ws.GetCalculatedOffset(layer)
		deltaX := totalDispX - st.PrevPosX
		deltaY := totalDispY - st.PrevPosY
		deltaRot := st.Angle - st.PrevRotation

		st.PrevPosX = totalDispX
		st.PrevPosY = totalDispY
		st.PrevRotation = st.Angle

		// Traverse children
		for _, child := range avatar.GetChildren(layer.Identification) {
			updateNode(child, deltaX, deltaY, deltaRot)
		}
	}

	// Update all root layers (parents receive global bounce as parentDelta)
	for _, root := range avatar.RootLayers {
		rootBounceDelta := float32(0)
		if !root.IgnoreBounce {
			rootBounceDelta = globalBounceDelta
		}
		updateNode(root, 0.0, rootBounceDelta, 0.0)
	}
}

// GetCalculatedOffset returns the combined displacement (spring + sinusoidal idle wave) for a layer.
func (ws *WobbleSystem) GetCalculatedOffset(layer *model.Layer) (float32, float32) {
	st := ws.GetState(layer.Identification)

	// Sinusoidal wave: amp * sin(2*pi * frq * tick) scaled by BobbingIntensity
	var sinX, sinY float32
	if layer.XAmp != 0 && layer.XFrq != 0 {
		sinX = layer.XAmp * ws.BobbingIntensity * float32(math.Sin(2.0*math.Pi*float64(layer.XFrq)*st.TickCount))
	}
	if layer.YAmp != 0 && layer.YFrq != 0 {
		sinY = layer.YAmp * ws.BobbingIntensity * float32(math.Sin(2.0*math.Pi*float64(layer.YFrq)*st.TickCount))
	}

	return (st.OffsetX * ws.WobbleIntensity) + sinX, (st.OffsetY * ws.WobbleIntensity) + sinY
}

// GetCalculatedRotation returns the current rotation angle in degrees scaled by WobbleIntensity.
func (ws *WobbleSystem) GetCalculatedRotation(layer *model.Layer) float32 {
	st := ws.GetState(layer.Identification)
	return st.Angle * ws.WobbleIntensity
}

// GetCalculatedStretch returns the current stretch factors for the layer.
func (ws *WobbleSystem) GetCalculatedStretch(layer *model.Layer) (float32, float32) {
	st := ws.GetState(layer.Identification)
	return st.StretchX, st.StretchY
}
