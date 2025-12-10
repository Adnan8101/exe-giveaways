package main

import (
	"discord-giveaway-bot/internal/bot"
	"discord-giveaway-bot/internal/database"
	"discord-giveaway-bot/internal/redis"
	"log"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/goccy/go-json"

	// Engine Imports
	"discord-giveaway-bot/internal/engine/acl"
	"discord-giveaway-bot/internal/engine/auditor"
	"discord-giveaway-bot/internal/engine/cde"
	"discord-giveaway-bot/internal/engine/fdl"
	"discord-giveaway-bot/internal/engine/ring"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Config struct {
	Token    string                  `json:"token"`
	Redis    redis.Config            `json:"redis"`
	Postgres database.PostgresConfig `json:"postgres"`
}

func main() {
	// CRITICAL: Panic recovery to prevent silent crashes
	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ FATAL PANIC: %v", r)
			log.Printf("Stack trace:\n%s", debug.Stack())
			os.Exit(1)
		}
	}()

	log.Println("🚀 Starting Discord Giveaway Bot...")

	// CRITICAL Performance optimizations for low latency
	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU) // Use all available CPU cores

	// Aggressive GC tuning for real-time performance
	// Higher GC percentage = less frequent GC = lower latency spikes
	gcPercent := 400
	debug.SetGCPercent(gcPercent) // Increased from 200 - trade memory for speed

	// Set memory limit to prevent OOM on 4GB RAM
	memoryLimit := int64(3 * 1024 * 1024 * 1024) // 3GB limit (leave 1GB for OS)
	debug.SetMemoryLimit(memoryLimit)

	log.Println("⚙️  Runtime optimized for low latency:")
	log.Printf("   • GOMAXPROCS: %d cores", numCPU)
	log.Printf("   • GC Percent: %d (reduced GC frequency)", gcPercent)
	log.Printf("   • Memory Limit: %d MB", memoryLimit/(1024*1024))

	// Load config
	log.Println("📄 Loading config.json...")
	file, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatalf("❌ Error reading config.json: %v", err)
	}

	var config Config
	if err := json.Unmarshal(file, &config); err != nil {
		log.Fatalf("❌ Error parsing config.json: %v", err)
	}

	// Validate token
	if config.Token == "" {
		log.Fatal("❌ Discord token is empty in config.json")
	}
	log.Println("✓ Config loaded successfully")

	// Initialize Redis
	log.Println("📦 Connecting to Redis...")
	rdb, err := redis.New(config.Redis)
	if err != nil {
		log.Fatalf("❌ Error initializing Redis: %v", err)
	}
	log.Println("✓ Redis connected")

	// Initialize Database
	log.Println("🗄️  Connecting to PostgreSQL...")
	db, err := database.NewDatabase(config.Postgres)
	if err != nil {
		log.Fatalf("❌ Error initializing Database: %v", err)
	}
	log.Println("✓ Database connected")

	// =========================================================================
	// HIGH-PERFORMANCE ENGINE INITIALIZATION
	// =========================================================================
	log.Println("🔧 Initializing High-Performance Engine...")

	// 1. Initialize Ring Buffer (The Highway)
	log.Println("   • Ring Buffer...")
	eventRing := ring.New()

	// 2. Start ACL Workers (The Async Executors)
	log.Println("   • ACL Workers...")
	acl.StartPunishWorker()
	acl.StartLogger()

	// 3. Initialize CDE with Database (CRITICAL)
	log.Println("   • Initializing CDE with database...")
	cde.InitCDE(db)

	// 4. Start CDE Workers (The Brains)
	// Pin 2-4 workers depending on core count
	numWorkers := numCPU / 2
	if numWorkers < 2 {
		numWorkers = 2
	}

	log.Printf("   • Starting %d CDE Workers...", numWorkers)
	for i := 0; i < numWorkers; i++ {
		worker := ring.Consumer{
			Ring:    eventRing,
			ID:      i,
			Handler: cde.ProcessEvent,
		}
		go worker.Start()
	}

	// 4. Start Time Ticker (1ms resolution) for zero-syscall time
	log.Println("   • Time Ticker (1ms)...")
	go func() {
		ticker := time.NewTicker(1 * time.Millisecond)
		for range ticker.C {
			cde.SetTime(time.Now().UnixNano())
		}
	}()

	log.Println("✓ Engine initialization complete")

	// =========================================================================

	// Initialize bot
	log.Println("🤖 Initializing Bot Session...")
	b, err := bot.New(config.Token, db, rdb)
	if err != nil {
		log.Fatalf("❌ Error initializing bot: %v", err)
	}
	log.Println("✓ Bot session created")

	// =========================================================================
	// WIRE ENGINE TO DISCORD SESSION
	// =========================================================================

	// Initialize ACL with Discord session
	acl.InitPunishWorker(b.Session)

	// Initialize logger with Discord session (channel mapping set below)
	acl.InitLogger(b.Session)

	// Initialize and start audit log monitor
	auditor := auditor.New(b.Session, eventRing)
	auditor.Start()

	log.Println("✅ Engine initialization complete")
	log.Println("   • ACL Workers: Running")
	log.Println("   • CDE Workers:", numWorkers)
	log.Println("   • Audit Log Monitor: Active")
	log.Println("   • Target Detection: <3µs")

	// =========================================================================
	// LOAD GUILD CONFIGURATIONS
	// =========================================================================
	log.Println("📚 Loading guild configurations...")

	// Add Ready handler to load configurations when bot connects
	b.Session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("[READY] Bot connected as %s, loading configs for %d guilds", r.User.Username, len(r.Guilds))

		// Load config for each guild
		for _, guild := range r.Guilds {
			guildID := guild.ID

			// Parse guild ID to uint64
			var gid uint64
			for i := 0; i < len(guildID); i++ {
				v := guildID[i] - '0'
				gid = gid*10 + uint64(v)
			}

			// Load guild config into CDE cache
			if err := cde.LoadGuildConfig(gid); err != nil {
				log.Printf("[READY] Failed to load config for guild %s: %v", guildID, err)
				continue
			}

			// Get log channel and configure logger
			config, err := db.GetAntiNukeConfig(guildID)
			if err == nil && config.Enabled && config.LogsChannel != "" {
				acl.SetGuildLogChannel(guildID, config.LogsChannel)
				log.Printf("[READY] Set log channel for guild %s: %s", guildID, config.LogsChannel)
			}
		}

		log.Printf("[READY] ✓ Loaded configurations for %d guilds", len(r.Guilds))
	})

	// Add periodic config refresh (every 30 seconds)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			// Get all guild IDs from session
			if b.Session.State == nil {
				continue
			}

			var guildIDs []uint64
			for _, guild := range b.Session.State.Guilds {
				var gid uint64
				for i := 0; i < len(guild.ID); i++ {
					v := guild.ID[i] - '0'
					gid = gid*10 + uint64(v)
				}
				guildIDs = append(guildIDs, gid)
			}

			cde.RefreshAllConfigs(guildIDs)
		}
	}()

	// =========================================================================

	// Start bot
	if err := b.Start(); err != nil {
		log.Fatalf("Error starting bot: %v", err)
	}

	// Hook into DiscordGo events for the Fast Path
	// Note: We use the Session from the bot to add a raw handler equivalent
	// Fast Handlers for AntiNuke Engine
	b.Session.AddHandler(func(s *discordgo.Session, e *discordgo.Event) {
		start := time.Now()

		// FAST PATH: Feed the Ring Buffer
		if len(e.RawData) == 0 {
			return
		}

		log.Printf("[MAIN] Received gateway event: Type=%s, Size=%d bytes", e.Type, len(e.RawData))

		fastEvt, err := fdl.ParseFrame(e.RawData)
		if err != nil {
			log.Printf("[MAIN] Failed to parse event: %v", err)
			return // Malformed or irrelevant event
		}
		if fastEvt != nil {
			log.Printf("[MAIN] ✓ Parsed event: Type=%d, GuildID=%d, UserID=%d - Pushing to ring buffer",
				fastEvt.ReqType, fastEvt.GuildID, fastEvt.UserID)
			if !eventRing.Push(fastEvt) {
				fdl.EventsDropped.Inc(0) // Increment dropped counter if buffer is full
				log.Printf("[MAIN] ❌ Ring buffer full, event dropped!")
			} else {
				fdl.EventsProcessed.Inc(fastEvt.UserID) // Tentative count
				log.Printf("[MAIN] ✓ Event pushed to ring buffer successfully")
			}
		} else {
			log.Printf("[MAIN] Event parsed but fastEvt is nil (unknown/ignored event type)")
		}

		// Track high-res latency in PerfMonitor
		b.GetPerfMonitor().TrackEvent(time.Since(start))
	})
}
