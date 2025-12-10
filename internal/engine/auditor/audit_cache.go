package auditor

import (
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// AuditCache stores recent audit log entries for a guild
type AuditCache struct {
	entries   []*discordgo.AuditLogEntry
	lastFetch time.Time
	mutex     sync.RWMutex
}

// AuditCacheManager manages audit log caches for all guilds
type AuditCacheManager struct {
	caches  map[string]*AuditCache // guildID -> cache
	session *discordgo.Session
	mutex   sync.RWMutex

	// Metrics
	totalFetches int
	cacheHits    int
	cacheMisses  int
}

const (
	// MinFetchInterval prevents rate limiting - fetch at most once per 200ms per guild
	MinFetchInterval = 200 * time.Millisecond

	// MaxCacheAge - cache entries older than 5 seconds are discarded
	MaxCacheAge = 5 * time.Second

	// MaxCacheSize - keep last 100 audit log entries per guild
	MaxCacheSize = 100
)

// NewAuditCacheManager creates a new audit cache manager
func NewAuditCacheManager(session *discordgo.Session) *AuditCacheManager {
	manager := &AuditCacheManager{
		caches:  make(map[string]*AuditCache),
		session: session,
	}

	// Start periodic cleanup goroutine
	go manager.periodicCleanup()

	return manager
}

// GetOrCreateCache gets or creates a cache for a guild
func (m *AuditCacheManager) GetOrCreateCache(guildID string) *AuditCache {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	cache, exists := m.caches[guildID]
	if !exists {
		cache = &AuditCache{
			entries:   make([]*discordgo.AuditLogEntry, 0, MaxCacheSize),
			lastFetch: time.Time{}, // Zero time - needs fetch
		}
		m.caches[guildID] = cache
	}

	return cache
}

// FetchAuditLogs fetches audit logs for a guild with rate limiting protection
// Returns immediately if we recently fetched (within MinFetchInterval)
func (m *AuditCacheManager) FetchAuditLogs(guildID string, actionType discordgo.AuditLogAction) error {
	cache := m.GetOrCreateCache(guildID)

	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	// Check if we can fetch (rate limit protection)
	timeSinceLastFetch := time.Since(cache.lastFetch)
	if timeSinceLastFetch < MinFetchInterval {
		// Too soon - use cached data
		log.Printf("[AUDIT-CACHE] Rate limit protection: Skipping fetch for guild %s (last fetch %v ago)",
			guildID, timeSinceLastFetch)
		return nil
	}

	// Fetch audit logs from Discord API
	log.Printf("[AUDIT-CACHE] Fetching audit logs for guild %s (action: %d)", guildID, actionType)
	auditLog, err := m.session.GuildAuditLog(guildID, "", "", int(actionType), 50)
	if err != nil {
		log.Printf("[AUDIT-CACHE] ⚠️  Failed to fetch audit logs for guild %s: %v", guildID, err)
		return err
	}

	// Update cache
	cache.entries = auditLog.AuditLogEntries
	cache.lastFetch = time.Now()
	m.totalFetches++

	log.Printf("[AUDIT-CACHE] ✅ Fetched %d audit log entries for guild %s", len(cache.entries), guildID)

	return nil
}

// GetRecentEntry finds the most recent audit log entry matching criteria
// Returns nil if no match found
func (m *AuditCacheManager) GetRecentEntry(guildID, targetID string, actionType discordgo.AuditLogAction, maxAge time.Duration) (*discordgo.AuditLogEntry, bool) {
	cache := m.GetOrCreateCache(guildID)

	cache.mutex.RLock()
	defer cache.mutex.RUnlock()

	now := time.Now()

	// Search for matching entry
	for _, entry := range cache.entries {
		// Check action type
		if entry.ActionType == nil || *entry.ActionType != actionType {
			continue
		}

		// Check age (entries have ID which encodes timestamp)
		entryAge := now.Sub(cache.lastFetch)
		if entryAge > maxAge {
			continue
		}

		// If targetID specified, must match
		// For some events (GuildUpdate, WebhooksUpdate), we accept partial matches
		if targetID != "" && entry.TargetID != "" && entry.TargetID != targetID {
			continue
		}

		// Found a match
		m.cacheHits++
		log.Printf("[AUDIT-CACHE] ✅ Cache HIT: Found entry for guild %s, action %d, user %s",
			guildID, actionType, entry.UserID)
		return entry, true
	}

	// No match found
	m.cacheMisses++
	log.Printf("[AUDIT-CACHE] ⚠️  Cache MISS: No entry found for guild %s, action %d, target %s",
		guildID, actionType, targetID)
	return nil, false
}

// GetUserIDForAction attempts to find the user who performed an action
// This is the main API used by the attribution engine
func (m *AuditCacheManager) GetUserIDForAction(guildID, targetID string, actionType discordgo.AuditLogAction) (string, bool) {
	// First try cache lookup
	entry, found := m.GetRecentEntry(guildID, targetID, actionType, MaxCacheAge)
	if found && entry.UserID != "" {
		return entry.UserID, true
	}

	// Cache miss - fetch fresh audit logs
	err := m.FetchAuditLogs(guildID, actionType)
	if err != nil {
		return "", false
	}

	// Try cache lookup again
	entry, found = m.GetRecentEntry(guildID, targetID, actionType, MaxCacheAge)
	if found && entry.UserID != "" {
		return entry.UserID, true
	}

	return "", false
}

// WarmCache preloads audit logs for all guilds on startup
func (m *AuditCacheManager) WarmCache(guilds []*discordgo.UserGuild) {
	log.Printf("[AUDIT-CACHE] 🔥 Warming audit log cache for %d guilds...", len(guilds))

	startTime := time.Now()
	successCount := 0

	// Common action types to preload
	actionTypes := []discordgo.AuditLogAction{
		discordgo.AuditLogActionChannelCreate,
		discordgo.AuditLogActionChannelDelete,
		discordgo.AuditLogActionRoleCreate,
		discordgo.AuditLogActionRoleDelete,
		discordgo.AuditLogActionMemberBanAdd,
		discordgo.AuditLogActionMemberKick,
	}

	for i, guild := range guilds {
		log.Printf("[AUDIT-CACHE] [%d/%d] Warming cache for guild: %s", i+1, len(guilds), guild.Name)

		// Fetch one action type per guild to warm the cache
		// We don't fetch all types to avoid rate limiting on startup
		err := m.FetchAuditLogs(guild.ID, actionTypes[0])
		if err == nil {
			successCount++
		}

		// Rate limit protection - 200ms between guilds
		if i < len(guilds)-1 {
			time.Sleep(MinFetchInterval)
		}
	}

	elapsed := time.Since(startTime)
	log.Printf("[AUDIT-CACHE] ✅ Cache warming complete: %d/%d guilds (took %v)",
		successCount, len(guilds), elapsed)
}

// periodicCleanup removes stale cache entries every minute
func (m *AuditCacheManager) periodicCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.cleanupStaleCaches()
	}
}

// cleanupStaleCaches removes cache entries older than MaxCacheAge
func (m *AuditCacheManager) cleanupStaleCaches() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	removedCount := 0

	for _, cache := range m.caches {
		cache.mutex.Lock()

		// If cache hasn't been used in MaxCacheAge, clear it
		if now.Sub(cache.lastFetch) > MaxCacheAge {
			cache.entries = cache.entries[:0] // Clear slice
			removedCount++
		}

		cache.mutex.Unlock()
	}

	if removedCount > 0 {
		log.Printf("[AUDIT-CACHE] 🧹 Cleaned up %d stale caches", removedCount)
	}
}

// PrintMetrics logs cache performance metrics
func (m *AuditCacheManager) PrintMetrics() {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	totalLookups := m.cacheHits + m.cacheMisses
	hitRate := float64(0)
	if totalLookups > 0 {
		hitRate = float64(m.cacheHits) / float64(totalLookups) * 100
	}

	log.Printf("[AUDIT-CACHE] Metrics:")
	log.Printf("   • Total Fetches: %d", m.totalFetches)
	log.Printf("   • Cache Hits: %d", m.cacheHits)
	log.Printf("   • Cache Misses: %d", m.cacheMisses)
	log.Printf("   • Hit Rate: %.1f%%", hitRate)
	log.Printf("   • Active Caches: %d guilds", len(m.caches))
}
