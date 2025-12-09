package antinuke

import (
	"discord-giveaway-bot/internal/database"
	"discord-giveaway-bot/internal/models"

	"github.com/bwmarrin/discordgo"
)

var SetLimit = &discordgo.ApplicationCommand{
	Name:        "setlimit",
	Description: "Set the action limit for AntiNuke protection",
	DefaultMemberPermissions: func() *int64 {
		perms := int64(discordgo.PermissionAdministrator)
		return &perms
	}(),
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "action",
			Description: "Action type to configure",
			Required:    true,
			Choices:     GetAllActionChoices(),
		},
		{
			Type:        discordgo.ApplicationCommandOptionInteger,
			Name:        "limit",
			Description: "Maximum number of actions allowed",
			Required:    true,
			MinValue:    func() *float64 { v := float64(1); return &v }(),
			MaxValue:    100,
		},
	},
}

// HandleSetLimit handles the /setlimit command
func HandleSetLimit(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database) {
	options := i.ApplicationCommandData().Options

	actionType := getStringOption(options, "action")
	limit := getIntOption(options, "limit", 3)

	// Update the limit
	err := db.UpdateActionLimit(i.GuildID, actionType, int(limit))
	if err != nil {
		respondWithError(s, i, "Database Error", "Failed to update limit: "+err.Error())
		return
	}

	// Get updated config
	config, err := db.GetActionConfig(i.GuildID, actionType)
	if err != nil {
		respondWithError(s, i, "Database Error", "Failed to retrieve configuration: "+err.Error())
		return
	}

	embed := CreateSuccessEmbed(
		"Limit Updated",
		"",
	)
	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:   "Action",
			Value:  models.GetActionDisplayName(actionType),
			Inline: true,
		},
		{
			Name:   "New Limit",
			Value:  models.FormatInt(config.LimitCount) + " actions",
			Inline: true,
		},
		{
			Name:   "Time Window",
			Value:  models.FormatWindowTime(config.WindowSeconds),
			Inline: true,
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
