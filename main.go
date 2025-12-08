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
	// Performance optimizations
	runtime.GOMAXPROCS(runtime.NumCPU()) // Use all available CPU cores
	debug.SetGCPercent(200)              // Reduce GC frequency for better throughput

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
