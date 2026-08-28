package render

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
	"pngtuber-lite/pkg/model"
)

// LayerTransform stores the calculated world-space transform for a layer.
type LayerTransform struct {
	WorldPos rl.Vector2
	Rotation float32 // in degrees
	Scale    float32
}

// ComputeWorldTransformsEx calculates the recursive world transforms for all layers with optional horizontal mirroring.
func ComputeWorldTransformsEx(
	avatar *model.Avatar,
	origin rl.Vector2,
	scale float32,
	globalBounceY float32,
	layerOffsets map[int64]model.Vector2,
	layerRotations map[int64]float32,
	flipHorizontal bool,
) map[int64]LayerTransform {
	if avatar == nil {
		return nil
	}
	transforms := make(map[int64]LayerTransform, len(avatar.Layers))
	ComputeWorldTransformsBuffer(avatar, origin, scale, globalBounceY, layerOffsets, layerRotations, flipHorizontal, transforms)
	return transforms
}

// ComputeWorldTransformsBuffer populates an existing transforms map with zero heap allocations and cycle protection.
func ComputeWorldTransformsBuffer(
	avatar *model.Avatar,
	origin rl.Vector2,
	scale float32,
	globalBounceY float32,
	layerOffsets map[int64]model.Vector2,
	layerRotations map[int64]float32,
	flipHorizontal bool,
	outTransforms map[int64]LayerTransform,
) {
	if avatar == nil || outTransforms == nil {
		return
	}

	visited := make(map[int64]bool, len(avatar.Layers))

	// Recursive helper to traverse the tree starting from root layers
	var computeNode func(layer *model.Layer, parentTransform LayerTransform, depth int)
	computeNode = func(layer *model.Layer, parentTransform LayerTransform, depth int) {
		if layer == nil || visited[layer.Identification] || depth > 50 {
			return // Break potential hierarchy cycle / prevent stack overflow
		}
		visited[layer.Identification] = true

		localPos := layer.Pos
		if offset, ok := layerOffsets[layer.Identification]; ok {
			localPos.X += offset.X
			localPos.Y += offset.Y
		}

		localRot := float32(0)
		if rot, ok := layerRotations[layer.Identification]; ok {
			localRot = rot
		}

		if flipHorizontal {
			localPos.X = -localPos.X
			localRot = -localRot
		}

		var worldPos rl.Vector2
		var worldRot float32

		if layer.ParentID == nil || *layer.ParentID == 0 || *layer.ParentID == layer.Identification {
			// Root layer
			bounce := float32(0)
			if !layer.IgnoreBounce {
				bounce = globalBounceY
			}
			worldPos = rl.Vector2{
				X: origin.X + localPos.X*scale,
				Y: origin.Y + (localPos.Y+bounce)*scale,
			}
			worldRot = localRot
		} else {
			// Child layer: rotate local position by parent's world rotation
			pAngleRad := float64(parentTransform.Rotation) * (math.Pi / 180.0)
			cosA := float32(math.Cos(pAngleRad))
			sinA := float32(math.Sin(pAngleRad))

			scaledLocalX := localPos.X * scale
			scaledLocalY := localPos.Y * scale

			rotX := scaledLocalX*cosA - scaledLocalY*sinA
			rotY := scaledLocalX*sinA + scaledLocalY*cosA

			worldPos = rl.Vector2{
				X: parentTransform.WorldPos.X + rotX,
				Y: parentTransform.WorldPos.Y + rotY,
			}
			worldRot = parentTransform.Rotation + localRot
		}

		currentTransform := LayerTransform{
			WorldPos: worldPos,
			Rotation: worldRot,
			Scale:    scale,
		}
		outTransforms[layer.Identification] = currentTransform

		// Traverse children
		for _, child := range avatar.GetChildren(layer.Identification) {
			computeNode(child, currentTransform, depth+1)
		}
		visited[layer.Identification] = false
	}

	// Compute for each root
	for _, root := range avatar.RootLayers {
		computeNode(root, LayerTransform{WorldPos: origin, Rotation: 0, Scale: scale}, 0)
	}
}

// ComputeWorldTransforms computes standard world transforms without horizontal flip.
func ComputeWorldTransforms(
	avatar *model.Avatar,
	origin rl.Vector2,
	scale float32,
	globalBounceY float32,
	layerOffsets map[int64]model.Vector2,
	layerRotations map[int64]float32,
) map[int64]LayerTransform {
	return ComputeWorldTransformsEx(avatar, origin, scale, globalBounceY, layerOffsets, layerRotations, false)
}

// ComputeAvatarExtents computes the maximum left, right, top, and bottom pixel extents
// relative to the avatar origin at the given scale, taking into account layer content bounds.
func ComputeAvatarExtents(avatar *model.Avatar, scale float32) (extLeft, extRight, extTop, extBottom float32) {
	if avatar == nil || len(avatar.Layers) == 0 {
		return 64 * scale, 64 * scale, 64 * scale, 64 * scale
	}

	transforms := ComputeWorldTransforms(avatar, rl.Vector2{X: 0, Y: 0}, scale, 0, nil, nil)

	minX := float32(0)
	maxX := float32(0)
	minY := float32(0)
	maxY := float32(0)
	first := true

	for id, layer := range avatar.Layers {
		tf, ok := transforms[id]
		if !ok {
			continue
		}

		w := float32(layer.ImageWidth)
		h := float32(layer.ImageHeight)
		if layer.Frames > 1 {
			w = w / float32(layer.Frames)
		}
		if w <= 0 {
			w = 128
		}
		if h <= 0 {
			h = 128
		}

		topLeftX := tf.WorldPos.X + (layer.Offset.X * tf.Scale)
		topLeftY := tf.WorldPos.Y + (layer.Offset.Y * tf.Scale)

		minX := float32(0)
		minY := float32(0)
		maxX := w
		maxY := h
		if layer.HasContentBounds {
			minX = layer.ContentMinX
			minY = layer.ContentMinY
			maxX = layer.ContentMaxX
			maxY = layer.ContentMaxY
		}

		corners := []rl.Vector2{
			{X: topLeftX + minX*tf.Scale, Y: topLeftY + minY*tf.Scale},
			{X: topLeftX + maxX*tf.Scale, Y: topLeftY + minY*tf.Scale},
			{X: topLeftX + minX*tf.Scale, Y: topLeftY + maxY*tf.Scale},
			{X: topLeftX + maxX*tf.Scale, Y: topLeftY + maxY*tf.Scale},
		}

		for _, c := range corners {
			if first {
				minX, maxX = c.X, c.X
				minY, maxY = c.Y, c.Y
				first = false
			} else {
				if c.X < minX {
					minX = c.X
				}
				if c.X > maxX {
					maxX = c.X
				}
				if c.Y < minY {
					minY = c.Y
				}
				if c.Y > maxY {
					maxY = c.Y
				}
			}
		}
	}

	extLeft = -minX
	if extLeft < 0 {
		extLeft = 0
	}
	extRight = maxX
	if extRight < 0 {
		extRight = 0
	}
	extTop = -minY
	if extTop < 0 {
		extTop = 0
	}
	extBottom = maxY
	if extBottom < 0 {
		extBottom = 0
	}

	return extLeft, extRight, extTop, extBottom
}
