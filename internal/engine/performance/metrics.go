package performance

import (
	"discord-giveaway-bot/internal/engine/fdl"
	"fmt"
	"log"
	"runtime"
	"sync/atomic"
	"time"
)

// PerformanceMetrics tracks system-wide performance statistics
type PerformanceMetrics struct {
	// Detection timing
	DetectionCount     uint64
	TotalDetectionTime int64 // nanoseconds
	MaxDetectionTime   int64 // nanoseconds
	MinDetectionTime   int64 // nanoseconds

	// Execution timing
	ExecutionCount     uint64
	TotalExecutionTime int64
	MaxExecutionTime   int64
	MinExecutionTime   int64

	// Throughput metrics
	EventsPerSecond      uint64
	PunishmentsPerSecond uint64

	// System metrics
	GoroutineCount  int64 // Changed to int64 for atomic operations
	HeapAllocMB     uint64
	CPUUsagePercent float64

	// Ring buffer metrics
	RingBufferSize     uint64
	RingBufferCapacity uint64

	// Queue metrics
	PunishQueueSize      uint64
	AttributionQueueSize uint64

	// Start time for uptime calculation
	StartTime time.Time
}

// Global metrics instance
var globalMetrics = &PerformanceMetrics{
	StartTime:        time.Now(),
	MinDetectionTime: 1<<63 - 1, // Max int64
	MinExecutionTime: 1<<63 - 1,
}

// RecordDetection records a detection event with timing
func RecordDetection(detectionTimeNs int64) {
	atomic.AddUint64(&globalMetrics.DetectionCount, 1)
	atomic.AddInt64(&globalMetrics.TotalDetectionTime, detectionTimeNs)

	// Update min (lockless)
	for {
		oldMin := atomic.LoadInt64(&globalMetrics.MinDetectionTime)
		if detectionTimeNs >= oldMin || atomic.CompareAndSwapInt64(&globalMetrics.MinDetectionTime, oldMin, detectionTimeNs) {
			break
		}
	}

	// Update max (lockless)
	for {
		oldMax := atomic.LoadInt64(&globalMetrics.MaxDetectionTime)
		if detectionTimeNs <= oldMax || atomic.CompareAndSwapInt64(&globalMetrics.MaxDetectionTime, oldMax, detectionTimeNs) {
			break
		}
	}
}

// RecordExecution records a punishment execution with timing
func RecordExecution(executionTimeNs int64) {
	atomic.AddUint64(&globalMetrics.ExecutionCount, 1)
	atomic.AddInt64(&globalMetrics.TotalExecutionTime, executionTimeNs)

	// Update min
	for {
		oldMin := atomic.LoadInt64(&globalMetrics.MinExecutionTime)
		if executionTimeNs >= oldMin || atomic.CompareAndSwapInt64(&globalMetrics.MinExecutionTime, oldMin, executionTimeNs) {
			break
		}
	}

	// Update max
	for {
		oldMax := atomic.LoadInt64(&globalMetrics.MaxExecutionTime)
		if executionTimeNs <= oldMax || atomic.CompareAndSwapInt64(&globalMetrics.MaxExecutionTime, oldMax, executionTimeNs) {
			break
		}
	}
}

// UpdateSystemMetrics updates runtime system metrics
func UpdateSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	atomic.StoreInt64(&globalMetrics.GoroutineCount, int64(runtime.NumGoroutine()))
	atomic.StoreUint64(&globalMetrics.HeapAllocMB, m.Alloc/1024/1024)
}

// GetMetrics returns a snapshot of current metrics
func GetMetrics() PerformanceMetrics {
	detectionCount := atomic.LoadUint64(&globalMetrics.DetectionCount)
	totalDetection := atomic.LoadInt64(&globalMetrics.TotalDetectionTime)

	executionCount := atomic.LoadUint64(&globalMetrics.ExecutionCount)
	totalExecution := atomic.LoadInt64(&globalMetrics.TotalExecutionTime)

	return PerformanceMetrics{
		DetectionCount:     detectionCount,
		TotalDetectionTime: totalDetection,
		MaxDetectionTime:   atomic.LoadInt64(&globalMetrics.MaxDetectionTime),
		MinDetectionTime:   atomic.LoadInt64(&globalMetrics.MinDetectionTime),

		ExecutionCount:     executionCount,
		TotalExecutionTime: totalExecution,
		MaxExecutionTime:   atomic.LoadInt64(&globalMetrics.MaxExecutionTime),
		MinExecutionTime:   atomic.LoadInt64(&globalMetrics.MinExecutionTime),

		GoroutineCount: atomic.LoadInt64(&globalMetrics.GoroutineCount),
		HeapAllocMB:    atomic.LoadUint64(&globalMetrics.HeapAllocMB),

		StartTime: globalMetrics.StartTime,
	}
}

// PrintMetrics outputs formatted performance metrics
func PrintMetrics() {
	UpdateSystemMetrics()

	metrics := GetMetrics()
	uptime := time.Since(metrics.StartTime)

	log.Println("╔════════════════════════════════════════════════════════════════════╗")
	log.Println("║              🚀 ULTRA-OPTIMIZED ANTI-NUKE METRICS 🚀               ║")
	log.Println("╠════════════════════════════════════════════════════════════════════╣")

	// Detection Performance
	if metrics.DetectionCount > 0 {
		avgDetection := time.Duration(metrics.TotalDetectionTime / int64(metrics.DetectionCount))
		minDetection := time.Duration(metrics.MinDetectionTime)
		maxDetection := time.Duration(metrics.MaxDetectionTime)

		log.Printf("║ ⚡ DETECTION SPEED                                                 ║")
		log.Printf("║   • Average: %-51s ║", avgDetection)
		log.Printf("║   • Minimum: %-51s ║", minDetection)
		log.Printf("║   • Maximum: %-51s ║", maxDetection)
		log.Printf("║   • Total Events: %-44d ║", metrics.DetectionCount)
	}

	// Execution Performance
	if metrics.ExecutionCount > 0 {
		log.Println("║                                                                    ║")
		log.Printf("║ 🎯 PUNISHMENT EXECUTION                                            ║")
		avgExecution := time.Duration(metrics.TotalExecutionTime / int64(metrics.ExecutionCount))
		minExecution := time.Duration(metrics.MinExecutionTime)
		maxExecution := time.Duration(metrics.MaxExecutionTime)

		log.Printf("║   • Average: %-51s ║", avgExecution)
		log.Printf("║   • Minimum: %-51s ║", minExecution)
		log.Printf("║   • Maximum: %-51s ║", maxExecution)
		log.Printf("║   • Total Punishments: %-39d ║", metrics.ExecutionCount)
	}

	// System Performance
	log.Println("║                                                                    ║")
	log.Printf("║ 💻 SYSTEM RESOURCES                                                ║")
	log.Printf("║   • Goroutines: %-47d ║", metrics.GoroutineCount)
	log.Printf("║   • Heap Memory: %-43d MB ║", metrics.HeapAllocMB)
	log.Printf("║   • CPU Cores: %-47d ║", runtime.NumCPU())
	log.Printf("║   • Uptime: %-51s ║", uptime.Round(time.Second))

	// Event counters
	totalEvents := fdl.TotalEvents.GetTotal()
	eventsProcessed := fdl.EventsProcessed.GetTotal()
	eventsDropped := fdl.EventsDropped.GetTotal()
	eventsDetected := fdl.EventsDetected.GetTotal()
	punishmentsIssued := fdl.PunishmentsIssued.GetTotal()

	log.Println("║                                                                    ║")
	log.Printf("║ 📊 EVENT STATISTICS                                                ║")
	log.Printf("║   • Total Events: %-44d ║", totalEvents)
	log.Printf("║   • Processed: %-47d ║", eventsProcessed)
	log.Printf("║   • Detected: %-48d ║", eventsDetected)
	log.Printf("║   • Dropped: %-49d ║", eventsDropped)
	log.Printf("║   • Punishments: %-45d ║", punishmentsIssued)

	// Throughput
	if uptime.Seconds() > 0 {
		eventsPerSec := float64(totalEvents) / uptime.Seconds()
		punishmentsPerSec := float64(punishmentsIssued) / uptime.Seconds()

		log.Println("║                                                                    ║")
		log.Printf("║ 🔥 THROUGHPUT                                                      ║")
		log.Printf("║   • Events/sec: %-46.2f ║", eventsPerSec)
		log.Printf("║   • Punishments/sec: %-41.2f ║", punishmentsPerSec)
	}

	log.Println("╚════════════════════════════════════════════════════════════════════╝")
}

// StartPeriodicMetrics starts periodic metrics reporting
func StartPeriodicMetrics(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			PrintMetrics()
		}
	}()
}

// GetPerformanceScore calculates overall performance score (0-100)
func GetPerformanceScore() float64 {
	metrics := GetMetrics()
	score := 100.0

	// Penalize high detection times
	if metrics.DetectionCount > 0 {
		avgDetectionUs := float64(metrics.TotalDetectionTime/int64(metrics.DetectionCount)) / 1000.0
		if avgDetectionUs > 10 {
			score -= (avgDetectionUs - 10) / 10.0
		}
	}

	// Penalize high execution times
	if metrics.ExecutionCount > 0 {
		avgExecutionMs := float64(metrics.TotalExecutionTime/int64(metrics.ExecutionCount)) / 1000000.0
		if avgExecutionMs > 100 {
			score -= (avgExecutionMs - 100) / 100.0
		}
	}

	// Cap score between 0-100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// FormatPerformanceReport generates a detailed performance report
func FormatPerformanceReport() string {
	metrics := GetMetrics()
	score := GetPerformanceScore()

	report := fmt.Sprintf(`
╔═══════════════════════════════════════════════════════════════════════╗
║                    ULTRA-PERFORMANCE REPORT                           ║
╠═══════════════════════════════════════════════════════════════════════╣
║ Performance Score: %.1f/100                                            ║
║                                                                       ║
║ Detection Speed:                                                      ║
║   Min: %v | Avg: %v | Max: %v                                        ║
║                                                                       ║
║ Execution Speed:                                                      ║
║   Min: %v | Avg: %v | Max: %v                                        ║
║                                                                       ║
║ System Health:                                                        ║
║   Goroutines: %d | Memory: %dMB | Uptime: %v                         ║
╚═══════════════════════════════════════════════════════════════════════╝
`, score,
		time.Duration(metrics.MinDetectionTime),
		time.Duration(metrics.TotalDetectionTime/int64(max(metrics.DetectionCount, 1))),
		time.Duration(metrics.MaxDetectionTime),
		time.Duration(metrics.MinExecutionTime),
		time.Duration(metrics.TotalExecutionTime/int64(max(metrics.ExecutionCount, 1))),
		time.Duration(metrics.MaxExecutionTime),
		metrics.GoroutineCount,
		metrics.HeapAllocMB,
		time.Since(metrics.StartTime).Round(time.Second))

	return report
}

func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
