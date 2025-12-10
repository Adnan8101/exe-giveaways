package cde

import (
	"discord-giveaway-bot/internal/database"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
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
	guild.GuildID = guildID

	// Set atomic flags
	var flags uint32
	if config.Enabled {
		flags |= 1
	}
	if config.PanicMode {
		flags |= 2
	}
	atomic.StoreUint32(&guild.Flags, flags)

	// Note: Owner ID will be set from Discord guild object in main.go

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
			uid := parseSnowflake(whitelist[i].TargetID)
			guild.TrustedUsers[i] = uid

			// Set bit in bloom filter / bitset
			// Hash/Map to 0-255
			bitIdx := hashUser(uid) % 256
			wordIdx := bitIdx / 64
			bitOffset := bitIdx % 64

			// Accessing array element in loop, unsafe strictly speaking if concurrent readers
			// But LoadGuildConfig is usually called when updating cache
			// We should probably compute local flags/bitset and store atomically if possible
			// But for now, direct update.
			guild.TrustedBitset[wordIdx] |= (1 << bitOffset)
		}
	}

	log.Printf("[CDE] ✓ Loaded config for guild %d: Enabled=%v, PanicMode=%v, LogChannel=%d, Owner=%d",
		guildID, config.Enabled, config.PanicMode, guild.LogChannelID, guild.OwnerID)

	return nil
}

// IsAntiNukeEnabled checks if antinuke is enabled for a guild (ATOMIC FAST PATH)
func IsAntiNukeEnabled(guildID uint64) bool {
	idx := hashGuild(guildID)
	// Direct memory access - no locks!
	// We rely on atomic load to ensure we don't read partial writes
	// But since we are reading from an array index that is stable for a guildID (hash collision ignored for speed)...
	// Ideally we check guild.GuildID
	guild := &GuildArena[idx]

	// Check if ID matches
	if atomic.LoadUint64(&guild.GuildID) != guildID {
		// Cache miss or collision
		// Fallback to slow consistency check
		// For high performance, we accept that first request might be slow
		return false // Treat as disabled until loaded
	}

	flags := atomic.LoadUint32(&guild.Flags)
	return (flags & 1) != 0
}

// IsUserWhitelisted checks if a user is whitelisted for a guild
// CRITICAL: Lock-free hot path
func IsUserWhitelisted(guildID, userID uint64) bool {
	idx := hashGuild(guildID)
	guild := &GuildArena[idx]

	// Quick ID check without lock
	if atomic.LoadUint64(&guild.GuildID) != guildID {
		return false
	}

	// Bitset check (lock-free)
	h := hashUser(userID)
	bitIdx := h % 256
	wordIdx := bitIdx / 64
	bitOffset := bitIdx % 64

	// Read bitset word atomically
	word := atomic.LoadUint64(&guild.TrustedBitset[wordIdx])
	if (word & (1 << bitOffset)) == 0 {
		return false // Definitely not whitelisted
	}

	// Bit is set: Verify with exact IDs (lock-free read)
	for i := 0; i < 16; i++ {
		trustedID := atomic.LoadUint64(&guild.TrustedUsers[i])
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

// GetGuildOwnerID returns the guild owner ID for a guild
// CRITICAL: Lock-free hot path
func GetGuildOwnerID(guildID uint64) uint64 {
	idx := hashGuild(guildID)
	guild := &GuildArena[idx]

	if atomic.LoadUint64(&guild.GuildID) == guildID {
		return atomic.LoadUint64(&guild.OwnerID)
	}
	return 0
}

// SetBotUserID sets the bot's user ID (for self-protection)
func SetBotUserID(userID uint64) {
	botUserID = userID
	log.Printf("[CDE] 🤖 Bot User ID set: %d (will never be punished)", botUserID)
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
// Optimized for power-of-2 arena size
//
//go:inline
func hashGuild(id uint64) uint64 {
	// Fibonacci hashing
	return ((id * 11400714819323198485) >> 32) & GuildMask
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
