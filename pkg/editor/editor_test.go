package editor

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"pngtuber-lite/pkg/model"
	"pngtuber-lite/pkg/render"
	"pngtuber-lite/pkg/ui"
)

const sample1x1PNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

func TestEditorStateNewAndModify(t *testing.T) {
	texCache := render.NewTextureCache()
	uiState := ui.NewUIState()
	editorState := NewEditorState(texCache, uiState)

	// 1. Create blank avatar
	editorState.NewBlankAvatar()
	if editorState.Avatar == nil {
		t.Fatalf("expected avatar instance")
	}

	// 2. Add layer via temp PNG file
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "layer1.png")
	pngBytes, _ := base64.StdEncoding.DecodeString(sample1x1PNG)
	if err := os.WriteFile(pngPath, pngBytes, 0644); err != nil {
		t.Fatalf("failed to write test PNG: %v", err)
	}

	layer := model.NewDefaultLayer(1001)
	layer.Path = "layer1.png"
	layer.ImageData = pngBytes
	layer.ImageWidth = 1
	layer.ImageHeight = 1
	editorState.Avatar.AddLayer(layer)
	editorState.Avatar.BuildHierarchy()
	editorState.SelectedLayerID = 1001

	if len(editorState.Avatar.Layers) != 1 {
		t.Fatalf("expected 1 layer, got %d", len(editorState.Avatar.Layers))
	}

	// 3. Duplicate layer
	editorState.DuplicateSelectedLayer()
	if len(editorState.Avatar.Layers) != 2 {
		t.Fatalf("expected 2 layers after duplicate, got %d", len(editorState.Avatar.Layers))
	}

	// 4. Remove selected layer
	editorState.RemoveSelectedLayer()
	if len(editorState.Avatar.Layers) != 1 {
		t.Fatalf("expected 1 layer after remove, got %d", len(editorState.Avatar.Layers))
	}

	// 5. Save avatar to file
	savePath := filepath.Join(tmpDir, "test_out.save")
	editorState.AvatarFilePath = savePath
	if err := editorState.SaveCurrentAvatar(); err != nil {
		t.Fatalf("failed to save avatar: %v", err)
	}

	// 6. Verify file was written and can be parsed
	reloaded, err := model.ParseSaveFile(savePath)
	if err != nil {
		t.Fatalf("failed to parse saved avatar: %v", err)
	}
	if len(reloaded.Layers) != 1 {
		t.Errorf("expected 1 layer in reloaded avatar, got %d", len(reloaded.Layers))
	}
}
