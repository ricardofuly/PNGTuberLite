package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config represents all persisted user settings for PNGTuber Lite.
type Config struct {
	AvatarPath        string    `json:"avatarPath"`
	AudioDevice       string    `json:"audioDevice"`
	AudioThreshold    float32   `json:"audioThreshold"`
	WindowWidth       int32     `json:"windowWidth"`
	WindowHeight      int32     `json:"windowHeight"`
	WindowX           int32     `json:"windowX"`
	WindowY           int32     `json:"windowY"`
	WindowTransparent bool      `json:"windowTransparent"`
	WindowAlwaysOnTop bool      `json:"windowAlwaysOnTop"`
	WindowBorderless  bool      `json:"windowBorderless"`
	TargetFPS         int32     `json:"targetFPS"`
	Scale             float32   `json:"scale"`
	FlipHorizontal    bool      `json:"flipHorizontal"`    // Invert avatar horizontally (mirroring)
	AvatarRelX        float32   `json:"avatarRelX"`        // Normalized horizontal position (0.0 to 1.0, default 0.5 = center)
	AvatarRelY        float32   `json:"avatarRelY"`        // Normalized vertical position (0.0 to 1.0, default 0.5 = center)
	BobbingIntensity  float32   `json:"bobbingIntensity"`  // Floating / idle breathing intensity (0.0 to 2.0, default 0.5)
	WobbleIntensity   float32   `json:"wobbleIntensity"`   // Inertia wobble multiplier (0.0 to 2.0, default 1.0)
	BounceStrength    float32   `json:"bounceStrength"`
	BounceGravity     float32   `json:"bounceGravity"`
	BounceOnCostume   bool      `json:"bounceOnCostume"`
	BlinkSpeed        float32   `json:"blinkSpeed"`
	BlinkChance       int       `json:"blinkChance"`
	BackgroundColor   [4]uint8  `json:"backgroundColor"` // RGBA (0,0,0,0 = transparent)
	Keybinds          Keybinds  `json:"keybinds"`
}

// DefaultConfig creates standard configuration defaults.
func DefaultConfig() *Config {
	return &Config{
		AvatarPath:        "assets/samples/defaultAvatar.save",
		AudioDevice:       "",
		AudioThreshold:    0.05,
		WindowWidth:       800,
		WindowHeight:      800,
		WindowX:           -1, // -1 indicates centered
		WindowY:           -1,
		WindowTransparent: true,
		WindowAlwaysOnTop: true,
		WindowBorderless:  false, // start with border for initial positioning, togglable via key F10
		TargetFPS:         60,
		Scale:             1.0,
		FlipHorizontal:    false,
		AvatarRelX:        0.5,
		AvatarRelY:        0.5,
		BobbingIntensity:  0.5,
		WobbleIntensity:   1.0,
		BounceStrength:    250.0,
		BounceGravity:     1000.0,
		BounceOnCostume:   false,
		BlinkSpeed:        1.0,
		BlinkChance:       200,
		BackgroundColor:   [4]uint8{0, 0, 0, 0},
		Keybinds:          DefaultKeybinds(),
	}
}

// LoadConfig loads configuration from a JSON file, or creates defaults if missing.
func LoadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			_ = SaveConfig(filePath, cfg)
			return cfg, nil
		}
		return nil, err
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if cfg.Keybinds.ToggleMenu == 0 {
		cfg.Keybinds = DefaultKeybinds()
	}
	return cfg, nil
}

// SaveConfig writes the configuration to a JSON file on disk.
func SaveConfig(filePath string, cfg *Config) error {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}
