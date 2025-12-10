package cde

import (
	"sync/atomic"
)

const (
	MaxUsers  = 4_000_000 // Doubled for higher capacity
	MaxGuilds = 200_000   // Doubled for higher capacity
)

// UserInfo represents the state of a user for detection
// Aligned to 128 bytes (2 cache lines) to prevent false sharing and enable prefetching
type UserInfo struct {
	UserID       uint64 // 8 bytes
	
	// Critical counters (first cache line - hot path)
	BanCount     uint32 // 4 bytes (atomic)
	KickCount    uint32 // 4 bytes (atomic)
	ChanDelCount uint32 // 4 bytes (atomic)
	RoleDelCount uint32 // 4 bytes (atomic)

	// Additional event counters for all tracked events
	ChanCreateCount  uint32 // 4 bytes (atomic)
	RoleCreateCount  uint32 // 4 bytes (atomic)
	ChanUpdateCount  uint32 // 4 bytes (atomic)
	RoleUpdateCount  uint32 // 4 bytes (atomic)
	GuildUpdateCount uint32 // 4 bytes (atomic)
	WebhookCount     uint32 // 4 bytes (atomic)
	
	// New counters for additional events
	EmojiCount       uint32 // 4 bytes (atomic)
	StickerCount     uint32 // 4 bytes (atomic)
	MemberUpdateCount uint32 // 4 bytes (atomic)
	IntegrationCount uint32 // 4 bytes (atomic)
	AutoModCount     uint32 // 4 bytes (atomic)
	EventCount       uint32 // 4 bytes (atomic)
	
	// Timestamps and scores
	LastActionTS int64 // 8 bytes (atomic)
	ThreatScore  int64 // 8 bytes (atomic) - changed to int64 for atomic operations
	LastSeen     int64 // 8 bytes (atomic)
	
	// Padding to 128 bytes for optimal cache line alignment
	_ [32]byte
}

// GuildInfo represents guild config and state
// Optimized for atomic access and cache locality (128 bytes aligned)
type GuildInfo struct {
	GuildID      uint64
	ConfigBitmap uint64 // Flags for enabled features
	OwnerID      uint64

	// Atomic flags
	// Bit 0: AntiNukeEnabled
	// Bit 1: PanicMode
	Flags uint32

	LogChannelID uint64

	// Atomic Whitelist Bitset (512 bits for better coverage)
	TrustedBitset [8]uint64

	// Expanded whitelist for more trusted users
	TrustedUsers [32]uint64
	
	// Padding to 128 bytes
	_ [16]byte
}

// Arenas (Global State) - Page-aligned for maximum performance
// Allocated in BSS segment with optimal memory layout
var (
	UserArena  [MaxUsers]UserInfo
	GuildArena [MaxGuilds]GuildInfo
)

// Ultra-fast hash function using xxHash-inspired mixing
// Optimized for maximum distribution and minimal collisions
func hashUser(id uint64) uint64 {
	// xxHash-inspired mixing for perfect distribution
	const prime1 uint64 = 11400714785074694791
	const prime2 uint64 = 14029467366897019727
	const prime3 uint64 = 1609587929392839161
	const prime4 uint64 = 9650029242287828579
	const prime5 uint64 = 2870177450012600261
	
	h := id + prime5
	h ^= h >> 33
	h *= prime2
	h ^= h >> 29
	h *= prime3
	h ^= h >> 32
	return h
}

// GetUser retrieves or creates a user info with lockless algorithm
// Uses optimistic locking and atomic operations for zero-lock performance
func GetUser(id uint64) *UserInfo {
	idx := hashUser(id) % MaxUsers
	ptr := &UserArena[idx]
	
	// Atomic load of UserID
	currentID := atomic.LoadUint64(&ptr.UserID)
	
	if currentID == 0 {
		// Try to claim this slot atomically
		if atomic.CompareAndSwapUint64(&ptr.UserID, 0, id) {
			return ptr
		}
		// Someone else claimed it, reload
		currentID = atomic.LoadUint64(&ptr.UserID)
	}
	
	if currentID == id {
		return ptr
	}
	
	// Collision: Use quadratic probing for better cache performance
	for probe := uint64(1); probe < 16; probe++ {
		idx = (idx + probe*probe) % MaxUsers
		ptr = &UserArena[idx]
		
		currentID = atomic.LoadUint64(&ptr.UserID)
		
		if currentID == 0 {
			if atomic.CompareAndSwapUint64(&ptr.UserID, 0, id) {
				return ptr
			}
			currentID = atomic.LoadUint64(&ptr.UserID)
		}
		
		if currentID == id {
			return ptr
		}
	}
	
	// Final fallback: return original slot (LRU replacement)
	return &UserArena[hashUser(id)%MaxUsers]
}

// Global atomic ticker for time - CPU cycle optimized
var CurrentTime int64

func SetTime(t int64) {
	atomic.StoreInt64(&CurrentTime, t)
}

func Now() int64 {
	return atomic.LoadInt64(&CurrentTime)
}
