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
}

func New(token string, db *database.Database, rdb *redis.Client) (*Bot, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("session error: %w", err)
	}

	s.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsGuildMessageReactions |
		discordgo.IntentsGuildMembers |
		discordgo.IntentsGuildInvites |
		discordgo.IntentsGuildVoiceStates |
		discordgo.IntentsMessageContent

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
	svc := services.NewGiveawayService(s, db, economySvc)
	blackjackCmd := economy.NewBlackjackCommand(db, economySvc)
	economyEvents := NewEconomyEvents(economySvc, svc, db, blackjackCmd)

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
	}

	// Register handlers - consolidated for maximum performance
	// Using single registration per event type with internal routing
	s.AddHandler(b.Ready)
	s.AddHandler(b.InteractionCreate)
	s.AddHandler(b.UnifiedMessageReactionAdd)    // Consolidated reaction handler
	s.AddHandler(b.UnifiedMessageReactionRemove) // Consolidated reaction remove handler
	s.AddHandler(b.UnifiedMessageCreate)         // Consolidated message handler
	s.AddHandler(b.UnifiedVoiceStateUpdate)      // Consolidated voice handler

	return b, nil
}

func (b *Bot) Start() error {
	err := b.Session.Open()
	if err != nil {
		return err
	}

	// Register commands
	log.Println("Registering commands...")
	for _, cmd := range commands.Commands {
		if cmd.Name == "gcreate" {
			log.Printf("Registering gcreate with %d options:", len(cmd.Options))
			for _, opt := range cmd.Options {
				log.Printf("- %s (%v)", opt.Name, opt.Type)
			}
		}
	}
	_, err = b.Session.ApplicationCommandBulkOverwrite(b.Session.State.User.ID, "", commands.Commands)
	if err != nil {
		return fmt.Errorf("failed to register commands: %w", err)
	}

	log.Println("Bot is running!")

	// Start pprof server
	go func() {
		log.Println("Starting pprof server on localhost:6060")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	// Start giveaway ticker
	go b.GiveawayTicker()

	// Wait for interrupt
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	return b.Close()
}

func (b *Bot) Close() error {
	log.Println("Shutting down...")
	b.DB.Close()
	b.Redis.Close()
	return b.Session.Close()
}
