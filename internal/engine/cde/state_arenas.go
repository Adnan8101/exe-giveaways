package cde

import "sync/atomic"

const (
	MaxUsers  = 2_000_000
	MaxGuilds = 100_000
)

// UserInfo represents the state of a user for detection
// It must be padded to cache line size if accessed frequently by different threads,
// but since CDE is partitioned, packing is better for cache density.
// Size: ~64 bytes
type UserInfo struct {
	UserID       uint64
	BanCount     uint16 // Rolling window count
	KickCount    uint16
	ChanDelCount uint16
	RoleDelCount uint16

	// Windows: Simple timestamps for the last N actions for complex rules
	LastActionTS int64

	// Score
	ThreatScore int32

	// Expiration
	LastSeen int64
}

// GuildInfo represents guild config and state
type GuildInfo struct {
	GuildID      uint64
	ConfigBitmap uint64 // Flags for enabled features
	OwnerID      uint64

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
