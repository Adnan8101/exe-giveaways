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

	// EXTREME GC tuning for absolute minimum latency
	// Disable GC entirely during critical operations - trade ALL memory for speed
	gcPercent := 2000 // EXTREME: 2000% - virtually disable GC
	debug.SetGCPercent(gcPercent)

	// Set memory limit to prevent OOM on 4GB RAM
	memoryLimit := int64(3200 * 1024 * 1024) // 3.2GB limit
	debug.SetMemoryLimit(memoryLimit)

	log.Println("⚙️  Runtime optimized for ABSOLUTE MINIMUM latency:")
	log.Printf("   • GOMAXPROCS: %d cores", numCPU)
	log.Printf("   • GC Percent: %d (GC virtually disabled - EXTREME SPEED MODE)", gcPercent)
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
		log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		log.Printf("🤖 Bot connected as %s#%s", r.User.Username, r.User.Discriminator)
		log.Printf("📊 Guild Discovery: Found %d guilds", len(r.Guilds))
		log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		log.Println("")

		if len(r.Guilds) == 0 {
			log.Println("⚠️  WARNING: Bot is not in any guilds!")
			log.Println("   Please invite the bot to a server to use antinuke features")
			return
		}

		log.Println("📚 Loading antinuke configurations for each guild...")
		log.Println("")

		successCount := 0
		failCount := 0
		startTime := time.Now()

		// Load config for each guild
		for i, guild := range r.Guilds {
			guildID := guild.ID

			log.Printf("[%d/%d] Loading guild: %s (ID: %s)", i+1, len(r.Guilds), guild.Name, guildID)

			// Parse guild ID to uint64
			var gid uint64
			for j := 0; j < len(guildID); j++ {
				v := guildID[j] - '0'
				gid = gid*10 + uint64(v)
			}

			// Load guild config into CDE cache
			if err := cde.LoadGuildConfig(gid); err != nil {
				log.Printf("   ❌ Failed to load config: %v", err)
				failCount++
				continue
			}

			// Get log channel and configure logger
			config, err := db.GetAntiNukeConfig(guildID)
			if err == nil {
				if config.Enabled {
					log.Printf("   ✅ AntiNuke: ENABLED")
					if config.LogsChannel != "" {
						acl.SetGuildLogChannel(guildID, config.LogsChannel)
						log.Printf("   📝 Log Channel: %s", config.LogsChannel)
					} else {
						log.Printf("   ⚠️  Log Channel: Not configured")
					}
				} else {
					log.Printf("   💤 AntiNuke: DISABLED")
				}
				successCount++
			} else {
				log.Printf("   ⚠️  Config not found, using defaults")
				successCount++
			}
			log.Println("")
		}

		elapsed := time.Since(startTime)
		log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		log.Printf("✅ Configuration Loading Complete:")
		log.Printf("   • Success: %d/%d guilds", successCount, len(r.Guilds))
		log.Printf("   • Failed: %d/%d guilds", failCount, len(r.Guilds))
		log.Printf("   • Time taken: %v", elapsed)
		log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		log.Println("")
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

	// Note: The generic Event handler has been removed.
	// Event detection is now handled by specific gateway event handlers
	// registered in the auditor package (event_handlers.go).
	// This eliminates conflicts and provides better performance.
}
