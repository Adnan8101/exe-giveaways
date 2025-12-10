package auditor

import (
	"discord-giveaway-bot/internal/engine/fdl"
	"discord-giveaway-bot/internal/engine/ring"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

// EventHandlers manages all Discord gateway event handlers for antinuke detection
type EventHandlers struct {
	session   *discordgo.Session
	eventRing *ring.RingBuffer
}

// NewEventHandlers creates a new event handlers manager
func NewEventHandlers(session *discordgo.Session, eventRing *ring.RingBuffer) *EventHandlers {
	return &EventHandlers{
		session:   session,
		eventRing: eventRing,
	}
}

// RegisterAll registers all antinuke event handlers with the Discord session
func (h *EventHandlers) RegisterAll() {
	log.Println("🔌 Registering antinuke event handlers...")

	// Channel events
	h.session.AddHandler(h.OnChannelCreate)
	log.Println("   ✓ Channel Create handler registered")

	h.session.AddHandler(h.OnChannelDelete)
	log.Println("   ✓ Channel Delete handler registered")

	h.session.AddHandler(h.OnChannelUpdate)
	log.Println("   ✓ Channel Update handler registered")

	// Role events
	h.session.AddHandler(h.OnRoleCreate)
	log.Println("   ✓ Role Create handler registered")

	h.session.AddHandler(h.OnRoleDelete)
	log.Println("   ✓ Role Delete handler registered")

	h.session.AddHandler(h.OnRoleUpdate)
	log.Println("   ✓ Role Update handler registered")

	// Member events
	h.session.AddHandler(h.OnGuildBanAdd)
	log.Println("   ✓ Guild Ban Add handler registered")

	h.session.AddHandler(h.OnGuildMemberRemove)
	log.Println("   ✓ Guild Member Remove handler registered")

	// Webhook events
	h.session.AddHandler(h.OnWebhooksUpdate)
	log.Println("   ✓ Webhooks Update handler registered")

	// Guild events
	h.session.AddHandler(h.OnGuildUpdate)
	log.Println("   ✓ Guild Update handler registered")

	log.Println("✅ All antinuke event handlers registered successfully")
}

// ============================================================================
// CHANNEL EVENTS
// ============================================================================

// OnChannelCreate detects malicious channel creation
func (h *EventHandlers) OnChannelCreate(s *discordgo.Session, e *discordgo.ChannelCreate) {
	start := time.Now()

	// Skip DM channels
	if e.GuildID == "" {
		return
	}

	log.Printf("[ANTINUKE] 🔵 Channel Create | Guild: %s | Channel: %s (%s) | Name: %s",
		e.GuildID, e.ID, e.Type, e.Name)

	// We need to fetch the audit log to find WHO created the channel
	// This is necessary because ChannelCreate event doesn't include the creator
	userID := h.fetchChannelCreator(e.GuildID, e.ID)

	if userID == 0 {
		log.Printf("[ANTINUKE] ⚠️  Could not determine channel creator, skipping detection")
		return
	}

	// Create FastEvent
	evt := &fdl.FastEvent{
		ReqType:        fdl.EvtChannelCreate,
		GuildID:        parseSnowflake(e.GuildID),
		UserID:         userID,
		EntityID:       parseSnowflake(e.ID),
		Timestamp:      time.Now().UnixNano(),
		DetectionStart: start.UnixNano(),
	}

	// Push to ring buffer
	if !h.eventRing.Push(evt) {
		fdl.EventsDropped.Inc(0)
		log.Printf("[ANTINUKE] ❌ Ring buffer full, event dropped!")
	} else {
		fdl.EventsProcessed.Inc(userID)
		latency := time.Since(start)
		log.Printf("[ANTINUKE] ✅ Channel Create detected | User: %d | Latency: %v", userID, latency)
	}
}

// OnChannelDelete detects malicious channel deletion
func (h *EventHandlers) OnChannelDelete(s *discordgo.Session, e *discordgo.ChannelDelete) {
	start := time.Now()

	// Skip DM channels
	if e.GuildID == "" {
		return
	}

	log.Printf("[ANTINUKE] 🔴 Channel Delete | Guild: %s | Channel: %s | Name: %s",
		e.GuildID, e.ID, e.Name)

	// Fetch who deleted the channel from audit log
	userID := h.fetchChannelDeleter(e.GuildID, e.ID)

	if userID == 0 {
		log.Printf("[ANTINUKE] ⚠️  Could not determine channel deleter, skipping detection")
		return
	}

	evt := &fdl.FastEvent{
		ReqType:        fdl.EvtChannelDelete,
		GuildID:        parseSnowflake(e.GuildID),
		UserID:         userID,
		EntityID:       parseSnowflake(e.ID),
		Timestamp:      time.Now().UnixNano(),
		DetectionStart: start.UnixNano(),
	}

	if !h.eventRing.Push(evt) {
		fdl.EventsDropped.Inc(0)
		log.Printf("[ANTINUKE] ❌ Ring buffer full, event dropped!")
	} else {
		fdl.EventsProcessed.Inc(userID)
		latency := time.Since(start)
		log.Printf("[ANTINUKE] ✅ Channel Delete detected | User: %d | Latency: %v", userID, latency)
	}
}

// OnChannelUpdate detects suspicious channel modifications
func (h *EventHandlers) OnChannelUpdate(s *discordgo.Session, e *discordgo.ChannelUpdate) {
	start := time.Now()

	if e.GuildID == "" {
		return
	}

	log.Printf("[ANTINUKE] 🟡 Channel Update | Guild: %s | Channel: %s | Name: %s",
		e.GuildID, e.ID, e.Name)

	userID := h.fetchChannelUpdater(e.GuildID, e.ID)

	if userID == 0 {
		return
	}

	evt := &fdl.FastEvent{
		ReqType:        fdl.EvtChannelUpdate,
		GuildID:        parseSnowflake(e.GuildID),
		UserID:         userID,
		EntityID:       parseSnowflake(e.ID),
		Timestamp:      time.Now().UnixNano(),
		DetectionStart: start.UnixNano(),
	}

	if !h.eventRing.Push(evt) {
		fdl.EventsDropped.Inc(0)
	} else {
		fdl.EventsProcessed.Inc(userID)
		latency := time.Since(start)
		log.Printf("[ANTINUKE] ✅ Channel Update detected | User: %d | Latency: %v", userID, latency)
	}
}

// ============================================================================
// ROLE EVENTS
// ============================================================================

// OnRoleCreate detects malicious role creation
func (h *EventHandlers) OnRoleCreate(s *discordgo.Session, e *discordgo.GuildRoleCreate) {
	start := time.Now()

	log.Printf("[ANTINUKE] 🔵 Role Create | Guild: %s | Role: %s | Name: %s | Perms: %d",
		e.GuildID, e.Role.ID, e.Role.Name, e.Role.Permissions)

	userID := h.fetchRoleCreator(e.GuildID, e.Role.ID)

	if userID == 0 {
		log.Printf("[ANTINUKE] ⚠️  Could not determine role creator, skipping detection")
		return
	}

	evt := &fdl.FastEvent{
		ReqType:        fdl.EvtRoleCreate,
		GuildID:        parseSnowflake(e.GuildID),
		UserID:         userID,
		EntityID:       parseSnowflake(e.Role.ID),
		Timestamp:      time.Now().UnixNano(),
		DetectionStart: start.UnixNano(),
	}

	if !h.eventRing.Push(evt) {
		fdl.EventsDropped.Inc(0)
		log.Printf("[ANTINUKE] ❌ Ring buffer full, event dropped!")
	} else {
		fdl.EventsProcessed.Inc(userID)
		latency := time.Since(start)
		log.Printf("[ANTINUKE] ✅ Role Create detected | User: %d | Latency: %v", userID, latency)
	}
}

// OnRoleDelete detects admin role deletions
func (h *EventHandlers) OnRoleDelete(s *discordgo.Session, e *discordgo.GuildRoleDelete) {
	start := time.Now()

	log.Printf("[ANTINUKE] 🔴 Role Delete | Guild: %s | Role: %s",
		e.GuildID, e.RoleID)

	userID := h.fetchRoleDeleter(e.GuildID, e.RoleID)

	if userID == 0 {
		log.Printf("[ANTINUKE] ⚠️  Could not determine role deleter, skipping detection")
		return
	}

	evt := &fdl.FastEvent{
		ReqType:        fdl.EvtRoleDelete,
		GuildID:        parseSnowflake(e.GuildID),
		UserID:         userID,
		EntityID:       parseSnowflake(e.RoleID),
		Timestamp:      time.Now().UnixNano(),
		DetectionStart: start.UnixNano(),
	}

	if !h.eventRing.Push(evt) {
		fdl.EventsDropped.Inc(0)
		log.Printf("[ANTINUKE] ❌ Ring buffer full, event dropped!")
	} else {
		fdl.EventsProcessed.Inc(userID)
		latency := time.Since(start)
		log.Printf("[ANTINUKE] ✅ Role Delete detected | User: %d | Latency: %v", userID, latency)
	}
}

// OnRoleUpdate detects permission escalation
func (h *EventHandlers) OnRoleUpdate(s *discordgo.Session, e *discordgo.GuildRoleUpdate) {
	start := time.Now()

	log.Printf("[ANTINUKE] 🟡 Role Update | Guild: %s | Role: %s | Name: %s | Perms: %d",
		e.GuildID, e.Role.ID, e.Role.Name, e.Role.Permissions)

	userID := h.fetchRoleUpdater(e.GuildID, e.Role.ID)

	if userID == 0 {
		return
	}

	evt := &fdl.FastEvent{
		ReqType:        fdl.EvtRoleUpdate,
		GuildID:        parseSnowflake(e.GuildID),
		UserID:         userID,
		EntityID:       parseSnowflake(e.Role.ID),
		Timestamp:      time.Now().UnixNano(),
		DetectionStart: start.UnixNano(),
	}

	if !h.eventRing.Push(evt) {
		fdl.EventsDropped.Inc(0)
	} else {
		fdl.EventsProcessed.Inc(userID)
		latency := time.Since(start)
		log.Printf("[ANTINUKE] ✅ Role Update detected | User: %d | Latency: %v", userID, latency)
	}
}

// ============================================================================
// MEMBER EVENTS
// ============================================================================

// OnGuildBanAdd detects mass ban attacks
func (h *EventHandlers) OnGuildBanAdd(s *discordgo.Session, e *discordgo.GuildBanAdd) {
	start := time.Now()

	log.Printf("[ANTINUKE] 🔨 Ban Add | Guild: %s | Banned User: %s",
		e.GuildID, e.User.ID)

	userID := h.fetchBanIssuer(e.GuildID, e.User.ID)

	if userID == 0 {
		log.Printf("[ANTINUKE] ⚠️  Could not determine ban issuer, skipping detection")
		return
	}

	evt := &fdl.FastEvent{
		ReqType:        fdl.EvtGuildBanAdd,
		GuildID:        parseSnowflake(e.GuildID),
		UserID:         userID,
		EntityID:       parseSnowflake(e.User.ID),
		Timestamp:      time.Now().UnixNano(),
		DetectionStart: start.UnixNano(),
	}

	if !h.eventRing.Push(evt) {
		fdl.EventsDropped.Inc(0)
		log.Printf("[ANTINUKE] ❌ Ring buffer full, event dropped!")
	} else {
		fdl.EventsProcessed.Inc(userID)
		latency := time.Since(start)
		log.Printf("[ANTINUKE] ✅ Ban detected | Issuer: %d | Target: %s | Latency: %v",
			userID, e.User.ID, latency)
	}
}

// OnGuildMemberRemove detects mass kicks
func (h *EventHandlers) OnGuildMemberRemove(s *discordgo.Session, e *discordgo.GuildMemberRemove) {
	start := time.Now()

	log.Printf("[ANTINUKE] 👢 Member Remove | Guild: %s | User: %s",
		e.GuildID, e.User.ID)

	// Check if it was a kick (not a leave or ban)
	userID := h.fetchKickIssuer(e.GuildID, e.User.ID)

	if userID == 0 {
		// Likely a voluntary leave, not a kick
		return
	}

	evt := &fdl.FastEvent{
		ReqType:        fdl.EvtGuildMemberRemove,
		GuildID:        parseSnowflake(e.GuildID),
		UserID:         userID,
		EntityID:       parseSnowflake(e.User.ID),
		Timestamp:      time.Now().UnixNano(),
		DetectionStart: start.UnixNano(),
	}

	if !h.eventRing.Push(evt) {
		fdl.EventsDropped.Inc(0)
	} else {
		fdl.EventsProcessed.Inc(userID)
		latency := time.Since(start)
		log.Printf("[ANTINUKE] ✅ Kick detected | Issuer: %d | Target: %s | Latency: %v",
			userID, e.User.ID, latency)
	}
}

// ============================================================================
// WEBHOOK & GUILD EVENTS
// ============================================================================

// OnWebhooksUpdate detects webhook creation attacks
func (h *EventHandlers) OnWebhooksUpdate(s *discordgo.Session, e *discordgo.WebhooksUpdate) {
	start := time.Now()

	log.Printf("[ANTINUKE] 🪝 Webhooks Update | Guild: %s | Channel: %s",
		e.GuildID, e.ChannelID)

	userID := h.fetchWebhookCreator(e.GuildID, e.ChannelID)

	if userID == 0 {
		return
	}

	evt := &fdl.FastEvent{
		ReqType:        fdl.EvtWebhookCreate,
		GuildID:        parseSnowflake(e.GuildID),
		UserID:         userID,
		EntityID:       parseSnowflake(e.ChannelID),
		Timestamp:      time.Now().UnixNano(),
		DetectionStart: start.UnixNano(),
	}

	if !h.eventRing.Push(evt) {
		fdl.EventsDropped.Inc(0)
	} else {
		fdl.EventsProcessed.Inc(userID)
		latency := time.Since(start)
		log.Printf("[ANTINUKE] ✅ Webhook event detected | User: %d | Latency: %v", userID, latency)
	}
}

// OnGuildUpdate detects server takeover attempts
func (h *EventHandlers) OnGuildUpdate(s *discordgo.Session, e *discordgo.GuildUpdate) {
	start := time.Now()

	log.Printf("[ANTINUKE] 🏰 Guild Update | Guild: %s | Name: %s",
		e.ID, e.Name)

	userID := h.fetchGuildUpdater(e.ID)

	if userID == 0 {
		return
	}

	evt := &fdl.FastEvent{
		ReqType:        fdl.EvtGuildUpdate,
		GuildID:        parseSnowflake(e.ID),
		UserID:         userID,
		EntityID:       0,
		Timestamp:      time.Now().UnixNano(),
		DetectionStart: start.UnixNano(),
	}

	if !h.eventRing.Push(evt) {
		fdl.EventsDropped.Inc(0)
	} else {
		fdl.EventsProcessed.Inc(userID)
		latency := time.Since(start)
		log.Printf("[ANTINUKE] ✅ Guild Update detected | User: %d | Latency: %v", userID, latency)
	}
}

// ============================================================================
// AUDIT LOG HELPERS
// ============================================================================

// fetchChannelCreator fetches who created a channel from audit logs
func (h *EventHandlers) fetchChannelCreator(guildID, channelID string) uint64 {
	return h.fetchAuditLogUser(guildID, channelID, discordgo.AuditLogActionChannelCreate)
}

// fetchChannelDeleter fetches who deleted a channel from audit logs
func (h *EventHandlers) fetchChannelDeleter(guildID, channelID string) uint64 {
	return h.fetchAuditLogUser(guildID, channelID, discordgo.AuditLogActionChannelDelete)
}

// fetchChannelUpdater fetches who updated a channel from audit logs
func (h *EventHandlers) fetchChannelUpdater(guildID, channelID string) uint64 {
	return h.fetchAuditLogUser(guildID, channelID, discordgo.AuditLogActionChannelUpdate)
}

// fetchRoleCreator fetches who created a role from audit logs
func (h *EventHandlers) fetchRoleCreator(guildID, roleID string) uint64 {
	return h.fetchAuditLogUser(guildID, roleID, discordgo.AuditLogActionRoleCreate)
}

// fetchRoleDeleter fetches who deleted a role from audit logs
func (h *EventHandlers) fetchRoleDeleter(guildID, roleID string) uint64 {
	return h.fetchAuditLogUser(guildID, roleID, discordgo.AuditLogActionRoleDelete)
}

// fetchRoleUpdater fetches who updated a role from audit logs
func (h *EventHandlers) fetchRoleUpdater(guildID, roleID string) uint64 {
	return h.fetchAuditLogUser(guildID, roleID, discordgo.AuditLogActionRoleUpdate)
}

// fetchBanIssuer fetches who issued a ban from audit logs
func (h *EventHandlers) fetchBanIssuer(guildID, userID string) uint64 {
	return h.fetchAuditLogUser(guildID, userID, discordgo.AuditLogActionMemberBanAdd)
}

// fetchKickIssuer fetches who issued a kick from audit logs
func (h *EventHandlers) fetchKickIssuer(guildID, userID string) uint64 {
	return h.fetchAuditLogUser(guildID, userID, discordgo.AuditLogActionMemberKick)
}

// fetchWebhookCreator fetches who created a webhook from audit logs
func (h *EventHandlers) fetchWebhookCreator(guildID, channelID string) uint64 {
	return h.fetchAuditLogUser(guildID, "", discordgo.AuditLogActionWebhookCreate)
}

// fetchGuildUpdater fetches who updated the guild from audit logs
func (h *EventHandlers) fetchGuildUpdater(guildID string) uint64 {
	return h.fetchAuditLogUser(guildID, "", discordgo.AuditLogActionGuildUpdate)
}

// fetchAuditLogUser is a generic function to fetch the user who performed an action
func (h *EventHandlers) fetchAuditLogUser(guildID, targetID string, actionType discordgo.AuditLogAction) uint64 {
	// Fetch recent audit log entries
	auditLog, err := h.session.GuildAuditLog(guildID, "", "", int(actionType), 1)
	if err != nil {
		log.Printf("[ANTINUKE] ⚠️  Failed to fetch audit log for guild %s: %v", guildID, err)
		return 0
	}

	if len(auditLog.AuditLogEntries) == 0 {
		return 0
	}

	// Get the most recent entry
	entry := auditLog.AuditLogEntries[0]

	// If targetID is specified, verify it matches
	if targetID != "" && entry.TargetID != targetID {
		return 0
	}

	// Parse and return the user ID
	return parseSnowflake(entry.UserID)
}
