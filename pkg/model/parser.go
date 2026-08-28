package model

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// RawLayerData represents the raw unparsed fields from the Godot .save file.
type RawLayerData struct {
	Identification interface{} `json:"identification"`
	ParentID       interface{} `json:"parentId"`
	Path           *string     `json:"path"`
	Type           *string     `json:"type"`
	ZIndex         interface{} `json:"zindex"`
	Pos            interface{} `json:"pos"`
	Offset         interface{} `json:"offset"`
	Frames         interface{} `json:"frames"`
	AnimSpeed      interface{} `json:"animSpeed"`
	Clipped        interface{} `json:"clipped"`
	StretchAmount  interface{} `json:"stretchAmount"`
	RLimitMin      interface{} `json:"rLimitMin"`
	RLimitMax      interface{} `json:"rLimitMax"`
	RotDrag        interface{} `json:"rotDrag"`
	Drag           interface{} `json:"drag"`
	XAmp           interface{} `json:"xAmp"`
	XFrq           interface{} `json:"xFrq"`
	YAmp           interface{} `json:"yAmp"`
	YFrq           interface{} `json:"yFrq"`
	IgnoreBounce   interface{} `json:"ignoreBounce"`
	ShowBlink      interface{} `json:"showBlink"`
	ShowTalk       interface{} `json:"showTalk"`
	CostumeLayers  interface{} `json:"costumeLayers"`
	ImageData      *string     `json:"imageData"`
}

// ParseSaveFile reads and parses a .save file from the filesystem.
func ParseSaveFile(filePath string) (*Avatar, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read save file %q: %w", filePath, err)
	}
	return ParseSaveData(data)
}

// ParseSaveData parses the byte content of a .save file.
func ParseSaveData(data []byte) (*Avatar, error) {
	// Clean potential BOM or leading/trailing whitespace
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, fmt.Errorf("empty save data")
	}

	var rawMap map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&rawMap); err != nil {
		return nil, fmt.Errorf("failed to decode JSON root structure: %w", err)
	}

	avatar := NewAvatar()

	for key, rawJSON := range rawMap {
		layer, err := parseLayer(rawJSON)
		if err != nil {
			// Forward compatibility: log or skip corrupted layer, but try to continue
			return nil, fmt.Errorf("error parsing layer key %q: %w", key, err)
		}
		if layer != nil {
			avatar.AddLayer(layer)
		}
	}

	avatar.BuildHierarchy()
	return avatar, nil
}

func parseLayer(data json.RawMessage) (*Layer, error) {
	var raw RawLayerData
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("invalid layer json: %w", err)
	}

	// 1. Identification
	id, err := toInt64(raw.Identification)
	if err != nil {
		return nil, fmt.Errorf("missing or invalid layer identification: %w", err)
	}

	layer := NewDefaultLayer(id)

	// 2. ParentID
	if raw.ParentID != nil {
		if pID, err := toInt64(raw.ParentID); err == nil && pID != 0 {
			layer.ParentID = &pID
		}
	}

	// 3. Path & Type
	if raw.Path != nil {
		layer.Path = *raw.Path
	}
	if raw.Type != nil {
		layer.Type = *raw.Type
	}

	// 4. ZIndex
	if raw.ZIndex != nil {
		if z, err := toInt(raw.ZIndex); err == nil {
			layer.ZIndex = z
		}
	}

	// 5. Pos & Offset
	if raw.Pos != nil {
		if posStr, ok := raw.Pos.(string); ok {
			if v, err := ParseVector2(posStr); err == nil {
				layer.Pos = v
			}
		}
	}
	if raw.Offset != nil {
		if offsetStr, ok := raw.Offset.(string); ok {
			if v, err := ParseVector2(offsetStr); err == nil {
				layer.Offset = v
			}
		}
	}

	// 6. Frames & AnimSpeed
	if raw.Frames != nil {
		if f, err := toInt(raw.Frames); err == nil && f > 0 {
			layer.Frames = f
		}
	}
	if raw.AnimSpeed != nil {
		if s, err := toFloat32(raw.AnimSpeed); err == nil {
			layer.AnimSpeed = s
		}
	}

	// 7. Clipped & StretchAmount
	if raw.Clipped != nil {
		if c, err := toBool(raw.Clipped); err == nil {
			layer.Clipped = c
		}
	}
	if raw.StretchAmount != nil {
		if s, err := toFloat32(raw.StretchAmount); err == nil {
			layer.StretchAmount = s
		}
	}

	// 8. Rotation limits & Physics damping
	if raw.RLimitMin != nil {
		if r, err := toFloat32(raw.RLimitMin); err == nil {
			layer.RLimitMin = r
		}
	}
	if raw.RLimitMax != nil {
		if r, err := toFloat32(raw.RLimitMax); err == nil {
			layer.RLimitMax = r
		}
	}
	if raw.RotDrag != nil {
		if r, err := toFloat32(raw.RotDrag); err == nil {
			layer.RotDrag = r
		}
	}
	if raw.Drag != nil {
		if d, err := toFloat32(raw.Drag); err == nil {
			layer.Drag = d
		}
	}

	// 9. Oscillation parameters
	if raw.XAmp != nil {
		if v, err := toFloat32(raw.XAmp); err == nil {
			layer.XAmp = v
		}
	}
	if raw.XFrq != nil {
		if v, err := toFloat32(raw.XFrq); err == nil {
			layer.XFrq = v
		}
	}
	if raw.YAmp != nil {
		if v, err := toFloat32(raw.YAmp); err == nil {
			layer.YAmp = v
		}
	}
	if raw.YFrq != nil {
		if v, err := toFloat32(raw.YFrq); err == nil {
			layer.YFrq = v
		}
	}

	// 10. Bounce & Visibility states
	if raw.IgnoreBounce != nil {
		if b, err := toBool(raw.IgnoreBounce); err == nil {
			layer.IgnoreBounce = b
		}
	}
	if raw.ShowBlink != nil {
		if b, err := toInt(raw.ShowBlink); err == nil {
			layer.ShowBlink = b
		}
	}
	if raw.ShowTalk != nil {
		if t, err := toInt(raw.ShowTalk); err == nil {
			layer.ShowTalk = t
		}
	}

	// 11. Costume layers
	if raw.CostumeLayers != nil {
		if costumes, err := parseCostumeLayers(raw.CostumeLayers); err == nil {
			layer.CostumeLayers = costumes
		}
	}

	// 12. Image Data (Base64 PNG)
	if raw.ImageData != nil && *raw.ImageData != "" {
		imgBytes, err := decodeBase64Image(*raw.ImageData)
		if err != nil {
			return nil, fmt.Errorf("failed to decode imageData for layer %d: %w", id, err)
		}
		layer.ImageData = imgBytes
		w, h := ExtractPNGDimensions(imgBytes)
		layer.ImageWidth = w
		layer.ImageHeight = h
		layer.UpdateContentBounds()
	}

	return layer, nil
}

// parseCostumeLayers handles string representation like "[1, 1, 1, 1, 1, 1, 1, 1, 1, 1]" or slice of ints.
func parseCostumeLayers(val interface{}) ([10]int, error) {
	result := [10]int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}

	switch v := val.(type) {
	case string:
		clean := strings.Trim(v, "[] \t\r\n")
		if clean == "" {
			return result, nil
		}
		parts := strings.Split(clean, ",")
		for i := 0; i < len(parts) && i < 10; i++ {
			p := strings.TrimSpace(parts[i])
			if n, err := strconv.Atoi(p); err == nil {
				result[i] = n
			}
		}
	case []interface{}:
		for i := 0; i < len(v) && i < 10; i++ {
			if n, err := toInt(v[i]); err == nil {
				result[i] = n
			}
		}
	}

	return result, nil
}

// decodeBase64Image decodes a base64 string, stripping data URI headers if present.
func decodeBase64Image(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	// Strip "data:image/png;base64," prefix if it exists
	if idx := strings.Index(raw, ","); idx != -1 && strings.HasPrefix(raw, "data:") {
		raw = raw[idx+1:]
	}
	// Remove all whitespace/newlines
	raw = strings.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
			return -1
		}
		return r
	}, raw)

	// Decode standard base64
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		// Fallback to URL-safe encoding
		var err2 error
		data, err2 = base64.URLEncoding.DecodeString(raw)
		if err2 != nil {
			return nil, err
		}
	}
	return data, nil
}

// ExtractPNGDimensions reads the width and height from a PNG IHDR header without loading the entire image.
func ExtractPNGDimensions(pngData []byte) (int, int) {
	// Minimum PNG length with IHDR is 33 bytes:
	// 8 bytes signature + 4 bytes length + 4 bytes "IHDR" + 13 bytes data + 4 bytes CRC
	if len(pngData) < 24 {
		return 0, 0
	}
	// Check PNG signature: \x89PNG\r\n\x1a\n
	pngSignature := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	if !bytes.HasPrefix(pngData, pngSignature) {
		return 0, 0
	}
	// In IHDR: width is at bytes 16..20, height is at bytes 20..24 (BigEndian uint32)
	width := int(binary.BigEndian.Uint32(pngData[16:20]))
	height := int(binary.BigEndian.Uint32(pngData[20:24]))
	return width, height
}

// Helper conversion functions for flexible JSON type parsing

func toInt64(val interface{}) (int64, error) {
	if val == nil {
		return 0, fmt.Errorf("nil value")
	}
	switch v := val.(type) {
	case json.Number:
		return v.Int64()
	case float64:
		return int64(v), nil
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case string:
		return strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", val)
	}
}

func toInt(val interface{}) (int, error) {
	n, err := toInt64(val)
	return int(n), err
}

func toFloat32(val interface{}) (float32, error) {
	if val == nil {
		return 0, fmt.Errorf("nil value")
	}
	switch v := val.(type) {
	case json.Number:
		f, err := v.Float64()
		return float32(f), err
	case float64:
		return float32(v), nil
	case float32:
		return v, nil
	case int64:
		return float32(v), nil
	case int:
		return float32(v), nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 32)
		return float32(f), err
	default:
		return 0, fmt.Errorf("cannot convert %T to float32", val)
	}
}

func toBool(val interface{}) (bool, error) {
	if val == nil {
		return false, fmt.Errorf("nil value")
	}
	switch v := val.(type) {
	case bool:
		return v, nil
	case json.Number:
		n, err := v.Int64()
		return n != 0, err
	case int, int64, float64:
		n, _ := toInt64(v)
		return n != 0, nil
	case string:
		s := strings.ToLower(strings.TrimSpace(v))
		if s == "true" || s == "1" {
			return true, nil
		}
		if s == "false" || s == "0" {
			return false, nil
		}
		return false, fmt.Errorf("cannot parse boolean from string %q", v)
	default:
		return false, fmt.Errorf("cannot convert %T to bool", val)
	}
}

// SerializeAvatar converts an Avatar into the official PNGTuber-Plus .save JSON format.
func SerializeAvatar(avatar *Avatar) ([]byte, error) {
	if avatar == nil {
		return nil, fmt.Errorf("nil avatar")
	}

	rawMap := make(map[string]map[string]interface{})

	index := 0
	for _, layer := range avatar.DrawOrder {
		layerMap := make(map[string]interface{})

		layerMap["identification"] = layer.Identification
		if layer.ParentID != nil && *layer.ParentID != 0 {
			layerMap["parentId"] = *layer.ParentID
		} else {
			layerMap["parentId"] = 0
		}

		layerMap["path"] = layer.Path
		if layer.Type != "" {
			layerMap["type"] = layer.Type
		} else {
			layerMap["type"] = "sprite"
		}

		layerMap["zindex"] = layer.ZIndex
		layerMap["pos"] = fmt.Sprintf("Vector2(%g, %g)", layer.Pos.X, layer.Pos.Y)
		layerMap["offset"] = fmt.Sprintf("Vector2(%g, %g)", layer.Offset.X, layer.Offset.Y)
		layerMap["frames"] = layer.Frames
		layerMap["animSpeed"] = layer.AnimSpeed
		layerMap["clipped"] = layer.Clipped
		layerMap["stretchAmount"] = layer.StretchAmount
		layerMap["rLimitMin"] = layer.RLimitMin
		layerMap["rLimitMax"] = layer.RLimitMax
		layerMap["rotDrag"] = layer.RotDrag
		layerMap["drag"] = layer.Drag
		layerMap["xAmp"] = layer.XAmp
		layerMap["xFrq"] = layer.XFrq
		layerMap["yAmp"] = layer.YAmp
		layerMap["yFrq"] = layer.YFrq
		layerMap["ignoreBounce"] = layer.IgnoreBounce
		layerMap["showBlink"] = layer.ShowBlink
		layerMap["showTalk"] = layer.ShowTalk

		// Format costumeLayers as string array: "[1, 1, 1, 1, 1, 1, 1, 1, 1, 1]"
		var costStrs []string
		for _, c := range layer.CostumeLayers {
			costStrs = append(costStrs, strconv.Itoa(c))
		}
		layerMap["costumeLayers"] = "[" + strings.Join(costStrs, ", ") + "]"

		// Base64 encode imageData
		if len(layer.ImageData) > 0 {
			layerMap["imageData"] = base64.StdEncoding.EncodeToString(layer.ImageData)
		} else {
			layerMap["imageData"] = ""
		}

		key := strconv.Itoa(index)
		rawMap[key] = layerMap
		index++
	}

	return json.MarshalIndent(rawMap, "", "  ")
}

// SaveAvatarToFile writes the avatar to a .save file on disk.
func SaveAvatarToFile(avatar *Avatar, filePath string) error {
	data, err := SerializeAvatar(avatar)
	if err != nil {
		return err
	}

	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return os.WriteFile(filePath, data, 0644)
}

