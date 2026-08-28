package model

import (
	"bytes"
	"image"
	_ "image/png"
)

// CalculateContentBounds scans the image for non-transparent pixels (Alpha > 15).
// It returns (minX, minY, maxX, maxY) relative to the frame (0 to frameWidth, 0 to frameHeight).
func CalculateContentBounds(imageData []byte, frames int) (minX, minY, maxX, maxY float32, found bool) {
	if len(imageData) == 0 {
		return 0, 0, 0, 0, false
	}

	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return 0, 0, 0, 0, false
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w <= 0 || h <= 0 {
		return 0, 0, 0, 0, false
	}

	if frames < 1 {
		frames = 1
	}
	if frames > w {
		frames = w
	}
	frameW := w / frames
	if frameW <= 0 {
		frameW = 1
	}

	minI := frameW
	minJ := h
	maxI := -1
	maxJ := -1

	for f := 0; f < frames; f++ {
		fStartX := bounds.Min.X + (f * frameW)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for localX := 0; localX < frameW; localX++ {
				x := fStartX + localX
				if x >= bounds.Max.X {
					continue
				}
				_, _, _, a := img.At(x, y).RGBA()
				if a > 3800 { // Alpha > ~15 / 255
					if localX < minI {
						minI = localX
					}
					if localX > maxI {
						maxI = localX
					}
					relY := y - bounds.Min.Y
					if relY < minJ {
						minJ = relY
					}
					if relY > maxJ {
						maxJ = relY
					}
				}
			}
		}
	}

	if maxI >= minI && maxJ >= minJ {
		pMinX := float32(minI)
		pMinY := float32(minJ)
		pMaxX := float32(maxI + 1)
		pMaxY := float32(maxJ + 1)
		return pMinX, pMinY, pMaxX, pMaxY, true
	}

	return 0, 0, float32(frameW), float32(h), false
}
