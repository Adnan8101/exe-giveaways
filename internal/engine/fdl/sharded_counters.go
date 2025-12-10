package fdl

import (
	"sync/atomic"
)

// CacheLine represents a cache-line padded uint64 to prevent false sharing
// Most CPUs have 64-byte cache lines
type CacheLine struct {
	Value uint64
	_     [7]uint64 // Padding to 64 bytes
}

// ShardedCounter provides distinct counters for each execution shard
// to avoid false sharing and cache line contention
type ShardedCounter struct {
	Shards [256]CacheLine
}

// Global counters for key metrics
var (
	TotalEvents     ShardedCounter
	EventsDropped   ShardedCounter
	EventsProcessed ShardedCounter
)

// Inc increments the counter for a specific ID (sharded by ID)
func (s *ShardedCounter) Inc(id uint64) {
	// No lock, just atomic increment on the specific cache line
	// The shard is determined by the ID modulo 256
	shard := id % 256
	atomic.AddUint64(&s.Shards[shard].Value, 1)
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

// Reset zeroes out all counters (non-atomic relative to other operations)
func (s *ShardedCounter) Reset() {
	for i := 0; i < 256; i++ {
		atomic.StoreUint64(&s.Shards[i].Value, 0)
	}
}
