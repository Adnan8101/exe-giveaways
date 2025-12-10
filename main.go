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

	log.Println("🚀 Runtime optimized for low latency:")
	log.Printf("   • GOMAXPROCS: %d cores", numCPU)
	log.Printf("   • GC Percent: %d (reduced GC frequency)", gcPercent)
	log.Printf("   • Memory Limit: %d MB", memoryLimit/(1024*1024))

	// Load config
	file, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatalf("Error reading config.json: %v", err)
	}

	var config Config
	if err := json.Unmarshal(file, &config); err != nil {
		log.Fatalf("Error parsing config.json: %v", err)
	}

	// Initialize Redis
	rdb, err := redis.New(config.Redis)
	if err != nil {
		log.Fatalf("Error initializing Redis: %v", err)
	}

	// Initialize Database
	db, err := database.NewDatabase(config.Postgres)
	if err != nil {
		log.Fatalf("Error initializing Database: %v", err)
	}

	// =========================================================================
	// HIGH-PERFORMANCE ENGINE INITIALIZATION
	// =========================================================================

	// 1. Initialize Ring Buffer (The Highway)
	eventRing := ring.New()

	// 2. Start ACL Workers (The Async Executors)
	acl.StartPunishWorker()
	acl.StartLogger()

	// 3. Start CDE Workers (The Brains)
	// Pin 2-4 workers depending on core count
	numWorkers := numCPU / 2
	if numWorkers < 2 {
		numWorkers = 2
	}

	for i := 0; i < numWorkers; i++ {
		worker := ring.Consumer{
			Ring:    eventRing,
			ID:      i,
			Handler: cde.ProcessEvent,
		}
		go worker.Start()
	}

	// 4. Start Time Ticker (1ms resolution) for zero-syscall time
	go func() {
		ticker := time.NewTicker(1 * time.Millisecond)
		for range ticker.C {
			cde.SetTime(time.Now().UnixNano())
		}
	}()

	// =========================================================================

	// Initialize bot
	b, err := bot.New(config.Token, db, rdb)
	if err != nil {
		log.Fatalf("Error initializing bot: %v", err)
	}

	// =========================================================================
	// WIRE ENGINE TO DISCORD SESSION
	// =========================================================================

	// Initialize ACL with Discord session
	acl.InitPunishWorker(b.Session)

	// Initialize logger with Discord session (will be set per-guild)
	// For now, we'll set it on the first Ready event
	// acl.InitLogger(b.Session, "log_channel_id")

	// Initialize and start audit log monitor
	auditor := auditor.New(b.Session, eventRing)
	auditor.Start()

	log.Println("✅ Engine initialization complete")
	log.Println("   • ACL Workers: Running")
	log.Println("   • CDE Workers:", numWorkers)
	log.Println("   • Audit Log Monitor: Active")
	log.Println("   • Target Detection: <3µs")

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
		fastEvt, err := fdl.ParseFrame(e.RawData)
		if err != nil {
			return // Malformed or irrelevant event
		}
		if fastEvt != nil {
			if !eventRing.Push(fastEvt) {
				fdl.EventsDropped.Inc(0) // Increment dropped counter if buffer is full
			} else {
				fdl.EventsProcessed.Inc(fastEvt.UserID) // Tentative count
			}
		}

		// Track high-res latency in PerfMonitor
		b.GetPerfMonitor().TrackEvent(time.Since(start))
	})
}
