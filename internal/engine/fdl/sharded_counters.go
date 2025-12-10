package fdl

import (
	"sync/atomic"
)

// ShardedCounter provides distinct counters for each execution shard
// to avoid false sharing and cache line contention
type ShardedCounter struct {
	Shards [256]uint64
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
	atomic.AddUint64(&s.Shards[id%256], 1)
}

// GetTotal sums all shards to get the total count
// This is a "slow" operation, used only for metrics reporting
func (s *ShardedCounter) GetTotal() uint64 {
	var total uint64
	for i := 0; i < 256; i++ {
		total += atomic.LoadUint64(&s.Shards[i])
	}
	return total
}

// Reset zeroes out all counters (non-atomic relative to other operations)
func (s *ShardedCounter) Reset() {
	for i := 0; i < 256; i++ {
		atomic.StoreUint64(&s.Shards[i], 0)
	}
}
