package audio

import (
	"math"
	"testing"
)

func TestCalculateRMS(t *testing.T) {
	// Silence
	silence := make([]float32, 100)
	if rms := CalculateRMS(silence); rms != 0 {
		t.Errorf("expected silence RMS 0, got %f", rms)
	}

	// Alternating AC signal (+0.5, -0.5)
	acSignal := make([]float32, 100)
	for i := range acSignal {
		if i%2 == 0 {
			acSignal[i] = 0.5
		} else {
			acSignal[i] = -0.5
		}
	}
	if rms := CalculateRMS(acSignal); math.Abs(float64(rms-0.5)) > 0.001 {
		t.Errorf("expected AC signal RMS ~0.5, got %f", rms)
	}
}

func TestVADHysteresisAndDebounce(t *testing.T) {
	vad := NewVoiceActivityDetector(0.1)
	vad.HoldOffFrames = 3
	vad.AttackFrames = 1

	// Silence input
	silence := make([]float32, 50)
	speaking, _ := vad.Process(silence)
	if speaking {
		t.Errorf("expected not speaking on silence")
	}

	// Loud AC speech buffer (0.5 amplitude)
	loud := make([]float32, 50)
	for i := range loud {
		if i%2 == 0 {
			loud[i] = 0.5
		} else {
			loud[i] = -0.5
		}
	}

	speaking, _ = vad.Process(loud)
	if !speaking {
		t.Errorf("expected speaking on loud audio")
	}

	// Mid-level AC audio (0.09) - below threshold 0.1 but above hysteresis threshold (0.1 * 0.80 = 0.08)
	mid := make([]float32, 50)
	for i := range mid {
		if i%2 == 0 {
			mid[i] = 0.09
		} else {
			mid[i] = -0.09
		}
	}

	speaking, _ = vad.Process(mid)
	if !speaking {
		t.Errorf("expected to stay speaking due to hysteresis threshold")
	}

	// 1st frame of silence (below hysteresis threshold): should still hold
	speaking, _ = vad.Process(silence)
	if !speaking {
		t.Errorf("expected to stay speaking during hold-off debounce frame 1")
	}

	// 2nd frame of silence: should still hold
	speaking, _ = vad.Process(silence)
	if !speaking {
		t.Errorf("expected to stay speaking during hold-off debounce frame 2")
	}

	// 3rd frame of silence: hold-off limit reached, should close mouth
	speaking, _ = vad.Process(silence)
	if speaking {
		t.Errorf("expected to stop speaking after hold-off debounce finished")
	}
}
