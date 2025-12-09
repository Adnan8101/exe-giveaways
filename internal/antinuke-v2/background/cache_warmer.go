package background

import (
	"discord-giveaway-bot/internal/antinuke-v2/core"
	"discord-giveaway-bot/internal/database"
	"log"
)

// CacheWarmer loads config from database into atomic cache
// NO automatic refresh - only on startup and manual warm calls
type CacheWarmer struct {
	cache *core.AtomicCache
	db    *database.Database
}

// NewCacheWarmer creates a new cache warmer
func NewCacheWarmer(cache *core.AtomicCache, db *database.Database) *CacheWarmer {
	return &CacheWarmer{
		cache: cache,
		db:    db,
	}
}

// WarmAll loads config for all guilds into cache (called on startup or config change)
func (cw *CacheWarmer) WarmAll(guildIDs []string) {
	warmedCount := 0

	for _, guildID := range guildIDs {
		// Warm guild config
		if cfg, err := cw.db.GetAntiNukeConfig(guildID); err == nil {
			cw.cache.SetConfig(&core.GuildConfig{
				GuildID:     cfg.GuildID,
				Enabled:     cfg.Enabled,
				OwnerID:     cfg.OwnerID,
				LogsChannel: cfg.LogsChannel,
				PanicMode:   cfg.PanicMode,
			})

			if cfg.Enabled {
				log.Printf("  ✓ Guild %s: Enabled (PanicMode=%v)", guildID, cfg.PanicMode)
				warmedCount++
			}
		}

		// Warm whitelist
		if entries, err := cw.db.GetWhitelistEntries(guildID); err == nil && len(entries) > 0 {
			ids := make([]string, len(entries))
			for i, entry := range entries {
				ids[i] = entry.TargetID
			}
			cw.cache.SetWhitelist(guildID, ids)
			log.Printf("   ├─ %d whitelist entries", len(ids))
		}

		// Warm action limits
		if configs, err := cw.db.GetAllActionConfigs(guildID); err == nil && len(configs) > 0 {
			for _, actionCfg := range configs {
				if actionCfg.Enabled {
					cw.cache.SetLimit(&core.LimitConfig{
						GuildID:       actionCfg.GuildID,
						ActionType:    actionCfg.ActionType,
						Enabled:       actionCfg.Enabled,
						LimitCount:    actionCfg.LimitCount,
						WindowSeconds: actionCfg.WindowSeconds,
						Punishment:    actionCfg.Punishment,
					})
				}
			}
			log.Printf("   └─ %d action configs", len(configs))
		}
	}

	if warmedCount > 0 {
		log.Printf("✅ Loaded config for %d guilds with AntiNuke enabled", warmedCount)
	} else {
		log.Println("⚠️  No guilds have AntiNuke enabled")
	}
}

// WarmGuild refreshes cache for a specific guild immediately
// Used when config changes via commands
func (cw *CacheWarmer) WarmGuild(guildID string) error {
	// Warm config
	cfg, err := cw.db.GetAntiNukeConfig(guildID)
	if err != nil {
		return err
	}

	cw.cache.SetConfig(&core.GuildConfig{
		GuildID:     cfg.GuildID,
		Enabled:     cfg.Enabled,
		OwnerID:     cfg.OwnerID,
		LogsChannel: cfg.LogsChannel,
		PanicMode:   cfg.PanicMode,
	})

	// Warm whitelist
	entries, err := cw.db.GetWhitelistEntries(guildID)
	if err != nil {
		return err
	}

	ids := make([]string, len(entries))
	for i, entry := range entries {
		ids[i] = entry.TargetID
	}
	cw.cache.SetWhitelist(guildID, ids)

	// Warm action limits
	configs, err := cw.db.GetAllActionConfigs(guildID)
	if err != nil {
		return err
	}

	for _, actionCfg := range configs {
		cw.cache.SetLimit(&core.LimitConfig{
			GuildID:       actionCfg.GuildID,
			ActionType:    actionCfg.ActionType,
			Enabled:       actionCfg.Enabled,
			LimitCount:    actionCfg.LimitCount,
			WindowSeconds: actionCfg.WindowSeconds,
			Punishment:    actionCfg.Punishment,
		})
	}

	log.Printf("✓ Warmed cache for guild %s", guildID)
	return nil
}
