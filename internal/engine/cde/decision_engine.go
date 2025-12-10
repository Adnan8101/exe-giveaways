package cde

import (
	"discord-giveaway-bot/internal/engine/acl"
	"discord-giveaway-bot/internal/engine/fdl"
	"log"
)

// ProcessEvent is the hot-path function called by the consumer
func ProcessEvent(evt fdl.FastEvent) {
	// DEBUG: Log all events being processed
	log.Printf("[CDE] Processing event: Type=%d, GuildID=%d, UserID=%d", evt.ReqType, evt.GuildID, evt.UserID)

	// 0. Check if AntiNuke is enabled for this guild (CRITICAL CHECK)
	if !IsAntiNukeEnabled(evt.GuildID) {
		log.Printf("[CDE] ⏭️  Skipping event: AntiNuke disabled for guild %d", evt.GuildID)
		return
	}

	// 1. Check Whitelist (CRITICAL CHECK)
	if IsUserWhitelisted(evt.GuildID, evt.UserID) {
		log.Printf("[CDE] ⏭️  Skipping event: User %d is whitelisted in guild %d", evt.UserID, evt.GuildID)
		return
	}

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
