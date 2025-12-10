package antinuke

import (
	"github.com/bwmarrin/discordgo"
)

var (
	// Permissions
	adminPerms = int64(discordgo.PermissionAdministrator)

	// Base Command for /antinuke
	AntiNukeCmd = &discordgo.ApplicationCommand{
		Name:        "antinuke",
		Description: "Configure the High-Performance AntiNuke System",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "enable",
				Description: "Enable the AntiNuke system for this server",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "disable",
				Description: "Disable the AntiNuke system",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "status",
				Description: "View current AntiNuke status and configuration",
			},
		},
		DefaultMemberPermissions: &adminPerms,
	}

	// /setlimit
	SetLimit = &discordgo.ApplicationCommand{
		Name:        "setlimit",
		Description: "Configure rate limits for specific actions",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "action",
				Description: "The action to limit",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Ban Members", Value: "ban_members"},
					{Name: "Kick Members", Value: "kick_members"},
					{Name: "Channel Delete", Value: "delete_channels"},
					{Name: "Role Delete", Value: "delete_roles"},
					{Name: "Bot Add", Value: "add_bots"},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "limit",
				Description: "Max number of actions allowed",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "seconds",
				Description: "Time window in seconds",
				Required:    true,
			},
		},
		DefaultMemberPermissions: &adminPerms,
	}

	// /punishment
	Punishment = &discordgo.ApplicationCommand{
		Name:        "punishment",
		Description: "Set punishment type for violations",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "action",
				Description: "The action to configure",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Ban Members", Value: "ban_members"},
					{Name: "Kick Members", Value: "kick_members"},
					{Name: "Channel Delete", Value: "delete_channels"},
					{Name: "Role Delete", Value: "delete_roles"},
					{Name: "All Actions", Value: "all"},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "type",
				Description: "Punishment to apply",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Ban", Value: "ban"},
					{Name: "Kick", Value: "kick"},
					{Name: "Quarantine (Remove Roles)", Value: "quarantine"},
				},
			},
		},
		DefaultMemberPermissions: &adminPerms,
	}

	// /whitelist
	Whitelist = &discordgo.ApplicationCommand{
		Name:        "whitelist",
		Description: "Manage trusted users/roles",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "add",
				Description: "Add a user or role to whitelist",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionUser,
						Name:        "user",
						Description: "User to whitelist",
						Required:    false,
					},
					{
						Type:        discordgo.ApplicationCommandOptionRole,
						Name:        "role",
						Description: "Role to whitelist",
						Required:    false,
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "remove",
				Description: "Remove a user or role from whitelist",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionUser,
						Name:        "user",
						Description: "User to remove",
						Required:    false,
					},
					{
						Type:        discordgo.ApplicationCommandOptionRole,
						Name:        "role",
						Description: "Role to remove",
						Required:    false,
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "list",
				Description: "List all whitelisted entities",
			},
		},
		DefaultMemberPermissions: &adminPerms,
	}

	// /panic_mode
	PanicModeCmd = &discordgo.ApplicationCommand{
		Name:                     "panic",
		Description:              "Toggle Panic Mode (Locks down server)",
		DefaultMemberPermissions: &adminPerms,
	}

	// /logs
	Logs = &discordgo.ApplicationCommand{
		Name:        "logs",
		Description: "Set the log channel for AntiNuke events",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionChannel,
				Name:         "channel",
				Description:  "Channel to send alerts to",
				Required:     true,
				ChannelTypes: []discordgo.ChannelType{discordgo.ChannelTypeGuildText},
			},
		},
		DefaultMemberPermissions: &adminPerms,
	}
)
