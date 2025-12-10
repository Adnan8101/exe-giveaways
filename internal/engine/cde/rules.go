package cde

import "discord-giveaway-bot/internal/engine/fdl"

// Rules constants - PANIC MODE: INSTANT DETECTION
const (
	ScoreThreshold    = 20 // Lowered from 100 - instant trigger
	MetricBan         = 30 // Instant ban
	MetricKick        = 30 // Instant ban
	MetricChanDel     = 30 // Instant ban
	MetricRoleDel     = 30 // Instant ban
	MetricChanCreate  = 30 // Instant ban on channel creation
	MetricRoleCreate  = 30 // Instant ban on role creation
	MetricChanUpdate  = 30 // Instant ban on channel update
	MetricRoleUpdate  = 30 // Instant ban on role update
	MetricGuildUpdate = 30 // Instant ban on guild update
	MetricWebhook     = 30 // Instant ban on webhook creation
)

// EvaluateRules checks the event against the user state and returns a punishment if needed
// Returns (ShouldPunish, PunishmentType)
func EvaluateRules(evt fdl.FastEvent, user *UserInfo) (bool, string) {
	now := Now()

	// Reset score if decay time passed (e.g. 5 seconds)
	if now-user.LastSeen > 5_000_000_000 {
		user.ThreatScore = 0
		user.BanCount = 0
		user.ChanDelCount = 0
		user.ChanCreateCount = 0
		user.RoleCreateCount = 0
		user.ChanUpdateCount = 0
		user.RoleUpdateCount = 0
		user.GuildUpdateCount = 0
		user.WebhookCount = 0
	}
	user.LastSeen = now

	weight := 0

	// Update Counts based on Event Type - ALL EVENTS TRACKED
	switch evt.ReqType {
	case fdl.EvtGuildBanAdd:
		user.BanCount++
		weight = MetricBan
	case fdl.EvtGuildMemberRemove: // Kick
		user.KickCount++
		weight = MetricKick
	case fdl.EvtChannelDelete:
		user.ChanDelCount++
		weight = MetricChanDel
	case fdl.EvtRoleDelete:
		user.RoleDelCount++
		weight = MetricRoleDel
	case fdl.EvtChannelCreate:
		user.ChanCreateCount++
		weight = MetricChanCreate
	case fdl.EvtRoleCreate:
		user.RoleCreateCount++
		weight = MetricRoleCreate
	case fdl.EvtChannelUpdate:
		user.ChanUpdateCount++
		weight = MetricChanUpdate
	case fdl.EvtRoleUpdate:
		user.RoleUpdateCount++
		weight = MetricRoleUpdate
	case fdl.EvtGuildUpdate:
		user.GuildUpdateCount++
		weight = MetricGuildUpdate
	case fdl.EvtWebhookCreate:
		user.WebhookCount++
		weight = MetricWebhook
	}

	// Accumulate Score
	user.ThreatScore += int32(weight)

	// Check Thresholds
	if user.ThreatScore > ScoreThreshold {
		return true, "BAN"
	}

	// Hard Limits (Instant Triggers) - PANIC MODE: ANY 1 ACTION = BAN
	if user.BanCount >= 1 {
		return true, "BAN"
	}
	if user.ChanDelCount >= 1 {
		return true, "BAN"
	}
	if user.RoleDelCount >= 1 {
		return true, "BAN"
	}
	if user.KickCount >= 1 {
		return true, "BAN"
	}
	if user.ChanCreateCount >= 1 {
		return true, "BAN"
	}
	if user.RoleCreateCount >= 1 {
		return true, "BAN"
	}
	if user.ChanUpdateCount >= 1 {
		return true, "BAN"
	}
	if user.RoleUpdateCount >= 1 {
		return true, "BAN"
	}
	if user.GuildUpdateCount >= 1 {
		return true, "BAN"
	}
	if user.WebhookCount >= 1 {
		return true, "BAN"
	}

	return false, ""
}
