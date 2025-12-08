package economy

import (
	crand "crypto/rand"
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/services"
	"discord-giveaway-bot/internal/utils"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var Coinflip = &discordgo.ApplicationCommand{
	Name:        "coinflip",
	Description: "Gamble your coins",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "amount",
			Description: "Amount to gamble (or 'all', 'half')",
			Required:    true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "choice",
			Description: "Heads or Tails (default: Tails)",
			Required:    false,
			Choices: []*discordgo.ApplicationCommandOptionChoice{
				{Name: "Heads", Value: "heads"},
				{Name: "Tails", Value: "tails"},
			},
		},
	},
}

func CoinflipCmd(ctx framework.Context, service *services.EconomyService) {
	config, err := service.GetConfig(ctx.GetGuildID())
	if err != nil {
		ctx.ReplyEphemeral(utils.EmojiCross + " Failed to get config.")
		return
	}

	if !config.GambleEnabled {
		ctx.ReplyEphemeral(utils.EmojiCross + " Gambling is currently disabled.")
		return
	}

	userID := ctx.GetAuthor().ID
	balance, err := service.GetUserBalance(ctx.GetGuildID(), userID)
	if err != nil {
		ctx.ReplyEphemeral(utils.EmojiCross + " Failed to check balance.")
		return
	}

	var amount int64
	var amountStr string
	choice := "tails" // Default

	if slashCtx, ok := ctx.(*framework.SlashContext); ok {
		amountStr = slashCtx.Interaction.ApplicationCommandData().Options[0].StringValue()
		if len(slashCtx.Interaction.ApplicationCommandData().Options) > 1 {
			choice = slashCtx.Interaction.ApplicationCommandData().Options[1].StringValue()
		}
	} else if prefixCtx, ok := ctx.(*framework.PrefixContext); ok {
		if len(prefixCtx.Args) < 1 {
			ctx.Reply("Usage: `!coinflip <amount|all> [h/t]`")
			return
		}
		amountStr = prefixCtx.Args[0]

		if len(prefixCtx.Args) > 1 {
			arg := strings.ToLower(prefixCtx.Args[1])
			if arg == "h" || arg == "head" || arg == "heads" {
				choice = "heads"
			} else if arg == "t" || arg == "tail" || arg == "tails" {
				choice = "tails"
			} else {
				ctx.Reply(utils.EmojiCross + " Invalid choice. Use h/t or heads/tails.")
				return
			}
		}
	}

	// Parse amount
	amountStr = strings.ToLower(amountStr)
	if amountStr == "all" || amountStr == "max" {
		amount = balance
	} else if amountStr == "half" {
		amount = balance / 2
	} else {
		parsed, err := strconv.ParseInt(amountStr, 10, 64)
		if err != nil || parsed < 1 {
			ctx.ReplyEphemeral(utils.EmojiCross + " Invalid amount. Use a number, 'all', or 'half'.")
			return
		}
		amount = parsed
	}

	if amount < 1 {
		ctx.ReplyEphemeral(utils.EmojiCross + " Amount must be at least 1.")
		return
	}

	// Check max limit
	if config.MaxGambleAmount > 0 && int(amount) > config.MaxGambleAmount {
		// If they said "all" but it's over limit, cap it? Or error?
		// Usually error is better or cap it. Let's error to be safe.
		ctx.ReplyEphemeral(fmt.Sprintf("%s Max gamble amount is **%d**.", utils.EmojiCross, config.MaxGambleAmount))
		return
	}

	if balance < amount {
		ctx.ReplyEphemeral(fmt.Sprintf("%s You don't have enough coins! You have **%d**.", utils.EmojiCross, balance))
		return
	}

	// Send loading embed
	var msg *discordgo.Message

	spinningEmbed := &discordgo.MessageEmbed{
		Description: fmt.Sprintf("%s Spinning...", utils.EmojiCoinflip),
		Color:       utils.ColorDark,
	}

	if slashCtx, ok := ctx.(*framework.SlashContext); ok {
		err := slashCtx.Session.InteractionRespond(slashCtx.Interaction.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{spinningEmbed},
			},
		})
		if err != nil {
			return
		}
	} else {
		m, err := ctx.GetSession().ChannelMessageSendEmbed(ctx.GetChannelID(), spinningEmbed)
		if err == nil {
			msg = m
		}
	}

	// Wait 2 seconds
	time.Sleep(2 * time.Second)

	// Flip coin
	// Use crypto/rand for better randomness
	b := make([]byte, 1)
	var isHeads bool
	_, err = crand.Read(b)
	if err != nil {
		// Fallback to math/rand if crypto fails (unlikely)
		rand.Seed(time.Now().UnixNano())
		isHeads = rand.Intn(2) == 0
	} else {
		isHeads = b[0]%2 == 0
	}

	result := "tails"
	if isHeads {
		result = "heads"
	}

	var description string
	var color int

	if choice == result {
		// Win
		err = service.AddCoins(ctx.GetGuildID(), userID, amount)
		if err != nil {
			description = utils.EmojiCross + " Transaction failed."
			color = utils.ColorRed
		} else {
			description = fmt.Sprintf("You won **%d** %s\n\n**Result:** %s", amount, config.CurrencyEmoji, strings.Title(result))
			color = utils.ColorGreen
		}
	} else {
		// Lose
		err = service.RemoveCoins(ctx.GetGuildID(), userID, amount)
		if err != nil {
			description = utils.EmojiCross + " Transaction failed."
			color = utils.ColorRed
		} else {
			description = fmt.Sprintf("You lost **%d** %s\n\n**Result:** %s", amount, config.CurrencyEmoji, strings.Title(result))
			color = utils.ColorRed
		}
	}

	resultEmbed := &discordgo.MessageEmbed{
		Description: description,
		Color:       color,
	}

	// Edit message
	if slashCtx, ok := ctx.(*framework.SlashContext); ok {
		slashCtx.Session.InteractionResponseEdit(slashCtx.Interaction.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{resultEmbed},
		})
	} else if msg != nil {
		ctx.GetSession().ChannelMessageEditEmbed(ctx.GetChannelID(), msg.ID, resultEmbed)
	}
}

func CoinflipHandler(s *discordgo.Session, i *discordgo.InteractionCreate, service *services.EconomyService) {
	ctx := framework.NewSlashContext(s, i)
	CoinflipCmd(ctx, service)
}
