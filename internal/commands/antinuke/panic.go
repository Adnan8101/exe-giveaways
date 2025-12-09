package antinuke

import (
	"discord-giveaway-bot/internal/database"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// HandlePanicMode toggles panic mode for the guild
func HandlePanicMode(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database) {
	// Check permissions (Admin only)
	if i.Member.Permissions&discordgo.PermissionAdministrator == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ You need Administrator permissions to use this command.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Please specify enable or disable.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	subCommand := options[0].Name
	enable := false

	if subCommand == "enable" {
		enable = true
	} else if subCommand == "disable" {
		enable = false
	} else {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Invalid option.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	err := db.SetPanicMode(i.GuildID, enable)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Failed to update panic mode: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	status := "DISABLED"
	color := 0x00ff00 // Green
	if enable {
		status = "ENABLED"
		color = 0xff0000 // Red
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🚨 AntiNuke Panic Mode",
		Description: fmt.Sprintf("**Panic Mode is now %s**\n\nWhen enabled:\n• All limits set to **1/1s**\n• Punishment set to **BAN**\n• Applies to ALL actions immediately", status),
		Color:       color,
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

var PanicModeCmd = &discordgo.ApplicationCommand{
	Name:        "panic_mode",
	Description: "Enable or disable AntiNuke Panic Mode (Strict 1/1s limits)",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Name:        "enable",
			Description: "Enable Panic Mode (Strict limits)",
			Type:        discordgo.ApplicationCommandOptionSubCommand,
		},
		{
			Name:        "disable",
			Description: "Disable Panic Mode (Return to normal config)",
			Type:        discordgo.ApplicationCommandOptionSubCommand,
		},
	},
}
