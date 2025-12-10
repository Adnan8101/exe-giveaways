package acl

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

// PunishTask represents an action to be taken via Discord API
type PunishTask struct {
	GuildID       uint64
	UserID        uint64
	Type          string
	Reason        string
	DetectionTime time.Duration // Time taken to detect the violation
}

// Buffered channel for tasks
// This allows CDE to push without blocking unless buffer is full (Backpressure)
// Size should be large enough to handle bursts
var punishQueue = make(chan PunishTask, 1000)

// Discord session (injected at startup)
var discordSession *discordgo.Session

// InitPunishWorker initializes the punishment worker with Discord session
func InitPunishWorker(session *discordgo.Session) {
	discordSession = session
}

// PushPunish adds a task to the queue
// Non-blocking drop if full to protect CDE?
// Or blocking? For security, we might want to ensure it goes through,
// but blocking CDE is fatal.
// We use a select with default to drop if full,
// OR we use a very large buffer.
func PushPunish(task PunishTask) {
	select {
	case punishQueue <- task:
	default:
		// ACL Overload - Drop or log error to atomic counter
		// For now, silent drop to preserve CDE latency
		log.Printf("[ACL] WARNING: Punishment queue full, dropping task for user %d", task.UserID)
	}
}

// StartPunishWorker starts the consumer for the punish queue
func StartPunishWorker() {
	go func() {
		for task := range punishQueue {
			executePunishment(task)
		}
	}()
}

func executePunishment(task PunishTask) {
	start := time.Now()

	log.Printf("[ACL] ⚡ Received punishment task: GuildID=%d, UserID=%d, Type=%s",
		task.GuildID, task.UserID, task.Type)

	if discordSession == nil {
		log.Println("[ACL] ERROR: Discord session not initialized")
		return
	}

	guildID := fmt.Sprintf("%d", task.GuildID)
	userID := fmt.Sprintf("%d", task.UserID)

	log.Printf("[ACL] Executing %s on user %s in guild %s...", task.Type, userID, guildID)

	var err error
	switch task.Type {
	case "BAN":
		log.Printf("[ACL] 🔨 EXECUTING BAN: User %s in Guild %s", userID, guildID)
		err = discordSession.GuildBanCreateWithReason(guildID, userID, task.Reason, 0)
		executionTime := time.Since(start)
		if err == nil {
			// Format detection time in microseconds
			detectionMicros := float64(task.DetectionTime.Nanoseconds()) / 1000.0

			log.Printf("[ACL] ✅ BAN SUCCESSFUL: User %s banned in guild %s", userID, guildID)
			log.Printf("    ⚡ Detection Speed: %.2fµs", detectionMicros)
			log.Printf("    ⏱️  Execution Time: %v", executionTime)

			PushLogEntry(LogEntry{
				Message:       fmt.Sprintf("Banned user %s after detecting 1 violation", userID),
				Level:         "critical",
				GuildID:       guildID,
				UserID:        userID,
				Action:        "BAN",
				Latency:       executionTime,
				DetectionTime: task.DetectionTime,
			})
		} else {
			log.Printf("[ACL] ❌ BAN FAILED: User %s in guild %s - Error: %v",
				userID, guildID, err)
		}

	case "KICK":
		err = discordSession.GuildMemberDeleteWithReason(guildID, userID, task.Reason)
		if err == nil {
			PushLogEntry(LogEntry{
				Message: fmt.Sprintf("Kicked user %s", userID),
				Level:   "error",
				GuildID: guildID,
				UserID:  userID,
				Action:  "KICK",
				Latency: time.Since(start),
			})
		}

	case "TIMEOUT":
		// Timeout for 5 minutes
		timeout := time.Now().Add(5 * time.Minute)
		err = discordSession.GuildMemberTimeout(guildID, userID, &timeout)
		if err == nil {
			PushLogEntry(LogEntry{
				Message: fmt.Sprintf("Timed out user %s for 5 minutes", userID),
				Level:   "warn",
				GuildID: guildID,
				UserID:  userID,
				Action:  "TIMEOUT",
				Latency: time.Since(start),
			})
		}

	case "QUARANTINE":
		// Remove all roles from the user
		member, err := discordSession.GuildMember(guildID, userID)
		if err == nil {
			for _, roleID := range member.Roles {
				discordSession.GuildMemberRoleRemove(guildID, userID, roleID)
			}
			PushLogEntry(LogEntry{
				Message: fmt.Sprintf("Quarantined user %s (removed all roles)", userID),
				Level:   "warn",
				GuildID: guildID,
				UserID:  userID,
				Action:  "QUARANTINE",
				Latency: time.Since(start),
			})
		}

	default:
		log.Printf("[ACL] Unknown punishment type: %s", task.Type)
		return
	}

	if err != nil {
		log.Printf("[ACL] Failed to execute %s on user %d: %v", task.Type, task.UserID, err)
		PushLogEntry(LogEntry{
			Message: fmt.Sprintf("Failed to %s user %s: %v", task.Type, userID, err),
			Level:   "error",
			GuildID: guildID,
			UserID:  userID,
			Action:  task.Type,
		})
	}
}
