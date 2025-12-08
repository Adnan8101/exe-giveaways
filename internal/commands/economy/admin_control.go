package economy

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/services"
	"discord-giveaway-bot/internal/utils"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var AdminCoins = &discordgo.ApplicationCommand{
	Name:        "admin-coins",
	Description: "Manage user balances",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "add",
			Description: "Add coins to a user",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "amount", Description: "Amount", Required: true},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "remove",
			Description: "Remove coins from a user",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "amount", Description: "Amount (or 'all')", Required: true},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "set",
			Description: "Set a user's balance",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "User", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "amount", Description: "Amount", Required: true},
			},
		},
	},
}

func AdminCoinsCmd(ctx framework.Context, service *services.EconomyService) {
	if ctx.GetMember().Permissions&discordgo.PermissionAdministrator == 0 {
		ctx.ReplyEphemeral(utils.EmojiCross + " You need Administrator permissions to use this command.")
		return
	}

	var subCommand string
	var targetUser *discordgo.User
	var amountStr string

	if slashCtx, ok := ctx.(*framework.SlashContext); ok {
		options := slashCtx.Interaction.ApplicationCommandData().Options
		subCommand = options[0].Name
		targetUser = options[0].Options[0].UserValue(slashCtx.Session)
		amountStr = options[0].Options[1].StringValue()
	} else if prefixCtx, ok := ctx.(*framework.PrefixContext); ok {
		if len(prefixCtx.Args) < 3 {
			ctx.Reply("Usage: `!admin-coins <add|remove|set> <user> <amount>`")
			return
		}
		subCommand = prefixCtx.Args[0]

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

		amountStr = prefixCtx.Args[2]
	}

	var err error
	var msg string

	config, err := service.GetConfig(ctx.GetGuildID())
	emoji := "<:Cash:1443554334670327848>"
	if err == nil {
		emoji = config.CurrencyEmoji
	}

	// Parse amount
	var amount int64
	isAll := strings.ToLower(amountStr) == "all"

	if !isAll {
		parsed, err := strconv.ParseInt(amountStr, 10, 64)
		if err != nil {
			ctx.ReplyEphemeral(utils.EmojiCross + " Invalid amount.")
			return
		}
		amount = parsed
	}

	switch subCommand {
	case "add":
		if isAll {
			ctx.ReplyEphemeral(utils.EmojiCross + " 'all' is not supported for add.")
			return
		}
		err = service.AddCoins(ctx.GetGuildID(), targetUser.ID, amount)
		msg = fmt.Sprintf("Added **%d** %s to **%s**.", amount, emoji, targetUser.Username)
	case "remove":
		if isAll {
			err = service.SetCoins(ctx.GetGuildID(), targetUser.ID, 0)
			msg = fmt.Sprintf("Removed **all** coins from **%s**.", targetUser.Username)
		} else {
			err = service.RemoveCoins(ctx.GetGuildID(), targetUser.ID, amount)
			msg = fmt.Sprintf("Removed **%d** %s from **%s**.", amount, emoji, targetUser.Username)
		}
	case "set":
		if isAll {
			ctx.ReplyEphemeral(utils.EmojiCross + " 'all' is not supported for set.")
			return
		}
		err = service.SetCoins(ctx.GetGuildID(), targetUser.ID, amount)
		msg = fmt.Sprintf("Set **%s**'s balance to **%d** %s.", targetUser.Username, amount, emoji)
	default:
		ctx.ReplyEphemeral(utils.EmojiCross + " Unknown subcommand.")
		return
	}

	if err != nil {
		ctx.ReplyEphemeral(fmt.Sprintf("%s Error: %s", utils.EmojiCross, err.Error()))
		return
	}

	embed := &discordgo.MessageEmbed{
		Description: fmt.Sprintf("%s %s", utils.EmojiTick, msg),
		Color:       0x00FF00, // Green
	}
	ctx.ReplyEmbed(embed)
}

func AdminCoinsHandler(s *discordgo.Session, i *discordgo.InteractionCreate, service *services.EconomyService) {
	ctx := framework.NewSlashContext(s, i)
	AdminCoinsCmd(ctx, service)
}
