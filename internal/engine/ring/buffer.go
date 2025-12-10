package ring

import (
	"sync/atomic"

	"discord-giveaway-bot/internal/engine/fdl"
)

// BufferSize must be power of 2 for fast modulo
const BufferSize = 1024 * 16
const IndexMask = BufferSize - 1

// RingBuffer is a Wait-Free Single-Producer Single-Consumer (SPSC) Ring Buffer.
// It uses padding to avoid false sharing between the producer (head) and consumer (tail).
type RingBuffer struct {
	// Pre-allocated data storage
	// We use value semantics to keep data contiguous in memory for cache locality
	data [BufferSize]fdl.FastEvent

	_ [64]byte // Padding to isolate data from head

	// Producer Write Index
	// Only modified by Producer
	head uint64

	_ [56]byte // Padding to isolate head from tail (Assuming 64-byte cache line: 8 bytes uint64 + 56 bytes pad)

	// Consumer Read Index
	// Only modified by Consumer
	tail uint64

	_ [56]byte // Padding to isolate tail from anything else
}

// New creates a new SPSC RingBuffer
func New() *RingBuffer {
	return &RingBuffer{}
}

// Push adds an item to the ring.
// WAIT-FREE. Single-Producer ONLY.
// Returns false if buffer is full.
func (r *RingBuffer) Push(e *fdl.FastEvent) bool {
	head := atomic.LoadUint64(&r.head)
	tail := atomic.LoadUint64(&r.tail)

	if head-tail >= BufferSize {
		return false // Buffer Full
	}

	// Write data
	// Since there is only one producer, we don't need CAS for the slot.
	// We just write and then publish the new head.
	// NOTE: We copy the value struct into the array slot (Pre-allocated slot usage)
	r.data[head&IndexMask] = *e

	// Publish write (Store Release)
	atomic.StoreUint64(&r.head, head+1)
	return true
}

// Pop returns the next item.
// WAIT-FREE. Single-Consumer ONLY.
// Returns false if empty.
func (r *RingBuffer) Pop() (fdl.FastEvent, bool) {
	tail := atomic.LoadUint64(&r.tail)
	head := atomic.LoadUint64(&r.head)

	if tail >= head {
		return fdl.FastEvent{}, false // Empty
	}

	// Read data
	item := r.data[tail&IndexMask]

	// Publish read (Store Release)
	atomic.StoreUint64(&r.tail, tail+1)

	return item, true
}

// Len returns approximate length
// Note: This is a racy read if called concurrently, but safe for metrics
func (r *RingBuffer) Len() uint64 {
	return atomic.LoadUint64(&r.head) - atomic.LoadUint64(&r.tail)
}

// GetSlot returns a pointer to the next write slot to allow zero-copy writing.
// The caller must call Commit() after writing.
func (r *RingBuffer) GetWriteSlot() *fdl.FastEvent {
	head := atomic.LoadUint64(&r.head)
	tail := atomic.LoadUint64(&r.tail)

	if head-tail >= BufferSize {
		return nil
	}
	return &r.data[head&IndexMask]
}

// Commit publishes the write slot
func (r *RingBuffer) Commit() {
	// We assume single producer, so r.head is owned by us (mostly), but for safety we load it again?
	// or we just atomic add?
	// To be completely safe/correct with GetWriteSlot pattern we should probably track head locally
	// or just atomic add.
	atomic.AddUint64(&r.head, 1)
}
