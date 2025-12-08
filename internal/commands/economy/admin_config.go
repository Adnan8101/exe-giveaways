package economy

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/models"
	"discord-giveaway-bot/internal/services"
	"discord-giveaway-bot/internal/utils"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var Economy = &discordgo.ApplicationCommand{
	Name:        "economy",
	Description: "Admin economy commands",
	Options: []*discordgo.ApplicationCommandOption{
		// Set Rewards
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "set-message-reward",
			Description: "Set coins per message",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "Amount", Required: true},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "set-vc-reward",
			Description: "Set coins per minute in VC",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "Amount", Required: true},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "set-daily",
			Description: "Set daily reward amount",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "Amount", Required: true},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "set-weekly",
			Description: "Set weekly reward amount",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "Amount", Required: true},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "set-hourly",
			Description: "Set hourly reward amount",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "Amount", Required: true},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "set-invite-reward",
			Description: "Set coins per invite",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "Amount", Required: true},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "set-react-reward",
			Description: "Set coins per reaction",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "Amount", Required: true},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "set-poll-reward",
			Description: "Set coins per poll vote",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "Amount", Required: true},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "set-event-reward",
			Description: "Set coins per event join",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "Amount", Required: true},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "set-upvote-reward",
			Description: "Set coins per upvote",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "Amount", Required: true},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "set-max-gamble",
			Description: "Set max gamble amount",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "amount", Description: "Amount", Required: true},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "set-currency-emoji",
			Description: "Set custom currency emoji",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "emoji", Description: "Emoji to use", Required: true},
			},
		},

		// Toggle Gamble
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "toggle-gamble",
			Description: "Enable or disable gambling",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "state",
					Description: "On or Off",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "On", Value: "on"},
						{Name: "Off", Value: "off"},
					},
				},
			},
		},
		// Reset
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "reset",
			Description: "Reset the economy",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "confirm",
					Description: "Type 'CONFIRM' to proceed",
					Required:    true,
				},
			},
		},
		// Stats
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "stats",
			Description: "View economy stats",
		},
	},
}

func EconomyCmd(ctx framework.Context, service *services.EconomyService) {
	// Check admin permissions
	if ctx.GetMember().Permissions&discordgo.PermissionAdministrator == 0 {
		ctx.ReplyEphemeral(utils.EmojiCross + " You need Administrator permissions to use this command.")
		return
	}

	var subCommand string
	var args []string

	if slashCtx, ok := ctx.(*framework.SlashContext); ok {
		subCommand = slashCtx.Interaction.ApplicationCommandData().Options[0].Name
		// For slash commands, options are nested
	} else if prefixCtx, ok := ctx.(*framework.PrefixContext); ok {
		if len(prefixCtx.Args) < 1 {
			ctx.Reply("Usage: `!economy <subcommand> [args]`")
			return
		}
		subCommand = prefixCtx.Args[0]
		args = prefixCtx.Args[1:]
	}

	config, err := service.GetConfig(ctx.GetGuildID())
	if err != nil {
		config = &models.EconomyConfig{GuildID: ctx.GetGuildID()}
	}

	var msg string

	switch subCommand {
	case "set-message-reward":
		var amount int64
		if slashCtx, ok := ctx.(*framework.SlashContext); ok {
			amount = slashCtx.Interaction.ApplicationCommandData().Options[0].Options[0].IntValue()
		} else {
			if len(args) < 1 {
				ctx.Reply(fmt.Sprintf("Usage: `!economy %s <amount>`", subCommand))
				return
			}
			parsed, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				ctx.Reply(utils.EmojiCross + " Invalid amount.")
				return
			}
			amount = parsed
		}
		config.MessageReward = int(amount)
		err = service.UpdateConfig(config)
		if err != nil {
			ctx.ReplyEphemeral(utils.EmojiCross + " Failed to update config.")
			return
		}

		// Send channel select menu
		if slashCtx, ok := ctx.(*framework.SlashContext); ok {
			err = ctx.GetSession().InteractionRespond(slashCtx.Interaction.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("%s Message reward set to **%d**.\n\nSelect channels where users can earn coins from messages (leave empty to allow all):", utils.EmojiTick, amount),
					Flags:   discordgo.MessageFlagsEphemeral,
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.SelectMenu{
									CustomID:    "select_allowed_channels",
									Placeholder: "Select channels...",
									MinValues:   intPtr(0),
									MaxValues:   25, // Max allowed by Discord
									MenuType:    discordgo.ChannelSelectMenu,
									ChannelTypes: []discordgo.ChannelType{
										discordgo.ChannelTypeGuildText,
									},
								},
							},
						},
					},
				},
			})
			if err != nil {
				log.Printf("Error sending select menu: %v", err)
			}
			return // Response sent manually
		} else {
			embed := &discordgo.MessageEmbed{
				Description: fmt.Sprintf("%s Message reward set to **%d**.", utils.EmojiTick, amount),
				Color:       0x00FF00,
			}
			ctx.ReplyEmbed(embed)
			return
		}

	case "set-vc-reward", "set-daily", "set-weekly", "set-hourly",
		"set-invite-reward", "set-react-reward", "set-poll-reward", "set-event-reward", "set-upvote-reward", "set-max-gamble":

		var amount int64
		if slashCtx, ok := ctx.(*framework.SlashContext); ok {
			amount = slashCtx.Interaction.ApplicationCommandData().Options[0].Options[0].IntValue()
		} else {
			if len(args) < 1 {
				ctx.Reply("Usage: `!economy <subcommand> <amount>`")
				return
			}
			parsed, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				ctx.Reply(utils.EmojiCross + " Invalid amount.")
				return
			}
			amount = parsed
		}

		switch subCommand {
		case "set-vc-reward":
			config.VCRewardPerMin = int(amount)
			msg = fmt.Sprintf("VC reward set to **%d** per minute.", amount)
		case "set-daily":
			config.DailyReward = int(amount)
			msg = fmt.Sprintf("Daily reward set to **%d**.", amount)
		case "set-weekly":
			config.WeeklyReward = int(amount)
			msg = fmt.Sprintf("Weekly reward set to **%d**.", amount)
		case "set-hourly":
			config.HourlyReward = int(amount)
			msg = fmt.Sprintf("Hourly reward set to **%d**.", amount)
		case "set-invite-reward":
			config.InviteReward = int(amount)
			msg = fmt.Sprintf("Invite reward set to **%d**.", amount)
		case "set-react-reward":
			config.ReactReward = int(amount)
			msg = fmt.Sprintf("React reward set to **%d**.", amount)
		case "set-poll-reward":
			config.PollReward = int(amount)
			msg = fmt.Sprintf("Poll reward set to **%d**.", amount)
		case "set-event-reward":
			config.EventReward = int(amount)
			msg = fmt.Sprintf("Event reward set to **%d**.", amount)
		case "set-upvote-reward":
			config.UpvoteReward = int(amount)
			msg = fmt.Sprintf("Upvote reward set to **%d**.", amount)
		case "set-max-gamble":
			config.MaxGambleAmount = int(amount)
			msg = fmt.Sprintf("Max gamble amount set to **%d**.", amount)
		}

	case "set-currency-emoji":
		var emoji string
		if slashCtx, ok := ctx.(*framework.SlashContext); ok {
			emoji = slashCtx.Interaction.ApplicationCommandData().Options[0].Options[0].StringValue()
		} else {
			if len(args) < 1 {
				ctx.Reply("Usage: `!economy set-currency-emoji <emoji>`")
				return
			}
			emoji = args[0]
		}
		config.CurrencyEmoji = emoji
		msg = fmt.Sprintf("Currency emoji set to %s", emoji)

	case "toggle-gamble":
		var state string
		if slashCtx, ok := ctx.(*framework.SlashContext); ok {
			state = slashCtx.Interaction.ApplicationCommandData().Options[0].Options[0].StringValue()
		} else {
			if len(args) < 1 {
				ctx.Reply("Usage: `!economy toggle-gamble <on/off>`")
				return
			}
			state = strings.ToLower(args[0])
		}

		if state != "on" && state != "off" {
			ctx.Reply(utils.EmojiCross + " Invalid state. Use 'on' or 'off'.")
			return
		}

		config.GambleEnabled = state == "on"
		status := "disabled"
		if config.GambleEnabled {
			status = "enabled"
		}
		msg = fmt.Sprintf("Gambling has been **%s**.", status)

	case "reset":
		// For prefix commands, we need to check the confirmation argument
		if prefixCtx, ok := ctx.(*framework.PrefixContext); ok {
			if len(prefixCtx.Args) < 2 || strings.ToUpper(prefixCtx.Args[1]) != "CONFIRM" {
				ctx.Reply(utils.EmojiCross + " Please confirm the reset by typing `!economy reset CONFIRM`.")
				return
			}
		} else if slashCtx, ok := ctx.(*framework.SlashContext); ok {
			// For slash commands, check the 'confirm' option
			confirm := slashCtx.Interaction.ApplicationCommandData().Options[0].Options[0].StringValue()
			if strings.ToUpper(confirm) != "CONFIRM" {
				ctx.ReplyEphemeral(utils.EmojiCross + " Please confirm the reset by typing 'CONFIRM'.")
				return
			}
		}

		if err := service.ResetEconomy(ctx.GetGuildID()); err != nil {
			ctx.ReplyEphemeral(utils.EmojiCross + " Failed to reset economy.")
			return
		}
		embed := &discordgo.MessageEmbed{
			Description: fmt.Sprintf("%s Economy has been reset for this guild.", utils.EmojiTick),
			Color:       0x00FF00,
		}
		ctx.ReplyEmbed(embed)
		return

	case "stats":
		totalUsers, totalCoins, err := service.GetTotalStats()
		if err != nil {
			ctx.ReplyEphemeral(utils.EmojiCross + " Failed to get stats.")
			return
		}

		embed := &discordgo.MessageEmbed{
			Title: "Economy Stats",
			Color: utils.ColorDark,
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Total Users", Value: fmt.Sprintf("%d", totalUsers), Inline: true},
				{Name: "Total Coins", Value: fmt.Sprintf("%d", totalCoins), Inline: true},
				{Name: "Message Reward", Value: fmt.Sprintf("%d", config.MessageReward), Inline: true},
				{Name: "Voice Reward", Value: fmt.Sprintf("%d/min", config.VCRewardPerMin), Inline: true},
				{Name: "Gamble Enabled", Value: fmt.Sprintf("%t", config.GambleEnabled), Inline: true},
				{Name: "Max Gamble", Value: fmt.Sprintf("%d", config.MaxGambleAmount), Inline: true},
				{Name: "Currency Emoji", Value: config.CurrencyEmoji, Inline: true},
			},
		}
		if config.AllowedChannels != "" {
			channels := strings.Split(config.AllowedChannels, ",")
			var channelMentions []string
			for _, ch := range channels {
				channelMentions = append(channelMentions, fmt.Sprintf("<#%s>", ch))
			}
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Allowed Channels",
				Value: strings.Join(channelMentions, ", "),
			})
		} else {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  "Allowed Channels",
				Value: "All Channels",
			})
		}

		ctx.ReplyEmbed(embed)
		return

	default:
		ctx.Reply(utils.EmojiCross + " Unknown subcommand.")
		return
	}

	if err := service.UpdateConfig(config); err != nil {
		ctx.ReplyEphemeral(fmt.Sprintf("%s Failed to update config: %s", utils.EmojiCross, err.Error()))
		return
	}

	embed := &discordgo.MessageEmbed{
		Description: fmt.Sprintf("%s %s", utils.EmojiTick, msg),
		Color:       0x00FF00,
	}
	ctx.ReplyEmbed(embed)
}

// HandleChannelSelect handles the interaction for selecting allowed channels
func HandleChannelSelect(s *discordgo.Session, i *discordgo.InteractionCreate, service *services.EconomyService) {
	data := i.MessageComponentData()
	if data.CustomID != "select_allowed_channels" {
		return
	}

	guildID := i.GuildID
	channels := data.Values // List of channel IDs

	config, err := service.GetConfig(guildID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " Failed to get config.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	config.AllowedChannels = strings.Join(channels, ",")
	err = service.UpdateConfig(config)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: utils.EmojiCross + " Failed to update config.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("%s Allowed channels updated! Users can earn coins in **%d** channels.", utils.EmojiTick, len(channels)),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func EconomyHandler(s *discordgo.Session, i *discordgo.InteractionCreate, service *services.EconomyService) {
	ctx := framework.NewSlashContext(s, i)
	EconomyCmd(ctx, service)
}

// intPtr returns a pointer to an int value.
func intPtr(i int) *int {
	return &i
}
