package antinukev2

import (
	"discord-giveaway-bot/internal/antinuke-v2/background"
	"discord-giveaway-bot/internal/antinuke-v2/core"
	"discord-giveaway-bot/internal/antinuke-v2/detector"
	"discord-giveaway-bot/internal/database"
	"log"

	"github.com/bwmarrin/discordgo"
)

// Service is the main AntiNuke V2 service coordinator
// PURE EVENT-DRIVEN - No polling, no background workers
// Config loaded ONCE on startup, then purely reactive to Discord events
type Service struct {
	session  *discordgo.Session
	db       *database.Database
	cache    *core.AtomicCache
	limiter  *core.FastRateLimiter
	detector *detector.Detector
}

// New creates a new AntiNuke V2 service
func New(session *discordgo.Session, db *database.Database) *Service {
	cache := core.NewAtomicCache()
	limiter := core.NewFastRateLimiter()

	return &Service{
		session:  session,
		db:       db,
		cache:    cache,
		limiter:  limiter,
		detector: detector.NewDetector(cache, limiter, session),
	}
}

// Start initializes the service - PURE EVENT-DRIVEN
func (s *Service) Start() {
	// Load ALL config into memory ONCE (single database hit)
	guildIDs := make([]string, 0, len(s.session.State.Guilds))
	for _, guild := range s.session.State.Guilds {
		guildIDs = append(guildIDs, guild.ID)
	}

	log.Printf("🔥 Loading AntiNuke config for %d guilds (ONE-TIME load)...", len(guildIDs))

	// Synchronous load - blocks until complete
	warmer := background.NewCacheWarmer(s.cache, s.db)
	warmer.WarmAll(guildIDs)

	log.Println("✓ AntiNuke config loaded into memory")

	// Register Discord gateway event handler - ONLY detection mechanism
	s.session.AddHandler(func(sess *discordgo.Session, e *discordgo.GuildAuditLogEntryCreate) {
		// Pure event-driven - 0ms lag
		s.detector.ProcessEvent(e.GuildID, e.AuditLogEntry)
	})

	log.Println("⚡ AntiNuke V2 started - PURE EVENT-DRIVEN (<0.3µs target)")
	log.Println("   ✓ Config loaded into memory")
	log.Println("   ✓ Listening to Discord gateway events")
	log.Println("   ✓ Zero polling, zero background workers")
}

// WarmCache manually refreshes cache for a guild (called when config changes via commands)
func (s *Service) WarmCache(guildID string) {
	warmer := background.NewCacheWarmer(s.cache, s.db)
	warmer.WarmAll([]string{guildID})
	log.Printf("🔄 Refreshed cache for guild %s", guildID)
}

// ProcessEvent manually processes an audit log entry (for testing/manual triggers)
func (s *Service) ProcessEvent(guildID string, entry *discordgo.AuditLogEntry) {
	s.detector.ProcessEvent(guildID, entry)
}
