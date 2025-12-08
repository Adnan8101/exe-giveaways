package cache

import (
	"context"
	"discord-giveaway-bot/internal/redis"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/ristretto"
	"golang.org/x/sync/singleflight"
)

// Cache provides a multi-layer caching system with L1 (in-memory) and L2 (Redis)
type Cache struct {
	l1           *ristretto.Cache
	l2           *redis.Client
	singleflight singleflight.Group

	// Metrics
	l1Hits   atomic.Uint64
	l1Misses atomic.Uint64
	l2Hits   atomic.Uint64
	l2Misses atomic.Uint64
}

// Config for cache initialization
type Config struct {
	L1MaxCost     int64         // Max cost in bytes for L1 cache (default: 10MB)
	L1NumCounters int64         // Number of keys to track frequency (default: 100k)
	DefaultTTL    time.Duration // Default TTL for cache entries
}

// NewCache creates a new multi-layer cache
func NewCache(redis *redis.Client, cfg Config) (*Cache, error) {
	if cfg.L1MaxCost == 0 {
		cfg.L1MaxCost = 10 << 20 // 10MB default
	}
	if cfg.L1NumCounters == 0 {
		cfg.L1NumCounters = 100000
	}
	if cfg.DefaultTTL == 0 {
		cfg.DefaultTTL = 5 * time.Minute
	}

	l1, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: cfg.L1NumCounters,
		MaxCost:     cfg.L1MaxCost,
		BufferItems: 64,
		Metrics:     true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create L1 cache: %w", err)
	}

	return &Cache{
		l1: l1,
		l2: redis,
	}, nil
}

// Get retrieves a value from cache with automatic L1->L2->L3 fallback
func (c *Cache) Get(ctx context.Context, key string, l3Fetch func() (interface{}, error)) (interface{}, error) {
	// Try L1 cache first
	if val, found := c.l1.Get(key); found {
		c.l1Hits.Add(1)
		return val, nil
	}
	c.l1Misses.Add(1)

	// Try L2 (Redis) cache
	if c.l2 != nil {
		if val, err := c.l2.Get(key); err == nil && val != "" {
			c.l2Hits.Add(1)
			// Store in L1 for next time
			c.l1.Set(key, val, 1)
			return val, nil
		}
		c.l2Misses.Add(1)
	}

	// L3 fetch with singleflight to prevent stampede
	val, err, _ := c.singleflight.Do(key, func() (interface{}, error) {
		return l3Fetch()
	})

	if err != nil {
		return nil, err
	}

	// Store in both caches
	c.Set(key, val, 5*time.Minute)
	return val, nil
}

// Set stores a value in both L1 and L2 caches
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	// L1 cache (in-memory)
	c.l1.SetWithTTL(key, value, 1, ttl)

	// L2 cache (Redis)
	if c.l2 != nil {
		c.l2.Set(key, value, ttl)
	}
}

// Delete removes a key from all cache layers
func (c *Cache) Delete(key string) {
	c.l1.Del(key)
	if c.l2 != nil {
		c.l2.Del(key)
	}
}

// GetMetrics returns cache performance metrics
func (c *Cache) GetMetrics() Metrics {
	l1Metrics := c.l1.Metrics

	l1Total := c.l1Hits.Load() + c.l1Misses.Load()
	l2Total := c.l2Hits.Load() + c.l2Misses.Load()

	var l1HitRate, l2HitRate float64
	if l1Total > 0 {
		l1HitRate = float64(c.l1Hits.Load()) / float64(l1Total)
	}
	if l2Total > 0 {
		l2HitRate = float64(c.l2Hits.Load()) / float64(l2Total)
	}

	return Metrics{
		L1Hits:        c.l1Hits.Load(),
		L1Misses:      c.l1Misses.Load(),
		L1HitRate:     l1HitRate,
		L2Hits:        c.l2Hits.Load(),
		L2Misses:      c.l2Misses.Load(),
		L2HitRate:     l2HitRate,
		L1KeysAdded:   l1Metrics.KeysAdded(),
		L1KeysEvicted: l1Metrics.KeysEvicted(),
		L1CostAdded:   l1Metrics.CostAdded(),
		L1CostEvicted: l1Metrics.CostEvicted(),
	}
}

// Metrics holds cache performance data
type Metrics struct {
	L1Hits        uint64
	L1Misses      uint64
	L1HitRate     float64
	L2Hits        uint64
	L2Misses      uint64
	L2HitRate     float64
	L1KeysAdded   uint64
	L1KeysEvicted uint64
	L1CostAdded   uint64
	L1CostEvicted uint64
}

// Close gracefully shuts down the cache
func (c *Cache) Close() {
	c.l1.Close()
}

// WarmUp pre-loads frequently accessed data into cache
func (c *Cache) WarmUp(items map[string]interface{}, ttl time.Duration) {
	for key, value := range items {
		c.Set(key, value, ttl)
	}
}
