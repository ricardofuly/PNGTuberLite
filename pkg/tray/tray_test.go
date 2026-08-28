package tray

import (
	"testing"
)

func TestTrayManagerSignals(t *testing.T) {
	tm := GetTrayManager()
	if tm == nil {
		t.Fatalf("expected non-nil TrayManager")
	}

	// Initially false
	if tm.CheckAndClearRestore() {
		t.Errorf("expected CheckAndClearRestore to be false initially")
	}
	if tm.CheckAndClearQuit() {
		t.Errorf("expected CheckAndClearQuit to be false initially")
	}

	// Test RequestRestore
	tm.RequestRestore()
	if !tm.CheckAndClearRestore() {
		t.Errorf("expected CheckAndClearRestore to be true after RequestRestore")
	}
	if tm.CheckAndClearRestore() {
		t.Errorf("expected CheckAndClearRestore to reset to false")
	}

	// Test RequestQuit
	tm.RequestQuit()
	if !tm.CheckAndClearQuit() {
		t.Errorf("expected CheckAndClearQuit to be true after RequestQuit")
	}
	if tm.CheckAndClearQuit() {
		t.Errorf("expected CheckAndClearQuit to reset to false")
	}
}
