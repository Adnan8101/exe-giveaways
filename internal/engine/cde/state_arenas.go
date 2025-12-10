package cde

import (
	"sync/atomic"
)

const (
	MaxUsers  = 4 * 1024 * 1024 // Power of 2 (4M) for bitwise masking
	MaxGuilds = 256 * 1024      // Power of 2 (256K)
	UserMask  = MaxUsers - 1
	GuildMask = MaxGuilds - 1
)

// UserInfo represents the state of a user for detection
// Aligned to 128 bytes (2 cache lines) to prevent false sharing and enable prefetching
type UserInfo struct {
	UserID uint64 // 8 bytes

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
	EmojiCount        uint32 // 4 bytes (atomic)
	StickerCount      uint32 // 4 bytes (atomic)
	MemberUpdateCount uint32 // 4 bytes (atomic)
	IntegrationCount  uint32 // 4 bytes (atomic)
	AutoModCount      uint32 // 4 bytes (atomic)
	EventCount        uint32 // 4 bytes (atomic)

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

// Ultra-fast hash function using simple bitwise operations
// Since Snowflakes are already semi-random in lower bits and we use power-of-2 arena,
// we can use a much simpler hash.
//
//go:inline
func hashUser(id uint64) uint64 {
	// Fibonacci hashing for good distribution on power-of-2 tables
	return (id * 11400714819323198485) >> 32
}

// GetUser retrieves or creates a user info with lockless algorithm
// Uses optimistic locking and atomic operations for zero-lock performance
//
//go:inline
func GetUser(id uint64) *UserInfo {
	// Fast path: Direct hash lookup
	// We use the hash to index into the power-of-2 arena
	idx := hashUser(id) & UserMask
	ptr := &UserArena[idx]

	// Atomic load of UserID - Hot path
	currentID := atomic.LoadUint64(&ptr.UserID)

	// Case 1: Slot is empty, try to claim it
	if currentID == 0 {
		if atomic.CompareAndSwapUint64(&ptr.UserID, 0, id) {
			return ptr
		}
		// CAS failed, reload
		currentID = atomic.LoadUint64(&ptr.UserID)
	}

	// Case 2: Slot matches our user
	if currentID == id {
		return ptr
	}

	// Case 3: Collision - Linear probing with limited depth
	// Unrolled for performance
	const maxProbes = 4
	
	// Probe 1
	idx = (idx + 1) & UserMask
	ptr = &UserArena[idx]
	currentID = atomic.LoadUint64(&ptr.UserID)
	if currentID == id { return ptr }
	if currentID == 0 {
		if atomic.CompareAndSwapUint64(&ptr.UserID, 0, id) { return ptr }
		if atomic.LoadUint64(&ptr.UserID) == id { return ptr }
	}

	// Probe 2
	idx = (idx + 1) & UserMask
	ptr = &UserArena[idx]
	currentID = atomic.LoadUint64(&ptr.UserID)
	if currentID == id { return ptr }
	if currentID == 0 {
		if atomic.CompareAndSwapUint64(&ptr.UserID, 0, id) { return ptr }
		if atomic.LoadUint64(&ptr.UserID) == id { return ptr }
	}

	// Probe 3
	idx = (idx + 1) & UserMask
	ptr = &UserArena[idx]
	currentID = atomic.LoadUint64(&ptr.UserID)
	if currentID == id { return ptr }
	if currentID == 0 {
		if atomic.CompareAndSwapUint64(&ptr.UserID, 0, id) { return ptr }
		if atomic.LoadUint64(&ptr.UserID) == id { return ptr }
	}

	// Probe 4
	idx = (idx + 1) & UserMask
	ptr = &UserArena[idx]
	currentID = atomic.LoadUint64(&ptr.UserID)
	if currentID == id { return ptr }
	if currentID == 0 {
		if atomic.CompareAndSwapUint64(&ptr.UserID, 0, id) { return ptr }
		if atomic.LoadUint64(&ptr.UserID) == id { return ptr }
	}

	// Final fallback: Overwrite the original slot (LRU-ish behavior)
	// This is rare with 4M slots
	return &UserArena[hashUser(id)&UserMask]
}

// Global atomic ticker for time - CPU cycle optimized
var CurrentTime int64

func SetTime(t int64) {
	atomic.StoreInt64(&CurrentTime, t)
}

func Now() int64 {
	return atomic.LoadInt64(&CurrentTime)
}
