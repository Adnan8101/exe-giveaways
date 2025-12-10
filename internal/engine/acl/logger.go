package acl

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Logger configuration
var (
	logQueue       = make(chan LogEntry, 5000)
	discordSess    *discordgo.Session
	logChannels    = make(map[string]string) // guildID -> channelID mapping
	logChannelLock sync.RWMutex
	batchInterval  = 100 * time.Millisecond
	batchSize      = 10
)

// LogEntry represents a single log entry
type LogEntry struct {
	Message       string
	Level         string // "info", "warn", "error", "critical"
	Timestamp     time.Time
	GuildID       string
	UserID        string
	Action        string
	Latency       time.Duration
	DetectionTime time.Duration // Time taken to detect the violation
}

// InitLogger initializes the logger with Discord session
func InitLogger(session *discordgo.Session) {
	discordSess = session
	log.Println("[LOGGER] Initialized with Discord session")
}

// SetGuildLogChannel sets the log channel for a specific guild
func SetGuildLogChannel(guildID, channelID string) {
	logChannelLock.Lock()
	defer logChannelLock.Unlock()
	logChannels[guildID] = channelID
	log.Printf("[LOGGER] Set log channel for guild %s: %s", guildID, channelID)
}

// GetGuildLogChannel retrieves the log channel for a guild
func GetGuildLogChannel(guildID string) string {
	logChannelLock.RLock()
	defer logChannelLock.RUnlock()
	return logChannels[guildID]
}

// PushLog adds a log entry to the queue
func PushLog(msg string) {
	select {
	case logQueue <- LogEntry{
		Message:   msg,
		Level:     "info",
		Timestamp: time.Now(),
	}:
	default:
		// Drop logs if queue is full to prevent blocking
		fmt.Println("[LOGGER] Queue full, dropping log:", msg)
	}
}

// PushLogEntry adds a structured log entry
func PushLogEntry(entry LogEntry) {
	entry.Timestamp = time.Now()
	select {
	case logQueue <- entry:
	default:
		fmt.Println("[LOGGER] Queue full, dropping entry")
	}
}

// StartLogger starts the log consumer with batching
func StartLogger() {
	go func() {
		batch := make([]LogEntry, 0, batchSize)
		ticker := time.NewTicker(batchInterval)
		defer ticker.Stop()

		flush := func() {
			if len(batch) == 0 {
				return
			}

			// Console output (always)
			for _, entry := range batch {
				if entry.DetectionTime > 0 {
					detectionMicros := float64(entry.DetectionTime.Nanoseconds()) / 1000.0
					fmt.Printf("[%s] %s | %s | Detection: %.2fµs | Execution: %v\n",
						entry.Level, entry.Timestamp.Format("15:04:05.000"),
						entry.Message, detectionMicros, entry.Latency)
				} else {
					fmt.Printf("[%s] %s | %s | Latency: %v\n",
						entry.Level, entry.Timestamp.Format("15:04:05.000"),
						entry.Message, entry.Latency)
				}
			}

			// Discord output (if configured)
			if discordSess != nil {
				sendToDiscord(batch)
			}

			batch = batch[:0] // Clear batch
		}

		for {
			select {
			case entry := <-logQueue:
				batch = append(batch, entry)
				if len(batch) >= batchSize {
					flush()
				}
			case <-ticker.C:
				flush()
			}
		}
	}()
}

// sendToDiscord sends batched logs to Discord channel
func sendToDiscord(entries []LogEntry) {
	if len(entries) == 0 {
		return
	}

	// Group entries by guild
	guildEntries := make(map[string][]LogEntry)
	for _, entry := range entries {
		if entry.GuildID != "" {
			guildEntries[entry.GuildID] = append(guildEntries[entry.GuildID], entry)
		}
	}

	// Send to each guild's log channel
	for guildID, guildLogs := range guildEntries {
		channelID := GetGuildLogChannel(guildID)
		if channelID == "" {
			continue // No log channel configured for this guild
		}
		sendToChannel(channelID, guildLogs)
	}
}

// sendToChannel sends logs to a specific channel
func sendToChannel(channelID string, entries []LogEntry) {
	if len(entries) == 0 {
		return
	}

	// Build embed
	var description strings.Builder
	for i, entry := range entries {
		if i >= 25 { // Discord embed limit
			description.WriteString(fmt.Sprintf("\n*...and %d more entries*", len(entries)-25))
			break
		}
		emoji := getEmojiForLevel(entry.Level)
		description.WriteString(fmt.Sprintf("%s **%s** | %s", emoji, entry.Action, entry.Message))
		if entry.DetectionTime > 0 {
			detectionMicros := float64(entry.DetectionTime.Nanoseconds()) / 1000.0
			description.WriteString(fmt.Sprintf(" `[Detection: %.2fµs, Execution: %v]`", detectionMicros, entry.Latency))
		} else if entry.Latency > 0 {
			description.WriteString(fmt.Sprintf(" `[%v]`", entry.Latency))
		}
		description.WriteString("\n")
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🛡️ AntiNuke Detection Log",
		Description: description.String(),
		Color:       getColorForLevel(entries[0].Level),
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("%d events logged", len(entries)),
		},
	}

	_, err := discordSess.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		fmt.Printf("[LOGGER] Failed to send to Discord channel %s: %v\n", channelID, err)
	}
}

func getEmojiForLevel(level string) string {
	switch level {
	case "critical":
		return "🚨"
	case "error":
		return "❌"
	case "warn":
		return "⚠️"
	default:
		return "ℹ️"
	}
}

func getColorForLevel(level string) int {
	switch level {
	case "critical":
		return 0xFF0000 // Red
	case "error":
		return 0xFF4500 // Orange-Red
	case "warn":
		return 0xFFA500 // Orange
	default:
		return 0x00FF00 // Green
	}
}
