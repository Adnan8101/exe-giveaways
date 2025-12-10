package acl

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Logger configuration
var (
	logQueue      = make(chan LogEntry, 5000)
	discordSess   *discordgo.Session
	logChannelID  string
	batchInterval = 100 * time.Millisecond
	batchSize     = 10
)

// LogEntry represents a single log entry
type LogEntry struct {
	Message   string
	Level     string // "info", "warn", "error", "critical"
	Timestamp time.Time
	GuildID   string
	UserID    string
	Action    string
	Latency   time.Duration
}

// InitLogger initializes the logger with Discord session
func InitLogger(session *discordgo.Session, channelID string) {
	discordSess = session
	logChannelID = channelID
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
				fmt.Printf("[%s] %s | %s | Latency: %v\n",
					entry.Level, entry.Timestamp.Format("15:04:05.000"),
					entry.Message, entry.Latency)
			}

			// Discord output (if configured)
			if discordSess != nil && logChannelID != "" {
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

	// Build embed
	var description strings.Builder
	for i, entry := range entries {
		if i >= 25 { // Discord embed limit
			description.WriteString(fmt.Sprintf("\n*...and %d more entries*", len(entries)-25))
			break
		}
		emoji := getEmojiForLevel(entry.Level)
		description.WriteString(fmt.Sprintf("%s **%s** | %s", emoji, entry.Action, entry.Message))
		if entry.Latency > 0 {
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

	_, err := discordSess.ChannelMessageSendEmbed(logChannelID, embed)
	if err != nil {
		fmt.Printf("[LOGGER] Failed to send to Discord: %v\n", err)
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
