package model

import (
	"os"
	"testing"
)

func TestBuildAvatarFromSlugcatDirectory(t *testing.T) {
	dir := "../../assets/samples/SlugcatPNGs"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skip("SlugcatPNGs directory not found, skipping")
	}

	avatar, err := BuildAvatarFromDirectory(dir)
	if err != nil {
		t.Fatalf("Failed to build avatar from Slugcat directory: %v", err)
	}

	if len(avatar.Layers) != 22 {
		t.Errorf("Expected 22 layers, got %d", len(avatar.Layers))
	}

	// Verify save to file
	targetSave := "../../assets/samples/slugcat.save"
	if err := SaveAvatarToFile(avatar, targetSave); err != nil {
		t.Fatalf("Failed to save slugcat.save: %v", err)
	}

	// Verify reloading the generated save file
	reloaded, err := ParseSaveFile(targetSave)
	if err != nil {
		t.Fatalf("Failed to reload generated slugcat.save: %v", err)
	}

	if len(reloaded.Layers) != 22 {
		t.Errorf("Expected 22 reloaded layers, got %d", len(reloaded.Layers))
	}

	t.Logf("Successfully generated %s with %d layers and verified reload!", targetSave, len(reloaded.Layers))
}

func TestDetectPNGTuberStates(t *testing.T) {
	tests := []struct {
		name        string
		wantCostume string
		wantBlink   int
		wantTalk    int
	}{
		{"DefaultClosed.png", "Default", 1, 1},
		{"DefaultBlink.png", "Default", 2, 1},
		{"DefaultOpenMouth.png", "Default", 1, 2},
		{"DefaultOpenBlink.png", "Default", 2, 2},
		{"AngryClosed.png", "Angry", 1, 1},
		{"AngryBlink.png", "Angry", 2, 1},
		{"AngryOpen.png", "Angry", 1, 2},
		{"AngryOpenBlink.png", "Angry", 2, 2},
		{"SadClosed.png", "Sad", 1, 1},
		{"SadOpenBlink.png", "Sad", 2, 2},
		{"ShockedOpen.png", "Shocked", 1, 2},
	}

	for _, tt := range tests {
		costume, blink, talk := DetectPNGTuberStates(tt.name)
		if costume != tt.wantCostume || blink != tt.wantBlink || talk != tt.wantTalk {
			t.Errorf("DetectPNGTuberStates(%q) = (%q, %d, %d); want (%q, %d, %d)",
				tt.name, costume, blink, talk, tt.wantCostume, tt.wantBlink, tt.wantTalk)
		}
	}
}
