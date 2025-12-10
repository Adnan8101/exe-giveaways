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

	// Start bot
	if err := b.Start(); err != nil {
		log.Fatalf("Error starting bot: %v", err)
	}

	// Hook into DiscordGo events for the Fast Path
	// Note: We use the Session from the bot to add a raw handler equivalent
	b.Session.AddHandler(func(s *discordgo.Session, e *discordgo.Event) {
		// FAST PATH: Feed the Ring Buffer
		// e.RawData contains the JSON of the inner data 'd' or the full event?
		// discordgo.Event.RawData is usually the full message or the data part.
		// Use ParseFrame on the RawData.

		// If RawData is empty (some events), skip
		if len(e.RawData) == 0 {
			return
		}

		// Parse (Zero Alloc-ish)
		fastEvt, err := fdl.ParseFrame(e.RawData)
		if err != nil {
			// Malformed or irrelevant event
			return
		}

		if fastEvt != nil {
			// Push to Ring (Non-blocking usually, but returns false if full)
			// If full, we increment dropped counter
			if !eventRing.Push(fastEvt) {
				fdl.EventsDropped.Inc(0)
			} else {
				fdl.EventsProcessed.Inc(fastEvt.UserID) // tentative count
			}
		}
	})
}
