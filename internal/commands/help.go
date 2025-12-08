package commands

import (
	"discord-giveaway-bot/internal/commands/framework"

	"github.com/bwmarrin/discordgo"
)

var Help = &discordgo.ApplicationCommand{
	Name:        "help",
	Description: "Show available commands",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "command",
			Description: "Specific command to get help for",
			Required:    false,
		},
	},
}

func HelpCmd(ctx framework.Context) {
	embed := &discordgo.MessageEmbed{
		Title:       "Bot Commands",
		Description: "Select a category below to view commands.",
		Color:       0x2b2d31, // Dark theme background (clean/colorless look)
	}

	menu := discordgo.SelectMenu{
		CustomID:    "help_category_select",
		Placeholder: "Select a category",
		Options: []discordgo.SelectMenuOption{
			{
				Label:       "Giveaways",
				Value:       "help_giveaways",
				Description: "Manage giveaways",
			},
			{
				Label:       "Economy",
				Value:       "help_economy",
				Description: "Manage coins and rewards",
			},
			{
				Label:       "Voice",
				Value:       "help_voice",
				Description: "Voice channel management",
			},
			{
				Label:       "Utility",
				Value:       "help_utility",
				Description: "General bot utilities",
			},
		},
	}

	ctx.ReplyComponent(embed, []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{menu},
		},
	})
}

func HandleHelpSelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.MessageComponentData()
	if len(data.Values) == 0 {
		return
	}

	category := data.Values[0]
	var embed *discordgo.MessageEmbed

	switch category {
	case "help_giveaways":
		embed = &discordgo.MessageEmbed{
			Title: "Giveaway Commands",
			Color: 0x2b2d31,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "/gcreate", Value: "Start a new giveaway", Inline: false},
				{Name: "/gend", Value: "End a giveaway", Inline: false},
				{Name: "/greroll", Value: "Reroll a giveaway", Inline: false},
				{Name: "/glist", Value: "List active giveaways", Inline: false},
				{Name: "/gcancel", Value: "Cancel a giveaway", Inline: false},
			},
		}
	case "help_economy":
		embed = &discordgo.MessageEmbed{
			Title: "Economy Commands",
			Color: 0x2b2d31,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "/daily", Value: "Claim daily reward", Inline: false},
				{Name: "/weekly", Value: "Claim weekly reward", Inline: false},
				{Name: "/hourly", Value: "Claim hourly reward", Inline: false},
				{Name: "/balance", Value: "Check your coin balance", Inline: false},
				{Name: "/leaderboard", Value: "View top users", Inline: false},
				{Name: "/cf <amount> [h/t]", Value: "Gamble coins", Inline: false},
			},
		}
	case "help_voice":
		embed = &discordgo.MessageEmbed{
			Title: "Voice Commands",
			Color: 0x2b2d31,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "/wv", Value: "Where Voice - Find a user", Inline: false},
				{Name: "/drag", Value: "Drag a user to your channel", Inline: false},
				{Name: "/to", Value: "Go to a user's channel", Inline: false},
				{Name: "/muteall", Value: "Mute all in channel", Inline: false},
				{Name: "/unmuteall", Value: "Unmute all in channel", Inline: false},
				{Name: "/deafenall", Value: "Deafen all in channel", Inline: false},
				{Name: "/undeafenall", Value: "Undeafen all in channel", Inline: false},
				{Name: "/vcclear", Value: "Kick all from channel", Inline: false},
			},
		}
	case "help_utility":
		embed = &discordgo.MessageEmbed{
			Title: "Utility Commands",
			Color: 0x2b2d31,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "/help", Value: "Show this menu", Inline: false},
				{Name: "/ping", Value: "Check bot latency", Inline: false},
				{Name: "/invites", Value: "Check your invites", Inline: false},
			},
		}
	}

	if embed != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
				// Keep the component so they can switch categories
				Components: i.Message.Components,
			},
		})
	}
}

func HandleHelp(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Parse options
	var args []string
	options := i.ApplicationCommandData().Options
	if len(options) > 0 {
		args = append(args, options[0].StringValue())
	}

	ctx := framework.NewSlashContextWithArgs(s, i, args)
	HelpCmd(ctx)
}
