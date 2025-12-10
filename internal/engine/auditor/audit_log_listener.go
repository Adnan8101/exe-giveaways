package auditor

import (
	"discord-giveaway-bot/internal/engine/acl"
	"discord-giveaway-bot/internal/engine/fdl"
	"discord-giveaway-bot/internal/engine/ring"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

// AuditLogMonitor monitors Discord audit log events
type AuditLogMonitor struct {
	session     *discordgo.Session
	eventRing   *ring.RingBuffer
	lastAuditID map[string]string // guildID -> last audit entry ID
}

// New creates a new audit log monitor
func New(session *discordgo.Session, eventRing *ring.RingBuffer) *AuditLogMonitor {
	return &AuditLogMonitor{
		session:     session,
		eventRing:   eventRing,
		lastAuditID: make(map[string]string),
	}
}

// Start begins monitoring audit logs
func (m *AuditLogMonitor) Start() {
	// Register handler for GUILD_AUDIT_LOG_ENTRY_CREATE (if available)
	m.session.AddHandler(m.handleAuditLogEntry)

	// Start polling fallback for guilds (every 500ms)
	go m.pollAuditLogs()

	acl.PushLog("Audit log monitoring started")
}

// handleAuditLogEntry processes real-time audit log events
func (m *AuditLogMonitor) handleAuditLogEntry(s *discordgo.Session, e *discordgo.Event) {
	// Check if this is an audit log entry event
	if e.Type != "GUILD_AUDIT_LOG_ENTRY_CREATE" {
		return
	}

	start := time.Now()

	// Parse the audit log entry from RawData
	fastEvt, err := fdl.ParseAuditLogEntry(e.RawData)
	if err != nil {
		return
	}

	if fastEvt != nil {
		// Feed directly into ring buffer
		if !m.eventRing.Push(fastEvt) {
			fdl.EventsDropped.Inc(0)
		} else {
			fdl.EventsProcessed.Inc(fastEvt.UserID)
		}

		// Log detection latency
		latency := time.Since(start)
		acl.PushLogEntry(acl.LogEntry{
			Message: fmt.Sprintf("Detected %s by %d", getActionName(fastEvt.ReqType), fastEvt.UserID),
			Level:   "info",
			GuildID: fmt.Sprintf("%d", fastEvt.GuildID),
			UserID:  fmt.Sprintf("%d", fastEvt.UserID),
			Action:  getActionName(fastEvt.ReqType),
			Latency: latency,
		})
	}
}

// pollAuditLogs polls Discord REST API for audit logs (fallback)
func (m *AuditLogMonitor) pollAuditLogs() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		// Get all guilds from session state
		// Note: This requires state tracking or we need to maintain guild list
		// For now, we'll skip polling and rely on event-based detection
		// Production implementation would poll for each guild
	}
}

// ProcessAuditLogManual allows manual audit log processing (for testing)
func (m *AuditLogMonitor) ProcessAuditLogManual(guildID string, entries []*discordgo.AuditLogEntry) {
	for _, entry := range entries {
		fastEvt := convertAuditLogToFastEvent(guildID, entry)
		if fastEvt != nil {
			m.eventRing.Push(fastEvt)
		}
	}
}

// convertAuditLogToFastEvent converts Discord audit log entry to FastEvent
func convertAuditLogToFastEvent(guildID string, entry *discordgo.AuditLogEntry) *fdl.FastEvent {
	evtType := mapAuditActionToEventType(*entry.ActionType)
	if evtType == fdl.EvtUnknown {
		return nil
	}

	userID := uint64(0)
	if entry.UserID != "" {
		userID = fdl.ParseSnowflakeString(entry.UserID)
	}

	targetID := uint64(0)
	if entry.TargetID != "" {
		targetID = fdl.ParseSnowflakeString(entry.TargetID)
	}

	return &fdl.FastEvent{
		ReqType:   evtType,
		GuildID:   fdl.ParseSnowflakeString(guildID),
		UserID:    userID,
		EntityID:  targetID,
		Timestamp: time.Now().UnixNano(),
	}
}

// mapAuditActionToEventType maps Discord audit log action types to internal event types
func mapAuditActionToEventType(actionType discordgo.AuditLogAction) uint8 {
	switch actionType {
	case discordgo.AuditLogActionChannelCreate:
		return fdl.EvtChannelCreate
	case discordgo.AuditLogActionChannelDelete:
		return fdl.EvtChannelDelete
	case discordgo.AuditLogActionChannelUpdate:
		return fdl.EvtChannelUpdate
	case discordgo.AuditLogActionMemberBanAdd:
		return fdl.EvtGuildBanAdd
	case discordgo.AuditLogActionMemberKick:
		return fdl.EvtGuildMemberRemove
	case discordgo.AuditLogActionRoleCreate:
		return fdl.EvtRoleCreate
	case discordgo.AuditLogActionRoleDelete:
		return fdl.EvtRoleDelete
	case discordgo.AuditLogActionRoleUpdate:
		return fdl.EvtRoleUpdate
	case discordgo.AuditLogActionWebhookCreate:
		return fdl.EvtWebhookCreate
	default:
		return fdl.EvtUnknown
	}
}

func getActionName(evtType uint8) string {
	switch evtType {
	case fdl.EvtChannelCreate:
		return "Channel Create"
	case fdl.EvtChannelDelete:
		return "Channel Delete"
	case fdl.EvtGuildBanAdd:
		return "Ban Add"
	case fdl.EvtGuildMemberRemove:
		return "Kick"
	case fdl.EvtRoleDelete:
		return "Role Delete"
	default:
		return "Unknown"
	}
}
