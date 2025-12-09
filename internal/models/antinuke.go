package models

import "fmt"

// AntiNukeConfig represents the guild-level antinuke configuration
type AntiNukeConfig struct {
	GuildID     string
	Enabled     bool
	LogsChannel string
	OwnerID     string // Bot owner ID that bypasses all antinuke checks
	PanicMode   bool   // If true, all limits are set to 1/1s and punishment is Ban
	CreatedAt   int64
	UpdatedAt   int64
}

// ActionConfig represents configuration for a specific action type
type ActionConfig struct {
	ID            int64
	GuildID       string
	ActionType    string
	Enabled       bool
	LimitCount    int
	WindowSeconds int
	Punishment    string
	CreatedAt     int64
	UpdatedAt     int64
}

// WhitelistEntry represents a whitelisted user or role
type WhitelistEntry struct {
	ID         int64
	GuildID    string
	TargetID   string
	TargetType string // "user" or "role"
	AddedBy    string
	CreatedAt  int64
}

// ActionEvent represents a tracked event for rate limiting
type ActionEvent struct {
	ID         int64
	GuildID    string
	ActionType string
	ExecutorID string
	TargetID   string
	Timestamp  int64
	Revoked    bool
}

// Action type constants
const (
	ActionBanMembers     = "ban_members"
	ActionKickMembers    = "kick_members"
	ActionDeleteRoles    = "delete_roles"
	ActionCreateRoles    = "create_roles"
	ActionDeleteChannels = "delete_channels"
	ActionCreateChannels = "create_channels"
	ActionAddBots        = "add_bots"
	ActionDangerousPerms = "dangerous_perms"
	ActionGiveAdminRoles = "give_admin_roles"
	ActionPruneMembers   = "prune_members"
	ActionCreateWebhooks = "create_webhooks"
	ActionDeleteEmojis   = "delete_emojis"
	ActionAll            = "all"
)

// Punishment type constants
const (
	PunishmentBan        = "ban"
	PunishmentKick       = "kick"
	PunishmentTimeout    = "timeout"
	PunishmentQuarantine = "quarantine"
)

// GetAllActionTypes returns all available action types
func GetAllActionTypes() []string {
	return []string{
		ActionBanMembers,
		ActionKickMembers,
		ActionDeleteRoles,
		ActionCreateRoles,
		ActionDeleteChannels,
		ActionCreateChannels,
		ActionAddBots,
		ActionDangerousPerms,
		ActionGiveAdminRoles,
		ActionPruneMembers,
		ActionCreateWebhooks,
		ActionDeleteEmojis,
	}
}

// GetActionDisplayName returns a human-readable name for an action type
func GetActionDisplayName(actionType string) string {
	switch actionType {
	case ActionBanMembers:
		return "Banning Members"
	case ActionKickMembers:
		return "Kicking Members"
	case ActionDeleteRoles:
		return "Deleting Roles"
	case ActionCreateRoles:
		return "Creating Roles"
	case ActionDeleteChannels:
		return "Deleting Channels"
	case ActionCreateChannels:
		return "Creating Channels"
	case ActionAddBots:
		return "Adding Bots"
	case ActionDangerousPerms:
		return "Dangerous Permissions"
	case ActionGiveAdminRoles:
		return "Giving Admin Roles"
	case ActionPruneMembers:
		return "Pruning Members"
	case ActionCreateWebhooks:
		return "Creating Webhooks"
	case ActionDeleteEmojis:
		return "Deleting Emojis"
	case ActionAll:
		return "All Actions"
	default:
		return actionType
	}
}

// ParseWindowTime parses a window time string (e.g., "10s", "1m") into seconds
func ParseWindowTime(window string) int {
	if len(window) < 2 {
		return 10 // default
	}

	unit := window[len(window)-1:]
	valueStr := window[:len(window)-1]

	// Simple parsing (production code would use proper parsing)
	value := 0
	for _, c := range valueStr {
		if c >= '0' && c <= '9' {
			value = value*10 + int(c-'0')
		}
	}

	switch unit {
	case "s":
		return value
	case "m":
		return value * 60
	case "h":
		return value * 3600
	default:
		return 10 // default
	}
}

// FormatWindowTime formats seconds into a readable string
func FormatWindowTime(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm", seconds/60)
	}
	return fmt.Sprintf("%dh", seconds/3600)
}

// FormatInt formats an integer as a string
func FormatInt(i int) string {
	return fmt.Sprintf("%d", i)
}
