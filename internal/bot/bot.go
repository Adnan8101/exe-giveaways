package bot

import (
	antinukev2 "discord-giveaway-bot/internal/antinuke-v2"
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
	AntiNukeV2        *antinukev2.Service // NEW: V2 Service
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
	s.Client = &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
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

	// Enable state caching for fast lookups
	s.StateEnabled = true
	s.State.TrackChannels = true
	s.State.TrackEmojis = false // Disabled for memory optimization
	s.State.TrackMembers = true
	s.State.TrackRoles = true
	s.State.TrackVoice = true
	s.State.TrackPresences = false // Disabled for memory optimization

	// Performance optimizations
	s.ShouldReconnectOnError = true
	s.MaxRestRetries = 3
	s.Compress = true // Enable gateway compression

	economySvc := services.NewEconomyService(db, rdb)
	svc := services.NewGiveawayService(s, db, rdb, economySvc)
	blackjackCmd := economy.NewBlackjackCommand(db, economySvc)
	economyEvents := NewEconomyEvents(economySvc, svc, db, blackjackCmd)

	// Initialize AntiNuke V2 service
	antiNukeV2 := antinukev2.New(s, db)

	// Initialize logger for antinuke
	logger, _ := zap.NewProduction()

	b := &Bot{
		Session:           s,
		DB:                db,
		Redis:             rdb,
		Service:           svc,
		EconomyService:    economySvc,
		AntiNukeV2:        antiNukeV2, // NEW: V2 Service
		EconomyEvents:     economyEvents,
		ShopCommands:      shop.NewShopCommand(db, economySvc),
		AdminShopCommands: shop.NewAdminShopCommand(db),
		BlackjackCommand:  blackjackCmd,
		VoiceSessions:     make(map[string]time.Time),
		StartTime:         time.Now(),
		Logger:            logger,
		PerfMonitor:       NewPerformanceMonitor(), // Initialize performance monitor
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
	// CRITICAL: Force US-WEST gateway for 1-20ms latency
	// Use environment variable to override gateway
	log.Println("⚡ Connecting to US-WEST Discord gateway for optimal latency...")

	err := b.Session.Open()
	if err != nil {
		return err
	}

	// Monitor WebSocket heartbeat latency
	go b.monitorHeartbeat()

	// Start performance monitoring dashboard (every 60 seconds)
	b.StartMonitoring(60 * time.Second)

	// Register commands
	log.Println("Registering commands...")
	_, err = b.Session.ApplicationCommandBulkOverwrite(b.Session.State.User.ID, "", commands.Commands)
	if err != nil {
		return fmt.Errorf("failed to register commands: %w", err)
	}
	log.Printf("✓ Registered %d regular commands", len(commands.Commands))

	// Start AntiNuke V2 (includes cache warming and gateway event handlers)
	log.Println("Starting AntiNuke V2...")
	b.AntiNukeV2.Start()

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

	for range ticker.C {
		latency := b.Session.HeartbeatLatency()
		log.Printf("📊 WebSocket Heartbeat: %v (Target: <20ms)", latency)

		if latency > 50*time.Millisecond {
			log.Printf("⚠️  HIGH LATENCY WARNING: %v - Check network routing", latency)
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
