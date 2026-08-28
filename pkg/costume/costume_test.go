package costume

import "testing"

func TestCostumeManager(t *testing.T) {
	cm := NewCostumeManager(true)
	if slot := cm.GetCostume(); slot != 1 {
		t.Errorf("expected initial costume 1, got %d", slot)
	}

	// Change to valid slot 3
	changed := cm.SetCostume(3)
	if !changed || cm.GetCostume() != 3 {
		t.Errorf("expected costume to change to 3")
	}

	// Change to same slot
	changed = cm.SetCostume(3)
	if changed {
		t.Errorf("expected changed=false when setting same costume")
	}

	// Invalid slot 11
	changed = cm.SetCostume(11)
	if changed || cm.GetCostume() != 3 {
		t.Errorf("expected invalid slot 11 to be rejected")
	}
}
