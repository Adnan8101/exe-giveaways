package ring

import (
	"runtime"
	"time"

	"discord-giveaway-bot/internal/engine/fdl"
)

// Consumer is a worker that processes events from the ring
type Consumer struct {
	Ring    *RingBuffer
	Handler func(fdl.FastEvent)
	ID      int
}

// Start begins the consumer loop
// This should be pinned to a core in main
func (c *Consumer) Start() {
	// Pin this goroutine to an OS thread for maximum performance
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Spin-wait loop for ultra-low latency
	for {
		evt, ok := c.Ring.Pop()
		if !ok {
			// Busy-wait optimization: spin for a bit before yielding
			for i := 0; i < 10000; i++ {
				evt, ok = c.Ring.Pop()
				if ok {
					goto process
				}
				runtime.Gosched()
			}
			// After spinning, sleep briefly to save CPU
			time.Sleep(100 * time.Nanosecond)
			continue
		}

	process:
		// NO LOGGING IN HOT PATH - Direct handler call
		c.Handler(evt)
	}
}
