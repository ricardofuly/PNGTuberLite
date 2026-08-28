package profiler

import (
	"testing"
	"time"
)

func TestSystemProfiler(t *testing.T) {
	p := NewSystemProfiler()
	if p == nil {
		t.Fatalf("expected non-nil profiler")
	}

	// Update with test frame metrics
	time.Sleep(10 * time.Millisecond)
	stats := p.Update(0.016, 60, 9, 1024*1024*4)

	if stats.FPS != 60 {
		t.Errorf("expected FPS 60, got %d", stats.FPS)
	}
	if stats.TextureCount != 9 {
		t.Errorf("expected TextureCount 9, got %d", stats.TextureCount)
	}
	if stats.VRAMMB != 4.0 {
		t.Errorf("expected VRAMMB 4.0, got %f", stats.VRAMMB)
	}

	summary := p.FormatStats()
	if summary == "" {
		t.Errorf("expected non-empty summary string")
	}
}
