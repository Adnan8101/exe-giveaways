package cde

import (
	"discord-giveaway-bot/internal/engine/acl"
	"discord-giveaway-bot/internal/engine/fdl"
)

// ProcessEvent is the hot-path function called by the consumer
func ProcessEvent(evt fdl.FastEvent) {
	// 1. Get State
	user := GetUser(evt.UserID)

	// TODO: Check Whitelist from GuildArena
	// if IsWhitelisted(evt.GuildID, evt.UserID) { return }

	// 2. Evaluate
	punish, pType := EvaluateRules(evt, user)

	// 3. Action
	if punish {
		// Create Async Task
		task := acl.PunishTask{
			GuildID: evt.GuildID,
			UserID:  evt.UserID,
			Type:    pType,
			Reason:  "Anti-Nuke Detection System",
		}

		// Push to ACL Queue
		acl.PushPunish(task)

		// Reset State to avoid double punishing immediately?
		// Or keep it high to block further actions?
		// Usually we keep it high.
	}
}
