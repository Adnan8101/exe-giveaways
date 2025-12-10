package bot

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

// PerformanceMonitor tracks critical performance metrics
type PerformanceMonitor struct {
	// Command execution metrics
	commandCount   atomic.Uint64
	commandLatency atomic.Int64 // nanoseconds

	// Event processing metrics
	eventCount   atomic.Uint64
	eventLatency atomic.Int64 // nanoseconds

	// REST API metrics
	restCallCount atomic.Uint64
	restLatency   atomic.Int64 // nanoseconds

	// WebSocket metrics
	wsLatency atomic.Int64 // milliseconds

	// AntiNuke detection metrics
	detectionCount   atomic.Uint64
	detectionLatency atomic.Int64 // nanoseconds

	startTime time.Time
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		startTime: time.Now(),
	}
}

// TrackCommand records command execution time
func (pm *PerformanceMonitor) TrackCommand(duration time.Duration) {
	pm.commandCount.Add(1)
	pm.commandLatency.Store(duration.Nanoseconds())
}

// TrackEvent records event processing time
func (pm *PerformanceMonitor) TrackEvent(duration time.Duration) {
	pm.eventCount.Add(1)
	pm.eventLatency.Store(duration.Nanoseconds())
}

// TrackREST records REST API call time
func (pm *PerformanceMonitor) TrackREST(duration time.Duration) {
	pm.restCallCount.Add(1)
	pm.restLatency.Store(duration.Nanoseconds())
}

// TrackDetection records AntiNuke detection time
func (pm *PerformanceMonitor) TrackDetection(duration time.Duration) {
	pm.detectionCount.Add(1)
	pm.detectionLatency.Store(duration.Nanoseconds())
}

// UpdateWSLatency updates WebSocket latency
func (pm *PerformanceMonitor) UpdateWSLatency(latency time.Duration) {
	pm.wsLatency.Store(latency.Milliseconds())
}

// GetStats returns current performance statistics
func (pm *PerformanceMonitor) GetStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"uptime_seconds":      time.Since(pm.startTime).Seconds(),
		"command_count":       pm.commandCount.Load(),
		"command_latency_ns":  pm.commandLatency.Load(),
		"event_count":         pm.eventCount.Load(),
		"event_latency_ns":    pm.eventLatency.Load(),
		"rest_call_count":     pm.restCallCount.Load(),
		"rest_latency_ns":     pm.restLatency.Load(),
		"ws_latency_ms":       pm.wsLatency.Load(),
		"detection_count":     pm.detectionCount.Load(),
		"detection_latency_ns": pm.detectionLatency.Load(),
		"goroutines":          runtime.NumGoroutine(),
		"memory_alloc_mb":     m.Alloc / 1024 / 1024,
		"memory_sys_mb":       m.Sys / 1024 / 1024,
		"gc_count":            m.NumGC,
		"cpu_cores":           runtime.NumCPU(),
	}
}

// PrintDashboard prints a performance dashboard
func (pm *PerformanceMonitor) PrintDashboard() {
	stats := pm.GetStats()

	fmt.Println("\n╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║          🚀 PERFORMANCE DASHBOARD                          ║")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ Uptime: %.0f seconds                                    \n", stats["uptime_seconds"])
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Println("║ 📊 LATENCY METRICS (Target vs Actual)                     ║")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")

	wsLatency := stats["ws_latency_ms"].(int64)
	wsStatus := "✅"
	if wsLatency > 20 {
		wsStatus = "❌"
	} else if wsLatency > 10 {
		wsStatus = "⚠️"
	}
	fmt.Printf("║ WebSocket:       %3dms (Target: <20ms)   %s          \n", wsLatency, wsStatus)

	restLatency := stats["rest_latency_ns"].(int64)
	restLatencyMs := float64(restLatency) / 1_000_000.0
	restStatus := "✅"
	if restLatencyMs > 150 {
		restStatus = "❌"
	} else if restLatencyMs > 100 {
		restStatus = "⚠️"
	}
	fmt.Printf("║ REST API:      %.2fms (Target: <100ms)  %s          \n", restLatencyMs, restStatus)

	// AntiNuke Detection latency
	if stats["detection_count"].(uint64) > 0 {
		detectionLatency := stats["detection_latency_ns"].(int64)
		detectionMicros := float64(detectionLatency) / 1000.0
		detectionStatus := "✅"
		if detectionMicros > 10 {
			detectionStatus = "❌"
		} else if detectionMicros > 3 {
			detectionStatus = "⚠️"
		}
		fmt.Printf("║ Detection:     %.2fµs (Target: <3µs)    %s          \n", detectionMicros, detectionStatus)
	}

	cmdLatency := stats["command_latency_ns"].(int64)
	cmdLatencyMs := float64(cmdLatency) / 1_000_000.0
	cmdStatus := "✅"
	if cmdLatencyMs > 5 {
		cmdStatus = "❌"
	} else if cmdLatencyMs > 2 {
		cmdStatus = "⚠️"
	}
	fmt.Printf("║ Command Exec:  %.2fms (Target: <5ms)    %s          \n", cmdLatencyMs, cmdStatus)

	eventLatency := stats["event_latency_ns"].(int64)
	eventLatencyMs := float64(eventLatency) / 1_000_000.0
	eventStatus := "✅"
	if eventLatencyMs > 1 {
		eventStatus = "⚠️"
	}
	fmt.Printf("║ Event Process: %.2fms (Target: <1ms)    %s          \n", eventLatencyMs, eventStatus)

	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Println("║ 📈 THROUGHPUT                                              ║")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ Commands:      %10d                                 \n", stats["command_count"])
	fmt.Printf("║ Events:        %10d                                 \n", stats["event_count"])
	fmt.Printf("║ Detections:    %10d                                 \n", stats["detection_count"])
	fmt.Printf("║ REST Calls:    %10d                                 \n", stats["rest_call_count"])
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Println("║ 💾 SYSTEM RESOURCES                                        ║")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ Memory Alloc:  %5d MB                                  \n", stats["memory_alloc_mb"])
	fmt.Printf("║ Memory Sys:    %5d MB                                  \n", stats["memory_sys_mb"])
	fmt.Printf("║ Goroutines:    %5d                                     \n", stats["goroutines"])
	fmt.Printf("║ GC Count:      %5d                                     \n", stats["gc_count"])
	fmt.Printf("║ CPU Cores:     %5d                                     \n", stats["cpu_cores"])
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// PerfTransport wraps http.RoundTripper to track REST latency
type PerfTransport struct {
	Base    http.RoundTripper
	Monitor *PerformanceMonitor
}

func (t *PerfTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := t.Base.RoundTrip(req)
	t.Monitor.TrackREST(time.Since(start))
	return resp, err
}

// StartMonitoring starts periodic performance monitoring
func (b *Bot) StartMonitoring(interval time.Duration) {
	if b.PerfMonitor == nil {
		b.PerfMonitor = NewPerformanceMonitor()
	}

	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			// Update WebSocket latency
			if b.Session != nil {
				b.PerfMonitor.UpdateWSLatency(b.Session.HeartbeatLatency())
			}

			// Print dashboard
			b.PerfMonitor.PrintDashboard()

			// Log warnings
			stats := b.PerfMonitor.GetStats()
			if wsLatency := stats["ws_latency_ms"].(int64); wsLatency > 50 {
				log.Printf("⚠️  CRITICAL: WebSocket latency is %dms - check network routing!", wsLatency)
			}
			if restLatency := stats["rest_latency_ns"].(int64); restLatency > 200_000_000 {
				log.Printf("⚠️  WARNING: REST API latency is %.2fms - check HTTP client configuration!", float64(restLatency)/1_000_000.0)
			}
			if detectionCount := stats["detection_count"].(uint64); detectionCount > 0 {
				detectionLatency := stats["detection_latency_ns"].(int64)
				detectionMicros := float64(detectionLatency) / 1000.0
				if detectionMicros > 5.0 {
					log.Printf("⚠️  WARNING: AntiNuke detection is %.2fµs - target is <3µs!", detectionMicros)
				}
			}
			if mem := stats["memory_alloc_mb"].(uint64); mem > 2500 {
				log.Printf("⚠️  WARNING: Memory usage is %d MB - approaching 3GB limit!", mem)
			}
		}
	}()

	log.Printf("📊 Performance monitoring started (interval: %v)", interval)
}
