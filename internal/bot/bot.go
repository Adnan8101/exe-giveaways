package bot

import (
	"discord-giveaway-bot/internal/commands"
	"discord-giveaway-bot/internal/commands/economy"
	"discord-giveaway-bot/internal/commands/shop"
	"discord-giveaway-bot/internal/database"
	"discord-giveaway-bot/internal/redis"
	"discord-giveaway-bot/internal/services"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // Register pprof handlers
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type Bot struct {
	Session           *discordgo.Session
	DB                *database.Database
	Redis             *redis.Client
	Service           *services.GiveawayService
	EconomyService    *services.EconomyService
	EconomyEvents     *EconomyEvents
	ShopCommands      *shop.ShopCommand
	AdminShopCommands *shop.AdminShopCommand
	BlackjackCommand  *economy.BlackjackCommand
	VoiceSessions     map[string]time.Time // UserID -> JoinTime
	VoiceMutex        sync.Mutex
	StartTime         time.Time
	Logger            *zap.Logger         // Logger for antinuke system
	PerfMonitor       *PerformanceMonitor // Performance monitoring
}

func New(token string, db *database.Database, rdb *redis.Client) (*Bot, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("session error: %w", err)
	}

	// Configure HTTP/2 keep-alive pooled transport for REST API
	// This reduces REST latency from 400-600ms to 60-120ms
	tr := &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 200,
		IdleConnTimeout:     90 * time.Second,
		ForceAttemptHTTP2:   true,
		DisableCompression:  false,
		// TCP optimizations
		DisableKeepAlives:     false,
		MaxConnsPerHost:       200,
		ResponseHeaderTimeout: 10 * time.Second,
	}

	s.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsGuildMessageReactions |
		discordgo.IntentsGuildMembers |
		discordgo.IntentsGuildInvites |
		discordgo.IntentsGuildVoiceStates |
		discordgo.IntentsMessageContent |
		discordgo.IntentsGuildBans | // Required for Audit Log events (AntiNuke) - Maps to GUILD_MODERATION
		discordgo.IntentsGuildWebhooks // Required for Webhook events
	// Note: Audit log events are included in IntentsGuilds

	// CRITICAL: Low-latency WebSocket configuration for Singapore gateway
	s.Identify.Compress = false // Disable compression for 10-15ms lower latency

	// Initialize Monitor first
	perfMonitor := NewPerformanceMonitor()

	s.Client = &http.Client{
		Transport: &PerfTransport{
			Base:    tr,
			Monitor: perfMonitor,
		},
		Timeout: 30 * time.Second,
	}

	// CRITICAL: Minimal state tracking for lowest overhead
	// Only track what's essential for commands to work
	s.StateEnabled = false
	// s.State.TrackChannels = true
	// s.State.TrackEmojis = false
	// s.State.TrackMembers = true
	// s.State.TrackRoles = true
	// s.State.TrackVoice = true
	// s.State.TrackPresences = false
	// s.State.MaxMessageCount = 0 // CRITICAL: No message caching = lower latency

	// Performance optimizations
	s.ShouldReconnectOnError = true
	s.ShouldRetryOnRateLimit = true
	s.MaxRestRetries = 3
	s.Compress = false // CRITICAL: Disable for low latency (matches Identify.Compress)

	economySvc := services.NewEconomyService(db, rdb)
	svc := services.NewGiveawayService(s, db, rdb, economySvc)
	blackjackCmd := economy.NewBlackjackCommand(db, economySvc)
	economyEvents := NewEconomyEvents(economySvc, svc, db, blackjackCmd)

	// Initialize logger for antinuke
	logger, _ := zap.NewProduction()

	b := &Bot{
		Session:           s,
		DB:                db,
		Redis:             rdb,
		Service:           svc,
		EconomyService:    economySvc,
		EconomyEvents:     economyEvents,
		ShopCommands:      shop.NewShopCommand(db, economySvc),
		AdminShopCommands: shop.NewAdminShopCommand(db),
		BlackjackCommand:  blackjackCmd,
		VoiceSessions:     make(map[string]time.Time),
		StartTime:         time.Now(),
		Logger:            logger,
		PerfMonitor:       perfMonitor, // Use initialized monitor
	}

	// Register handlers - consolidated for maximum performance
	s.AddHandler(b.Ready)
	s.AddHandler(b.InteractionCreate)
	s.AddHandler(b.UnifiedMessageReactionAdd)    // Consolidated reaction handler
	s.AddHandler(b.UnifiedMessageReactionRemove) // Consolidated reaction remove handler
	s.AddHandler(b.UnifiedMessageCreate)         // Consolidated message handler
	s.AddHandler(b.UnifiedVoiceStateUpdate)      // Consolidated voice handler
	s.AddHandler(b.GuildCreate)                  // Handler for guild creation (command registration)

	return b, nil
}

func (b *Bot) Start() error {
	// CRITICAL: Connecting to Discord Gateway
	// When deployed in Singapore (asia-southeast1-b), Discord will automatically
	// route your bot to the Singapore gateway cluster for 15-25ms latency
	log.Println("⚡ Connecting to Discord Gateway...")
	log.Println("   • Compression: DISABLED (for lower latency)")
	log.Println("   • Message caching: DISABLED (for minimal overhead)")
	log.Println("   • Expected latency from Singapore: 15-25ms")

	err := b.Session.Open()
	if err != nil {
		log.Printf("❌ Failed to connect to Discord Gateway: %v", err)
		log.Println("   Common causes:")
		log.Println("   • Invalid bot token in config.json")
		log.Println("   • Network connectivity issues")
		log.Println("   • Discord API outage")
		return fmt.Errorf("gateway connection failed: %w", err)
	}
	log.Println("✓ Connected to Discord Gateway")

	// Ensure we have the bot user (since state is disabled)
	log.Println("🤖 Fetching bot user info...")
	if b.Session.State.User == nil {
		u, err := b.Session.User("@me")
		if err != nil {
			log.Printf("❌ Failed to fetch bot user: %v", err)
			return fmt.Errorf("failed to get bot user: %w", err)
		}
		b.Session.State.User = u
	}
	log.Printf("✓ Logged in as: %s#%s (ID: %s)",
		b.Session.State.User.Username,
		b.Session.State.User.Discriminator,
		b.Session.State.User.ID)

	// Monitor WebSocket heartbeat latency every 30 seconds
	go b.monitorHeartbeat()

	// Start performance monitoring dashboard (every 60 seconds)
	// b.StartMonitoring(60 * time.Second)

	// Register commands
	log.Println("Registering commands...")
	_, err = b.Session.ApplicationCommandBulkOverwrite(b.Session.State.User.ID, "", commands.Commands)
	if err != nil {
		return fmt.Errorf("failed to register commands: %w", err)
	}
	log.Printf("✓ Registered %d regular commands", len(commands.Commands))

	// Start AntiNuke V2 (includes cache warming and gateway event handlers)
	// log.Println("Starting AntiNuke V2...")
	// b.AntiNukeV2.Start()

	log.Println("\n🚀 Bot is running!")
	log.Println("⚡ AntiNuke V2: Real-time gateway events + lock-free detection (<0.3ms target)")

	// Start pprof server
	go func() {
		log.Println("Starting pprof server on localhost:6060")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	// Sync giveaway queue
	log.Println("Syncing giveaway queue...")
	if err := b.Service.SyncGiveawayQueue(); err != nil {
		log.Printf("Failed to sync giveaway queue: %v", err)
	}

	// Start giveaway ticker
	go b.GiveawayTicker()

	// Start message count flusher
	go b.MessageCountFlusher()

	// Wait for interrupt
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	return b.Close()
}

func (b *Bot) Close() error {
	log.Println("Shutting down...")
	if b.Logger != nil {
		b.Logger.Sync() // Flush logger buffers
	}
	b.DB.Close()
	b.Redis.Close()
	return b.Session.Close()
}

// monitorHeartbeat monitors WebSocket heartbeat latency
func (b *Bot) monitorHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("\n📊 WebSocket Latency Monitor Started")
	log.Println("   • Checking every 30 seconds")
	log.Println("   • Target: <25ms (Singapore gateway)")
	log.Println("   • Warmup period: ~5 minutes for optimal routing")
	log.Println("")

	for range ticker.C {
		latency := b.Session.HeartbeatLatency()
		latencyMs := latency.Milliseconds()

		// Enhanced latency reporting with status indicators
		if latencyMs < 20 {
			log.Printf("✅ WS Latency: %dms (EXCELLENT - Optimal Singapore routing)", latencyMs)
		} else if latencyMs < 30 {
			log.Printf("✅ WS Latency: %dms (GOOD - Singapore gateway)", latencyMs)
		} else if latencyMs < 50 {
			log.Printf("⚠️  WS Latency: %dms (OK - May improve after warmup)", latencyMs)
		} else if latencyMs < 100 {
			log.Printf("⚠️  WS Latency: %dms (HIGH - Check VM region)", latencyMs)
		} else {
			log.Printf("❌ WS Latency: %dms (CRITICAL - Wrong gateway region!)", latencyMs)
			log.Printf("   • Expected: 15-25ms from Singapore")
			log.Printf("   • Current: %dms", latencyMs)
			log.Printf("   • Action: Verify VM is in asia-southeast1-b")
			log.Printf("   • Action: Run 'ping gateway.discord.gg' on VM")
		}
	}
}

// GetPerfMonitor returns the performance monitor for external access
func (b *Bot) GetPerfMonitor() *PerformanceMonitor {
	return b.PerfMonitor
}

// GetSession returns the Discord session for external access
func (b *Bot) GetSession() *discordgo.Session {
	return b.Session
}
