package auditor

import (
	"discord-giveaway-bot/internal/engine/fdl"
	"discord-giveaway-bot/internal/engine/ring"
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

var eventPool = sync.Pool{
	New: func() interface{} {
		return &PendingEvent{}
	},
}

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
	eventsChan  chan *PendingEvent
	ringBuffer  *ring.RingBuffer
	auditCache  *AuditCacheManager
	batchWindow time.Duration
	ticker      *time.Ticker
	stopChan    chan struct{}
}

const (
	// Ultra-aggressive batch window for faster attribution
	DefaultBatchWindow = 100 * time.Millisecond

	// MaxRetries for attribution attempts
	MaxRetries = 5 // Increased retries

	// RetryDelays for different event types
	ImmediateRetry = 0 * time.Millisecond
	ShortRetry     = 100 * time.Millisecond
	LongRetry      = 250 * time.Millisecond
)

// NewAttributionEngine creates a new attribution engine with optimized settings
func NewAttributionEngine(ringBuffer *ring.RingBuffer, auditCache *AuditCacheManager) *AttributionEngine {
	return &AttributionEngine{
		eventsChan:  make(chan *PendingEvent, 8192), // Massively increased buffer
		ringBuffer:  ringBuffer,
		auditCache:  auditCache,
		batchWindow: DefaultBatchWindow,
		stopChan:    make(chan struct{}),
	}
}

// Start begins the attribution engine processing loop with parallelization
func (a *AttributionEngine) Start() {
	log.Println("[ATTRIBUTION] 🚀 Starting ultra-fast parallel attribution engine...")
	log.Printf("[ATTRIBUTION] Batch window: %v", a.batchWindow)
	log.Println("[ATTRIBUTION] Using parallel batch processing for maximum throughput")

	a.ticker = time.NewTicker(a.batchWindow)

	// Start multiple worker goroutines for parallel processing
	numWorkers := 4 // Parallel attribution workers
	for i := 0; i < numWorkers; i++ {
		go a.attributionWorker(i)
	}

	log.Printf("[ATTRIBUTION] ✅ Attribution engine started with %d parallel workers", numWorkers)
}

// attributionWorker processes attribution in parallel
func (a *AttributionEngine) attributionWorker(id int) {
	batch := make([]*PendingEvent, 0, 512)
	
	for {
		select {
		case evt := <-a.eventsChan:
			batch = append(batch, evt)
			if len(batch) >= 400 { // Early flush if batch is full
				a.processBatch(batch)
				batch = batch[:0]
			}
		case <-a.ticker.C:
			if len(batch) > 0 {
				a.processBatch(batch)
				batch = batch[:0]
			}
		case <-a.stopChan:
			if len(batch) > 0 {
				a.processBatch(batch)
			}
			return
		}
	}
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
	// Zero-allocation: Get from pool
	pending := eventPool.Get().(*PendingEvent)
	pending.event = event
	pending.guildID = guildID
	pending.targetID = targetID
	pending.actionType = actionType
	pending.receivedAt = time.Now()
	pending.retries = 0

	select {
	case a.eventsChan <- pending:
	default:
		// Drop if full to prevent blocking hot path
		// In production, we might want to log this drop periodically
		eventPool.Put(pending)
	}
}

// processBatch processes all valid events in the current batch
func (a *AttributionEngine) processBatch(batch []*PendingEvent) {
	// No logs here - hot path

	successCount := 0
	failCount := 0
	// We don't reuse retryList here to simplifiy logic for now, or we can use another pool
	var retryList []*PendingEvent

	// Group events by guild to batch audit log fetches
	eventsByGuild := make(map[string][]*PendingEvent)
	for _, pending := range batch {
		eventsByGuild[pending.guildID] = append(eventsByGuild[pending.guildID], pending)
	}

	// Process each guild's events
	for guildID, guildEvents := range eventsByGuild {
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
			success := a.attributeEvent(pending)
			if success {
				successCount++
				// Done with this event, return to pool
				eventPool.Put(pending)
			} else {
				// Check if we should retry
				if a.shouldRetry(pending) {
					pending.retries++
					retryList = append(retryList, pending)
				} else {
					failCount++
					// Push with UserID=0 (unknown attacker)
					a.ringBuffer.Push(pending.event)
					// Failed event, but processed. Return to pool.
					eventPool.Put(pending)
				}
			}
		}
	}

	// Re-queue events for retry
	for _, retry := range retryList {
		select {
		case a.eventsChan <- retry:
		default:
			// Queue full on retry, drop and return to pool
			eventPool.Put(retry)
		}
	}
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
	// Zero-copy optimization?
	// RingBuffer.Push takes a pointer and copies it.
	// SPSC Push is efficient.
	if !a.ringBuffer.Push(pending.event) {
		fdl.EventsDropped.Inc(0)
		return false
	}

	fdl.EventsProcessed.Inc(pending.event.UserID)

	// No logging in hot path
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
	return len(a.eventsChan)
}

// PrintMetrics logs attribution engine metrics
func (a *AttributionEngine) PrintMetrics() {
	queueSize := a.GetQueueSize()
	log.Printf("[ATTRIBUTION] Metrics:")
	log.Printf("   • Queue Size: %d events", queueSize)
	log.Printf("   • Batch Window: %v", a.batchWindow)
}
