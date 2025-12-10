package ring

import (
	"runtime"
	"sync/atomic"
	"time"

	"discord-giveaway-bot/internal/engine/fdl"
)

// Consumer is a worker that processes events from the ring
// Optimized for maximum CPU utilization and minimal latency
type Consumer struct {
	Ring      *RingBuffer
	Handler   func(fdl.FastEvent)
	ID        int
	BatchSize int // Process events in batches for efficiency
	SpinCount int // How many times to spin before yielding
	running   uint32
	stopChan  chan struct{}
}

// NewConsumer creates an optimized consumer with default settings
func NewConsumer(ring *RingBuffer, handler func(fdl.FastEvent), id int) *Consumer {
	return &Consumer{
		Ring:      ring,
		Handler:   handler,
		ID:        id,
		BatchSize: 128,    // Process up to 128 events per batch
		SpinCount: 100000, // Aggressive spinning for ultra-low latency
		stopChan:  make(chan struct{}),
	}
}

// Start begins the consumer loop with CPU pinning and ultra-low latency optimizations
// This should be pinned to a core in main
func (c *Consumer) Start() {
	// Pin this goroutine to an OS thread for maximum performance
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	atomic.StoreUint32(&c.running, 1)

	// Ultra-aggressive spin-wait loop for sub-microsecond latency
	spinCounter := 0
	emptyLoops := 0

	for atomic.LoadUint32(&c.running) == 1 {
		evt, ok := c.Ring.Pop()

		if !ok {
			// Empty queue - use adaptive spinning strategy
			spinCounter++

			if spinCounter < c.SpinCount {
				// Phase 1: Aggressive spin (0-100k iterations)
				// Busy-wait for immediate response to incoming events
				runtime.Gosched() // Hint to scheduler but keep spinning
				continue
			} else if spinCounter < c.SpinCount*2 {
				// Phase 2: Moderate spin with CPU yield
				for i := 0; i < 10; i++ {
					evt, ok = c.Ring.Pop()
					if ok {
						goto process
					}
				}
				runtime.Gosched()
				continue
			} else {
				// Phase 3: Brief sleep to reduce CPU usage when idle
				emptyLoops++
				if emptyLoops > 100 {
					time.Sleep(50 * time.Nanosecond) // Minimal sleep
					emptyLoops = 0
				}
				spinCounter = 0
				continue
			}
		}

	process:
		// Reset spin counter on successful pop
		spinCounter = 0
		emptyLoops = 0

		// NO LOGGING IN HOT PATH - Direct handler call
		// Handler is inlined by compiler for zero-overhead function call
		c.Handler(evt)

		// Try to process more events in batch for better throughput
		// This amortizes the loop overhead across multiple events
		for i := 1; i < c.BatchSize; i++ {
			evt, ok = c.Ring.Pop()
			if !ok {
				break
			}
			c.Handler(evt)
		}
	}
}

// StartBatch begins the consumer loop with batch processing for maximum throughput
func (c *Consumer) StartBatch() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	atomic.StoreUint32(&c.running, 1)

	for atomic.LoadUint32(&c.running) == 1 {
		// Use batch pop for better cache efficiency
		batch, count := c.Ring.PopBatch(c.BatchSize)

		if count == 0 {
			// Empty - adaptive backoff
			time.Sleep(10 * time.Nanosecond)
			continue
		}

		// Process entire batch with minimal overhead
		for i := 0; i < count; i++ {
			c.Handler(batch[i])
		}
	}
}

// Stop gracefully stops the consumer
func (c *Consumer) Stop() {
	atomic.StoreUint32(&c.running, 0)
	close(c.stopChan)
}
