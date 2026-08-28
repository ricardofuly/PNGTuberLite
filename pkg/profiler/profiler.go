package profiler

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ProfilerStats stores current measured system resource metrics.
type ProfilerStats struct {
	CPUPercent   float32 // Process CPU % (0 to 100%)
	RamAllocMB   float32 // Go Heap Alloc memory in MB
	RamSysMB     float32 // Go System reserved memory in MB
	RamRSSMB     float32 // Real OS Physical Resident Set Size (RSS) in MB
	NumGoroutine int     // Active Goroutines
	NumGC        uint32  // Garbage collector cycle count
	FrameTimeMS  float32 // Frame time in milliseconds
	FPS          int32   // Frames per second
	VRAMMB       float32 // Estimated GPU VRAM used by loaded textures in MB
	TextureCount int     // Number of GPU textures loaded
}

// SystemProfiler tracks process CPU, Memory and Rendering metrics.
type SystemProfiler struct {
	lastSampleTime   time.Time
	lastProcessTicks int64
	cachedStats      ProfilerStats
	numCPU           float64
}

// NewSystemProfiler creates and initializes a lightweight resource profiler.
func NewSystemProfiler() *SystemProfiler {
	p := &SystemProfiler{
		lastSampleTime: time.Now(),
		numCPU:         float64(runtime.NumCPU()),
	}
	if p.numCPU < 1 {
		p.numCPU = 1
	}
	p.lastProcessTicks = getProcessCPUTicks()
	p.Update(0, 0, 0, 0)
	return p
}

// Update samples resource usage. Sampling of CPU & OS RAM is throttled to every 300ms for minimal overhead.
func (p *SystemProfiler) Update(frameTimeSec float32, fps int32, textureCount int, vramBytes int64) ProfilerStats {
	now := time.Now()
	elapsed := now.Sub(p.lastSampleTime)

	if elapsed >= 300*time.Millisecond {
		// 1. Calculate CPU %
		currentTicks := getProcessCPUTicks()
		if p.lastProcessTicks > 0 && elapsed.Seconds() > 0 {
			diffTicks := float64(currentTicks - p.lastProcessTicks)
			// Linux USER_HZ is typically 100 clock ticks per second
			cpuSeconds := diffTicks / 100.0
			cpuPct := (cpuSeconds / elapsed.Seconds()) / p.numCPU * 100.0
			if cpuPct < 0 {
				cpuPct = 0
			}
			if cpuPct > 100.0 {
				cpuPct = 100.0
			}
			p.cachedStats.CPUPercent = float32(cpuPct)
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

		// 3. Linux Process Physical RSS (Resident Set Size)
		rssMB := getProcessRSSMB()
		if rssMB > 0 {
			p.cachedStats.RamRSSMB = rssMB
		} else {
			p.cachedStats.RamRSSMB = p.cachedStats.RamAllocMB
		}
	}

	// 4. GPU & Frame stats (updated each frame)
	p.cachedStats.FrameTimeMS = frameTimeSec * 1000.0
	p.cachedStats.FPS = fps
	p.cachedStats.TextureCount = textureCount
	p.cachedStats.VRAMMB = float32(vramBytes) / (1024 * 1024)

	return p.cachedStats
}

// getProcessCPUTicks reads utime + stime from /proc/self/stat on Linux.
func getProcessCPUTicks() int64 {
	data, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	// Field 13 (utime) and 14 (stime) in 0-indexed slice
	if len(fields) < 15 {
		return 0
	}
	utime, _ := strconv.ParseInt(fields[13], 10, 64)
	stime, _ := strconv.ParseInt(fields[14], 10, 64)
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
	// Page size on Linux x86_64 / arm64 is 4096 bytes (4 KB)
	bytesUsed := residentPages * 4096
	return float32(bytesUsed) / (1024 * 1024)
}

// FormatStats returns a human-readable summary string of current stats.
func (p *SystemProfiler) FormatStats() string {
	s := p.cachedStats
	return fmt.Sprintf("CPU: %.1f%% | RAM: %.1f MB (Heap: %.1f MB) | GPU: %.1f MB (%d tex) | Frame: %.1fms (%d FPS)",
		s.CPUPercent, s.RamRSSMB, s.RamAllocMB, s.VRAMMB, s.TextureCount, s.FrameTimeMS, s.FPS)
}
