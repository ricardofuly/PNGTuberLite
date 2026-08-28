package ui

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
	"pngtuber-lite/assets"
)

func TestEmbeddedAssets(t *testing.T) {
	requiredAssets := map[string][]byte{
		"IconAdd":        assets.IconAddPNG,
		"IconAudio":      assets.IconAudioPNG,
		"IconAvatar":     assets.IconAvatarPNG,
		"IconChange":     assets.IconChangePNG,
		"IconClose":      assets.IconClosePNG,
		"IconDelete":     assets.IconDeletePNG,
		"IconDuplicate":  assets.IconDuplicatePNG,
		"IconEditor":     assets.IconEditorPNG,
		"IconEnable":     assets.IconEnablePNG,
		"IconFavorite":   assets.IconFavoritePNG,
		"IconPhysics":    assets.IconPhysicsPNG,
		"IconKeys":       assets.IconKeysPNG,
		"AppLogo":        assets.AppLogoPNG,
		"AppIconICO":     assets.AppIconICO,
		"AppTrayPNG":     assets.AppTrayPNG,
		"IconOBS":        assets.IconOBSPNG,
		"IconOpenEditor": assets.IconOpenEditorPNG,
		"IconPNGFile":    assets.IconPNGFilePNG,
		"IconRemove":     assets.IconRemovePNG,
		"IconRestart":    assets.IconRestartPNG,
		"IconRestore":    assets.IconRestorePNG,
		"IconCostumes":   assets.IconCostumesPNG,
		"IconSave":       assets.IconSavePNG,
		"IconSelected":   assets.IconSelectedPNG,
		"IconSettings":   assets.IconSettingsPNG,
		"IconUpdate":     assets.IconUpdatePNG,
	}

	for name, data := range requiredAssets {
		if len(data) == 0 {
			t.Fatalf("embedded asset %s is empty", name)
		}
	}

	logoImg := rl.LoadImageFromMemory(".png", assets.AppLogoPNG, int32(len(assets.AppLogoPNG)))
	if logoImg.Width == 0 || logoImg.Height == 0 {
		t.Fatalf("invalid logo image dimensions: %dx%d", logoImg.Width, logoImg.Height)
	}
	rl.UnloadImage(logoImg)
}
