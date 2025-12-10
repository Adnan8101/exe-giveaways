package cde

import (
	"discord-giveaway-bot/internal/engine/acl"
	"discord-giveaway-bot/internal/engine/fdl"
	"log"
)

// Bot user ID (set during initialization) - NEVER punish this user
var botUserID uint64

// ProcessEvent is the hot-path function called by the consumer
func ProcessEvent(evt fdl.FastEvent) {
	// DEBUG: Log all events being processed
	log.Printf("[CDE] Processing event: Type=%d, GuildID=%d, UserID=%d", evt.ReqType, evt.GuildID, evt.UserID)

	// ═══════════════════════════════════════════════════════════════════
	// CRITICAL SAFETY CHECKS - MUST BE FIRST
	// ═══════════════════════════════════════════════════════════════════

	// SAFETY 1: NEVER punish own bot (self-protection)
	if evt.UserID == botUserID && botUserID != 0 {
		log.Printf("[CDE] 🛡️  SKIPPED: Own bot action (self-protection) - UserID=%d", evt.UserID)
		return
	}

	// SAFETY 2: NEVER punish guild owner
	ownerID := GetGuildOwnerID(evt.GuildID)
	if evt.UserID == ownerID && ownerID != 0 {
		log.Printf("[CDE] 👑 SKIPPED: Guild owner action - UserID=%d", evt.UserID)
		return
	}

	// SAFETY 3: Check if AntiNuke is enabled for this guild
	if !IsAntiNukeEnabled(evt.GuildID) {
		log.Printf("[CDE] ⏭️  SKIPPED: AntiNuke disabled for guild %d", evt.GuildID)
		return
	}

	// SAFETY 4: Check Whitelist (applies to ALL users including bots)
	if IsUserWhitelisted(evt.GuildID, evt.UserID) {
		log.Printf("[CDE] ⏭️  SKIPPED: User %d whitelisted in guild %d", evt.UserID, evt.GuildID)
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

	// DEBUG: Log evaluation result
	log.Printf("[CDE] Evaluation: UserID=%d, ThreatScore=%d, Punish=%v, Type=%s",
		evt.UserID, user.ThreatScore, punish, pType)

	// 4. Execute Punishment
	if punish {
		log.Printf("[CDE] 🚨 TRIGGERING PUNISHMENT: UserID=%d, Type=%s, Reason=Anti-Nuke",
			evt.UserID, pType)

		// Create Async Task
		task := acl.PunishTask{
			GuildID: evt.GuildID,
			UserID:  evt.UserID,
			Type:    pType,
			Reason:  "Anti-Nuke Detection System - PANIC MODE",
		}

		// Push to ACL Queue
		acl.PushPunish(task)

		log.Printf("[CDE] ✓ Punishment task sent to ACL queue")

		// Reset State to avoid double punishing immediately?
		// Or keep it high to block further actions?
		// Usually we keep it high.
	}
}
