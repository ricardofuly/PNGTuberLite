package model

import (
	"encoding/base64"
	"testing"
)

// Minimal 1x1 valid transparent PNG encoded in Base64
const sample1x1PNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

func TestParseVector2(t *testing.T) {
	tests := []struct {
		input    string
		expected Vector2
		wantErr  bool
	}{
		{"Vector2(0, 0)", Vector2{0, 0}, false},
		{"Vector2(10.5, -20.25)", Vector2{10.5, -20.25}, false},
		{"Vector2( 100 , 200 )", Vector2{100, 200}, false},
		{"10, 20", Vector2{10, 20}, false},
		{"", Vector2{0, 0}, false},
		{"invalid", Vector2{0, 0}, true},
	}

	for _, tt := range tests {
		got, err := ParseVector2(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseVector2(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && (got.X != tt.expected.X || got.Y != tt.expected.Y) {
			t.Errorf("ParseVector2(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestParseCostumeLayers(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected [10]int
	}{
		{"[1, 1, 1, 1, 1, 1, 1, 1, 1, 1]", [10]int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}},
		{"[1, 0, 1, 0, 1, 0, 1, 0, 1, 0]", [10]int{1, 0, 1, 0, 1, 0, 1, 0, 1, 0}},
		{"[0, 1]", [10]int{0, 1, 1, 1, 1, 1, 1, 1, 1, 1}},
	}

	for _, tt := range tests {
		got, err := parseCostumeLayers(tt.input)
		if err != nil {
			t.Errorf("parseCostumeLayers(%v) error = %v", tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("parseCostumeLayers(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestPNGDimensionsExtraction(t *testing.T) {
	pngBytes, err := base64.StdEncoding.DecodeString(sample1x1PNG)
	if err != nil {
		t.Fatalf("failed to decode sample PNG: %v", err)
	}

	w, h := ExtractPNGDimensions(pngBytes)
	if w != 1 || h != 1 {
		t.Errorf("ExtractPNGDimensions got (%d, %d), expected (1, 1)", w, h)
	}
}

func TestParseSaveDataAndHierarchy(t *testing.T) {
	saveJSON := `{
		"0": {
			"identification": 100,
			"parentId": null,
			"pos": "Vector2(0, 0)",
			"offset": "Vector2(0, 0)",
			"zindex": -1,
			"type": "sprite",
			"frames": 1,
			"animSpeed": 0,
			"costumeLayers": "[1, 1, 1, 1, 1, 1, 1, 1, 1, 1]",
			"imageData": "` + sample1x1PNG + `"
		},
		"1": {
			"identification": 200,
			"parentId": 100,
			"pos": "Vector2(10, -50)",
			"offset": "Vector2(5, 5)",
			"zindex": 0,
			"type": "sprite",
			"frames": 2,
			"animSpeed": 0.5,
			"showBlink": 0,
			"showTalk": 0,
			"costumeLayers": "[1, 1, 1, 1, 1, 1, 1, 1, 1, 1]",
			"imageData": "` + sample1x1PNG + `"
		},
		"2": {
			"identification": 300,
			"parentId": 200,
			"pos": "Vector2(0, 0)",
			"offset": "Vector2(0, 0)",
			"zindex": 2,
			"type": "sprite",
			"showBlink": 0,
			"showTalk": 2,
			"costumeLayers": "[1, 0, 1, 1, 1, 1, 1, 1, 1, 1]",
			"imageData": "` + sample1x1PNG + `"
		},
		"3": {
			"identification": 400,
			"parentId": 200,
			"pos": "Vector2(0, 0)",
			"offset": "Vector2(0, 0)",
			"zindex": 1,
			"type": "sprite",
			"showBlink": 2,
			"showTalk": 0,
			"costumeLayers": "[1, 1, 1, 1, 1, 1, 1, 1, 1, 1]",
			"imageData": "` + sample1x1PNG + `"
		}
	}`

	avatar, err := ParseSaveData([]byte(saveJSON))
	if err != nil {
		t.Fatalf("ParseSaveData failed: %v", err)
	}

	if len(avatar.Layers) != 4 {
		t.Fatalf("expected 4 layers, got %d", len(avatar.Layers))
	}

	// Verify root layer
	if len(avatar.RootLayers) != 1 {
		t.Fatalf("expected 1 root layer, got %d", len(avatar.RootLayers))
	}
	if avatar.RootLayers[0].Identification != 100 {
		t.Errorf("expected root layer ID 100, got %d", avatar.RootLayers[0].Identification)
	}

	// Verify children of root (ID 100) -> should be ID 200
	children100 := avatar.GetChildren(100)
	if len(children100) != 1 || children100[0].Identification != 200 {
		t.Fatalf("expected children of 100 to be [200], got %v", children100)
	}

	// Verify children of ID 200 -> should be 300 and 400
	children200 := avatar.GetChildren(200)
	if len(children200) != 2 {
		t.Fatalf("expected 2 children for layer 200, got %d", len(children200))
	}

	// Verify draw order sorting by ZIndex: -1 (ID 100), 0 (ID 200), 1 (ID 400), 2 (ID 300)
	expectedDrawOrder := []int64{100, 200, 400, 300}
	for i, expectedID := range expectedDrawOrder {
		if avatar.DrawOrder[i].Identification != expectedID {
			t.Errorf("drawOrder[%d] ID = %d, want %d", i, avatar.DrawOrder[i].Identification, expectedID)
		}
	}

	// Test Layer visibility rules
	mouthLayer := avatar.GetLayer(300) // showTalk = 2 (only speaking)
	eyeLayer := avatar.GetLayer(400)   // showBlink = 2 (only blinking)

	// Mouth: speaking vs quiet
	if !mouthLayer.IsVisible(1, false, true) {
		t.Errorf("mouth should be visible when talking in costume 1")
	}
	if mouthLayer.IsVisible(1, false, false) {
		t.Errorf("mouth should NOT be visible when quiet")
	}
	// Mouth in costume 2 (costumeLayers[1] = 0)
	if mouthLayer.IsVisible(2, false, true) {
		t.Errorf("mouth should NOT be visible in costume slot 2")
	}

	// Eyes: blinking vs not blinking
	if !eyeLayer.IsVisible(1, true, false) {
		t.Errorf("blink eyes should be visible when blinking")
	}
	if eyeLayer.IsVisible(1, false, false) {
		t.Errorf("blink eyes should NOT be visible when not blinking")
	}
}

func TestParseRealDefaultAvatar(t *testing.T) {
	avatar, err := ParseSaveFile("../../assets/samples/defaultAvatar.save")
	if err != nil {
		t.Fatalf("failed to parse defaultAvatar.save: %v", err)
	}

	if len(avatar.Layers) == 0 {
		t.Fatalf("expected layers in defaultAvatar.save, got 0")
	}

	t.Logf("Successfully parsed defaultAvatar.save with %d layers", len(avatar.Layers))
	t.Logf("Root layers count: %d", len(avatar.RootLayers))
	t.Logf("Draw order count: %d", len(avatar.DrawOrder))

	for id, layer := range avatar.Layers {
		pID := int64(0)
		if layer.ParentID != nil {
			pID = *layer.ParentID
		}
		t.Logf("Layer ID: %d | Path: %s | Parent: %d | Pos: (%.1f, %.1f) | Offset: (%.1f, %.1f) | Size: %dx%d",
			id, layer.Path, pID, layer.Pos.X, layer.Pos.Y, layer.Offset.X, layer.Offset.Y, layer.ImageWidth, layer.ImageHeight)
	}
}

func TestSaveAndReloadAvatar(t *testing.T) {
	origAvatar, err := ParseSaveFile("../../assets/samples/defaultAvatar.save")
	if err != nil {
		t.Fatalf("failed to parse original avatar: %v", err)
	}

	// 1. Serialize avatar
	savedData, err := SerializeAvatar(origAvatar)
	if err != nil {
		t.Fatalf("failed to serialize avatar: %v", err)
	}

	// 2. Re-parse serialized avatar
	reloaded, err := ParseSaveData(savedData)
	if err != nil {
		t.Fatalf("failed to parse reloaded avatar: %v", err)
	}

	// 3. Compare layer counts and properties
	if len(reloaded.Layers) != len(origAvatar.Layers) {
		t.Fatalf("expected %d layers, got %d", len(origAvatar.Layers), len(reloaded.Layers))
	}
	if len(reloaded.RootLayers) != len(origAvatar.RootLayers) {
		t.Fatalf("expected %d root layers, got %d", len(origAvatar.RootLayers), len(reloaded.RootLayers))
	}
	if len(reloaded.DrawOrder) != len(origAvatar.DrawOrder) {
		t.Fatalf("expected %d draw order layers, got %d", len(origAvatar.DrawOrder), len(reloaded.DrawOrder))
	}

	for id, origLayer := range origAvatar.Layers {
		reloadedLayer := reloaded.GetLayer(id)
		if reloadedLayer == nil {
			t.Errorf("missing layer %d in reloaded avatar", id)
			continue
		}
		if reloadedLayer.ZIndex != origLayer.ZIndex {
			t.Errorf("layer %d: ZIndex %d != %d", id, reloadedLayer.ZIndex, origLayer.ZIndex)
		}
		if reloadedLayer.ShowBlink != origLayer.ShowBlink {
			t.Errorf("layer %d: ShowBlink %d != %d", id, reloadedLayer.ShowBlink, origLayer.ShowBlink)
		}
		if reloadedLayer.ShowTalk != origLayer.ShowTalk {
			t.Errorf("layer %d: ShowTalk %d != %d", id, reloadedLayer.ShowTalk, origLayer.ShowTalk)
		}
		if len(reloadedLayer.ImageData) != len(origLayer.ImageData) {
			t.Errorf("layer %d: ImageData length %d != %d", id, len(reloadedLayer.ImageData), len(origLayer.ImageData))
		}
	}
}

