package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigLoadSave(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pngtuber_config_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")

	// 1. Loading non-existent file should create and return default config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed on new file: %v", err)
	}
	if cfg.TargetFPS != 60 || cfg.WindowWidth != 800 {
		t.Errorf("unexpected default config values: %+v", cfg)
	}

	// 2. Modify and save
	cfg.Scale = 1.5
	cfg.AudioThreshold = 0.08
	if err := SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// 3. Reload and verify persistence
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("re-LoadConfig failed: %v", err)
	}
	if loaded.Scale != 1.5 || loaded.AudioThreshold != 0.08 {
		t.Errorf("config was not correctly persisted: scale=%f, threshold=%f", loaded.Scale, loaded.AudioThreshold)
	}
}

func TestKeybinds(t *testing.T) {
	kb := DefaultKeybinds()
	if kb.ToggleMenu != 258 {
		t.Errorf("expected DefaultKeybinds ToggleMenu = 258 (Tab), got %d", kb.ToggleMenu)
	}

	if GetKeyName(258) != "TAB" {
		t.Errorf("expected GetKeyName(258) == 'TAB', got %q", GetKeyName(258))
	}
	if GetKeyName(32) != "ESPAÇO" {
		t.Errorf("expected GetKeyName(32) == 'ESPAÇO', got %q", GetKeyName(32))
	}
	if GetKeyName(69) != "E" {
		t.Errorf("expected GetKeyName(69) == 'E', got %q", GetKeyName(69))
	}
	if GetKeyName(298) != "F9" {
		t.Errorf("expected GetKeyName(298) == 'F9', got %q", GetKeyName(298))
	}
}
