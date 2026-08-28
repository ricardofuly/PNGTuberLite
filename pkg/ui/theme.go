package ui

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// PICO-8 & Cozy Modern Color Palette (Extracted from assets/refs/color-pallet.hex)
var (
	// Base Palette Colors
	ColBlack     = rl.NewColor(0, 0, 0, 255)         // #000000
	ColDarkNavy  = rl.NewColor(29, 43, 83, 255)      // #1D2B53
	ColWine      = rl.NewColor(126, 37, 83, 255)     // #7E2553
	ColDarkGreen = rl.NewColor(0, 135, 81, 255)      // #008751
	ColBrown     = rl.NewColor(171, 82, 54, 255)     // #AB5236
	ColDarkGray  = rl.NewColor(95, 87, 79, 255)      // #5F574F
	ColLightGray = rl.NewColor(194, 195, 199, 255)   // #C2C3C7
	ColWhite     = rl.NewColor(255, 241, 232, 255)   // #FFF1E8 Crisp Cream White
	ColPureWhite = rl.NewColor(255, 255, 255, 255)   // #FFFFFF
	ColRed       = rl.NewColor(255, 0, 77, 255)      // #FF004D
	ColOrange    = rl.NewColor(255, 163, 0, 255)     // #FFA300
	ColYellow    = rl.NewColor(255, 236, 39, 255)    // #FFEC27
	ColLime      = rl.NewColor(0, 228, 54, 255)      // #00E436
	ColSkyBlue   = rl.NewColor(41, 173, 255, 255)    // #29ADFF
	ColLavender  = rl.NewColor(131, 118, 156, 255)   // #83769C
	ColPink      = rl.NewColor(255, 119, 168, 255)   // #FF77A8
	ColPeach     = rl.NewColor(255, 204, 170, 255)   // #FFCCAA

	// Semantic Cozy UI Theme (Deep contrast, warm shadows, and high legibility)
	ColPanelBg        = rl.NewColor(16, 22, 38, 252)    // Deep warm navy background
	ColPanelBorder    = rl.NewColor(52, 70, 110, 255)   // Subtle warm border
	ColCardBg         = rl.NewColor(26, 36, 62, 240)    // High-contrast card surface
	ColCardHover      = rl.NewColor(38, 52, 88, 255)    // Elevated hover state
	ColCardActive     = rl.NewColor(28, 62, 112, 255)   // Active selected surface
	ColCardBorder     = rl.NewColor(48, 66, 104, 220)   // Smooth card border
	ColCardBorderAct  = rl.NewColor(41, 173, 255, 255)  // Glowing active border
	ColScrollTrack    = rl.NewColor(12, 16, 28, 200)    // Deep track
	ColScrollThumb    = rl.NewColor(68, 90, 140, 230)   // Cozy thumb
	ColScrollThumbHov = rl.NewColor(100, 130, 195, 255) // Glowing thumb
	ColPillBg         = rl.NewColor(22, 30, 52, 240)    // Pill bar background
	ColPillActive     = rl.NewColor(41, 173, 255, 255)  // Sky blue active pill
	ColTextTitle      = rl.NewColor(255, 241, 232, 255) // #FFF1E8 Crisp heading
	ColTextBody       = rl.NewColor(235, 242, 255, 255) // Crisp readable text
	ColTextMuted      = rl.NewColor(160, 174, 204, 255) // Soft secondary text
	ColIconBoxBg      = rl.NewColor(34, 46, 76, 255)    // Icon rounded box
)

// DrawCard draws a cozy card with soft rounded corners, border and hover effects.
func DrawCard(rec rl.Rectangle, isHovered, isSelected bool) {
	bg := ColCardBg
	border := ColCardBorder

	if isSelected {
		bg = ColCardActive
		border = ColCardBorderAct
	} else if isHovered {
		bg = ColCardHover
		border = rl.NewColor(75, 100, 155, 255)
	}

	rl.DrawRectangleRounded(rec, 0.16, 4, bg)
	rl.DrawRectangleRoundedLines(rec, 0.16, 4, border)
}

// DrawIconBadge draws an icon centered in a rounded pill/square box.
func (ui *UIState) DrawIconBadge(x, y, size float32, iconID int, iconTint, boxBg rl.Color) {
	badgeRec := rl.NewRectangle(x, y, size, size)
	rl.DrawRectangleRounded(badgeRec, 0.28, 4, boxBg)
	rl.DrawRectangleRoundedLines(badgeRec, 0.28, 4, rl.NewColor(55, 75, 120, 200))
	iconPadding := size * 0.18
	iconSize := size - (iconPadding * 2)
	GlobalIcons.DrawIcon(iconID, x+iconPadding, y+iconPadding, iconSize, iconTint)
}

// DrawBadge draws a small pill badge with text with comfortable horizontal and vertical breathing room.
func (ui *UIState) DrawBadge(x, y float32, text string, bgCol, textCol rl.Color) float32 {
	paddingH := float32(10)
	paddingV := float32(4)
	fontSize := float32(11)
	textW := ui.MeasureTextBold(text, fontSize)
	badgeW := textW + (paddingH * 2)
	badgeH := fontSize + (paddingV * 2) + 2

	badgeRec := rl.NewRectangle(x, y, badgeW, badgeH)
	rl.DrawRectangleRounded(badgeRec, 0.5, 4, bgCol)
	ui.DrawTextBold(text, int32(x+paddingH), int32(y+paddingV), fontSize, textCol)

	return badgeW
}

// DrawToggle renders a modern rounded toggle switch with smooth state indicator.
func (ui *UIState) DrawToggle(rec rl.Rectangle, label string, enabled bool, mousePos rl.Vector2) bool {
	hovered := rl.CheckCollisionPointRec(mousePos, rec)

	// Draw label
	ui.DrawText(label, int32(rec.X), int32(rec.Y+5), 13, ColTextBody)

	// Switch geometry
	switchW := float32(46)
	switchH := float32(24)
	switchX := rec.X + rec.Width - switchW
	switchY := rec.Y + (rec.Height-switchH)/2
	switchRec := rl.NewRectangle(switchX, switchY, switchW, switchH)

	switchHovered := rl.CheckCollisionPointRec(mousePos, switchRec)

	bgCol := ColDarkGray
	if enabled {
		bgCol = ColLime
	}
	if switchHovered || hovered {
		if enabled {
			bgCol = rl.NewColor(40, 245, 90, 255)
		} else {
			bgCol = rl.NewColor(120, 112, 102, 255)
		}
	}

	rl.DrawRectangleRounded(switchRec, 0.5, 4, bgCol)

	// Knob
	knobRadius := float32(9)
	knobX := switchX + knobRadius + 3
	if enabled {
		knobX = switchX + switchW - knobRadius - 3
	}
	knobY := switchY + switchH/2
	rl.DrawCircle(int32(knobX), int32(knobY), knobRadius, ColWhite)

	if (switchHovered || hovered) && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		return !enabled
	}
	return enabled
}

// DrawSliderControl renders a sleek slider with active fill, track and thumb handle.
func (ui *UIState) DrawSliderControl(
	trackRec rl.Rectangle,
	value, min, max float32,
	mousePos rl.Vector2,
	accentCol rl.Color,
) float32 {
	if max <= min {
		max = min + 1.0
	}
	ratio := (value - min) / (max - min)
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	hitRec := rl.NewRectangle(trackRec.X-10, trackRec.Y-10, trackRec.Width+20, trackRec.Height+20)
	isHovered := rl.CheckCollisionPointRec(mousePos, hitRec)

	if rl.IsMouseButtonDown(rl.MouseLeftButton) && isHovered {
		ratio = (mousePos.X - trackRec.X) / trackRec.Width
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 1 {
			ratio = 1
		}
		value = min + ratio*(max-min)
	}

	// 1. Draw Track
	rl.DrawRectangleRounded(trackRec, 0.5, 4, ColScrollTrack)
	rl.DrawRectangleRoundedLines(trackRec, 0.5, 4, rl.NewColor(35, 48, 76, 200))

	// 2. Draw Active Fill
	fillW := ratio * trackRec.Width
	if fillW > 0 {
		fillRec := rl.NewRectangle(trackRec.X, trackRec.Y, fillW, trackRec.Height)
		rl.DrawRectangleRounded(fillRec, 0.5, 4, accentCol)
	}

	// 3. Draw Thumb Handle
	thumbX := trackRec.X + fillW
	thumbY := trackRec.Y + trackRec.Height/2
	thumbRadius := float32(7.5)
	if isHovered {
		thumbRadius = 9.5
	}
	rl.DrawCircle(int32(thumbX), int32(thumbY), thumbRadius+1, ColDarkNavy)
	rl.DrawCircle(int32(thumbX), int32(thumbY), thumbRadius, ColWhite)

	return value
}
