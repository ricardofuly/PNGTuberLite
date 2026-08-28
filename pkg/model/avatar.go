package model

import (
	"sort"
)

// Avatar holds the entire collection of layers, hierarchy relationships, and rendering order.
type Avatar struct {
	Layers     map[int64]*Layer   `json:"layers"`
	RootLayers []*Layer           `json:"-"`
	Children   map[int64][]*Layer `json:"-"`
	DrawOrder  []*Layer           `json:"-"`
}

// NewAvatar creates an empty avatar structure.
func NewAvatar() *Avatar {
	return &Avatar{
		Layers:     make(map[int64]*Layer),
		RootLayers: make([]*Layer, 0),
		Children:   make(map[int64][]*Layer),
		DrawOrder:  make([]*Layer, 0),
	}
}

// AddLayer adds a layer to the avatar.
func (a *Avatar) AddLayer(layer *Layer) {
	if layer == nil {
		return
	}
	a.Layers[layer.Identification] = layer
}

// RemoveLayer deletes a layer by its identification ID and updates child parentIDs.
func (a *Avatar) RemoveLayer(id int64) {
	delete(a.Layers, id)
	for _, layer := range a.Layers {
		if layer.ParentID != nil && *layer.ParentID == id {
			layer.ParentID = nil
		}
	}
}

// GetLayer returns a layer by its identification ID, or nil if not found.
func (a *Avatar) GetLayer(id int64) *Layer {
	return a.Layers[id]
}

// GetChildren returns the immediate children of the specified parent layer ID.
func (a *Avatar) GetChildren(parentID int64) []*Layer {
	return a.Children[parentID]
}

// BuildHierarchy resolves the parent-child relationships and constructs the draw order.
func (a *Avatar) BuildHierarchy() {
	a.RootLayers = make([]*Layer, 0)
	a.Children = make(map[int64][]*Layer)
	a.DrawOrder = make([]*Layer, 0, len(a.Layers))

	for _, layer := range a.Layers {
		a.DrawOrder = append(a.DrawOrder, layer)

		if layer.ParentID == nil || *layer.ParentID == 0 || *layer.ParentID == layer.Identification {
			a.RootLayers = append(a.RootLayers, layer)
		} else {
			pID := *layer.ParentID
			// Check if the parent actually exists
			if _, exists := a.Layers[pID]; exists {
				a.Children[pID] = append(a.Children[pID], layer)
			} else {
				// Parent not found, treat as root
				a.RootLayers = append(a.RootLayers, layer)
			}
		}
	}

	// Calculate tree depth for each layer (roots = 0, children = 1, grandchildren = 2)
	depthMap := make(map[int64]int, len(a.Layers))
	var calcDepth func(id int64, currentDepth int)
	calcDepth = func(id int64, currentDepth int) {
		depthMap[id] = currentDepth
		for _, child := range a.Children[id] {
			calcDepth(child.Identification, currentDepth+1)
		}
	}
	for _, root := range a.RootLayers {
		calcDepth(root.Identification, 0)
	}

	// Sort draw order:
	// 1. ZIndex ascending (lower ZIndex is drawn behind, higher ZIndex in front)
	// 2. When ZIndex is equal: Tree Depth ascending (parents drawn before children, so children render ON TOP of parent)
	// 3. Identification tie-breaker
	sort.SliceStable(a.DrawOrder, func(i, j int) bool {
		if a.DrawOrder[i].ZIndex != a.DrawOrder[j].ZIndex {
			return a.DrawOrder[i].ZIndex < a.DrawOrder[j].ZIndex
		}
		depthI := depthMap[a.DrawOrder[i].Identification]
		depthJ := depthMap[a.DrawOrder[j].Identification]
		if depthI != depthJ {
			return depthI < depthJ
		}
		return a.DrawOrder[i].Identification < a.DrawOrder[j].Identification
	})
}
