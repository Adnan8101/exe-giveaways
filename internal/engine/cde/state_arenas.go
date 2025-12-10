package cde

import (
	"sync/atomic"
)

const (
	MaxUsers  = 2_000_000
	MaxGuilds = 100_000
)

// UserInfo represents the state of a user for detection
// Padded to cache line (64 bytes) to prevent false sharing
type UserInfo struct {
	UserID       uint64 // 8 bytes
	BanCount     uint16 // 2 bytes
	KickCount    uint16 // 2 bytes
	ChanDelCount uint16 // 2 bytes
	RoleDelCount uint16 // 2 bytes

	// Additional event counters for all tracked events
	ChanCreateCount  uint16 // 2 bytes
	RoleCreateCount  uint16 // 2 bytes
	ChanUpdateCount  uint16 // 2 bytes
	RoleUpdateCount  uint16 // 2 bytes
	GuildUpdateCount uint16 // 2 bytes
	WebhookCount     uint16 // 2 bytes

	// Windows: Simple timestamps for the last N actions
	LastActionTS int64 // 8 bytes

	// Score
	ThreatScore int32 // 4 bytes

	// Expiration
	LastSeen int64 // 8 bytes

	// Padding to 64 bytes (current: 52 bytes, need 12 more)
	_ [12]byte
}

// GuildInfo represents guild config and state
// Optimized for atomic access and cache locality
type GuildInfo struct {
	GuildID      uint64
	ConfigBitmap uint64 // Flags for enabled features
	OwnerID      uint64

	// Atomic flags
	// Bit 0: AntiNukeEnabled
	// Bit 1: PanicMode
	Flags uint32

	LogChannelID uint64

	// Atomic Whitelist Bitset (256 bits)
	TrustedBitset [4]uint64

	// Whitelists (IDs hashed into bloom filter or fixed array for O(1)?)
	// For simplicity in hot path, we might just store a small fixed array of trusted IDs
	// or point to a read-only structure.
	TrustedUsers [16]uint64
}

// Arenas (Global State)
// Allocated in BSS segment usually
var (
	UserArena  [MaxUsers]UserInfo
	GuildArena [MaxGuilds]GuildInfo
)

// Hasher for mapping IDs to Arena indices
// Simple modulo for now, but should ideally be XOR-shift or WyHash
func hashUser(id uint64) uint64 {
	// Simple mixing
	x := id
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	x ^= x >> 31
	return x
}

func GetUser(id uint64) *UserInfo {
	idx := hashUser(id) % MaxUsers
	// Collision handling: Linear probing could go here
	// For MVP we just return the slot.
	// If UserID doesn't match, we overwrite (LRU style replacement essentially)
	ptr := &UserArena[idx]
	if ptr.UserID == 0 {
		ptr.UserID = id
	}
	// If ptr.UserID != id -> Collision.
	// In a real system we'd probe next slot.
	// Spec says "Mask & Probe".
	if ptr.UserID != id {
		// Simple probe
		idx = (idx + 1) % MaxUsers
		ptr = &UserArena[idx]
		if ptr.UserID == 0 {
			ptr.UserID = id
		}
	}
	return ptr
}

// Global atomic ticker for time
var CurrentTime int64

func SetTime(t int64) {
	atomic.StoreInt64(&CurrentTime, t)
}

func Now() int64 {
	return atomic.LoadInt64(&CurrentTime)
}
