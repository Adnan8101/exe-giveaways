package acl

import (
	"fmt"
	"log"
	"runtime"
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
// Massively increased buffer size for extreme burst handling
var punishQueue = make(chan PunishTask, 10000)

// Discord session (injected at startup)
var discordSession *discordgo.Session

// Worker pool for parallel punishment execution
var (
	workerCount    = 200 // MASSIVE worker pool for maximum parallel execution
	workerPoolOnce sync.Once
	fastBanQueue   = make(chan PunishTask, 5000) // Dedicated fast lane for bans
)

// String pools to avoid allocations (expanded pool size)
var stringPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 0, 64)
		return &b
	},
}

// Pre-allocated string buffers for ID conversion (per-worker) with larger capacity
var idBufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 32) // Increased size
		return &buf
	},
}

// InitPunishWorker initializes the punishment worker with Discord session
func InitPunishWorker(session *discordgo.Session) {
	discordSession = session
}

// Fast uint64 to string conversion with pooled buffer (zero allocation)
func uitoaPooled(n uint64) string {
	if n == 0 {
		return "0"
	}

	// Get buffer from pool
	bufPtr := idBufferPool.Get().(*[]byte)
	buf := *bufPtr

	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}

	// Copy to string (necessary to return from pool safely)
	result := string(buf[i:])

	// Return buffer to pool
	idBufferPool.Put(bufPtr)

	return result
}

// Legacy version for compatibility
func uitoa(n uint64) string {
	return uitoaPooled(n)
}

// PushPunish adds a task to the queue
// ULTRA-OPTIMIZED: BAN actions use dedicated fast lane for minimum latency
func PushPunish(task PunishTask) {
	// EXTREME OPTIMIZATION: BAN actions use dedicated high-priority queue
	if task.Type == "BAN" {
		// Try fast lane first (non-blocking)
		select {
		case fastBanQueue <- task:
			return
		default:
			// Fast lane full, execute immediately in goroutine
			go executePunishmentDirect(task)
			return
		}
	}

	// Other punishment types use standard queue
	select {
	case punishQueue <- task:
	default:
		// ACL Overload - Drop or log error to atomic counter
		log.Printf("[ACL] WARNING: Punishment queue full, dropping task for user %d", task.UserID)
	}
}

// StartPunishWorker starts multiple worker goroutines for parallel execution
func StartPunishWorker() {
	workerPoolOnce.Do(func() {
		log.Printf("[ACL] Starting %d punishment workers...", workerCount)

		// Start dedicated fast ban workers (75% of pool for ban priority)
		fastWorkerCount := (workerCount * 3) / 4
		for i := 0; i < fastWorkerCount; i++ {
			go fastBanWorker(i)
		}

		// Start standard workers for other punishment types
		standardWorkerCount := workerCount - fastWorkerCount
		for i := 0; i < standardWorkerCount; i++ {
			go punishmentWorker(i + fastWorkerCount)
		}

		log.Printf("[ACL] ✅ All %d workers ready (%d fast-ban, %d standard)",
			workerCount, fastWorkerCount, standardWorkerCount)
	})
}

// fastBanWorker is a dedicated high-priority worker for ban actions
func fastBanWorker(id int) {
	runtime.LockOSThread() // Pin to OS thread for consistent performance
	defer runtime.UnlockOSThread()

	for task := range fastBanQueue {
		executePunishment(task)
	}
}

// punishmentWorker is a dedicated goroutine that processes punishment tasks
func punishmentWorker(id int) {
	for task := range punishQueue {
		executePunishment(task)
	}
}

// executeFastBan performs an optimized ban with minimal overhead
// Uses direct HTTP client access for maximum speed, bypassing discordgo overhead
func executeFastBan(guildID, userID uint64, reason string) error {
	// Use ultra-fast direct API call (bypasses discordgo overhead)
	err := FastBanRequest(guildID, userID, reason)
	if err != nil {
		// Fallback to standard discordgo method if fast path fails
		return discordSession.GuildBanCreateWithReason(uitoaPooled(guildID), uitoaPooled(userID), reason, 0)
	}
	return nil
}

// executePunishmentDirect executes punishment without queueing (EXTREME SPEED MODE)
// Used for BAN actions to minimize latency
func executePunishmentDirect(task PunishTask) {
	executePunishment(task)
}

func executePunishment(task PunishTask) {
	start := time.Now()

	if discordSession == nil {
		return
	}

	// Fast uint64 to string conversion with pooled buffers (zero allocation)
	guildID := uitoaPooled(task.GuildID)
	userID := uitoaPooled(task.UserID)

	var err error
	switch task.Type {
	case "BAN":
		// ULTRA-FAST BAN EXECUTION
		// Use direct API call with minimal overhead
		err = executeFastBan(task.GuildID, task.UserID, task.Reason)
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
				log.Printf("[ACL] ⚡ BAN | User %s | Detection: %.2fµs | Exec: %v | Total: %v",
					userID, detectionMicros, executionTime, totalTime)
			} else {
				log.Printf("[ACL] ⚡ BAN | User %s | Detection: %.2fµs | Exec: %v",
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
