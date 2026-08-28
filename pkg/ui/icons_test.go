package ui

import (
	"encoding/json"
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
	"pngtuber-lite/assets"
)

func TestEmbeddedAssets(t *testing.T) {
	if len(assets.IconsAtlasJSON) == 0 {
		t.Fatalf("assets.IconsAtlasJSON is empty")
	}
	if len(assets.IconsAtlasPNG) == 0 {
		t.Fatalf("assets.IconsAtlasPNG is empty")
	}
	if len(assets.AppLogoPNG) == 0 {
		t.Fatalf("assets.AppLogoPNG is empty")
	}

	var a atlasJSON
	if err := json.Unmarshal(assets.IconsAtlasJSON, &a); err != nil {
		t.Fatalf("failed to unmarshal embedded atlas JSON: %v", err)
	}
	if len(a.Frames) == 0 {
		t.Fatalf("no frames parsed from embedded atlas JSON")
	}

	atlasImg := rl.LoadImageFromMemory(".png", assets.IconsAtlasPNG, int32(len(assets.IconsAtlasPNG)))
	if atlasImg.Width != 320 || atlasImg.Height != 320 {
		t.Fatalf("unexpected atlas image dimensions: %dx%d", atlasImg.Width, atlasImg.Height)
	}
	rl.UnloadImage(atlasImg)

	logoImg := rl.LoadImageFromMemory(".png", assets.AppLogoPNG, int32(len(assets.AppLogoPNG)))
	if logoImg.Width == 0 || logoImg.Height == 0 {
		t.Fatalf("invalid logo image dimensions: %dx%d", logoImg.Width, logoImg.Height)
	}
	rl.UnloadImage(logoImg)
}
