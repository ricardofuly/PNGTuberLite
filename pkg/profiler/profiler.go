package profiler

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const MaxHistoryPoints = 50

// ProfilerStats stores current measured system resource metrics and time-series history.
type ProfilerStats struct {
	CPUPercent       float32   // Process CPU % (matching top / btop / htop, where 100% = 1 full core)
	CPUTotalPercent  float32   // Normalized CPU % across all machine cores (0 to 100%)
	SmoothCPU        float32   // Smoothly interpolated CPU % for fluid UI animation
	RamAllocMB       float32   // Go Heap Alloc memory in MB
	RamSysMB         float32   // Go System reserved memory in MB
	RamRSSMB         float32   // Real OS Physical Resident Set Size (RSS) in MB (matching top / btop RES)
	SmoothRAM        float32   // Smoothly interpolated RAM RSS
	NumGoroutine     int       // Active Goroutines
	NumGC            uint32    // Garbage collector cycle count
	FrameTimeMS      float32   // Frame time in milliseconds
	SmoothFrameTime  float32   // Smoothly interpolated frametime
	FPS              int32     // Frames per second
	VRAMMB           float32   // Estimated GPU VRAM used by loaded textures in MB
	TextureCount     int       // Number of GPU textures loaded
	CPUHistory       []float32 // Historical CPU % samples for sparkline graph
	FrametimeHistory []float32 // Historical frametime (ms) samples for sparkline graph
	RAMHistory       []float32 // Historical RAM RSS (MB) samples for sparkline graph
	SampleProgress   float32   // Normalized 0.0..1.0 time progress between samples for subpixel graph scroll
}

// SystemProfiler tracks process CPU, Memory and Rendering metrics.
type SystemProfiler struct {
	lastSampleTime   time.Time
	lastProcessTicks int64
	cachedStats      ProfilerStats
	numCPU           float64
	cpuHistory       []float32
	frameHistory     []float32
	ramHistory       []float32
	frameAccumTime   float32
}

// NewSystemProfiler creates and initializes a lightweight resource profiler.
func NewSystemProfiler() *SystemProfiler {
	p := &SystemProfiler{
		lastSampleTime: time.Now(),
		numCPU:         float64(runtime.NumCPU()),
		cpuHistory:     make([]float32, 0, MaxHistoryPoints),
		frameHistory:   make([]float32, 0, MaxHistoryPoints),
		ramHistory:     make([]float32, 0, MaxHistoryPoints),
	}
	if p.numCPU < 1 {
		p.numCPU = 1
	}
	p.lastProcessTicks = getProcessCPUTicks()
	p.Update(0, 0, 0, 0)
	return p
}

// Update samples resource usage. Sampling of CPU & OS RAM is throttled to every 200ms for minimal overhead.
func (p *SystemProfiler) Update(dt float32, fps int32, textureCount int, vramBytes int64) ProfilerStats {
	now := time.Now()
	elapsed := now.Sub(p.lastSampleTime)

	const sampleInterval = 200 * time.Millisecond

	// Subpixel scroll fraction (0.0 to 1.0)
	p.cachedStats.SampleProgress = float32(elapsed.Seconds()) / float32(sampleInterval.Seconds())
	if p.cachedStats.SampleProgress > 1.0 {
		p.cachedStats.SampleProgress = 1.0
	}

	if elapsed >= sampleInterval {
		// 1. Calculate CPU % (Matching Linux top / btop / htop standard)
		currentTicks := getProcessCPUTicks()
		if p.lastProcessTicks > 0 && elapsed.Seconds() > 0 {
			diffTicks := float64(currentTicks - p.lastProcessTicks)
			// Linux USER_HZ is 100 clock ticks per second
			cpuSeconds := diffTicks / 100.0

			// Standard Process CPU % (e.g. 8.0% in top)
			processCPU := (cpuSeconds / elapsed.Seconds()) * 100.0
			if processCPU < 0 {
				processCPU = 0
			}

			// Normalized Whole-System CPU %
			totalCPU := processCPU / p.numCPU
			if totalCPU > 100.0 {
				totalCPU = 100.0
			}

			p.cachedStats.CPUPercent = float32(processCPU)
			p.cachedStats.CPUTotalPercent = float32(totalCPU)

			// Push to CPU History
			if len(p.cpuHistory) < MaxHistoryPoints {
				p.cpuHistory = append(p.cpuHistory, p.cachedStats.CPUPercent)
			} else {
				p.cpuHistory = append(p.cpuHistory[1:], p.cachedStats.CPUPercent)
			}
		}
		p.lastProcessTicks = currentTicks
		p.lastSampleTime = now

		// 2. Go Runtime Memory
		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		p.cachedStats.RamAllocMB = float32(mem.Alloc) / (1024 * 1024)
		p.cachedStats.RamSysMB = float32(mem.Sys) / (1024 * 1024)
		p.cachedStats.NumGC = mem.NumGC
		p.cachedStats.NumGoroutine = runtime.NumGoroutine()

		// 3. Linux Process Physical RSS (Resident Set Size matching top RES)
		rssMB := getProcessRSSMB()
		if rssMB > 0 {
			p.cachedStats.RamRSSMB = rssMB
		} else {
			p.cachedStats.RamRSSMB = p.cachedStats.RamAllocMB
		}

		// Push to RAM History
		if len(p.ramHistory) < MaxHistoryPoints {
			p.ramHistory = append(p.ramHistory, p.cachedStats.RamRSSMB)
		} else {
			p.ramHistory = append(p.ramHistory[1:], p.cachedStats.RamRSSMB)
		}
	}

	// 4. GPU & Frame stats (updated each frame)
	frameMS := dt * 1000.0
	if frameMS <= 0 && fps > 0 {
		frameMS = 1000.0 / float32(fps)
	}
	p.cachedStats.FrameTimeMS = frameMS
	p.cachedStats.FPS = fps
	p.cachedStats.TextureCount = textureCount
	p.cachedStats.VRAMMB = float32(vramBytes) / (1024 * 1024)

	// Smooth continuous frame sampling (~30 samples/sec for silky smooth stream)
	p.frameAccumTime += dt
	if p.frameAccumTime >= 0.033 {
		p.frameAccumTime = 0
		if len(p.frameHistory) < MaxHistoryPoints {
			p.frameHistory = append(p.frameHistory, frameMS)
		} else {
			p.frameHistory = append(p.frameHistory[1:], frameMS)
		}
	}

	// 5. Smooth exponential lerp for display values (60fps fluid UI animation)
	lerpRate := float32(1.0 - math.Exp(float64(-12.0*dt)))
	if lerpRate > 1.0 {
		lerpRate = 1.0
	}
	p.cachedStats.SmoothCPU += (p.cachedStats.CPUPercent - p.cachedStats.SmoothCPU) * lerpRate
	p.cachedStats.SmoothRAM += (p.cachedStats.RamRSSMB - p.cachedStats.SmoothRAM) * lerpRate
	p.cachedStats.SmoothFrameTime += (p.cachedStats.FrameTimeMS - p.cachedStats.SmoothFrameTime) * lerpRate

	// Attach history slices
	p.cachedStats.CPUHistory = p.cpuHistory
	p.cachedStats.FrametimeHistory = p.frameHistory
	p.cachedStats.RAMHistory = p.ramHistory

	return p.cachedStats
}

// getProcessCPUTicks reads utime + stime from /proc/self/stat on Linux.
func getProcessCPUTicks() int64 {
	data, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return 0
	}
	s := string(data)
	idx := strings.LastIndex(s, ")")
	if idx == -1 || idx+1 >= len(s) {
		return 0
	}

	fields := strings.Fields(s[idx+1:])
	// Fields after ')':
	// 0: state (field 3 of /proc/self/stat)
	// ...
	// 11: utime (field 14 of /proc/self/stat)
	// 12: stime (field 15 of /proc/self/stat)
	if len(fields) < 13 {
		return 0
	}

	utime, _ := strconv.ParseInt(fields[11], 10, 64)
	stime, _ := strconv.ParseInt(fields[12], 10, 64)
	return utime + stime
}

// getProcessRSSMB reads real physical memory pages from /proc/self/statm on Linux.
func getProcessRSSMB() float32 {
	data, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0
	}
	residentPages, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	// Standard Linux page size is 4096 bytes (4 KB)
	bytesUsed := residentPages * 4096
	return float32(bytesUsed) / (1024 * 1024)
}

// FormatStats returns a human-readable summary string of current stats.
func (p *SystemProfiler) FormatStats() string {
	s := p.cachedStats
	return fmt.Sprintf("CPU: %.1f%% | RAM: %.1f MB (Heap: %.1f MB) | GPU: %.1f MB (%d tex) | Frame: %.1fms (%d FPS)",
		s.CPUPercent, s.RamRSSMB, s.RamAllocMB, s.VRAMMB, s.TextureCount, s.FrameTimeMS, s.FPS)
}
