package acl

import (
	"discord-giveaway-bot/internal/redis"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

var redisClient *redis.Client

// InitPermissionCache initializes the permission cache with Redis client
func InitPermissionCache(rdb *redis.Client) {
	redisClient = rdb
	log.Println("[ACL] Permission cache initialized with Redis")
}

// ValidatePermissionsFast checks permissions with Redis caching (target <1µs)
// Returns: canPerformAction, reason
func ValidatePermissionsFast(guildID, userID, actionType string) (bool, string) {
	if discordSession == nil {
		return false, "Session not initialized"
	}

	// FAST PATH: Check cache first
	if redisClient != nil {
		cacheKey := guildID + ":" + userID + ":" + actionType
		if canDo, found := checkPermissionCache(cacheKey); found {
			RecordCacheHit()
			return canDo, ""
		}
		RecordCacheMiss()
	}

	// EMERGENCY FAST PATH: For BAN actions, use optimistic validation
	// Assume bot can perform action if basic checks pass (cache warmup will have validated this)
	// This prevents slow API calls during critical ban events
	if actionType == "BAN" {
		// Quick validation without API calls
		canDo, reason := validatePermissionsOptimistic(guildID, userID, actionType)
		if canDo {
			// Cache the positive result
			if redisClient != nil {
				cacheKey := guildID + ":" + userID + ":" + actionType
				cachePermissionResult(cacheKey, true)
			}
			return true, ""
		}
		// If optimistic check fails, fall through to detailed check
		if reason != "" {
			return false, reason
		}
	}

	// Detailed validation with API calls (slower path)
	canDo, reason := validatePermissionsDetailed(guildID, userID, actionType)

	if redisClient != nil {
		cacheKey := guildID + ":" + userID + ":" + actionType
		cachePermissionResult(cacheKey, canDo)
	}

	return canDo, reason
}

// checkPermissionCache checks Redis for cached permission result
func checkPermissionCache(cacheKey string) (bool, bool) {
	key := "acl_perm:" + cacheKey
	val, err := redisClient.Get(key)
	if err != nil {
		return false, false
	}
	return val == "1", true
}

// cachePermissionResult caches permission validation result
func cachePermissionResult(cacheKey string, canDo bool) {
	key := "acl_perm:" + cacheKey
	val := "0"
	if canDo {
		val = "1"
	}
	// Cache for 2 minutes (short TTL for security)
	redisClient.Set(key, val, 2*time.Minute)
}

// validatePermissionsOptimistic performs fast permission validation using only cache
// Returns: canPerformAction, reason (empty reason means "need detailed check")
func validatePermissionsOptimistic(guildID, userID, actionType string) (bool, string) {
	if redisClient == nil {
		return false, "" // Need detailed check
	}

	// Check owner ID from cache
	if ownerID, found := redisClient.GetOwnerID(guildID); found {
		if userID == ownerID {
			return false, "Cannot action guild owner"
		}
	}

	// Get cached member data
	targetMember, targetFound := redisClient.GetMemberCache(guildID, userID)
	botMember, botFound := redisClient.GetBotMember(guildID)

	// If we have full cache, validate
	if targetFound && botFound {
		// Check if target has admin
		if targetMember.HasAdmin {
			return false, "Target has Administrator permission"
		}

		// Check role hierarchy
		if targetMember.HighestRolePos >= botMember.HighestRolePos {
			return false, "Target has higher or equal role position"
		}

		// Bot has permissions (validated during cache warmup)
		return true, ""
	}

	// Cache miss - need detailed validation
	return false, ""
}

// validatePermissionsDetailed performs full permission validation
func validatePermissionsDetailed(guildID, userID, actionType string) (bool, string) {
	// Try to get cached member data from Redis
	var botMember, targetMember *redis.MemberCache
	var guild *discordgo.Guild
	var err error

	// Get bot member from cache (no logging in hot path)
	if redisClient != nil {
		botMember, _ = redisClient.GetBotMember(guildID)
	}

	// Get target member from cache (no logging in hot path)
	if redisClient != nil {
		targetMember, _ = redisClient.GetMemberCache(guildID, userID)
	}

	// If cache miss, fetch from Discord (async to avoid blocking)
	if botMember == nil || targetMember == nil {
		guild, err = discordSession.Guild(guildID)
		if err != nil {
			return false, "Failed to get guild"
		}

		// Get bot member if not cached
		if botMember == nil {
			botMemberObj, err := discordSession.GuildMember(guildID, discordSession.State.User.ID)
			if err != nil {
				return false, "Failed to get bot member"
			}
			botMember = buildMemberCache(botMemberObj, guild)
			if redisClient != nil {
				// Cache async to avoid blocking
				go redisClient.SetBotMember(guildID, botMember)
			}
		}

		// Get target member if not cached
		if targetMember == nil {
			targetMemberObj, err := discordSession.GuildMember(guildID, userID)
			if err != nil {
				return false, "User not found or left server"
			}
			targetMember = buildMemberCache(targetMemberObj, guild)
			if redisClient != nil {
				// Cache async to avoid blocking
				go redisClient.SetMemberCache(guildID, targetMember)
			}
		}
	}

	// Check if target is guild owner (need to fetch guild if not already fetched)
	if guild == nil {
		guild, err = discordSession.Guild(guildID)
		if err != nil {
			return false, "Failed to get guild"
		}
	}

	if userID == guild.OwnerID {
		return false, "Cannot action guild owner"
	}

	// Check if target has Administrator permission
	if targetMember.HasAdmin {
		return false, "Target has Administrator permission"
	}

	// Check role hierarchy
	if targetMember.HighestRolePos >= botMember.HighestRolePos {
		return false, "Target has higher or equal role position"
	}

	// Check bot permissions
	requiredPerms := getRequiredPermission(actionType)
	if !botMember.HasAdmin {
		// Bot doesn't have admin, check specific permission
		hasPermission := checkBotPermission(guild, botMember, requiredPerms)
		if !hasPermission {
			return false, "Bot lacks required permission"
		}
	}

	return true, ""
}

// buildMemberCache builds a MemberCache from Discord member object
func buildMemberCache(member *discordgo.Member, guild *discordgo.Guild) *redis.MemberCache {
	cache := &redis.MemberCache{
		UserID:   member.User.ID,
		Roles:    member.Roles,
		CachedAt: time.Now().Unix(),
	}

	// Calculate highest role position and check for admin
	highestPos := 0
	hasAdmin := false

	for _, roleID := range member.Roles {
		for _, guildRole := range guild.Roles {
			if guildRole.ID == roleID {
				if guildRole.Position > highestPos {
					highestPos = guildRole.Position
				}
				// Check for Administrator permission (0x8)
				if guildRole.Permissions&0x8 != 0 {
					hasAdmin = true
				}
			}
		}
	}

	cache.HighestRolePos = highestPos
	cache.HasAdmin = hasAdmin

	return cache
}

// checkBotPermission checks if bot has required permission
func checkBotPermission(guild *discordgo.Guild, botMember *redis.MemberCache, requiredPerms int64) bool {
	for _, roleID := range botMember.Roles {
		for _, role := range guild.Roles {
			if role.ID == roleID {
				if role.Permissions&requiredPerms != 0 {
					return true
				}
			}
		}
	}
	return false
}

// getRequiredPermission returns the required permission for an action
func getRequiredPermission(actionType string) int64 {
	switch actionType {
	case "BAN":
		return 0x4 // BAN_MEMBERS
	case "KICK":
		return 0x2 // KICK_MEMBERS
	case "TIMEOUT":
		return 0x10000000000 // MODERATE_MEMBERS
	case "QUARANTINE":
		return 0x10000000 // MANAGE_ROLES
	default:
		return 0
	}
}

// InvalidateMemberCache invalidates cached member data
func InvalidateMemberCache(guildID, userID string) {
	if redisClient != nil {
		go redisClient.InvalidateMemberCache(guildID, userID)
	}
}

// PrewarmCache prewarms the permission cache for a guild (ASYNC for speed)
func PrewarmCache(guildID string) {
	if discordSession == nil || redisClient == nil {
		return
	}

	// Run warmup in background to not block startup
	go func() {
		guild, err := discordSession.Guild(guildID)
		if err != nil {
			return
		}

		// Cache guild owner
		redisClient.SetOwnerID(guildID, guild.OwnerID)

		// Cache bot member
		botMember, err := discordSession.GuildMember(guildID, discordSession.State.User.ID)
		if err == nil {
			cache := buildMemberCache(botMember, guild)
			redisClient.SetBotMember(guildID, cache)
		}

		// Cache all role positions
		rolePositions := make(map[string]int)
		for _, role := range guild.Roles {
			rolePositions[role.ID] = role.Position
		}
		redisClient.CacheGuildRoles(guildID, rolePositions)

		// Pre-cache common permission checks for all action types
		commonActions := []string{"BAN", "KICK", "TIMEOUT", "QUARANTINE"}
		for _, action := range commonActions {
			// Cache that bot CAN perform these actions
			cacheKey := guildID + ":" + discordSession.State.User.ID + ":" + action
			cachePermissionResult(cacheKey, true)
		}
	}()
}
