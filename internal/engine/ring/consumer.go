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
	for {
		evt, ok := c.Ring.Pop()
		if !ok {
			// Spin-wait optimization:
			// Instead of immediate sleep, we spin for a bit, then yield.
			// For ultra-low latency, busy-wait is preferred if CPU allows.
			for i := 0; i < 1000; i++ {
				evt, ok = c.Ring.Pop()
				if ok {
					break
				}
				runtime.Gosched()
			}
			if !ok {
				// If still empty sleep tiny amount to save CPU
				// In pure HFT this would be strictly busy wait
				time.Sleep(1 * time.Microsecond)
				continue
			}
		}

		c.Handler(evt)
	}
}
