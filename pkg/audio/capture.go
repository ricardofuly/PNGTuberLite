package audio

/*
#cgo linux LDFLAGS: -Wl,--allow-multiple-definition
#cgo darwin LDFLAGS: -Wl,-multiply_defined,suppress
#cgo windows LDFLAGS: -Wl,--allow-multiple-definition
*/
import "C"

import (
	"fmt"
	"math"
	"strings"
	"sync/atomic"
	"unsafe"

	"github.com/gen2brain/malgo"
)

// CaptureEngine manages the microphone recording and speech state detection.
type CaptureEngine struct {
	ctx           *malgo.AllocatedContext
	device        *malgo.Device
	vad           *VoiceActivityDetector
	isTalking     atomic.Bool
	currentVolume atomic.Uint32 // stores math.Float32bits
	isRunning     atomic.Bool
}

// NewCaptureEngine creates a new audio capture engine.
func NewCaptureEngine(threshold float32) *CaptureEngine {
	return &CaptureEngine{
		vad: NewVoiceActivityDetector(threshold),
	}
}

// Start begins audio recording on the specified device (pass empty string for system default).
func (ce *CaptureEngine) Start(deviceName string) error {
	if ce.isRunning.Load() {
		ce.Stop()
	}

	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(message string) {
		// Miniaudio log message handler
	})
	if err != nil {
		return fmt.Errorf("failed to initialize audio context: %w", err)
	}
	ce.ctx = ctx

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatF32
	deviceConfig.Capture.Channels = 1
	deviceConfig.SampleRate = 44100
	deviceConfig.PeriodSizeInMilliseconds = 25 // 25ms low latency buffers

	// If a specific device is selected
	if deviceName != "" {
		infos, err := ctx.Devices(malgo.Capture)
		if err == nil {
			for _, info := range infos {
				if info.Name() == deviceName {
					deviceConfig.Capture.DeviceID = info.ID.Pointer()
					break
				}
			}
		}
	}

	onRecvSamples := func(pOutput, pInput []byte, frameCount uint32) {
		if frameCount == 0 || len(pInput) == 0 {
			return
		}

		// Convert raw byte buffer to []float32
		sampleCount := int(frameCount)
		if len(pInput) < sampleCount*4 {
			sampleCount = len(pInput) / 4
		}

		samples := unsafe.Slice((*float32)(unsafe.Pointer(&pInput[0])), sampleCount)

		// Process VAD
		speaking, rms := ce.vad.Process(samples)
		ce.isTalking.Store(speaking)
		ce.currentVolume.Store(math.Float32bits(rms))
	}

	deviceCallbacks := malgo.DeviceCallbacks{
		Data: onRecvSamples,
	}

	device, err := malgo.InitDevice(ctx.Context, deviceConfig, deviceCallbacks)
	if err != nil {
		ctx.Uninit()
		ctx.Free()
		return fmt.Errorf("failed to initialize audio capture device: %w", err)
	}

	if err := device.Start(); err != nil {
		device.Uninit()
		ctx.Uninit()
		ctx.Free()
		return fmt.Errorf("failed to start audio device stream: %w", err)
	}

	ce.device = device
	ce.isRunning.Store(true)
	return nil
}

// Stop stops the audio capture and frees system audio resources.
func (ce *CaptureEngine) Stop() {
	if !ce.isRunning.Swap(false) {
		return
	}

	if ce.device != nil {
		ce.device.Stop()
		ce.device.Uninit()
		ce.device = nil
	}
	if ce.ctx != nil {
		ce.ctx.Uninit()
		ce.ctx.Free()
		ce.ctx = nil
	}

	ce.isTalking.Store(false)
	ce.currentVolume.Store(0)
}

// IsTalking returns whether speech is currently detected (thread-safe).
func (ce *CaptureEngine) IsTalking() bool {
	return ce.isTalking.Load()
}

// GetVolume returns the current RMS volume level (thread-safe).
func (ce *CaptureEngine) GetVolume() float32 {
	bits := ce.currentVolume.Load()
	return math.Float32frombits(bits)
}

// SetThreshold dynamically updates the activation threshold.
func (ce *CaptureEngine) SetThreshold(threshold float32) {
	if threshold > 0 {
		ce.vad.Threshold = threshold
	}
}

// ListDevices returns the names of all available physical and virtual audio capture microphones,
// completely filtering out internal loopback/monitor devices.
func (ce *CaptureEngine) ListDevices() ([]string, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		ctx.Uninit()
		ctx.Free()
	}()

	infos, err := ctx.Devices(malgo.Capture)
	if err != nil {
		return nil, err
	}

	var devices []string
	for _, info := range infos {
		name := info.Name()
		if name == "" {
			continue
		}
		lower := strings.ToLower(name)
		// Filter out any monitor / loopback output sinks
		if strings.Contains(lower, "monitor") {
			continue
		}
		devices = append(devices, name)
	}

	return devices, nil
}
