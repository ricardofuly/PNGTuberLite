package costume

// CostumeManager manages the active costume slot (1 to 10) and hotkey mappings.
type CostumeManager struct {
	ActiveCostume  int  // 1-indexed costume slot (1..10)
	BounceOnChange bool // trigger physical jump when costume changes
}

// NewCostumeManager creates a costume manager with slot 1 active.
func NewCostumeManager(bounceOnChange bool) *CostumeManager {
	return &CostumeManager{
		ActiveCostume:  1,
		BounceOnChange: bounceOnChange,
	}
}

// SetCostume changes the active costume slot. Returns true if slot changed.
func (cm *CostumeManager) SetCostume(slot int) bool {
	if slot < 1 || slot > 10 {
		return false
	}
	if cm.ActiveCostume == slot {
		return false
	}
	cm.ActiveCostume = slot
	return true
}

// GetCostume returns the current active costume slot.
func (cm *CostumeManager) GetCostume() int {
	if cm.ActiveCostume < 1 {
		return 1
	}
	return cm.ActiveCostume
}
