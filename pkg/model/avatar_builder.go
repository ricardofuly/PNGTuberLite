package model

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ParsedPNGMeta represents detected PNGTuber metadata for a single PNG sprite.
type ParsedPNGMeta struct {
	FilePath     string
	FileName     string
	CostumeName  string
	CostumeIndex int
	ShowBlink    int // 0: always, 1: open eyes, 2: blinking
	ShowTalk     int // 0: always, 1: silence, 2: talking
	ImageData    []byte
	Width        int
	Height       int
}

// BuildAvatarFromDirectory automatically creates a complete .save avatar structure from a directory of PNGs.
func BuildAvatarFromDirectory(dirPath string) (*Avatar, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler diretório de imagens: %w", err)
	}

	var pngFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext == ".png" {
			pngFiles = append(pngFiles, filepath.Join(dirPath, entry.Name()))
		}
	}

	if len(pngFiles) == 0 {
		return nil, fmt.Errorf("nenhum arquivo .png encontrado em %q", dirPath)
	}

	return BuildAvatarFromPNGFiles(pngFiles)
}

// BuildAvatarFromPNGFiles parses a list of PNG filepaths into a fully rigged Avatar model.
func BuildAvatarFromPNGFiles(filePaths []string) (*Avatar, error) {
	if len(filePaths) == 0 {
		return nil, fmt.Errorf("nenhum arquivo informado")
	}

	var parsedList []ParsedPNGMeta

	// 1. Read files and extract dimensions
	for _, fp := range filePaths {
		data, err := os.ReadFile(fp)
		if err != nil {
			continue
		}
		w, h := ExtractPNGDimensions(data)
		if w == 0 || h == 0 {
			continue
		}

		base := filepath.Base(fp)
		nameWithoutExt := strings.TrimSuffix(base, filepath.Ext(base))

		costume, blink, talk := DetectPNGTuberStates(nameWithoutExt)

		parsedList = append(parsedList, ParsedPNGMeta{
			FilePath:    fp,
			FileName:    base,
			CostumeName: costume,
			ShowBlink:   blink,
			ShowTalk:    talk,
			ImageData:   data,
			Width:       w,
			Height:      h,
		})
	}

	if len(parsedList) == 0 {
		return nil, fmt.Errorf("nenhuma imagem PNG válida encontrada")
	}

	// 2. Map and assign costume indexes (1 to 10)
	costumeMap := make(map[string]bool)
	for _, p := range parsedList {
		if p.CostumeName != "" {
			costumeMap[p.CostumeName] = true
		}
	}

	var costumeNames []string
	for c := range costumeMap {
		costumeNames = append(costumeNames, c)
	}

	// Sort so "Default", "Normal", "Idle" is costume 1, others alphabetical
	sort.Slice(costumeNames, func(i, j int) bool {
		lowerI := strings.ToLower(costumeNames[i])
		lowerJ := strings.ToLower(costumeNames[j])
		if isDefaultCostumeName(lowerI) && !isDefaultCostumeName(lowerJ) {
			return true
		}
		if !isDefaultCostumeName(lowerI) && isDefaultCostumeName(lowerJ) {
			return false
		}
		return lowerI < lowerJ
	})

	costumeIndexMap := make(map[string]int)
	for i, c := range costumeNames {
		if i < 10 {
			costumeIndexMap[c] = i + 1 // 1-indexed (1..10)
		} else {
			costumeIndexMap[c] = 10
		}
	}

	// 3. Construct Avatar and Layers
	avatar := NewAvatar()
	baseID := time.Now().UnixMilli()

	for i, p := range parsedList {
		layerID := baseID + int64(i)
		layer := NewDefaultLayer(layerID)
		layer.Path = p.FileName
		layer.ImageData = p.ImageData
		layer.ImageWidth = p.Width
		layer.ImageHeight = p.Height
		layer.ShowBlink = p.ShowBlink
		layer.ShowTalk = p.ShowTalk
		layer.ZIndex = i

		// Set costume active slot
		cIdx, exists := costumeIndexMap[p.CostumeName]
		if exists && len(costumeNames) > 1 {
			var costumes [10]int
			costumes[cIdx-1] = 1
			layer.CostumeLayers = costumes
		} else {
			// Single costume: active on all 10 slots
			layer.CostumeLayers = [10]int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
		}

		layer.UpdateContentBounds()
		avatar.AddLayer(layer)
	}

	avatar.BuildHierarchy()
	return avatar, nil
}

func isDefaultCostumeName(name string) bool {
	return name == "default" || name == "normal" || name == "idle" || name == "padrao" || name == "base"
}

// DetectPNGTuberStates automatically inspects a sprite filename to determine its costume name, blink state, and talk state.
func DetectPNGTuberStates(name string) (costume string, showBlink int, showTalk int) {
	name = strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	lower := strings.ToLower(name)

	// Detect Blink State (1: Open, 2: Blinking, 0: Always)
	isBlink := strings.Contains(lower, "blink") || strings.Contains(lower, "pisc") || strings.Contains(lower, "fechad") || strings.Contains(lower, "eyes_closed")
	if isBlink {
		showBlink = 2
	} else {
		showBlink = 1
	}

	// Detect Talk State (1: Silence/Closed, 2: Talking/Open, 0: Always)
	isOpenMouth := strings.Contains(lower, "open") || strings.Contains(lower, "talk") || strings.Contains(lower, "mouth") || strings.Contains(lower, "fala") || strings.Contains(lower, "abert")
	if isOpenMouth {
		showTalk = 2
	} else {
		showTalk = 1
	}

	// Extract Costume/Expression Prefix
	// Strip state words: "open", "closed", "blink", "mouth", "openmouth", "openblink"
	cleaned := name
	stripWords := []string{
		"OpenMouth", "OpenBlink", "ClosedBlink", "Open_Blink", "Closed_Blink",
		"Open", "Closed", "Blink", "Mouth", "Talk", "Quiet", "Silence",
		"Aberto", "Fechado", "Piscando", "Piscar", "Falando", "Calado",
	}

	for _, w := range stripWords {
		cleaned = strings.ReplaceAll(cleaned, w, "")
		cleaned = strings.ReplaceAll(cleaned, strings.ToLower(w), "")
		cleaned = strings.ReplaceAll(cleaned, strings.ToUpper(w), "")
	}

	cleaned = strings.Trim(cleaned, "_- ")
	if cleaned == "" {
		cleaned = "Default"
	}

	return cleaned, showBlink, showTalk
}
