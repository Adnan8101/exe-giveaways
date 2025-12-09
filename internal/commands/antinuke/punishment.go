package antinuke

import (
	"discord-giveaway-bot/internal/database"
	"discord-giveaway-bot/internal/models"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var Punishment = &discordgo.ApplicationCommand{
	Name:        "punishment",
	Description: "Set the punishment type for AntiNuke violations",
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
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "punishment",
			Description: "Punishment to apply when limit is exceeded",
			Required:    true,
			Choices:     GetPunishmentChoices(),
		},
	},
}

// HandlePunishment handles the /punishment command
func HandlePunishment(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database) {
	options := i.ApplicationCommandData().Options

	actionType := getStringOption(options, "action")
	punishment := getStringOption(options, "punishment")

	// Update the punishment
	err := db.UpdateActionPunishment(i.GuildID, actionType, punishment)
	if err != nil {
		respondWithError(s, i, "Database Error", "Failed to update punishment: "+err.Error())
		return
	}

	// Get updated config
	config, err := db.GetActionConfig(i.GuildID, actionType)
	if err != nil {
		respondWithError(s, i, "Database Error", "Failed to retrieve configuration: "+err.Error())
		return
	}

	embed := CreateSuccessEmbed(
		"Punishment Updated",
		"",
	)
	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:   "Action",
			Value:  models.GetActionDisplayName(actionType),
			Inline: true,
		},
		{
			Name:   "New Punishment",
			Value:  strings.Title(config.Punishment),
			Inline: true,
		},
		{
			Name:   "Current Limit",
			Value:  models.FormatInt(config.LimitCount) + " actions in " + models.FormatWindowTime(config.WindowSeconds),
			Inline: false,
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
