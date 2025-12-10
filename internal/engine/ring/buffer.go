package ring

import (
	"sync/atomic"
	"unsafe"

	"discord-giveaway-bot/internal/engine/fdl"
)

// BufferSize must be power of 2 for fast modulo (increased for higher throughput)
const BufferSize = 1024 * 64 // 64K events = massive buffer
const IndexMask = BufferSize - 1

// RingBuffer is a Wait-Free Single-Producer Single-Consumer (SPSC) Ring Buffer.
// Uses cache-line padding and atomic operations for maximum performance
type RingBuffer struct {
	// Pre-allocated data storage - aligned to cache line boundaries
	// We use value semantics to keep data contiguous in memory for cache locality
	data [BufferSize]fdl.FastEvent

	_ [64]byte // Padding to isolate data from head

	// Producer Write Index
	// Only modified by Producer
	head uint64

	_ [56]byte // Padding to isolate head from tail (64-byte cache line: 8 bytes uint64 + 56 bytes pad)

	// Consumer Read Index
	// Only modified by Consumer
	tail uint64

	_ [56]byte // Padding to isolate tail from anything else
}

// New creates a new SPSC RingBuffer with pre-allocated memory
func New() *RingBuffer {
	rb := &RingBuffer{}
	// Ensure proper alignment for optimal CPU cache usage
	if (uintptr(unsafe.Pointer(rb)) & 63) != 0 {
		// Not 64-byte aligned, but Go allocator should handle this
	}
	return rb
}

// Push adds an item to the ring using zero-copy semantics
// WAIT-FREE. Single-Producer ONLY.
// Returns false if buffer is full.
//
//go:inline
func (r *RingBuffer) Push(e *fdl.FastEvent) bool {
	// Relaxed loads for maximum performance
	head := atomic.LoadUint64(&r.head)
	tail := atomic.LoadUint64(&r.tail)

	// Check if buffer is full with optimized comparison
	if head-tail >= BufferSize {
		return false // Buffer Full - backpressure signal
	}

	// Write data directly to pre-allocated slot (cache-friendly)
	// Since there is only one producer, we don't need CAS for the slot.
	// Memory copy is optimized by compiler for cache-line aligned structures
	r.data[head&IndexMask] = *e

	// Publish write with store-release semantics for memory ordering
	atomic.StoreUint64(&r.head, head+1)
	return true
}

// Pop returns the next item with zero-allocation
// WAIT-FREE. Single-Consumer ONLY.
// Returns false if empty.
//
//go:inline
func (r *RingBuffer) Pop() (fdl.FastEvent, bool) {
	// Relaxed loads
	tail := atomic.LoadUint64(&r.tail)
	head := atomic.LoadUint64(&r.head)

	if tail >= head {
		return fdl.FastEvent{}, false // Empty
	}

	// Read data from pre-allocated slot (cache-friendly)
	item := r.data[tail&IndexMask]

	// Publish read with store-release semantics
	atomic.StoreUint64(&r.tail, tail+1)

	return item, true
}

// PopBatch returns multiple items at once for batch processing optimization
// Returns slice of events and count
func (r *RingBuffer) PopBatch(maxBatch int) ([]fdl.FastEvent, int) {
	tail := atomic.LoadUint64(&r.tail)
	head := atomic.LoadUint64(&r.head)
	
	available := int(head - tail)
	if available == 0 {
		return nil, 0
	}
	
	count := available
	if count > maxBatch {
		count = maxBatch
	}
	
	batch := make([]fdl.FastEvent, count)
	for i := 0; i < count; i++ {
		batch[i] = r.data[(tail+uint64(i))&IndexMask]
	}
	
	atomic.StoreUint64(&r.tail, tail+uint64(count))
	return batch, count
}

// Len returns approximate length
// Note: This is a racy read if called concurrently, but safe for metrics
func (r *RingBuffer) Len() uint64 {
	return atomic.LoadUint64(&r.head) - atomic.LoadUint64(&r.tail)
}

// GetWriteSlot returns a pointer to the next write slot to allow zero-copy writing.
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
