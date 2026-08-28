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

// ComputeWorldTransforms calculates the recursive world transforms for all layers in the avatar.
func ComputeWorldTransforms(
	avatar *model.Avatar,
	origin rl.Vector2,
	scale float32,
	globalBounceY float32,
	layerOffsets map[int64]model.Vector2,
	layerRotations map[int64]float32,
) map[int64]LayerTransform {
	transforms := make(map[int64]LayerTransform, len(avatar.Layers))

	// Recursive helper to traverse the tree starting from root layers
	var computeNode func(layer *model.Layer, parentTransform LayerTransform)
	computeNode = func(layer *model.Layer, parentTransform LayerTransform) {
		localPos := layer.Pos
		if offset, ok := layerOffsets[layer.Identification]; ok {
			localPos.X += offset.X
			localPos.Y += offset.Y
		}

		localRot := float32(0)
		if rot, ok := layerRotations[layer.Identification]; ok {
			localRot = rot
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
		transforms[layer.Identification] = currentTransform

		// Traverse children
		for _, child := range avatar.GetChildren(layer.Identification) {
			computeNode(child, currentTransform)
		}
	}

	// Compute for each root
	for _, root := range avatar.RootLayers {
		computeNode(root, LayerTransform{WorldPos: origin, Rotation: 0, Scale: scale})
	}

	return transforms
}

// ComputeAvatarExtents calculates the maximum bounding extents of all visible layers from the origin point.
func ComputeAvatarExtents(avatar *model.Avatar, scale float32) (extLeft, extRight, extTop, extBottom float32) {
	if avatar == nil || len(avatar.Layers) == 0 {
		return 150.0 * scale, 150.0 * scale, 150.0 * scale, 150.0 * scale
	}

	transforms := ComputeWorldTransforms(avatar, rl.Vector2{X: 0, Y: 0}, scale, 0, nil, nil)

	for id, layer := range avatar.Layers {
		tf, ok := transforms[id]
		if !ok {
			continue
		}

		frames := layer.Frames
		if frames < 1 {
			frames = 1
		}

		w := (float32(layer.ImageWidth) / float32(frames)) * scale
		h := float32(layer.ImageHeight) * scale

		halfW := w * 0.5
		halfH := h * 0.5

		left := -tf.WorldPos.X + halfW - (layer.Offset.X * scale)
		right := tf.WorldPos.X + halfW + (layer.Offset.X * scale)
		top := -tf.WorldPos.Y + halfH - (layer.Offset.Y * scale)
		bottom := tf.WorldPos.Y + halfH + (layer.Offset.Y * scale)

		if left > extLeft {
			extLeft = left
		}
		if right > extRight {
			extRight = right
		}
		if top > extTop {
			extTop = top
		}
		if bottom > extBottom {
			extBottom = bottom
		}
	}

	if extLeft < 50*scale {
		extLeft = 150 * scale
	}
	if extRight < 50*scale {
		extRight = 150 * scale
	}
	if extTop < 50*scale {
		extTop = 150 * scale
	}
	if extBottom < 50*scale {
		extBottom = 150 * scale
	}

	return extLeft, extRight, extTop, extBottom
}
