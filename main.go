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

	// Initialize bot
	b, err := bot.New(config.Token, db, rdb)
	if err != nil {
		log.Fatalf("Error initializing bot: %v", err)
	}

	// Start bot
	if err := b.Start(); err != nil {
		log.Fatalf("Error starting bot: %v", err)
	}
}
