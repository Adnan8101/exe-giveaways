package cde

import (
	"discord-giveaway-bot/internal/engine/acl"
	"discord-giveaway-bot/internal/engine/fdl"
	"time"
)

// Bot user ID (set during initialization) - NEVER punish this user
var botUserID uint64

// ProcessEvent is the ULTRA-OPTIMIZED hot-path function called by the consumer
// CRITICAL: ZERO LOGGING IN THIS FUNCTION - EVERY NANOSECOND COUNTS
// Target: Sub-microsecond detection time
func ProcessEvent(evt fdl.FastEvent) {
	// ═══════════════════════════════════════════════════════════════════
	// CRITICAL SAFETY CHECKS - MUST BE FIRST (Optimized for branch prediction)
	// ═══════════════════════════════════════════════════════════════════

	// SAFETY 1: NEVER punish own bot (self-protection) - Most common case first
	if evt.UserID == botUserID && botUserID != 0 {
		return
	}

	// SAFETY 2: NEVER punish guild owner (Inline check for speed)
	ownerID := GetGuildOwnerID(evt.GuildID)
	if evt.UserID == ownerID && ownerID != 0 {
		return
	}

	// SAFETY 3: Check if AntiNuke is enabled for this guild
	if !IsAntiNukeEnabled(evt.GuildID) {
		return
	}

	// SAFETY 4: Check Whitelist (applies to ALL users including bots)
	if IsUserWhitelisted(evt.GuildID, evt.UserID) {
		return
	}

	// NOTE: Other bots (not own bot) CAN be punished normally
	// Many attacks use malicious/compromised bots

	// ═══════════════════════════════════════════════════════════════════
	// END SAFETY CHECKS - PROCEED WITH ULTRA-FAST DETECTION
	// ═══════════════════════════════════════════════════════════════════

	// Calculate detection speed (only for threats that pass safety checks)
	detectionTime := time.Now().UnixNano() - evt.DetectionStart
	detectionSpeed := time.Duration(detectionTime)

	// Get User State with lockless algorithm
	user := GetUser(evt.UserID)

	// Evaluate Rules with table-driven zero-allocation approach
	punish, pType := EvaluateRules(evt, user)

	// Execute Punishment if needed
	if punish {
		// Increment detection counter
		fdl.EventsDetected.Inc(evt.UserID)
		fdl.PunishmentsIssued.Inc(evt.UserID)

		// Create Async Task (Logging handled by ACL worker, not hot path)
		task := acl.PunishTask{
			GuildID:        evt.GuildID,
			UserID:         evt.UserID,
			Type:           pType,
			Reason:         "🚨 Anti-Nuke ULTRA Detection - Instant Response",
			DetectionTime:  detectionSpeed,
			DetectionStart: time.Unix(0, evt.DetectionStart),
		}

		// Push to ACL Queue (Fast lane for bans)
		acl.PushPunish(task)

		// State management: Keep threat score high to block further actions
		// The decay window will reset it after the timeout period
	}
}
