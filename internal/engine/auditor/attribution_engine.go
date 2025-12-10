package auditor

import (
	"discord-giveaway-bot/internal/engine/fdl"
	"discord-giveaway-bot/internal/engine/ring"
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// PendingEvent represents an event waiting for user attribution
type PendingEvent struct {
	event      *fdl.FastEvent
	guildID    string
	targetID   string
	actionType discordgo.AuditLogAction
	receivedAt time.Time
	retries    int
}

// AttributionEngine handles delayed attribution of events to users
type AttributionEngine struct {
	pendingEvents []PendingEvent
	mutex         sync.Mutex
	ringBuffer    *ring.RingBuffer
	auditCache    *AuditCacheManager
	batchWindow   time.Duration
	ticker        *time.Ticker
	stopChan      chan struct{}
}

const (
	// DefaultBatchWindow is the time to wait before processing a batch of events
	DefaultBatchWindow = 300 * time.Millisecond

	// MaxRetries for attribution attempts
	MaxRetries = 3

	// RetryDelays for different event types
	ImmediateRetry = 0 * time.Millisecond
	ShortRetry     = 200 * time.Millisecond
	LongRetry      = 400 * time.Millisecond
)

// NewAttributionEngine creates a new attribution engine
func NewAttributionEngine(ringBuffer *ring.RingBuffer, auditCache *AuditCacheManager) *AttributionEngine {
	return &AttributionEngine{
		pendingEvents: make([]PendingEvent, 0, 100),
		ringBuffer:    ringBuffer,
		auditCache:    auditCache,
		batchWindow:   DefaultBatchWindow,
		stopChan:      make(chan struct{}),
	}
}

// Start begins the attribution engine processing loop
func (a *AttributionEngine) Start() {
	log.Println("[ATTRIBUTION] 🚀 Starting delayed attribution engine...")
	log.Printf("[ATTRIBUTION] Batch window: %v", a.batchWindow)

	a.ticker = time.NewTicker(a.batchWindow)

	go func() {
		for {
			select {
			case <-a.ticker.C:
				a.processBatch()
			case <-a.stopChan:
				log.Println("[ATTRIBUTION] Stopping attribution engine...")
				return
			}
		}
	}()

	log.Println("[ATTRIBUTION] ✅ Attribution engine started")
}

// Stop stops the attribution engine
func (a *AttributionEngine) Stop() {
	close(a.stopChan)
	if a.ticker != nil {
		a.ticker.Stop()
	}
}

// PushEvent adds an event to the attribution queue
// The event's UserID is 0 (unknown) and will be filled by attribution
func (a *AttributionEngine) PushEvent(event *fdl.FastEvent, guildID, targetID string, actionType discordgo.AuditLogAction) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	pending := PendingEvent{
		event:      event,
		guildID:    guildID,
		targetID:   targetID,
		actionType: actionType,
		receivedAt: time.Now(),
		retries:    0,
	}

	a.pendingEvents = append(a.pendingEvents, pending)

	log.Printf("[ATTRIBUTION] 📌 Queued event: Type=%d, Guild=%s, Target=%s (queue size: %d)",
		event.ReqType, guildID, targetID, len(a.pendingEvents))
}

// processBatch processes all pending events in the current batch
func (a *AttributionEngine) processBatch() {
	a.mutex.Lock()

	if len(a.pendingEvents) == 0 {
		a.mutex.Unlock()
		return
	}

	// Take snapshot of pending events
	batch := make([]PendingEvent, len(a.pendingEvents))
	copy(batch, a.pendingEvents)
	a.pendingEvents = a.pendingEvents[:0] // Clear pending queue

	a.mutex.Unlock()

	log.Printf("[ATTRIBUTION] ⚡ Processing batch of %d events...", len(batch))

	startTime := time.Now()
	successCount := 0
	failCount := 0
	retryList := make([]PendingEvent, 0)

	// Group events by guild to batch audit log fetches
	eventsByGuild := make(map[string][]PendingEvent)
	for _, pending := range batch {
		eventsByGuild[pending.guildID] = append(eventsByGuild[pending.guildID], pending)
	}

	// Process each guild's events
	for guildID, guildEvents := range eventsByGuild {
		log.Printf("[ATTRIBUTION] Processing %d events for guild %s", len(guildEvents), guildID)

		// Fetch audit logs once for this guild
		// The cache manager will handle rate limiting
		actionTypes := make(map[discordgo.AuditLogAction]bool)
		for _, pending := range guildEvents {
			actionTypes[pending.actionType] = true
		}

		// Fetch logs for each unique action type
		for actionType := range actionTypes {
			a.auditCache.FetchAuditLogs(guildID, actionType)
		}

		// Attribute each event
		for _, pending := range guildEvents {
			success := a.attributeEvent(&pending)
			if success {
				successCount++
			} else {
				// Check if we should retry
				if a.shouldRetry(&pending) {
					pending.retries++
					retryList = append(retryList, pending)
					log.Printf("[ATTRIBUTION] ⏳ Retry queued for event (attempt %d/%d)",
						pending.retries+1, MaxRetries)
				} else {
					failCount++
					// Push with UserID=0 (unknown attacker)
					a.ringBuffer.Push(pending.event)
					log.Printf("[ATTRIBUTION] ⚠️  Failed to attribute event after %d retries, pushed with UserID=0",
						pending.retries)
				}
			}
		}
	}

	// Re-queue events for retry
	if len(retryList) > 0 {
		a.mutex.Lock()
		a.pendingEvents = append(a.pendingEvents, retryList...)
		a.mutex.Unlock()
		log.Printf("[ATTRIBUTION] 🔄 Re-queued %d events for retry", len(retryList))
	}

	elapsed := time.Since(startTime)
	log.Printf("[ATTRIBUTION] ✅ Batch complete: %d succeeded, %d failed, %d retry (took %v)",
		successCount, failCount, len(retryList), elapsed)
}

// attributeEvent attempts to attribute a single event to a user
func (a *AttributionEngine) attributeEvent(pending *PendingEvent) bool {
	// Try to get user ID from audit cache
	userID, found := a.auditCache.GetUserIDForAction(pending.guildID, pending.targetID, pending.actionType)

	if !found || userID == "" {
		log.Printf("[ATTRIBUTION] ❌ Could not attribute event: Type=%d, Guild=%s, Target=%s",
			pending.event.ReqType, pending.guildID, pending.targetID)
		return false
	}

	// Parse user ID and fill in the event
	pending.event.UserID = parseSnowflake(userID)

	// Push to ring buffer for detection
	if !a.ringBuffer.Push(pending.event) {
		fdl.EventsDropped.Inc(0)
		log.Printf("[ATTRIBUTION] ❌ Ring buffer full, event dropped!")
		return false
	}

	fdl.EventsProcessed.Inc(pending.event.UserID)

	attributionLatency := time.Since(pending.receivedAt)
	log.Printf("[ATTRIBUTION] ✅ Attributed event: Type=%d, User=%d, Guild=%s, Latency=%v",
		pending.event.ReqType, pending.event.UserID, pending.guildID, attributionLatency)

	return true
}

// shouldRetry determines if we should retry attribution for an event
func (a *AttributionEngine) shouldRetry(pending *PendingEvent) bool {
	if pending.retries >= MaxRetries {
		return false
	}

	// ChannelCreate needs more retries (audit logs are most delayed for this)
	if pending.actionType == discordgo.AuditLogActionChannelCreate {
		return pending.retries < MaxRetries
	}

	// GuildUpdate may never get reliable audit logs
	if pending.actionType == discordgo.AuditLogActionGuildUpdate {
		return pending.retries < 1 // Only retry once
	}

	return pending.retries < MaxRetries
}

// parseSnowflake converts Discord snowflake string to uint64
func parseSnowflake(s string) uint64 {
	if s == "" {
		return 0
	}
	var n uint64
	for i := 0; i < len(s); i++ {
		v := s[i] - '0'
		n = n*10 + uint64(v)
	}
	return n
}

// GetQueueSize returns the current attribution queue size
func (a *AttributionEngine) GetQueueSize() int {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return len(a.pendingEvents)
}

// PrintMetrics logs attribution engine metrics
func (a *AttributionEngine) PrintMetrics() {
	queueSize := a.GetQueueSize()
	log.Printf("[ATTRIBUTION] Metrics:")
	log.Printf("   • Queue Size: %d events", queueSize)
	log.Printf("   • Batch Window: %v", a.batchWindow)
}
