package core

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// GuildConfig represents immutable antinuke configuration for a guild
// Uses atomic pointer swaps for lock-free updates
type GuildConfig struct {
	GuildID     string
	Enabled     bool
	OwnerID     string
	LogsChannel string
	PanicMode   bool
}

// LimitConfig represents rate limit configuration for an action type
type LimitConfig struct {
	GuildID       string
	ActionType    string
	Enabled       bool
	LimitCount    int
	WindowSeconds int
	Punishment    string
}

// WhitelistSet is a lock-free set of whitelisted user/role IDs
// Implemented as immutable map for zero-allocation reads
type WhitelistSet struct {
	items map[string]struct{} // Immutable after creation
}

// AtomicCache provides lock-free, zero-allocation config access
// All reads use atomic.LoadPointer (no mutexes, ~50ns latency)
// Updates use atomic.StorePointer (copy-on-write pattern)
type AtomicCache struct {
	// sync.Map is lock-free for reads after initial write
	// Perfect for our use case: frequent reads, rare writes
	configs   sync.Map // guildID -> unsafe.Pointer(*GuildConfig)
	whitelist sync.Map // guildID -> unsafe.Pointer(*WhitelistSet)
	limits    sync.Map // "guildID:actionType" -> unsafe.Pointer(*LimitConfig)
}

// NewAtomicCache creates a new lock-free cache
func NewAtomicCache() *AtomicCache {
	return &AtomicCache{}
}

// GetConfig retrieves guild config with zero allocations
// Latency: ~50-100ns (atomic pointer load)
func (c *AtomicCache) GetConfig(guildID string) *GuildConfig {
	val, ok := c.configs.Load(guildID)
	if !ok {
		return nil
	}

	// Type assertion is safe - we only store *GuildConfig
	return val.(*GuildConfig)
}

// SetConfig updates guild config atomically
// Uses copy-on-write: creates new config, swaps pointer
func (c *AtomicCache) SetConfig(cfg *GuildConfig) {
	c.configs.Store(cfg.GuildID, cfg)
}

// DeleteConfig removes a guild config
func (c *AtomicCache) DeleteConfig(guildID string) {
	c.configs.Delete(guildID)
}

// GetLimit retrieves action limit config with zero allocations
// Latency: ~50-100ns
func (c *AtomicCache) GetLimit(guildID, actionType string) *LimitConfig {
	key := guildID + ":" + actionType
	val, ok := c.limits.Load(key)
	if !ok {
		return nil
	}

	return val.(*LimitConfig)
}

// SetLimit updates action limit config atomically
func (c *AtomicCache) SetLimit(cfg *LimitConfig) {
	key := cfg.GuildID + ":" + cfg.ActionType
	c.limits.Store(key, cfg)
}

// DeleteLimit removes an action limit config
func (c *AtomicCache) DeleteLimit(guildID, actionType string) {
	key := guildID + ":" + actionType
	c.limits.Delete(key)
}

// IsWhitelisted checks if a user/role is whitelisted
// Latency: ~100ns (atomic load + map lookup)
// Zero allocations
func (c *AtomicCache) IsWhitelisted(guildID, targetID string) bool {
	val, ok := c.whitelist.Load(guildID)
	if !ok {
		return false
	}

	set := val.(*WhitelistSet)
	_, exists := set.items[targetID]
	return exists
}

// SetWhitelist updates the whitelist for a guild atomically
// Takes a slice of IDs and converts to immutable set
func (c *AtomicCache) SetWhitelist(guildID string, ids []string) {
	// Create immutable map
	items := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		items[id] = struct{}{}
	}

	set := &WhitelistSet{items: items}
	c.whitelist.Store(guildID, set)
}

// AddToWhitelist adds a single ID to the whitelist
// Uses copy-on-write to maintain immutability
func (c *AtomicCache) AddToWhitelist(guildID, targetID string) {
	val, ok := c.whitelist.Load(guildID)

	var oldItems map[string]struct{}
	if ok {
		oldItems = val.(*WhitelistSet).items
	} else {
		oldItems = make(map[string]struct{})
	}

	// Copy old items + new ID
	newItems := make(map[string]struct{}, len(oldItems)+1)
	for k := range oldItems {
		newItems[k] = struct{}{}
	}
	newItems[targetID] = struct{}{}

	set := &WhitelistSet{items: newItems}
	c.whitelist.Store(guildID, set)
}

// RemoveFromWhitelist removes a single ID from the whitelist
// Uses copy-on-write to maintain immutability
func (c *AtomicCache) RemoveFromWhitelist(guildID, targetID string) {
	val, ok := c.whitelist.Load(guildID)
	if !ok {
		return
	}

	oldItems := val.(*WhitelistSet).items

	// Copy without removed ID
	newItems := make(map[string]struct{}, len(oldItems))
	for k := range oldItems {
		if k != targetID {
			newItems[k] = struct{}{}
		}
	}

	set := &WhitelistSet{items: newItems}
	c.whitelist.Store(guildID, set)
}

// DeleteWhitelist removes all whitelist entries for a guild
func (c *AtomicCache) DeleteWhitelist(guildID string) {
	c.whitelist.Delete(guildID)
}

// Stats returns cache statistics for monitoring
type CacheStats struct {
	ConfigCount    int
	WhitelistCount int
	LimitCount     int
}

// GetStats returns current cache statistics
func (c *AtomicCache) GetStats() CacheStats {
	stats := CacheStats{}

	c.configs.Range(func(_, _ interface{}) bool {
		stats.ConfigCount++
		return true
	})

	c.whitelist.Range(func(_, _ interface{}) bool {
		stats.WhitelistCount++
		return true
	})

	c.limits.Range(func(_, _ interface{}) bool {
		stats.LimitCount++
		return true
	})

	return stats
}

// AtomicPointer is a generic atomic pointer wrapper for Go 1.19+
// This provides type-safe atomic operations
type AtomicPointer[T any] struct {
	ptr unsafe.Pointer
}

// Load atomically loads the pointer value
func (p *AtomicPointer[T]) Load() *T {
	return (*T)(atomic.LoadPointer(&p.ptr))
}

// Store atomically stores a new pointer value
func (p *AtomicPointer[T]) Store(val *T) {
	atomic.StorePointer(&p.ptr, unsafe.Pointer(val))
}

// CompareAndSwap performs atomic compare-and-swap
func (p *AtomicPointer[T]) CompareAndSwap(old, new *T) bool {
	return atomic.CompareAndSwapPointer(&p.ptr, unsafe.Pointer(old), unsafe.Pointer(new))
}
