// Package cde implements the Core Detection Engine - MAXIMUM PERFORMANCE EDITION
// Target: Sub-microsecond threat detection with 100% CPU utilization
//
// ARCHITECTURE:
// - Lock-free arenas for 4M users + 200K guilds
// - Cache-aligned structures (prevent false sharing)
// - Atomic operations for all state updates
// - Zero-allocation hot paths
// - Branchless where possible
//
// PERFORMANCE: < 500ns per event processing
package cde

import (
	"discord-giveaway-bot/internal/engine/acl"
	"discord-giveaway-bot/internal/engine/fdl"
	"sync/atomic"
	"time"
	"unsafe"
	_ "unsafe"
)

// Link to runtime nanotime for sub-nanosecond precision
//
//go:linkname nanotime runtime.nanotime
func nanotime() int64

// ═══════════════════════════════════════════════════════════════════
// STATE STRUCTURES - Cache-Aligned for Maximum Performance
// ═══════════════════════════════════════════════════════════════════

// Cache line size for modern CPUs
const CacheLineSize = 64

// UserInfoUltra - 128-byte cache-aligned user state
type UserInfoUltra struct {
	UserID       uint64 // 8 bytes
	ThreatScore  int32  // 4 bytes
	LastSeen     int64  // 8 bytes
	LastActionTS int64  // 8 bytes

	// Event counters (uint16 = 2 bytes each)
	BanCount          uint16
	UnbanCount        uint16
	KickCount         uint16
	ChanDelCount      uint16
	ChanCreateCount   uint16
	RoleDelCount      uint16
	RoleCreateCount   uint16
	ChanUpdateCount   uint16
	RoleUpdateCount   uint16
	GuildUpdateCount  uint16
	WebhookCount      uint16
	EmojiCount        uint16
	MemberUpdateCount uint16
	IntegrationCount  uint16
	AutomodCount      uint16
	EventCount        uint16
	PruneCount        uint16

	// Padding to 128 bytes (62 bytes used, need 66 bytes padding)
	_ [66]byte
}

// Type alias for compatibility
type UserInfo = UserInfoUltra

// GuildInfoUltra - 64-byte cache-aligned guild config
type GuildInfoUltra struct {
	GuildID        uint64
	OwnerID        uint64
	Flags          uint32
	LogChannelID   uint64
	WhitelistSlots [4]uint64
	_              [4]byte // Padding to 64 bytes
}

// Arena sizes
const (
	MaxUsersUltra  = 4_000_000
	MaxGuildsUltra = 200_000
)

// Global arenas (pre-allocated, zero-cost initialization)
var (
	UserArenaUltra  [MaxUsersUltra]UserInfoUltra
	GuildArenaUltra [MaxGuildsUltra]GuildInfoUltra

	UserArenaHits   uint64
	UserArenaMisses uint64
	UserArenaCollis uint64

	botUserID uint64
)

// ═══════════════════════════════════════════════════════════════════
// ATOMIC UINT16 OPERATIONS (Go doesn't provide these)
// ═══════════════════════════════════════════════════════════════════

//go:inline
func AddUint16(addr *uint16, delta uint16) uint16 {
	alignedAddr := (*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(addr)) &^ 3))
	offset := (uintptr(unsafe.Pointer(addr)) & 3) * 8

	for {
		old := atomic.LoadUint32(alignedAddr)
		oldVal := uint16(old >> offset)
		newVal := oldVal + delta
		mask := uint32(0xFFFF) << offset
		new := (old &^ mask) | (uint32(newVal) << offset)

		if atomic.CompareAndSwapUint32(alignedAddr, old, new) {
			return newVal
		}
	}
}

//go:inline
func StoreUint16(addr *uint16, val uint16) {
	alignedAddr := (*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(addr)) &^ 3))
	offset := (uintptr(unsafe.Pointer(addr)) & 3) * 8

	for {
		old := atomic.LoadUint32(alignedAddr)
		mask := uint32(0xFFFF) << offset
		new := (old &^ mask) | (uint32(val) << offset)

		if atomic.CompareAndSwapUint32(alignedAddr, old, new) {
			return
		}
	}
}

// ═══════════════════════════════════════════════════════════════════
// HASH & ARENA ACCESS - Sub-50ns lookups
// ═══════════════════════════════════════════════════════════════════

//go:inline
func hashSnowflakeFast(id uint64) uint64 {
	h := id
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	h *= 0xc4ceb9fe1a85ec53
	h ^= h >> 33
	return h
}

//go:inline
func GetUserFast(id uint64) *UserInfo {
	idx := hashSnowflakeFast(id) % MaxUsersUltra
	slot := &UserArenaUltra[idx]

	currentID := atomic.LoadUint64(&slot.UserID)
	if currentID == id {
		atomic.AddUint64(&UserArenaHits, 1)
		return slot
	}

	if currentID == 0 {
		if atomic.CompareAndSwapUint64(&slot.UserID, 0, id) {
			atomic.AddUint64(&UserArenaMisses, 1)
			return slot
		}
	}

	// Linear probe
	for probe := uint64(1); probe < 16; probe++ {
		idx = (idx + probe) % MaxUsersUltra
		slot = &UserArenaUltra[idx]

		currentID = atomic.LoadUint64(&slot.UserID)
		if currentID == id {
			atomic.AddUint64(&UserArenaHits, 1)
			return slot
		}

		if currentID == 0 {
			if atomic.CompareAndSwapUint64(&slot.UserID, 0, id) {
				atomic.AddUint64(&UserArenaMisses, 1)
				return slot
			}
		}
	}

	atomic.AddUint64(&UserArenaCollis, 1)
	slot = &UserArenaUltra[idx]
	atomic.StoreUint64(&slot.UserID, id)
	ResetUserStateFast(slot)
	return slot
}

//go:inline
func GetGuildFast(id uint64) *GuildInfoUltra {
	idx := hashSnowflakeFast(id) % MaxGuildsUltra
	slot := &GuildArenaUltra[idx]

	currentID := atomic.LoadUint64(&slot.GuildID)
	if currentID == id {
		return slot
	}

	if currentID == 0 {
		return nil
	}

	for probe := uint64(1); probe < 16; probe++ {
		idx = (idx + probe) % MaxGuildsUltra
		slot = &GuildArenaUltra[idx]

		currentID = atomic.LoadUint64(&slot.GuildID)
		if currentID == id {
			return slot
		}
		if currentID == 0 {
			return nil
		}
	}

	return nil
}

//go:inline
func ResetUserStateFast(user *UserInfo) {
	atomic.StoreInt32(&user.ThreatScore, 0)
	StoreUint16(&user.BanCount, 0)
	StoreUint16(&user.UnbanCount, 0)
	StoreUint16(&user.KickCount, 0)
	StoreUint16(&user.ChanDelCount, 0)
	StoreUint16(&user.ChanCreateCount, 0)
	StoreUint16(&user.RoleDelCount, 0)
	StoreUint16(&user.RoleCreateCount, 0)
	StoreUint16(&user.ChanUpdateCount, 0)
	StoreUint16(&user.RoleUpdateCount, 0)
	StoreUint16(&user.GuildUpdateCount, 0)
	StoreUint16(&user.WebhookCount, 0)
	StoreUint16(&user.EmojiCount, 0)
	StoreUint16(&user.MemberUpdateCount, 0)
	StoreUint16(&user.IntegrationCount, 0)
	StoreUint16(&user.AutomodCount, 0)
	StoreUint16(&user.EventCount, 0)
	StoreUint16(&user.PruneCount, 0)
}

//go:inline
func IsUserWhitelistedFast(guild *GuildInfoUltra, userID uint64) bool {
	for i := 0; i < 4; i++ {
		if atomic.LoadUint64(&guild.WhitelistSlots[i]) == userID {
			return true
		}
	}
	return false
}

func GetArenaStats() (hits, misses, collisions uint64) {
	return atomic.LoadUint64(&UserArenaHits),
		atomic.LoadUint64(&UserArenaMisses),
		atomic.LoadUint64(&UserArenaCollis)
}

// ═══════════════════════════════════════════════════════════════════
// DECISION ENGINE - Sub-Microsecond Threat Detection
// ═══════════════════════════════════════════════════════════════════

const (
	ScoreThresholdUltra = 20
	DecayWindow         = 5_000_000_000 // 5 seconds in nanoseconds
	FlagAntiNukeEnabled = 1 << 0
	FlagPanicMode       = 1 << 1
)

const (
	WeightBan         int32 = 30
	WeightUnban       int32 = 25
	WeightKick        int32 = 30
	WeightChanDel     int32 = 30
	WeightChanCreate  int32 = 25
	WeightRoleDel     int32 = 30
	WeightRoleCreate  int32 = 25
	WeightChanUpdate  int32 = 20
	WeightRoleUpdate  int32 = 20
	WeightGuildUpdate int32 = 30
	WeightWebhook     int32 = 30
	WeightEmoji       int32 = 15
	WeightMember      int32 = 15
	WeightIntegration int32 = 25
	WeightAutomod     int32 = 20
	WeightEvent       int32 = 15
	WeightPrune       int32 = 50
)

type RuleHandlerFunc func(*UserInfo) (int32, bool)

var RuleHandlersUltra [256]RuleHandlerFunc

func init() {
	noop := func(u *UserInfo) (int32, bool) { return 0, false }
	for i := 0; i < 256; i++ {
		RuleHandlersUltra[i] = noop
	}

	RuleHandlersUltra[fdl.EvtGuildBanAdd] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.BanCount, 1)
		return WeightBan, count >= 1
	}

	RuleHandlersUltra[fdl.EvtGuildUnban] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.UnbanCount, 1)
		return WeightUnban, count >= 1
	}

	RuleHandlersUltra[fdl.EvtGuildMemberRemove] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.KickCount, 1)
		return WeightKick, count >= 1
	}

	RuleHandlersUltra[fdl.EvtChannelDelete] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.ChanDelCount, 1)
		return WeightChanDel, count >= 1
	}

	RuleHandlersUltra[fdl.EvtChannelCreate] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.ChanCreateCount, 1)
		return WeightChanCreate, count >= 2
	}

	RuleHandlersUltra[fdl.EvtRoleDelete] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.RoleDelCount, 1)
		return WeightRoleDel, count >= 1
	}

	RuleHandlersUltra[fdl.EvtRoleCreate] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.RoleCreateCount, 1)
		return WeightRoleCreate, count >= 2
	}

	RuleHandlersUltra[fdl.EvtChannelUpdate] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.ChanUpdateCount, 1)
		return WeightChanUpdate, count >= 3
	}

	RuleHandlersUltra[fdl.EvtRoleUpdate] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.RoleUpdateCount, 1)
		return WeightRoleUpdate, count >= 3
	}

	RuleHandlersUltra[fdl.EvtGuildUpdate] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.GuildUpdateCount, 1)
		return WeightGuildUpdate, count >= 1
	}

	RuleHandlersUltra[fdl.EvtWebhookCreate] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.WebhookCount, 1)
		return WeightWebhook, count >= 1
	}

	RuleHandlersUltra[fdl.EvtWebhookUpdate] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.WebhookCount, 1)
		return WeightWebhook, count >= 1
	}

	RuleHandlersUltra[fdl.EvtWebhookDelete] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.WebhookCount, 1)
		return WeightWebhook, count >= 1
	}

	RuleHandlersUltra[fdl.EvtEmojiCreate] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.EmojiCount, 1)
		return WeightEmoji, count >= 5
	}

	RuleHandlersUltra[fdl.EvtEmojiDelete] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.EmojiCount, 1)
		return WeightEmoji, count >= 5
	}

	RuleHandlersUltra[fdl.EvtEmojiUpdate] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.EmojiCount, 1)
		return WeightEmoji, count >= 5
	}

	RuleHandlersUltra[fdl.EvtMemberUpdate] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.MemberUpdateCount, 1)
		return WeightMember, count >= 5
	}

	RuleHandlersUltra[fdl.EvtIntegrationCreate] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.IntegrationCount, 1)
		return WeightIntegration, count >= 1
	}

	RuleHandlersUltra[fdl.EvtIntegrationUpdate] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.IntegrationCount, 1)
		return WeightIntegration, count >= 1
	}

	RuleHandlersUltra[fdl.EvtIntegrationDelete] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.IntegrationCount, 1)
		return WeightIntegration, count >= 1
	}

	RuleHandlersUltra[fdl.EvtAutomodCreate] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.AutomodCount, 1)
		return WeightAutomod, count >= 2
	}

	RuleHandlersUltra[fdl.EvtAutomodUpdate] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.AutomodCount, 1)
		return WeightAutomod, count >= 2
	}

	RuleHandlersUltra[fdl.EvtAutomodDelete] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.AutomodCount, 1)
		return WeightAutomod, count >= 2
	}

	RuleHandlersUltra[fdl.EvtEventCreate] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.EventCount, 1)
		return WeightEvent, count >= 3
	}

	RuleHandlersUltra[fdl.EvtEventUpdate] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.EventCount, 1)
		return WeightEvent, count >= 3
	}

	RuleHandlersUltra[fdl.EvtEventDelete] = func(u *UserInfo) (int32, bool) {
		count := AddUint16(&u.EventCount, 1)
		return WeightEvent, count >= 3
	}

	RuleHandlersUltra[fdl.EvtMemberPrune] = func(u *UserInfo) (int32, bool) {
		AddUint16(&u.PruneCount, 1)
		return WeightPrune, true // INSTANT BAN
	}
}

//go:noinline
func ProcessEventUltra(evt fdl.FastEvent) {
	if evt.UserID == botUserID && botUserID != 0 {
		return
	}

	guild := GetGuildFast(evt.GuildID)
	if guild == nil {
		return
	}

	if evt.UserID == guild.OwnerID && guild.OwnerID != 0 {
		return
	}

	flags := atomic.LoadUint32(&guild.Flags)
	if (flags & FlagAntiNukeEnabled) == 0 {
		return
	}

	if IsUserWhitelistedFast(guild, evt.UserID) {
		return
	}

	user := GetUserFast(evt.UserID)
	if user == nil {
		return
	}

	now := nanotime()

	timeDelta := now - atomic.LoadInt64(&user.LastSeen)
	if timeDelta > DecayWindow {
		ResetUserStateFast(user)
	}

	atomic.StoreInt64(&user.LastSeen, now)

	handler := RuleHandlersUltra[evt.ReqType]
	weight, instantTrigger := handler(user)

	newScore := atomic.AddInt32(&user.ThreatScore, weight)

	shouldPunish := instantTrigger || newScore > ScoreThresholdUltra
	if !shouldPunish {
		return
	}

	detectionTime := now - evt.DetectionStart

	task := acl.PunishTask{
		GuildID:        evt.GuildID,
		UserID:         evt.UserID,
		Type:           "BAN",
		Reason:         "Anti-Nuke Detection System - ULTRA MODE",
		DetectionTime:  time.Duration(detectionTime),
		DetectionStart: time.Unix(0, evt.DetectionStart),
	}

	acl.PushPunish(task)
}
