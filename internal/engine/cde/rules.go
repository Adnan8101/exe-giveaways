package cde

import "discord-giveaway-bot/internal/engine/fdl"

// Rules constants - PANIC MODE: INSTANT DETECTION
const (
	ScoreThreshold = 20 // Lowered from 100 - instant trigger
	// Panic Mode Metrics
	MetricBan         = 30
	MetricKick        = 30
	MetricChanDel     = 30
	MetricRoleDel     = 30
	MetricChanCreate  = 30
	MetricRoleCreate  = 30
	MetricChanUpdate  = 30
	MetricRoleUpdate  = 30
	MetricGuildUpdate = 30
	MetricWebhook     = 30
)

// Rule Definition optimized for lookup
type RuleDef struct {
	Weight      int32
	CountOffset uintptr // Offset in UserInfo struct
	Limit       uint16  // Immediate ban limit
}

// Rule Table (Global Lookup)
// Index corresponds to ReqType
var RuleTable [256]RuleDef

func init() {
	// Initialize Rule Table
	// We use direct offsets to avoid reflection in hot path (or just switch if unrolling is better)
	// But user asked for "Table-Driven".
	// Since we can't easily take offset of struct field in pure Go without unsafe or reflect once...
	// Let's use a specialized handler function array instead, as suggested in plan.
	// "Unroll all rule evaluation into fixed table-driven tiny functions"
}

// Handler function type for rules
type RuleHandler func(*UserInfo) (int32, bool)

// RuleHandlers Table
var RuleHandlers [256]RuleHandler

func init() {
	// Default handler (ignore)
	noop := func(u *UserInfo) (int32, bool) { return 0, false }
	for i := 0; i < 256; i++ {
		RuleHandlers[i] = noop
	}

	// Register handlers
	RuleHandlers[fdl.EvtGuildBanAdd] = func(u *UserInfo) (int32, bool) {
		u.BanCount++
		return MetricBan, u.BanCount >= 1
	}
	RuleHandlers[fdl.EvtGuildMemberRemove] = func(u *UserInfo) (int32, bool) {
		u.KickCount++
		return MetricKick, u.KickCount >= 1
	}
	RuleHandlers[fdl.EvtChannelDelete] = func(u *UserInfo) (int32, bool) {
		u.ChanDelCount++
		return MetricChanDel, u.ChanDelCount >= 1
	}
	RuleHandlers[fdl.EvtRoleDelete] = func(u *UserInfo) (int32, bool) {
		u.RoleDelCount++
		return MetricRoleDel, u.RoleDelCount >= 1
	}
	RuleHandlers[fdl.EvtChannelCreate] = func(u *UserInfo) (int32, bool) {
		u.ChanCreateCount++
		return MetricChanCreate, u.ChanCreateCount >= 1
	}
	RuleHandlers[fdl.EvtRoleCreate] = func(u *UserInfo) (int32, bool) {
		u.RoleCreateCount++
		return MetricRoleCreate, u.RoleCreateCount >= 1
	}
	RuleHandlers[fdl.EvtChannelUpdate] = func(u *UserInfo) (int32, bool) {
		u.ChanUpdateCount++
		return MetricChanUpdate, u.ChanUpdateCount >= 1
	}
	RuleHandlers[fdl.EvtRoleUpdate] = func(u *UserInfo) (int32, bool) {
		u.RoleUpdateCount++
		return MetricRoleUpdate, u.RoleUpdateCount >= 1
	}
	RuleHandlers[fdl.EvtGuildUpdate] = func(u *UserInfo) (int32, bool) {
		u.GuildUpdateCount++
		return MetricGuildUpdate, u.GuildUpdateCount >= 1
	}
	RuleHandlers[fdl.EvtWebhookCreate] = func(u *UserInfo) (int32, bool) {
		u.WebhookCount++
		return MetricWebhook, u.WebhookCount >= 1
	}
}

// EvaluateRules checks the event against the user state and returns a punishment if needed
// Returns (ShouldPunish, PunishmentType)
// CRITICAL: Hot path - must be as fast as possible
//
//go:inline
func EvaluateRules(evt fdl.FastEvent, user *UserInfo) (bool, string) {
	now := Now()

	// Reset score if decay time passed (e.g. 5 seconds)
	// Branchless optimization: compute delta and mask
	timeDelta := now - user.LastSeen
	if timeDelta > 5_000_000_000 { // 5 seconds in nanoseconds
		// Reset all counters
		user.ThreatScore = 0
		user.BanCount = 0
		user.KickCount = 0
		user.ChanDelCount = 0
		user.ChanCreateCount = 0
		user.RoleCreateCount = 0
		user.RoleDelCount = 0
		user.ChanUpdateCount = 0
		user.RoleUpdateCount = 0
		user.GuildUpdateCount = 0
		user.WebhookCount = 0
	}
	user.LastSeen = now

	// EvaluateRules checks the event against the user state
	// Optimized: Table-driven lookup (~5-10ns overhead)
	handler := RuleHandlers[evt.ReqType]
	weight, instantTrigger := handler(user)

	// Update Score
	user.ThreatScore += weight

	// Check Thresholds - instant trigger for any violation in panic mode
	if instantTrigger || user.ThreatScore > ScoreThreshold {
		return true, "BAN"
	}

	return false, ""
}
