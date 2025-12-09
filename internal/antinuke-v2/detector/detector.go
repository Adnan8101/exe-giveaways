package detector

import (
	"discord-giveaway-bot/internal/antinuke-v2/core"
	"discord-giveaway-bot/internal/models"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Detector is the main antinuke detection engine
// Target: <0.3ms from event to decision
type Detector struct {
	cache       *core.AtomicCache
	rateLimiter *core.FastRateLimiter
	session     *discordgo.Session

	// Async action queues (not on hot path)
	loggingQueue chan *ViolationEvent

	// Object pools for zero-allocation event handling
	violationPool sync.Pool
	taskPool      sync.Pool
}

// PunishmentTask represents a punishment to execute asynchronously
type PunishmentTask struct {
	GuildID    string
	UserID     string
	Punishment string
	Reason     string
}

// RevocationTask represents actions to revoke asynchronously
type RevocationTask struct {
	GuildID    string
	ActionType string
	UserID     string
	TargetID   string
}

// ViolationEvent represents a violation for logging
type ViolationEvent struct {
	GuildID          string
	ActionType       string
	ExecutorID       string
	Count            int
	Limit            int
	DetectionLatency time.Duration
	LogsChannel      string
}

// NewDetector creates a new detection engine
func NewDetector(cache *core.AtomicCache, rateLimiter *core.FastRateLimiter, session *discordgo.Session) *Detector {
	d := &Detector{
		cache:        cache,
		rateLimiter:  rateLimiter,
		session:      session,
		loggingQueue: make(chan *ViolationEvent, 1000),
	}

	// Initialize object pools
	d.violationPool.New = func() interface{} {
		return &ViolationEvent{}
	}
	d.taskPool.New = func() interface{} {
		return &PunishmentTask{}
	}

	// Start background workers
	go d.loggingWorker()

	return d
}

// ProcessEvent is a wrapper for ProcessEventWithGuild that extracts guild ID from entry
// For gateway events, use ProcessEventWithGuild directly which is more efficient
func (d *Detector) ProcessEvent(guildID string, entry *discordgo.AuditLogEntry) {
	// Delegate to the main processing method
	d.ProcessEventWithGuild(guildID, entry)
}

func (d *Detector) ProcessEventWithGuild(guildID string, entry *discordgo.AuditLogEntry) {
	// Start latency measurement immediately
	start := time.Now()

	// TRACE: Log EVERY event received - COMMENTED OUT FOR SPEED
	// log.Printf("📥 [DETECTOR] Processing event: Guild=%s, Action=%d, User=%s",
	// 	guildID, *entry.ActionType, entry.UserID)

	// Fast path 1: Validate guild ID
	if guildID == "" {
		// log.Printf("⏭️  [DETECTOR] Empty guild ID, skipping")
		return // Invalid entry
	}

	// Fast path 1.5: Ignore self (prevent bot from flagging its own punishments)
	if d.session.State.User != nil && entry.UserID == d.session.State.User.ID {
		// log.Printf("⏭️  [DETECTOR] Ignoring self action by %s", entry.UserID)
		return
	}

	// Fast path 2: Atomic config load (~50ns)
	cfg := d.cache.GetConfig(guildID)
	if cfg == nil {
		// log.Printf("⏭️  [DETECTOR] Guild %s: Config is nil, skipping", guildID)
		return
	}
	if !cfg.Enabled {
		// log.Printf("⏭️  [DETECTOR] Guild %s: AntiNuke disabled, skipping", guildID)
		return
	}

	// Map Discord action type to our action type
	actionType := mapAuditLogAction(*entry.ActionType)
	if actionType == "" {
		// log.Printf("⏭️  [DETECTOR] Unmapped action type: %d, skipping", *entry.ActionType)
		return // Not a monitored action
	}

	// PANIC MODE FAST TRACK - PRIORITY 1
	if cfg.PanicMode {
		// 1. Check Owner (Fastest)
		if entry.UserID == cfg.OwnerID {
			return
		}

		// 2. Check User Whitelist (Fast)
		if d.cache.IsWhitelisted(guildID, entry.UserID) {
			return
		}

		// 3. Check Role Whitelist
		if member, err := d.session.State.Member(guildID, entry.UserID); err == nil {
			for _, roleID := range member.Roles {
				if d.cache.IsWhitelisted(guildID, roleID) {
					return
				}
			}
		}

		// 4. EXECUTE BAN IMMEDIATELY
		// No rate limit check, no further logic.
		go func() {
			d.session.GuildBanCreateWithReason(guildID, entry.UserID, "AntiNuke: Panic Mode Violation", 0)
		}()

		// 5. Async Logging
		detectionLatency := time.Since(start)
		// log.Printf("🚨 [DETECTOR] Panic Mode triggered: %s by %s in %v", actionType, entry.UserID, detectionLatency)

		violation := &ViolationEvent{
			GuildID:          guildID,
			ActionType:       actionType,
			ExecutorID:       entry.UserID,
			Count:            1,
			Limit:            0,
			DetectionLatency: detectionLatency,
			LogsChannel:      cfg.LogsChannel,
		}
		select {
		case d.loggingQueue <- violation:
		default:
		}
		
		// Attempt revocation in parallel
		go func() {
			revocation := &RevocationTask{
				GuildID:    guildID,
				ActionType: actionType,
				UserID:     entry.UserID,
				TargetID:   entry.TargetID,
			}
			d.revokeAction(revocation)
		}()

		return
	}

	// Fast path 6: Get action limit config (~50ns) - Moved UP check faster
	limit := d.cache.GetLimit(guildID, actionType)

	// Determine effective limits and punishment
	var limitCount int
	var windowSeconds int
	var punishment string

	// Dangerous actions that should be strict by default if not configured
	isDangerous := actionType == models.ActionAddBots ||
		actionType == models.ActionCreateWebhooks ||
		actionType == models.ActionBanMembers

	if limit != nil && limit.Enabled {
		// Normal configured limits
		limitCount = limit.LimitCount
		windowSeconds = limit.WindowSeconds
		punishment = limit.Punishment
	} else if isDangerous {
		// Dangerous action with no specific config - apply strict defaults
		limitCount = 1
		windowSeconds = 1
		punishment = models.PunishmentBan
		// log.Printf("⚠️ [DETECTOR] Dangerous action %s detected with no config - applying strict defaults", actionType)
	} else {
		// No config and not dangerous - skip
		// log.Printf("⏭️  [DETECTOR] No limit config for action %s, skipping", actionType)
		return
	}

	executorID := entry.UserID

	// Fast path 3: Guild owner bypass (~100ns)
	if executorID == cfg.OwnerID {
		// log.Printf("⏭️  [DETECTOR] User %s is guild owner, skipping", executorID)
		return
	}

	// Fast path 4: Whitelist check (~100ns)
	if d.cache.IsWhitelisted(guildID, executorID) {
		// log.Printf("⏭️  [DETECTOR] User %s is explicitly whitelisted, skipping", executorID)
		return
	}

	// Fast path 5: Role-based whitelist (check member roles)
	// This adds ~1-2µs but necessary for role whitelisting
	if member, err := d.session.State.Member(guildID, executorID); err == nil {
		for _, roleID := range member.Roles {
			if d.cache.IsWhitelisted(guildID, roleID) {
				// log.Printf("⏭️  [DETECTOR] User %s has whitelisted role %s, skipping", executorID, roleID)
				return
			}
		}
	}

	// Fast path 7: Atomic rate limit check (~5-10µs)
	triggered, count := d.rateLimiter.Check(
		guildID,
		actionType,
		executorID,
		limitCount,
		windowSeconds,
	)

	// If not triggered, we're done (fast path complete)
	if !triggered {
		return
	}

	// 🚨 CRITICAL PATH: Violation detected
	// PRIORITY 1: Execute punishment IMMEDIATELY
	// We spawn this goroutine BEFORE doing anything else (logging, metrics, etc.)
	go func(guildID, userID, punishment, actionType string) {
		// Construct task inside goroutine to avoid allocation on critical path
		task := &PunishmentTask{
			GuildID:    guildID,
			UserID:     userID,
			Punishment: punishment,
			Reason:     "AntiNuke: Exceeded " + actionType + " limit",
		}

		// start := time.Now()
		d.executePunishment(task)
		// latency := time.Since(start)
		// log.Printf("✓ Punished %s: %s (took %v)", userID, punishment, latency)
	}(guildID, executorID, punishment, actionType)

	// PRIORITY 2: Execute revocation
	go func(guildID, actionType, userID, targetID string) {
		revocation := &RevocationTask{
			GuildID:    guildID,
			ActionType: actionType,
			UserID:     userID,
			TargetID:   targetID,
		}

		d.revokeAction(revocation)
	}(guildID, actionType, executorID, entry.TargetID)

	// PRIORITY 3: Logging and Metrics (Post-Action)
	detectionLatency := time.Since(start)
	// log.Printf("⚡ AntiNuke triggered: %s by %s in %s (count: %d/%d, latency: %v)",
	// 	actionType, executorID, guildID, count, limitCount, detectionLatency)

	// Queue logging (async)
	violation := &ViolationEvent{
		GuildID:          guildID,
		ActionType:       actionType,
		ExecutorID:       executorID,
		Count:            count,
		Limit:            limitCount,
		DetectionLatency: detectionLatency,
		LogsChannel:      cfg.LogsChannel,
	}
	select {
	case d.loggingQueue <- violation:
	default:
		// log.Printf("⚠️ Logging queue full, dropping event")
	}
}

// mapAuditLogAction maps Discord audit log actions to our action types
// Inlined for performance (no function call overhead)
func mapAuditLogAction(actionType discordgo.AuditLogAction) string {
	switch actionType {
	case discordgo.AuditLogActionMemberBanAdd:
		return models.ActionBanMembers
	case discordgo.AuditLogActionMemberKick:
		return models.ActionKickMembers
	case discordgo.AuditLogActionRoleDelete:
		return models.ActionDeleteRoles
	case discordgo.AuditLogActionRoleCreate:
		return models.ActionCreateRoles
	case discordgo.AuditLogActionChannelDelete:
		return models.ActionDeleteChannels
	case discordgo.AuditLogActionChannelCreate:
		return models.ActionCreateChannels
	case discordgo.AuditLogActionBotAdd:
		return models.ActionAddBots
	case discordgo.AuditLogActionMemberPrune:
		return models.ActionPruneMembers
	case discordgo.AuditLogActionWebhookCreate:
		return models.ActionCreateWebhooks
	case discordgo.AuditLogActionEmojiDelete:
		return models.ActionDeleteEmojis
	default:
		return ""
	}
}

// punishmentWorker processes punishment queue asynchronously
// Deprecated: Now executed immediately in goroutines
func (d *Detector) punishmentWorker() {
	// No-op
}

// executePunishment applies the configured punishment
func (d *Detector) executePunishment(task *PunishmentTask) error {
	switch task.Punishment {
	case models.PunishmentBan:
		return d.session.GuildBanCreateWithReason(task.GuildID, task.UserID, task.Reason, 0)

	case models.PunishmentKick:
		return d.session.GuildMemberDeleteWithReason(task.GuildID, task.UserID, task.Reason)

	case models.PunishmentTimeout:
		timeout := time.Now().Add(1 * time.Hour)
		return d.session.GuildMemberTimeout(task.GuildID, task.UserID, &timeout)

	case models.PunishmentQuarantine:
		member, err := d.session.GuildMember(task.GuildID, task.UserID)
		if err != nil {
			return err
		}
		// Remove all roles
		for _, roleID := range member.Roles {
			d.session.GuildMemberRoleRemove(task.GuildID, task.UserID, roleID)
		}
		return nil

	default:
		log.Printf("⚠️ Unknown punishment type: %s", task.Punishment)
		return nil
	}
}

// revocationWorker processes revocation queue asynchronously
// Deprecated: Now executed immediately in goroutines
func (d *Detector) revocationWorker() {
	// No-op
}

// revokeAction attempts to undo an action - AGGRESSIVE MODE
func (d *Detector) revokeAction(task *RevocationTask) bool {
	if task.TargetID == "" {
		log.Printf("⚠️ Cannot revoke %s: no target ID", task.ActionType)
		return false
	}

	var err error
	var success = false

	log.Printf("🔄 Attempting to revoke %s action on %s...", task.ActionType, task.TargetID)

	switch task.ActionType {
	case models.ActionCreateChannels:
		_, err = d.session.ChannelDelete(task.TargetID)
		if err == nil {
			log.Printf("✅ DELETED malicious channel %s", task.TargetID)
			success = true
		} else {
			log.Printf("❌ Failed to delete channel: %v", err)
		}

	case models.ActionCreateRoles:
		err = d.session.GuildRoleDelete(task.GuildID, task.TargetID)
		if err == nil {
			log.Printf("✅ DELETED malicious role %s", task.TargetID)
			success = true
		} else {
			log.Printf("❌ Failed to delete role: %v", err)
		}

	case models.ActionBanMembers:
		err = d.session.GuildBanDelete(task.GuildID, task.TargetID)
		if err == nil {
			log.Printf("✅ UNBANNED user %s", task.TargetID)
			success = true
		} else {
			log.Printf("❌ Failed to unban: %v", err)
		}

	case models.ActionCreateWebhooks:
		err = d.session.WebhookDelete(task.TargetID)
		if err == nil {
			log.Printf("✅ DELETED malicious webhook %s", task.TargetID)
			success = true
		} else {
			log.Printf("❌ Failed to delete webhook: %v", err)
		}

	default:
		log.Printf("⚠️ No revocation available for: %s", task.ActionType)
	}

	return success
}

// loggingWorker processes logging queue asynchronously
func (d *Detector) loggingWorker() {
	for violation := range d.loggingQueue {
		if violation.LogsChannel == "" {
			continue
		}

		// Create embed with violation details
		embed := &discordgo.MessageEmbed{
			Title:     "🛡️ AntiNuke Violation Detected",
			Color:     0xed4245,
			Timestamp: time.Now().Format(time.RFC3339),
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Action",
					Value:  models.GetActionDisplayName(violation.ActionType),
					Inline: true,
				},
				{
					Name:   "Executor",
					Value:  "<@" + violation.ExecutorID + ">",
					Inline: true,
				},
				{
					Name:   "Count",
					Value:  fmt.Sprintf("%d/%d", violation.Count, violation.Limit),
					Inline: true,
				},
				{
					Name:   "Detection Speed",
					Value:  fmt.Sprintf("%.2fµs", float64(violation.DetectionLatency.Microseconds())),
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("Target: <300µs | Actual: %.2fµs", float64(violation.DetectionLatency.Microseconds())),
			},
		}

		d.session.ChannelMessageSendEmbed(violation.LogsChannel, embed)
	}
}
