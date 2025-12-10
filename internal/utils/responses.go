package utils

import (
	"github.com/bwmarrin/discordgo"
)

// SendError sends an ephemeral error message
func SendError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "❌ Error",
					Description: message,
					Color:       0xFF0000,
				},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

// SendSuccess sends an ephemeral success message
func SendSuccess(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{
				{
					Description: message,
					Color:       0x00FF00,
				},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}
