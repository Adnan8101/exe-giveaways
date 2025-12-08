package economy

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/models"
	"discord-giveaway-bot/internal/services"
	"discord-giveaway-bot/internal/utils"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var Give = &discordgo.ApplicationCommand{
	Name:        "give",
	Description: "Transfer coins to another user",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionInteger,
			Name:        "amount",
			Description: "Amount of coins to transfer",
			Required:    true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionUser,
			Name:        "user",
			Description: "User to transfer to",
			Required:    true,
		},
	},
}

func GiveCmd(ctx framework.Context, service *services.EconomyService) {
	var amount int64
	var targetUser *discordgo.User

	if slashCtx, ok := ctx.(*framework.SlashContext); ok {
		amount = slashCtx.Interaction.ApplicationCommandData().Options[0].IntValue()
		targetUser = slashCtx.Interaction.ApplicationCommandData().Options[1].UserValue(slashCtx.Session)
	} else if prefixCtx, ok := ctx.(*framework.PrefixContext); ok {
		if len(prefixCtx.Args) < 2 {
			ctx.Reply("Usage: `!give <amount> <user>`")
			return
		}

		// Parse amount
		parsed, err := strconv.ParseInt(prefixCtx.Args[0], 10, 64)
		if err != nil {
			ctx.Reply(utils.EmojiCross + " Invalid amount.")
			return
		}
		amount = parsed

		// Parse user
		arg := prefixCtx.Args[1]
		if strings.HasPrefix(arg, "<@") && strings.HasSuffix(arg, ">") {
			id := strings.Trim(arg, "<@!>")
			u, err := ctx.GetSession().User(id)
			if err == nil {
				targetUser = u
			}
		} else {
			u, err := ctx.GetSession().User(arg)
			if err == nil {
				targetUser = u
			}
		}

		if targetUser == nil {
			ctx.Reply(utils.EmojiCross + " Invalid user.")
			return
		}
	}

	if amount <= 0 {
		ctx.Reply(utils.EmojiCross + " Amount must be positive.")
		return
	}

	if targetUser.ID == ctx.GetAuthor().ID {
		ctx.Reply(utils.EmojiCross + " You cannot give coins to yourself.")
		return
	}

	if targetUser.Bot {
		ctx.Reply(utils.EmojiCross + " You cannot give coins to bots.")
		return
	}

	config, err := service.GetConfig(ctx.GetGuildID())
	if err != nil {
		config = &models.EconomyConfig{CurrencyEmoji: "<:Cash:1443554334670327848>"}
	}

	// Check balance
	bal, err := service.GetUserBalance(ctx.GetGuildID(), ctx.GetAuthor().ID)
	if err != nil {
		ctx.Reply(utils.EmojiCross + " Error fetching balance.")
		return
	}

	if bal < amount {
		ctx.Reply(fmt.Sprintf("%s Insufficient funds. You have **%d** %s.", utils.EmojiCross, bal, config.CurrencyEmoji))
		return
	}

	// Send confirmation embed
	embed := &discordgo.MessageEmbed{
		Title:       "Confirm Transfer",
		Description: fmt.Sprintf("Are you sure you want to transfer **%d** %s to **%s**?", amount, config.CurrencyEmoji, targetUser.Username),
		Color:       0xFFA500, // Orange
		Footer: &discordgo.MessageEmbedFooter{
			Text: "This action cannot be undone.",
		},
	}

	// Buttons
	btnContinue := discordgo.Button{
		Label:    "Continue",
		Style:    discordgo.SuccessButton,
		CustomID: fmt.Sprintf("give_confirm_%d_%s_%s", amount, targetUser.ID, ctx.GetAuthor().ID),
	}
	btnCancel := discordgo.Button{
		Label:    "Cancel",
		Style:    discordgo.DangerButton,
		CustomID: fmt.Sprintf("give_cancel_%s", ctx.GetAuthor().ID),
	}

	msg := &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{btnContinue, btnCancel},
			},
		},
	}

	if slashCtx, ok := ctx.(*framework.SlashContext); ok {
		slashCtx.Session.InteractionRespond(slashCtx.Interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds:     msg.Embeds,
				Components: msg.Components,
			},
		})
	} else if prefixCtx, ok := ctx.(*framework.PrefixContext); ok {
		prefixCtx.Session.ChannelMessageSendComplex(prefixCtx.Message.ChannelID, msg)
	}
}

func HandleGiveButton(s *discordgo.Session, i *discordgo.InteractionCreate, service *services.EconomyService) {
	parts := strings.Split(i.MessageComponentData().CustomID, "_")
	action := parts[1] // confirm or cancel

	// Check user
	var authorID string
	if action == "confirm" {
		authorID = parts[4]
	} else {
		authorID = parts[2]
	}

	if i.Member.User.ID != authorID {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: utils.EmojiCross + " This is not your confirmation.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if action == "cancel" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Description: "❌ Transfer cancelled.",
						Color:       0xFF0000,
					},
				},
				Components: []discordgo.MessageComponent{},
			},
		})
		return
	}

	if action == "confirm" {
		amount, _ := strconv.ParseInt(parts[2], 10, 64)
		targetID := parts[3]

		err := service.TransferCoins(i.GuildID, authorID, targetID, amount)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("%s Error: %s", utils.EmojiCross, err.Error()),
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		targetUser, _ := s.User(targetID)
		targetName := "Unknown"
		if targetUser != nil {
			targetName = targetUser.Username
		}

		config, err := service.GetConfig(i.GuildID)
		if err != nil {
			config = &models.EconomyConfig{CurrencyEmoji: "<:Cash:1443554334670327848>"}
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Description: fmt.Sprintf("✅ Successfully transferred **%d** %s to **%s**.", amount, config.CurrencyEmoji, targetName),
						Color:       0x00FF00,
					},
				},
				Components: []discordgo.MessageComponent{},
			},
		})
	}
}
