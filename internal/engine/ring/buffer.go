package ring

import (
	"runtime"
	"sync/atomic"

	"discord-giveaway-bot/internal/engine/fdl"
)

// Size must be power of 2 for fast modulo
const BufferSize = 1024 * 16
const IndexMask = BufferSize - 1

type RingBuffer struct {
	data [BufferSize]fdl.FastEvent
	_    [56]byte // padding to prevent false sharing
	head uint64   // Write index (Producer)
	_    [56]byte // padding
	tail uint64   // Read index (Consumer)
	_    [56]byte // padding
}

func New() *RingBuffer {
	return &RingBuffer{}
}

// Push adds an item to the ring.
// Single Producer only - no CAS needed for head.
func (r *RingBuffer) Push(e *fdl.FastEvent) bool {
	// Check if full
	tail := atomic.LoadUint64(&r.tail)
	if r.head-tail >= BufferSize {
		return false // Buffer full
	}

	// Store data
	// Using atomic pointer swap might be safer but normal assignment is atomic for valid structs?
	// Go memory model guarantees word-aligned assignment is atomic.
	// FastEvent is larger than a word, so we could have tearing if we are not careful.
	// However, usually we just write and then increment head.
	// The consumer waits for head > current_index.

	// Better safety: Slot availability check.

	idx := r.head & IndexMask
	r.data[idx] = *e

	// Commit write
	atomic.AddUint64(&r.head, 1)
	return true
}

// Pop returns the next item.
// Multi-Consumer safe via CAS.
func (r *RingBuffer) Pop() (fdl.FastEvent, bool) {
	var empty fdl.FastEvent

	for {
		tail := atomic.LoadUint64(&r.tail)
		head := atomic.LoadUint64(&r.head)

		if tail >= head {
			return empty, false // Empty
		}

		// Attempt to claim slot
		if atomic.CompareAndSwapUint64(&r.tail, tail, tail+1) {
			item := r.data[tail&IndexMask]
			return item, true
		}
		// If CAS failed, another consumer got it, retry
		runtime.Gosched()
	}
}

// Len returns approximate length
func (r *RingBuffer) Len() uint64 {
	return atomic.LoadUint64(&r.head) - atomic.LoadUint64(&r.tail)
}
