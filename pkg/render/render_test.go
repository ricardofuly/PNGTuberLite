package render

import (
	"math"
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
	"pngtuber-lite/pkg/model"
)

func TestComputeWorldTransforms(t *testing.T) {
	avatar := model.NewAvatar()

	// Root Layer (ID 1)
	root := model.NewDefaultLayer(1)
	root.Pos = model.Vector2{X: 10, Y: 20}
	root.Offset = model.Vector2{X: 0, Y: 0}
	avatar.AddLayer(root)

	// Child Layer (ID 2) attached to root
	pID := int64(1)
	child := model.NewDefaultLayer(2)
	child.ParentID = &pID
	child.Pos = model.Vector2{X: 0, Y: 50} // 50px below parent
	avatar.AddLayer(child)

	avatar.BuildHierarchy()

	origin := rl.Vector2{X: 100, Y: 100}
	scale := float32(1.0)
	bounceY := float32(-10.0)

	layerOffsets := map[int64]model.Vector2{
		1: {X: 5, Y: 0},
	}
	// Root rotated 90 degrees clockwise
	layerRotations := map[int64]float32{
		1: 90.0,
		2: 0.0,
	}

	transforms := ComputeWorldTransforms(
		avatar,
		origin,
		scale,
		bounceY,
		layerOffsets,
		layerRotations,
	)

	// 1. Verify Root Transform
	rootT, ok := transforms[1]
	if !ok {
		t.Fatalf("missing transform for root layer 1")
	}
	// Root WorldPos.X = origin.X + (root.Pos.X + layerOffset.X) = 100 + (10 + 5) = 115
	// Root WorldPos.Y = origin.Y + (root.Pos.Y + bounceY) = 100 + (20 - 10) = 110
	if rootT.WorldPos.X != 115 || rootT.WorldPos.Y != 110 {
		t.Errorf("expected root WorldPos (115, 110), got (%f, %f)", rootT.WorldPos.X, rootT.WorldPos.Y)
	}
	if rootT.Rotation != 90.0 {
		t.Errorf("expected root rotation 90.0, got %f", rootT.Rotation)
	}

	// 2. Verify Child Transform (Rotated 90 deg by parent)
	childT, ok := transforms[2]
	if !ok {
		t.Fatalf("missing transform for child layer 2")
	}
	// Child local pos is (0, 50). Rotated 90 deg (cos(90)=0, sin(90)=1):
	// rotX = 0*0 - 50*1 = -50
	// rotY = 0*1 + 50*0 = 0
	// Child WorldPos = (root.WorldPos.X - 50, root.WorldPos.Y + 0) = (115 - 50, 110) = (65, 110)
	if math.Abs(float64(childT.WorldPos.X-65)) > 0.01 || math.Abs(float64(childT.WorldPos.Y-110)) > 0.01 {
		t.Errorf("expected child WorldPos (65, 110), got (%f, %f)", childT.WorldPos.X, childT.WorldPos.Y)
	}
	if childT.Rotation != 90.0 {
		t.Errorf("expected child accumulated rotation 90.0, got %f", childT.Rotation)
	}
}
