package antinuke

import (
	antinukev2 "discord-giveaway-bot/internal/antinuke-v2"
	"discord-giveaway-bot/internal/database"
	"discord-giveaway-bot/internal/models"

	"github.com/bwmarrin/discordgo"
)

var Enable = &discordgo.ApplicationCommand{
	Name:        "antinuke",
	Description: "Configure AntiNuke protection",
	DefaultMemberPermissions: func() *int64 {
		perms := int64(discordgo.PermissionAdministrator)
		return &perms
	}(),
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "enable",
			Description: "Enable AntiNuke protection for an action",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "action",
					Description: "Action type to protect against",
					Required:    true,
					Choices:     GetAllActionChoices(),
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "limit",
					Description: "Maximum number of actions allowed (default: 3)",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "window",
					Description: "Time window for rate limiting (default: 10s)",
					Required:    false,
					Choices:     GetWindowChoices(),
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "punishment",
					Description: "Punishment to apply (default: ban)",
					Required:    false,
					Choices:     GetPunishmentChoices(),
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "disable",
			Description: "Disable AntiNuke protection for an action",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "action",
					Description: "Action type to disable",
					Required:    true,
					Choices:     GetAllActionChoices(),
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "status",
			Description: "View current AntiNuke configuration",
		},
	},
}

// HandleAntiNuke handles the /antinuke command
// Now with V2 cache warming after config changes
func HandleAntiNuke(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database, antiNukeV2 *antinukev2.Service) {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		respondWithError(s, i, "Invalid command", "No subcommand provided")
		return
	}

	subcommand := options[0]

	switch subcommand.Name {
	case "enable":
		handleEnable(s, i, db, antiNukeV2, subcommand.Options)
	case "disable":
		handleDisable(s, i, db, antiNukeV2, subcommand.Options)
	case "status":
		handleStatus(s, i, db)
	default:
		respondWithError(s, i, "Invalid subcommand", "Unknown subcommand: "+subcommand.Name)
	}
}

func handleEnable(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database, antiNukeV2 *antinukev2.Service, options []*discordgo.ApplicationCommandInteractionDataOption) {
	// Defer response immediately to prevent timeout
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	// Parse options
	actionType := getStringOption(options, "action")
	limit := getIntOption(options, "limit", 3)
	windowStr := getStringOption(options, "window")
	if windowStr == "" {
		windowStr = "10s"
	}
	punishment := getStringOption(options, "punishment")
	if punishment == "" {
		punishment = models.PunishmentBan
	}

	windowSeconds := models.ParseWindowTime(windowStr)

	// Enable antinuke if not already enabled
	err := db.EnableAntiNuke(i.GuildID)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				CreateErrorEmbed("Database Error", "Failed to enable AntiNuke: "+err.Error()),
			},
		})
		return
	}

	// Set action configuration
	err = db.SetActionConfig(i.GuildID, actionType, int(limit), windowSeconds, punishment)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{
				CreateErrorEmbed("Configuration Error", "Failed to configure action: "+err.Error()),
			},
		})
		return
	}

	// IMPORTANT: Warm cache after config change (V2 integration)
	antiNukeV2.WarmCache(i.GuildID)

	// Get the updated config for display
	config, _ := db.GetActionConfig(i.GuildID, actionType)

	// Send success response
	embed := CreateConfigEmbed(actionType, config)

	// Add permission warning if punishment is ban/kick
	if punishment == models.PunishmentBan || punishment == models.PunishmentKick {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "⚠️ Permission Required",
			Value:  "Ensure the bot has **Administrator** permission or a role higher than all members to execute punishments.",
			Inline: false,
		})
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}

func handleDisable(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database, antiNukeV2 *antinukev2.Service, options []*discordgo.ApplicationCommandInteractionDataOption) {
	actionType := getStringOption(options, "action")

	err := db.DisableAction(i.GuildID, actionType)
	if err != nil {
		respondWithError(s, i, "Database Error", "Failed to disable action: "+err.Error())
		return
	}

	// IMPORTANT: Warm cache after config change (V2 integration)
	antiNukeV2.WarmCache(i.GuildID)

	embed := CreateSuccessEmbed(
		"Action Disabled",
		"AntiNuke protection for **"+models.GetActionDisplayName(actionType)+"** has been disabled.",
	)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func handleStatus(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database) {
	configs, err := db.GetAllActionConfigs(i.GuildID)
	if err != nil {
		respondWithError(s, i, "Database Error", "Failed to retrieve configuration: "+err.Error())
		return
	}

	embed := CreateStatusEmbed(configs)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

// Helper functions

func getStringOption(options []*discordgo.ApplicationCommandInteractionDataOption, name string) string {
	for _, opt := range options {
		if opt.Name == name {
			return opt.StringValue()
		}
	}
	return ""
}

func getIntOption(options []*discordgo.ApplicationCommandInteractionDataOption, name string, defaultValue int64) int64 {
	for _, opt := range options {
		if opt.Name == name {
			return opt.IntValue()
		}
	}
	return defaultValue
}

func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, title, desc string) {
	embed := CreateErrorEmbed(title, desc)
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}
