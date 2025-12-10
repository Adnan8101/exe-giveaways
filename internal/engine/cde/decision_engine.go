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

	// 1. Get State
	user := GetUser(evt.UserID)

	// TODO: Check Whitelist from GuildArena
	// if IsWhitelisted(evt.GuildID, evt.UserID) { return }

	// 2. Evaluate
	punish, pType := EvaluateRules(evt, user)

	// DEBUG: Log evaluation result
	log.Printf("[CDE] Evaluation: UserID=%d, ThreatScore=%d, Punish=%v, Type=%s",
		evt.UserID, user.ThreatScore, punish, pType)

	// 3. Action
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
