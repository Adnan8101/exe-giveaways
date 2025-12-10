package acl

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// PunishTask represents an action to be taken via Discord API
type PunishTask struct {
	GuildID        uint64
	UserID         uint64
	Type           string
	Reason         string
	DetectionTime  time.Duration // Time taken to detect the violation
	DetectionStart time.Time     // When detection started (for total latency tracking)
}

// Buffered channel for tasks
// This allows CDE to push without blocking unless buffer is full (Backpressure)
// Size should be large enough to handle bursts
var punishQueue = make(chan PunishTask, 1000)

// Discord session (injected at startup)
var discordSession *discordgo.Session

// String pools to avoid allocations
var stringPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 0, 32)
		return &b
	},
}

// InitPunishWorker initializes the punishment worker with Discord session
func InitPunishWorker(session *discordgo.Session) {
	discordSession = session
}

// Fast uint64 to string conversion
func uitoa(n uint64) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 20)
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
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

	if discordSession == nil {
		return
	}

	// Fast uint64 to string conversion without allocations
	guildID := uitoa(task.GuildID)
	userID := uitoa(task.UserID)

	var err error
	switch task.Type {
	case "BAN":
		// EXECUTE BAN IMMEDIATELY - NO BLOCKING
		err = discordSession.GuildBanCreateWithReason(guildID, userID, task.Reason, 0)
		executionTime := time.Since(start)
		
		// Format detection time in microseconds
		detectionMicros := float64(task.DetectionTime.Nanoseconds()) / 1000.0
		
		// Calculate total time from detection to ban completion
		var totalTime time.Duration
		if !task.DetectionStart.IsZero() {
			totalTime = time.Since(task.DetectionStart)
		}

		if err == nil {
			// Console log immediately (fast)
			if totalTime > 0 {
				log.Printf("[ACL] ⚡ BAN | User %s | Detection: %.2fµs | Execution: %v | Total: %v", 
					userID, detectionMicros, executionTime, totalTime)
			} else {
				log.Printf("[ACL] ⚡ BAN | User %s | Detection: %.2fµs | Execution: %v", 
					userID, detectionMicros, executionTime)
			}

			// Push to async Discord logger (non-blocking)
			go PushLogEntry(LogEntry{
				Message:       fmt.Sprintf("Banned user %s after detecting 1 violation", userID),
				Level:         "critical",
				GuildID:       guildID,
				UserID:        userID,
				Action:        "BAN",
				Latency:       executionTime,
				DetectionTime: task.DetectionTime,
			})
		} else {
			log.Printf("[ACL] ❌ BAN FAILED: %v", err)
		}

	case "KICK":
		err = discordSession.GuildMemberDeleteWithReason(guildID, userID, task.Reason)
		executionTime := time.Since(start)
		if err == nil {
			log.Printf("[ACL] ✅ KICK | User %s | Execution: %v", userID, executionTime)
			go PushLogEntry(LogEntry{
				Message: fmt.Sprintf("Kicked user %s", userID),
				Level:   "error",
				GuildID: guildID,
				UserID:  userID,
				Action:  "KICK",
				Latency: executionTime,
			})
		}

	case "TIMEOUT":
		// Timeout for 5 minutes
		timeout := time.Now().Add(5 * time.Minute)
		err = discordSession.GuildMemberTimeout(guildID, userID, &timeout)
		executionTime := time.Since(start)
		if err == nil {
			log.Printf("[ACL] ✅ TIMEOUT | User %s | Execution: %v", userID, executionTime)
			go PushLogEntry(LogEntry{
				Message: fmt.Sprintf("Timed out user %s for 5 minutes", userID),
				Level:   "warn",
				GuildID: guildID,
				UserID:  userID,
				Action:  "TIMEOUT",
				Latency: executionTime,
			})
		}

	case "QUARANTINE":
		// Remove all roles from the user concurrently
		member, err := discordSession.GuildMember(guildID, userID)
		if err == nil {
			var wg sync.WaitGroup
			for _, roleID := range member.Roles {
				wg.Add(1)
				go func(rID string) {
					defer wg.Done()
					discordSession.GuildMemberRoleRemove(guildID, userID, rID)
				}(roleID)
			}
			wg.Wait()
			executionTime := time.Since(start)
			log.Printf("[ACL] ✅ QUARANTINE | User %s | Execution: %v", userID, executionTime)
			go PushLogEntry(LogEntry{
				Message: fmt.Sprintf("Quarantined user %s (removed all roles)", userID),
				Level:   "warn",
				GuildID: guildID,
				UserID:  userID,
				Action:  "QUARANTINE",
				Latency: executionTime,
			})
		}

	default:
		log.Printf("[ACL] Unknown punishment type: %s", task.Type)
		return
	}

	if err != nil {
		log.Printf("[ACL] Failed to execute %s on user %d: %v", task.Type, task.UserID, err)
		go PushLogEntry(LogEntry{
			Message: fmt.Sprintf("Failed to %s user %s: %v", task.Type, userID, err),
			Level:   "error",
			GuildID: guildID,
			UserID:  userID,
			Action:  task.Type,
		})
	}
}
