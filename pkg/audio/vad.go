package audio

import (
	"math"
)

// VoiceActivityDetector processes audio PCM samples and determines speaking state with hysteresis and debounce.
type VoiceActivityDetector struct {
	Threshold         float32 // RMS activation threshold (default ~0.05)
	HysteresisRatio   float32 // Deactivation threshold ratio (e.g. 0.80 of Threshold)
	HoldOffFrames     int     // Frames of silence before closing mouth (debounce, e.g. 6 frames / ~100ms)
	AttackFrames      int     // Frames of speech required before opening mouth (e.g. 1 frame)

	isSpeaking        bool
	consecutiveSpeech int
	consecutiveSilent int
	currentRMS        float32
}

// NewVoiceActivityDetector creates a new VAD with anti-flicker settings.
func NewVoiceActivityDetector(threshold float32) *VoiceActivityDetector {
	if threshold <= 0 {
		threshold = 0.05
	}
	return &VoiceActivityDetector{
		Threshold:       threshold,
		HysteresisRatio: 0.80,
		HoldOffFrames:   6, // ~100ms hold for snappy, responsive mouth closing
		AttackFrames:    1,
	}
}

// CalculateRMS computes the root mean square energy of a float32 audio sample slice with DC bias removal.
func CalculateRMS(samples []float32) float32 {
	if len(samples) == 0 {
		return 0
	}

	// 1. Calculate DC mean to eliminate microphone hardware bias / DC offset
	var sum float64
	for _, s := range samples {
		sum += float64(s)
	}
	mean := sum / float64(len(samples))

	// 2. Compute variance / AC power
	var sumSq float64
	for _, s := range samples {
		diff := float64(s) - mean
		sumSq += diff * diff
	}

	return float32(math.Sqrt(sumSq / float64(len(samples))))
}

// Process evaluates a new buffer of audio samples and updates the speaking state.
func (vad *VoiceActivityDetector) Process(samples []float32) (bool, float32) {
	rms := CalculateRMS(samples)
	vad.currentRMS = rms

	onThreshold := vad.Threshold
	offThreshold := vad.Threshold * vad.HysteresisRatio

	if vad.isSpeaking {
		if rms < offThreshold {
			vad.consecutiveSilent++
			if vad.consecutiveSilent >= vad.HoldOffFrames {
				vad.isSpeaking = false
				vad.consecutiveSilent = 0
				vad.consecutiveSpeech = 0
			}
		} else {
			vad.consecutiveSilent = 0
		}
	} else {
		if rms >= onThreshold {
			vad.consecutiveSpeech++
			if vad.consecutiveSpeech >= vad.AttackFrames {
				vad.isSpeaking = true
				vad.consecutiveSpeech = 0
				vad.consecutiveSilent = 0
			}
		} else {
			vad.consecutiveSpeech = 0
		}
	}

	return vad.isSpeaking, rms
}

// IsSpeaking returns the current filtered speech state.
func (vad *VoiceActivityDetector) IsSpeaking() bool {
	return vad.isSpeaking
}

// CurrentRMS returns the most recently computed RMS volume level.
func (vad *VoiceActivityDetector) CurrentRMS() float32 {
	return vad.currentRMS
}
