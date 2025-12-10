package cde

import "discord-giveaway-bot/internal/engine/fdl"

// Rules constants - PANIC MODE: INSTANT DETECTION
const (
	ScoreThreshold = 20 // Lowered from 100 - instant trigger
	MetricBan      = 25 // Higher weight for bans
	MetricKick     = 15 // Higher weight for kicks
	MetricChanDel  = 30 // 1 channel delete = instant ban
	MetricRoleDel  = 30 // 1 role delete = instant ban
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
	}
	user.LastSeen = now

	weight := 0

	// Update Counts based on Event Type
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
	}

	// Accumulate Score
	user.ThreatScore += int32(weight)

	// Check Thresholds
	if user.ThreatScore > ScoreThreshold {
		return true, "BAN"
	}

	// Hard Limits (Instant Triggers) - PANIC MODE
	if user.BanCount >= 1 {
		return true, "BAN" // 1 ban = instant trigger
	}
	if user.ChanDelCount >= 1 {
		return true, "BAN" // 1 channel delete = instant trigger
	}
	if user.RoleDelCount >= 1 {
		return true, "BAN" // 1 role delete = instant trigger
	}
	if user.KickCount >= 1 {
		return true, "BAN" // 1 kick = instant trigger
	}

	return false, ""
}
