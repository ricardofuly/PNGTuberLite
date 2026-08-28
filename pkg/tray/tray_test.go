package tray

import (
	"testing"
)

func TestTrayManagerSetup(t *testing.T) {
	tm := GetTrayManager()
	if tm == nil {
		t.Fatalf("expected non-nil TrayManager")
	}

	openCalled := false
	quitCalled := false

	tm.Setup(func() {
		openCalled = true
	}, func() {
		quitCalled = true
	})

	if tm.onOpen == nil {
		t.Errorf("expected onOpen to be registered")
	}
	if tm.onQuit == nil {
		t.Errorf("expected onQuit to be registered")
	}

	// Test calling registered callbacks
	tm.onOpen()
	if !openCalled {
		t.Errorf("expected onOpen to set openCalled to true")
	}

	tm.onQuit()
	if !quitCalled {
		t.Errorf("expected onQuit to set quitCalled to true")
	}
}
