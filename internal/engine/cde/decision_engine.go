package cde

import (
	"discord-giveaway-bot/internal/engine/acl"
	"discord-giveaway-bot/internal/engine/fdl"
	"log"
	"time"
)

// Bot user ID (set during initialization) - NEVER punish this user
var botUserID uint64

// ProcessEvent is the hot-path function called by the consumer
func ProcessEvent(evt fdl.FastEvent) {
	// DEBUG: Log removal for speed
	// log.Printf("[CDE] Processing event: Type=%d, GuildID=%d, UserID=%d", evt.ReqType, evt.GuildID, evt.UserID)

	// Calculate detection speed (time from event start to processing)
	detectionTime := time.Now().UnixNano() - evt.DetectionStart
	detectionSpeed := time.Duration(detectionTime)

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

	// 2. Get User State
	user := GetUser(evt.UserID)

	// 3. Evaluate Rules
	punish, pType := EvaluateRules(evt, user)

	// DEBUG: Log evaluation result with detection speed
	log.Printf("[CDE] Evaluation: UserID=%d, ThreatScore=%d, Punish=%v, Type=%s, DetectionSpeed=%v",
		evt.UserID, user.ThreatScore, punish, pType, detectionSpeed)

	// 4. Execute Punishment
	if punish {
		// Log only on punishment (rare event compared to normal traffic) but maybe async?
		// For 80ns target, even this log is too slow.
		// We will rely on the Punishment Task to log.

		// Create Async Task

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
