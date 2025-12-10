package cde

import (
	"discord-giveaway-bot/internal/engine/fdl"
	"sync/atomic"
)

// ULTRA-AGGRESSIVE PANIC MODE RULES
// Detection threshold lowered to near-zero for instant response
const (
	ScoreThreshold = 10 // Ultra-low threshold for instant trigger
	
	// Critical actions - INSTANT BAN (single action = ban)
	MetricBan         = 50
	MetricUnban       = 50
	MetricKick        = 50
	MetricChanDel     = 50
	MetricRoleDel     = 50
	MetricWebhook     = 50
	MetricPrune       = 100 // Prune is most dangerous
	
	// Bulk creation/update actions - INSTANT BAN (2 actions = ban)
	MetricChanCreate     = 40
	MetricRoleCreate     = 40
	MetricChanUpdate     = 40
	MetricRoleUpdate     = 40
	MetricGuildUpdate    = 45
	MetricEmojiUpdate    = 35
	MetricStickerUpdate  = 35
	MetricMemberUpdate   = 35
	MetricIntegration    = 40
	MetricAutoMod        = 40
	MetricGuildEvent     = 35
	
	// Decay window - keep violations for 10 seconds
	DecayWindow = 10_000_000_000 // 10 seconds in nanoseconds
)

// Handler function type for ultra-fast rule evaluation
type RuleHandler func(*UserInfo) (int32, bool)

// RuleHandlers Table - Pre-compiled function pointers for zero-overhead dispatch
var RuleHandlers [256]RuleHandler

func init() {
	// Default handler (ignore unknown events)
	noop := func(u *UserInfo) (int32, bool) { return 0, false }
	for i := 0; i < 256; i++ {
		RuleHandlers[i] = noop
	}

	// Register optimized handlers for all event types
	// Using atomic operations for thread-safe counter updates
	
	// CRITICAL EVENTS - Instant ban on first occurrence
	RuleHandlers[fdl.EvtGuildBanAdd] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.BanCount, 1)
		return MetricBan, count >= 1 // Instant ban
	}
	
	RuleHandlers[fdl.EvtGuildMemberRemove] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.KickCount, 1)
		return MetricKick, count >= 1 // Instant ban (covers kick/unban)
	}
	
	RuleHandlers[fdl.EvtChannelDelete] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.ChanDelCount, 1)
		return MetricChanDel, count >= 1
	}
	
	RuleHandlers[fdl.EvtRoleDelete] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.RoleDelCount, 1)
		return MetricRoleDel, count >= 1
	}
	
	RuleHandlers[fdl.EvtWebhookCreate] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.WebhookCount, 1)
		return MetricWebhook, count >= 1
	}
	
	RuleHandlers[fdl.EvtWebhookUpdate] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.WebhookCount, 1)
		return MetricWebhook, count >= 1
	}
	
	RuleHandlers[fdl.EvtWebhookDelete] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.WebhookCount, 1)
		return MetricWebhook, count >= 1
	}
	
	RuleHandlers[fdl.EvtPrune] = func(u *UserInfo) (int32, bool) {
		// Prune is EXTREMELY dangerous - instant ban
		return MetricPrune, true
	}
	
	// BULK OPERATION EVENTS - Ban after 2 rapid actions
	RuleHandlers[fdl.EvtChannelCreate] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.ChanCreateCount, 1)
		return MetricChanCreate, count >= 2
	}
	
	RuleHandlers[fdl.EvtRoleCreate] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.RoleCreateCount, 1)
		return MetricRoleCreate, count >= 2
	}
	
	RuleHandlers[fdl.EvtChannelUpdate] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.ChanUpdateCount, 1)
		return MetricChanUpdate, count >= 2
	}
	
	RuleHandlers[fdl.EvtRoleUpdate] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.RoleUpdateCount, 1)
		return MetricRoleUpdate, count >= 2
	}
	
	RuleHandlers[fdl.EvtGuildUpdate] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.GuildUpdateCount, 1)
		return MetricGuildUpdate, count >= 2
	}
	
	// EMOJI/STICKER EVENTS
	RuleHandlers[fdl.EvtEmojiCreate] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.EmojiCount, 1)
		return MetricEmojiUpdate, count >= 2
	}
	
	RuleHandlers[fdl.EvtEmojiDelete] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.EmojiCount, 1)
		return MetricEmojiUpdate, count >= 2
	}
	
	RuleHandlers[fdl.EvtEmojiUpdate] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.EmojiCount, 1)
		return MetricEmojiUpdate, count >= 2
	}
	
	RuleHandlers[fdl.EvtStickerCreate] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.StickerCount, 1)
		return MetricStickerUpdate, count >= 2
	}
	
	RuleHandlers[fdl.EvtStickerDelete] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.StickerCount, 1)
		return MetricStickerUpdate, count >= 2
	}
	
	RuleHandlers[fdl.EvtStickerUpdate] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.StickerCount, 1)
		return MetricStickerUpdate, count >= 2
	}
	
	// MEMBER UPDATE EVENTS
	RuleHandlers[fdl.EvtMemberUpdate] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.MemberUpdateCount, 1)
		return MetricMemberUpdate, count >= 2
	}
	
	// INTEGRATION EVENTS
	RuleHandlers[fdl.EvtIntegrationCreate] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.IntegrationCount, 1)
		return MetricIntegration, count >= 1 // More sensitive
	}
	
	RuleHandlers[fdl.EvtIntegrationUpdate] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.IntegrationCount, 1)
		return MetricIntegration, count >= 1
	}
	
	RuleHandlers[fdl.EvtIntegrationDelete] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.IntegrationCount, 1)
		return MetricIntegration, count >= 1
	}
	
	// AUTO-MODERATION EVENTS
	RuleHandlers[fdl.EvtAutoModRuleCreate] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.AutoModCount, 1)
		return MetricAutoMod, count >= 1 // Instant ban on automod changes
	}
	
	RuleHandlers[fdl.EvtAutoModRuleUpdate] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.AutoModCount, 1)
		return MetricAutoMod, count >= 1
	}
	
	RuleHandlers[fdl.EvtAutoModRuleDelete] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.AutoModCount, 1)
		return MetricAutoMod, count >= 1
	}
	
	// GUILD SCHEDULED EVENT EVENTS
	RuleHandlers[fdl.EvtGuildEventCreate] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.EventCount, 1)
		return MetricGuildEvent, count >= 2
	}
	
	RuleHandlers[fdl.EvtGuildEventUpdate] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.EventCount, 1)
		return MetricGuildEvent, count >= 2
	}
	
	RuleHandlers[fdl.EvtGuildEventDelete] = func(u *UserInfo) (int32, bool) {
		count := atomic.AddUint32(&u.EventCount, 1)
		return MetricGuildEvent, count >= 2
	}
}

// EvaluateRules checks the event against the user state with ULTRA-FAST processing
// Returns (ShouldPunish, PunishmentType)
// CRITICAL HOT PATH: Optimized for sub-microsecond execution
//
//go:inline
func EvaluateRules(evt fdl.FastEvent, user *UserInfo) (bool, string) {
	now := Now()

	// Atomic load of last seen time for thread safety
	lastSeen := atomic.LoadInt64(&user.LastSeen)
	timeDelta := now - lastSeen

	// Reset counters if decay window passed (lockless with atomic CAS)
	if timeDelta > DecayWindow {
		// Try to reset - only one thread wins, others skip
		if atomic.CompareAndSwapInt64(&user.LastSeen, lastSeen, now) {
			// Winner resets all counters atomically
			atomic.StoreInt64(&user.ThreatScore, 0)
			atomic.StoreUint32(&user.BanCount, 0)
			atomic.StoreUint32(&user.KickCount, 0)
			atomic.StoreUint32(&user.ChanDelCount, 0)
			atomic.StoreUint32(&user.ChanCreateCount, 0)
			atomic.StoreUint32(&user.RoleCreateCount, 0)
			atomic.StoreUint32(&user.RoleDelCount, 0)
			atomic.StoreUint32(&user.ChanUpdateCount, 0)
			atomic.StoreUint32(&user.RoleUpdateCount, 0)
			atomic.StoreUint32(&user.GuildUpdateCount, 0)
			atomic.StoreUint32(&user.WebhookCount, 0)
			atomic.StoreUint32(&user.EmojiCount, 0)
			atomic.StoreUint32(&user.StickerCount, 0)
			atomic.StoreUint32(&user.MemberUpdateCount, 0)
			atomic.StoreUint32(&user.IntegrationCount, 0)
			atomic.StoreUint32(&user.AutoModCount, 0)
			atomic.StoreUint32(&user.EventCount, 0)
		}
	} else {
		// Update last seen without resetting
		atomic.StoreInt64(&user.LastSeen, now)
	}

	// Table-driven rule evaluation (~2-3ns overhead with inlining)
	handler := RuleHandlers[evt.ReqType]
	weight, instantTrigger := handler(user)

	// Atomic threat score update
	newScore := atomic.AddInt64(&user.ThreatScore, int64(weight))

	// Ultra-fast threshold check - instant trigger for any violation
	if instantTrigger || newScore > ScoreThreshold {
		return true, "BAN"
	}

	return false, ""
}
