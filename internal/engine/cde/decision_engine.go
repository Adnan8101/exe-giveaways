package cde

import (
	"discord-giveaway-bot/internal/engine/acl"
	"discord-giveaway-bot/internal/engine/fdl"
	"sync/atomic"
	"time"
)

// Bot user ID (set during initialization) - NEVER punish this user
var botUserID uint64

// ProcessEvent is the ULTRA-OPTIMIZED hot-path function called by the consumer
// CRITICAL: ZERO LOGGING IN THIS FUNCTION - EVERY NANOSECOND COUNTS
// Target: Sub-microsecond detection time
//
//go:inline
func ProcessEvent(evt fdl.FastEvent) {
	// ═══════════════════════════════════════════════════════════════════
	// CRITICAL SAFETY CHECKS - MUST BE FIRST (Optimized for branch prediction)
	// ═══════════════════════════════════════════════════════════════════

	// SAFETY 1: NEVER punish own bot (self-protection) - Most common case first
	if evt.UserID == botUserID && botUserID != 0 {
		return
	}

	// SAFETY 2: Check if AntiNuke is enabled for this guild
	// Inlined IsAntiNukeEnabled for speed
	idx := hashGuild(evt.GuildID)
	guild := &GuildArena[idx]
	
	// Check ID match (atomic load)
	if atomic.LoadUint64(&guild.GuildID) != evt.GuildID {
		return // Cache miss or disabled
	}

	// Check enabled flag
	if (atomic.LoadUint32(&guild.Flags) & 1) == 0 {
		return
	}

	// SAFETY 3: NEVER punish guild owner (Inline check for speed)
	if evt.UserID == guild.OwnerID {
		return
	}

	// SAFETY 4: Check Whitelist (applies to ALL users including bots)
	// Inlined IsUserWhitelisted
	// Check TrustedUsers array (linear scan of 16 items is faster than map)
	for i := 0; i < 16; i++ {
		if guild.TrustedUsers[i] == evt.UserID {
			return
		}
		if guild.TrustedUsers[i] == 0 {
			break // End of list
		}
	}
	
	// Check Bitset
	// Hash/Map to 0-511
	bitIdx := hashUser(evt.UserID) & 511
	wordIdx := bitIdx >> 6 // div 64
	bitOffset := bitIdx & 63
	if (guild.TrustedBitset[wordIdx] & (1 << bitOffset)) != 0 {
		return
	}

	// ═══════════════════════════════════════════════════════════════════
	// END SAFETY CHECKS - PROCEED WITH ULTRA-FAST DETECTION
	// ═══════════════════════════════════════════════════════════════════

	// Get User State with lockless algorithm
	user := GetUser(evt.UserID)

	// Evaluate Rules with table-driven zero-allocation approach
	punish, pType := EvaluateRules(evt, user)

	// Execute Punishment if needed
	if punish {
		// Calculate detection speed (only for threats that pass safety checks)
		// Use RDTSC or similar if possible, but for now rely on monotonic clock diff
		// We avoid time.Now() if possible, but we need it for the log
		now := time.Now()
		detectionTime := now.UnixNano() - evt.DetectionStart
		detectionSpeed := time.Duration(detectionTime)

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
		// This is non-blocking if the queue has space
		acl.PushPunish(task)
	}
}
