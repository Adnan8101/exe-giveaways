package cde

import (
	"discord-giveaway-bot/internal/database"
	"fmt"
	"log"
	"sync"
)

// Database instance for config queries
var dbInstance *database.Database
var configMutex sync.RWMutex

// InitCDE initializes the CDE with database connection
func InitCDE(db *database.Database) {
	dbInstance = db
	log.Println("[CDE] Initialized with database connection")
}

// LoadGuildConfig loads a guild's antinuke configuration from database into cache
func LoadGuildConfig(guildID uint64) error {
	if dbInstance == nil {
		return fmt.Errorf("database not initialized")
	}

	guildIDStr := fmt.Sprintf("%d", guildID)
	config, err := dbInstance.GetAntiNukeConfig(guildIDStr)
	if err != nil {
		log.Printf("[CDE] Failed to load config for guild %d: %v", guildID, err)
		return err
	}

	// Get guild info from arena
	idx := hashGuild(guildID)
	configMutex.Lock()
	defer configMutex.Unlock()

	guild := &GuildArena[idx]
	guild.GuildID = guildID
	guild.AntiNukeEnabled = config.Enabled
	guild.PanicMode = config.PanicMode

	// Parse log channel ID if present
	if config.LogsChannel != "" {
		guild.LogChannelID = parseSnowflake(config.LogsChannel)
	}

	// Load whitelist
	whitelist, err := dbInstance.GetWhitelistEntries(guildIDStr)
	if err == nil && len(whitelist) > 0 {
		// Store first 16 whitelisted user IDs in TrustedUsers array
		count := len(whitelist)
		if count > 16 {
			count = 16
		}
		for i := 0; i < count; i++ {
			guild.TrustedUsers[i] = parseSnowflake(whitelist[i].TargetID)
		}
	}

	log.Printf("[CDE] ✓ Loaded config for guild %d: Enabled=%v, PanicMode=%v, LogChannel=%d",
		guildID, guild.AntiNukeEnabled, guild.PanicMode, guild.LogChannelID)

	return nil
}

// IsAntiNukeEnabled checks if antinuke is enabled for a guild (fast cache check)
func IsAntiNukeEnabled(guildID uint64) bool {
	idx := hashGuild(guildID)
	configMutex.RLock()
	defer configMutex.RUnlock()

	guild := &GuildArena[idx]

	// If guild not in cache or ID mismatch, try to load it
	if guild.GuildID != guildID {
		configMutex.RUnlock()
		LoadGuildConfig(guildID) // Load in background
		configMutex.RLock()
		// Re-check after load attempt
		if guild.GuildID != guildID {
			return false // Still not loaded, assume disabled
		}
	}

	return guild.AntiNukeEnabled
}

// IsUserWhitelisted checks if a user is whitelisted for a guild
func IsUserWhitelisted(guildID, userID uint64) bool {
	idx := hashGuild(guildID)
	configMutex.RLock()
	defer configMutex.RUnlock()

	guild := &GuildArena[idx]
	if guild.GuildID != guildID {
		return false
	}

	// Check TrustedUsers array (fast O(1) check for first 16)
	for _, trustedID := range guild.TrustedUsers {
		if trustedID == userID {
			return true
		}
	}

	// TODO: For more than 16 whitelisted users, check bloom filter or DB
	return false
}

// GetLogChannelID returns the log channel ID for a guild
func GetLogChannelID(guildID uint64) uint64 {
	idx := hashGuild(guildID)
	configMutex.RLock()
	defer configMutex.RUnlock()

	guild := &GuildArena[idx]
	if guild.GuildID == guildID {
		return guild.LogChannelID
	}
	return 0
}

// RefreshAllConfigs refreshes configurations for all active guilds
func RefreshAllConfigs(guildIDs []uint64) {
	log.Printf("[CDE] Refreshing configs for %d guilds...", len(guildIDs))
	for _, guildID := range guildIDs {
		if err := LoadGuildConfig(guildID); err != nil {
			log.Printf("[CDE] Failed to refresh config for guild %d: %v", guildID, err)
		}
	}
	log.Printf("[CDE] ✓ Config refresh complete")
}

// hashGuild hashes guild ID to arena index
func hashGuild(id uint64) uint64 {
	x := id
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	x ^= x >> 31
	return x % MaxGuilds
}

// parseSnowflake converts string snowflake to uint64
func parseSnowflake(s string) uint64 {
	if s == "" {
		return 0
	}
	var n uint64
	for i := 0; i < len(s); i++ {
		v := s[i] - '0'
		n = n*10 + uint64(v)
	}
	return n
}
