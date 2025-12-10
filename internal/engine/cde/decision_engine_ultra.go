package cde

import (
	"discord-giveaway-bot/internal/engine/fdl"
)

var punishQueueChan chan PunishTask

type PunishTask struct {
	GuildID       uint64
	UserID        uint64
	Type          string
	Reason        string
	DetectionTime int64
}

func InitPunishQueue(ch chan PunishTask) {
	punishQueueChan = ch
}

func ProcessEventUltra(evt fdl.FastEvent) {
	// CRITICAL PATH: Optimized for <3µs detection
	// Fast rejection checks first (most common path)

	// Skip bot's own actions (common case)
	if evt.UserID == botUserID {
		return
	}

	idx := hashGuild(evt.GuildID)
	guild := &GuildArena[idx]

	// Guild mismatch or disabled (uncommon)
	if guild.GuildID != evt.GuildID || (guild.Flags&1) == 0 {
		return
	}

	// Owner check (rare)
	if evt.UserID == guild.OwnerID {
		return
	}

	// Whitelist check (optimized bitset)
	h := hashUser(evt.UserID)
	if (guild.TrustedBitset[(h%256)/64] & (1 << ((h % 256) % 64))) != 0 {
		// Full whitelist verification (only if bit is set)
		for i := 0; i < 16; i++ {
			if guild.TrustedUsers[i] == evt.UserID {
				return
			}
		}
	}

	// Get user state (arena access - very fast)
	user := &UserArena[hashUser(evt.UserID)%MaxUsers]
	if user.UserID != evt.UserID {
		// Initialize new user (uncommon path)
		user.UserID = evt.UserID
		user.ThreatScore = 0
		user.BanCount = 0
		user.KickCount = 0
		user.ChanDelCount = 0
		user.RoleDelCount = 0
		user.ChanCreateCount = 0
		user.RoleCreateCount = 0
		user.ChanUpdateCount = 0
		user.RoleUpdateCount = 0
		user.GuildUpdateCount = 0
		user.WebhookCount = 0
		user.LastPunished = 0
		user.LastSeen = 0
	}

	now := Now()

	// Cooldown check (common for repeated offenders)
	if (now - user.LastPunished) < 60_000_000_000 {
		return
	}

	// State reset if inactive
	if (now - user.LastSeen) > 5_000_000_000 {
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

	// Apply rule handler (function pointer call)
	weight, instantTrigger := RuleHandlers[evt.ReqType](user)
	user.ThreatScore += weight

	// Ban decision (hot path)
	if instantTrigger || user.ThreatScore > ScoreThreshold {
		// Non-blocking push to punishment queue
		select {
		case punishQueueChan <- PunishTask{
			GuildID:       evt.GuildID,
			UserID:        evt.UserID,
			Type:          "BAN",
			Reason:        "AntiNuke",
			DetectionTime: now - evt.DetectionStart,
		}:
		default:
			// Queue full - drop (prevents blocking)
		}
	}
}
