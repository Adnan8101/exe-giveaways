package fdl

import (
	"sync/atomic"
)

// CacheLine represents a cache-line padded uint64 to prevent false sharing
// x86_64 CPUs have 64-byte cache lines
type CacheLine struct {
	Value uint64
	_     [7]uint64 // Padding to 64 bytes total
}

// ShardedCounter provides distinct counters for each execution shard
// to avoid false sharing and cache line contention
// Using 256 shards for optimal distribution across CPU cores
type ShardedCounter struct {
	Shards [256]CacheLine
}

// Global counters for key metrics - Pre-allocated for zero overhead
var (
	TotalEvents       ShardedCounter
	EventsDropped     ShardedCounter
	EventsProcessed   ShardedCounter
	EventsDetected    ShardedCounter
	PunishmentsIssued ShardedCounter
)

// Inc increments the counter for a specific ID (sharded by ID)
// Uses atomic operations for thread-safe lock-free increments
//
//go:inline
func (s *ShardedCounter) Inc(id uint64) {
	// No lock, just atomic increment on the specific cache line
	// The shard is determined by the ID modulo 256 for perfect distribution
	shard := id & 255 // Bitwise AND is faster than modulo for power of 2
	atomic.AddUint64(&s.Shards[shard].Value, 1)
}

// IncBy increments the counter by a specific amount
//
//go:inline
func (s *ShardedCounter) IncBy(id uint64, delta uint64) {
	shard := id & 255
	atomic.AddUint64(&s.Shards[shard].Value, delta)
}

// GetTotal sums all shards to get the total count
// This is a "slow" operation, used only for metrics reporting
func (s *ShardedCounter) GetTotal() uint64 {
	var total uint64
	for i := 0; i < 256; i++ {
		total += atomic.LoadUint64(&s.Shards[i].Value)
	}
	return total
}

// GetShard returns the value for a specific shard (for debugging)
func (s *ShardedCounter) GetShard(shard int) uint64 {
	if shard < 0 || shard >= 256 {
		return 0
	}
	return atomic.LoadUint64(&s.Shards[shard].Value)
}

// Reset zeroes out all counters (non-atomic relative to other operations)
// Should only be called when no other operations are in progress
func (s *ShardedCounter) Reset() {
	for i := 0; i < 256; i++ {
		atomic.StoreUint64(&s.Shards[i].Value, 0)
	}
}

// Snapshot returns a copy of all shard values for consistent metrics
func (s *ShardedCounter) Snapshot() [256]uint64 {
	var snapshot [256]uint64
	for i := 0; i < 256; i++ {
		snapshot[i] = atomic.LoadUint64(&s.Shards[i].Value)
	}
	return snapshot
}
