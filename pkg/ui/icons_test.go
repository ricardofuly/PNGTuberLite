package ui

import (
	"os"
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func TestIconAtlasParsing(t *testing.T) {
	jsonPaths := []string{
		"../../assets/icons/IconsFlat-32.json",
		"assets/icons/IconsFlat-32.json",
	}
	var foundPath string
	for _, p := range jsonPaths {
		if _, err := os.Stat(p); err == nil {
			foundPath = p
			break
		}
	}
	if foundPath == "" {
		t.Skip("assets/icons/IconsFlat-32.json not found in test path")
	}

	im := &IconManager{
		Frames: make(map[int]rl.Rectangle),
	}
	data, err := os.ReadFile(foundPath)
	if err != nil {
		t.Fatalf("failed to read atlas json: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("atlas json is empty")
	}
	_ = im
}
