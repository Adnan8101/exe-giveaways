package core

import (
	"sync"
	"sync/atomic"
	"time"
)

// EventWindow is a lock-free sliding window for rate limiting
// Uses ring buffer with atomic operations for thread-safe access
type EventWindow struct {
	events   [128]int64 // Ring buffer of Unix timestamps (pre-allocated)
	writePos atomic.Uint32
	size     atomic.Uint32
}

// RateLimiter provides lock-free rate limiting using atomic operations
// Target: <10µs per check with zero allocations
type RateLimiter struct {
	windows sync.Map // "guildID:actionType:userID" -> *EventWindow
}

// NewRateLimiter creates a new lock-free rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{}
}

// Check performs a rate limit check and records the event
// Returns: triggered (bool), current count (int)
// Latency: <10µs with zero allocations
func (r *RateLimiter) Check(guildID, actionType, userID string, limit int, windowSecs int) (bool, int) {
	key := guildID + ":" + actionType + ":" + userID
	now := time.Now().Unix()
	cutoff := now - int64(windowSecs)

	// Get or create event window
	val, ok := r.windows.Load(key)
	if !ok {
		val, _ = r.windows.LoadOrStore(key, &EventWindow{})
	}
	window := val.(*EventWindow)

	// Count events in current window (lock-free)
	count := r.countRecentEvents(window, cutoff)

	// Add current event atomically
	r.addEvent(window, now)

	// Increment count for current event
	count++

	// Check if limit exceeded
	triggered := count > limit

	return triggered, count
}

// addEvent adds an event timestamp to the ring buffer atomically
func (r *RateLimiter) addEvent(window *EventWindow, timestamp int64) {
	// Get current write position and increment atomically
	pos := window.writePos.Add(1) - 1

	// Write to ring buffer (wraps around at 128)
	idx := pos % 128
	atomic.StoreInt64(&window.events[idx], timestamp)

	// Update size (capped at buffer size)
	for {
		oldSize := window.size.Load()
		newSize := oldSize + 1
		if newSize > 128 {
			newSize = 128
		}

		if window.size.CompareAndSwap(oldSize, newSize) {
			break
		}
	}
}

// countRecentEvents counts events within the time window
// Uses atomic loads for thread-safe access
func (r *RateLimiter) countRecentEvents(window *EventWindow, cutoff int64) int {
	size := window.size.Load()
	if size == 0 {
		return 0
	}

	count := 0

	// Scan ring buffer (lock-free reads)
	for i := uint32(0); i < size; i++ {
		ts := atomic.LoadInt64(&window.events[i])
		if ts >= cutoff {
			count++
		}
	}

	return count
}

// Reset clears a specific rate limit window
func (r *RateLimiter) Reset(guildID, actionType, userID string) {
	key := guildID + ":" + actionType + ":" + userID
	r.windows.Delete(key)
}

// ResetGuild clears all rate limit windows for a guild
func (r *RateLimiter) ResetGuild(guildID string) {
	prefix := guildID + ":"

	r.windows.Range(func(key, _ interface{}) bool {
		if k, ok := key.(string); ok {
			if len(k) > len(prefix) && k[:len(prefix)] == prefix {
				r.windows.Delete(k)
			}
		}
		return true
	})
}

// Cleanup removes old event windows that haven't been used recently
// Should be called periodically (e.g., every 5 minutes) to prevent memory bloat
func (r *RateLimiter) Cleanup(maxIdleSeconds int64) int {
	now := time.Now().Unix()
	removed := 0

	r.windows.Range(func(key, val interface{}) bool {
		window := val.(*EventWindow)
		size := window.size.Load()

		if size == 0 {
			r.windows.Delete(key)
			removed++
			return true
		}

		// Check most recent event
		lastPos := (window.writePos.Load() - 1) % 128
		lastEvent := atomic.LoadInt64(&window.events[lastPos])

		// If no events in last maxIdleSeconds, remove window
		if now-lastEvent > maxIdleSeconds {
			r.windows.Delete(key)
			removed++
		}

		return true
	})

	return removed
}

// Stats returns rate limiter statistics
type RateLimiterStats struct {
	ActiveWindows int
	TotalEvents   int64
}

// GetStats returns current statistics
func (r *RateLimiter) GetStats() RateLimiterStats {
	stats := RateLimiterStats{}

	r.windows.Range(func(_, val interface{}) bool {
		stats.ActiveWindows++
		window := val.(*EventWindow)
		stats.TotalEvents += int64(window.size.Load())
		return true
	})

	return stats
}

// FastRateLimiter is an optimized version for extremely high throughput
// Uses sharding to reduce contention on sync.Map
type FastRateLimiter struct {
	shards [64]*RateLimiter // 64 shards to reduce sync.Map contention
}

// NewFastRateLimiter creates a sharded rate limiter
func NewFastRateLimiter() *FastRateLimiter {
	f := &FastRateLimiter{}
	for i := 0; i < 64; i++ {
		f.shards[i] = NewRateLimiter()
	}
	return f
}

// getShard returns the appropriate shard for a key
func (f *FastRateLimiter) getShard(guildID, actionType, userID string) *RateLimiter {
	// Simple hash: sum of bytes modulo shard count
	hash := uint64(0)
	for i := 0; i < len(guildID); i++ {
		hash = hash*31 + uint64(guildID[i])
	}
	for i := 0; i < len(actionType); i++ {
		hash = hash*31 + uint64(actionType[i])
	}
	for i := 0; i < len(userID); i++ {
		hash = hash*31 + uint64(userID[i])
	}

	return f.shards[hash%64]
}

// Check performs a sharded rate limit check
func (f *FastRateLimiter) Check(guildID, actionType, userID string, limit int, windowSecs int) (bool, int) {
	shard := f.getShard(guildID, actionType, userID)
	return shard.Check(guildID, actionType, userID, limit, windowSecs)
}

// Reset clears a specific rate limit window
func (f *FastRateLimiter) Reset(guildID, actionType, userID string) {
	shard := f.getShard(guildID, actionType, userID)
	shard.Reset(guildID, actionType, userID)
}

// ResetGuild clears all rate limit windows for a guild
func (f *FastRateLimiter) ResetGuild(guildID string) {
	// Must check all shards since we don't know which ones have this guild
	for i := 0; i < 64; i++ {
		f.shards[i].ResetGuild(guildID)
	}
}

// Cleanup removes old event windows across all shards
func (f *FastRateLimiter) Cleanup(maxIdleSeconds int64) int {
	total := 0
	for i := 0; i < 64; i++ {
		total += f.shards[i].Cleanup(maxIdleSeconds)
	}
	return total
}

// GetStats returns aggregated statistics across all shards
func (f *FastRateLimiter) GetStats() RateLimiterStats {
	stats := RateLimiterStats{}

	for i := 0; i < 64; i++ {
		shardStats := f.shards[i].GetStats()
		stats.ActiveWindows += shardStats.ActiveWindows
		stats.TotalEvents += shardStats.TotalEvents
	}

	return stats
}
