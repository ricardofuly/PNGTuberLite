package config

import (
	"fmt"
)

// Keybinds holds customizable Raylib keyboard key codes for application actions.
type Keybinds struct {
	ToggleMenu         int32 `json:"toggleMenu"`
	ToggleEditor       int32 `json:"toggleEditor"`
	ToggleHUD          int32 `json:"toggleHUD"`
	ToggleClickThrough int32 `json:"toggleClickThrough"`
	ToggleBorderless   int32 `json:"toggleBorderless"`
	ToggleAlwaysOnTop  int32 `json:"toggleAlwaysOnTop"`
	TestBounce         int32 `json:"testBounce"`
	ResetAvatar        int32 `json:"resetAvatar"`
	IncreaseSens       int32 `json:"increaseSens"`
	DecreaseSens       int32 `json:"decreaseSens"`
}

// DefaultKeybinds returns the standard key mapping.
func DefaultKeybinds() Keybinds {
	return Keybinds{
		ToggleMenu:         258, // KeyTab
		ToggleEditor:       69,  // KeyE
		ToggleHUD:          72,  // KeyH
		ToggleClickThrough: 298, // KeyF9
		ToggleBorderless:   299, // KeyF10
		ToggleAlwaysOnTop:  300, // KeyF11
		TestBounce:         32,  // KeySpace
		ResetAvatar:        82,  // KeyR
		IncreaseSens:       61,  // KeyEqual (=/+)
		DecreaseSens:       45,  // KeyMinus (-)
	}
}

// GetKeyName returns a user-friendly display string for a Raylib key code.
func GetKeyName(key int32) string {
	switch key {
	case 32:
		return "ESPAÇO"
	case 256:
		return "ESC"
	case 257:
		return "ENTER"
	case 258:
		return "TAB"
	case 259:
		return "BACKSPACE"
	case 260:
		return "INSERT"
	case 261:
		return "DELETE"
	case 262:
		return "DIREITA"
	case 263:
		return "ESQUERDA"
	case 264:
		return "BAIXO"
	case 265:
		return "CIMA"
	case 266:
		return "PAGE UP"
	case 267:
		return "PAGE DOWN"
	case 268:
		return "HOME"
	case 269:
		return "END"
	case 280:
		return "CAPS LOCK"
	case 281:
		return "SCROLL LOCK"
	case 282:
		return "NUM LOCK"
	case 283:
		return "PRINT SCREEN"
	case 284:
		return "PAUSE"
	case 290:
		return "F1"
	case 291:
		return "F2"
	case 292:
		return "F3"
	case 293:
		return "F4"
	case 294:
		return "F5"
	case 295:
		return "F6"
	case 296:
		return "F7"
	case 297:
		return "F8"
	case 298:
		return "F9"
	case 299:
		return "F10"
	case 300:
		return "F11"
	case 301:
		return "F12"
	case 334:
		return "NUM +"
	case 333:
		return "NUM -"
	case 340:
		return "L-SHIFT"
	case 341:
		return "L-CTRL"
	case 342:
		return "L-ALT"
	case 344:
		return "R-SHIFT"
	case 345:
		return "R-CTRL"
	case 346:
		return "R-ALT"
	case 45:
		return "-"
	case 61:
		return "+"
	case 91:
		return "["
	case 93:
		return "]"
	case 59:
		return ";"
	case 39:
		return "'"
	case 44:
		return ","
	case 46:
		return "."
	case 47:
		return "/"
	case 92:
		return "\\"
	case 96:
		return "`"
	}

	// Printable ASCII letters & numbers
	if key >= 65 && key <= 90 {
		return string(rune(key))
	}
	if key >= 48 && key <= 57 {
		return string(rune(key))
	}

	if key == 0 {
		return "NENHUMA"
	}
	return fmt.Sprintf("TECLA %d", key)
}
