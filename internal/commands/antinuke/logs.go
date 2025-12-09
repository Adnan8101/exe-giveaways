package antinuke

import (
	"discord-giveaway-bot/internal/database"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var Logs = &discordgo.ApplicationCommand{
	Name:        "logs",
	Description: "Configure AntiNuke logging channel",
	DefaultMemberPermissions: func() *int64 {
		perms := int64(discordgo.PermissionAdministrator)
		return &perms
	}(),
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionChannel,
			Name:        "channel",
			Description: "Channel to send AntiNuke logs",
			Required:    true,
			ChannelTypes: []discordgo.ChannelType{
				discordgo.ChannelTypeGuildText,
			},
		},
	},
}

// HandleLogs handles the /logs command
func HandleLogs(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database) {
	options := i.ApplicationCommandData().Options

	channel := options[0].ChannelValue(s)
	if channel == nil {
		respondWithError(s, i, "Invalid Channel", "Could not find the specified channel")
		return
	}

	// Save logs channel to database
	err := db.SetAntiNukeLogsChannel(i.GuildID, channel.ID)
	if err != nil {
		respondWithError(s, i, "Database Error", "Failed to set logs channel: "+err.Error())
		return
	}

	// Success embed
	embed := CreateSuccessEmbed(
		"Logs Channel Configured",
		fmt.Sprintf("AntiNuke logs will now be sent to <#%s>", channel.ID),
	)

	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:   "Logged Information",
			Value:  "• Action detected timestamp\n• Executor information\n• Detection latency\n• Punishment applied\n• Punishment execution time\n• Actions revoked",
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
