package cde

import (
	"discord-giveaway-bot/internal/engine/acl"
	"discord-giveaway-bot/internal/engine/fdl"
	"time"
)

// Bot user ID (set during initialization) - NEVER punish this user
var botUserID uint64

// ProcessEvent is the hot-path function called by the consumer
// CRITICAL: ZERO LOGGING IN THIS FUNCTION - EVERY NANOSECOND COUNTS
func ProcessEvent(evt fdl.FastEvent) {
	// ═══════════════════════════════════════════════════════════════════
	// CRITICAL SAFETY CHECKS - MUST BE FIRST
	// ═══════════════════════════════════════════════════════════════════

	// SAFETY 1: NEVER punish own bot (self-protection)
	if evt.UserID == botUserID && botUserID != 0 {
		return
	}

	// SAFETY 2: NEVER punish guild owner
	// Inline owner check for speed
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
	// END SAFETY CHECKS - PROCEED WITH DETECTION
	// ═══════════════════════════════════════════════════════════════════

	// Calculate detection speed AFTER safety checks (only for threats)
	detectionTime := time.Now().UnixNano() - evt.DetectionStart
	detectionSpeed := time.Duration(detectionTime)

	// 2. Get User State
	user := GetUser(evt.UserID)

	// 3. Evaluate Rules
	punish, pType := EvaluateRules(evt, user)

	// 4. Execute Punishment
	if punish {
		// Create Async Task (Logging handled by ACL worker, not hot path)
		task := acl.PunishTask{
			GuildID:       evt.GuildID,
			UserID:        evt.UserID,
			Type:          pType,
			Reason:        "Anti-Nuke Detection System - PANIC MODE",
			DetectionTime: detectionSpeed,
		}

		// Push to ACL Queue
		acl.PushPunish(task)

		// log.Printf("[CDE] ✓ Punishment task sent to ACL queue")

		// Reset State to avoid double punishing immediately?
		// Or keep it high to block further actions?
		// Usually we keep it high.
	}
}
