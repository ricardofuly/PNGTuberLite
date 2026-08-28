package model

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Vector2 represents a 2D vector with X and Y float32 coordinates.
type Vector2 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

// NewVector2 creates a new Vector2.
func NewVector2(x, y float32) Vector2 {
	return Vector2{X: x, Y: y}
}

// Add returns the sum of v and other.
func (v Vector2) Add(other Vector2) Vector2 {
	return Vector2{X: v.X + other.X, Y: v.Y + other.Y}
}

// Sub returns the difference of v and other.
func (v Vector2) Sub(other Vector2) Vector2 {
	return Vector2{X: v.X - other.X, Y: v.Y - other.Y}
}

// Scale multiplies the vector by a scalar.
func (v Vector2) Scale(s float32) Vector2 {
	return Vector2{X: v.X * s, Y: v.Y * s}
}

// String returns the Godot-formatted string representation.
func (v Vector2) String() string {
	return fmt.Sprintf("Vector2(%g, %g)", v.X, v.Y)
}

var vec2Regex = regexp.MustCompile(`Vector2\(\s*([-\d.eE+]+)\s*,\s*([-\d.eE+]+)\s*\)`)

// ParseVector2 parses a Godot Vector2 string representation such as "Vector2(10.5, -20.0)".
func ParseVector2(s string) (Vector2, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Vector2{}, nil
	}

	matches := vec2Regex.FindStringSubmatch(s)
	if len(matches) == 3 {
		x, errX := strconv.ParseFloat(matches[1], 32)
		y, errY := strconv.ParseFloat(matches[2], 32)
		if errX != nil || errY != nil {
			return Vector2{}, fmt.Errorf("invalid coordinates in Vector2 string %q: %v, %v", s, errX, errY)
		}
		return Vector2{X: float32(x), Y: float32(y)}, nil
	}

	// Fallback: try parsing comma-separated "x, y"
	parts := strings.Split(s, ",")
	if len(parts) == 2 {
		cleanX := strings.Trim(parts[0], " ()[]\"'")
		cleanY := strings.Trim(parts[1], " ()[]\"'")
		x, errX := strconv.ParseFloat(cleanX, 32)
		y, errY := strconv.ParseFloat(cleanY, 32)
		if errX == nil && errY == nil {
			return Vector2{X: float32(x), Y: float32(y)}, nil
		}
	}

	return Vector2{}, fmt.Errorf("cannot parse Vector2 from string %q", s)
}
