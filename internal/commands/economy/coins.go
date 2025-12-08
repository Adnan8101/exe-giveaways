package economy

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/models"
	"discord-giveaway-bot/internal/services"
	"discord-giveaway-bot/internal/utils"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var Coins = &discordgo.ApplicationCommand{
	Name:        "coins",
	Description: "Check your coin balance",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionUser,
			Name:        "user",
			Description: "User to check balance for",
			Required:    false,
		},
	},
}

func CoinsCmd(ctx framework.Context, service *services.EconomyService) {
	targetUser := ctx.GetAuthor()

	// Handle arguments
	// Slash: Options
	// Prefix: Args

	if slashCtx, ok := ctx.(*framework.SlashContext); ok {
		if len(slashCtx.Interaction.ApplicationCommandData().Options) > 0 {
			targetUser = slashCtx.Interaction.ApplicationCommandData().Options[0].UserValue(slashCtx.Session)
		}
	} else if prefixCtx, ok := ctx.(*framework.PrefixContext); ok {
		if len(prefixCtx.Args) > 0 {
			// Try to parse mention or ID
			arg := prefixCtx.Args[0]
			if strings.HasPrefix(arg, "<@") && strings.HasSuffix(arg, ">") {
				id := strings.Trim(arg, "<@!>")
				u, err := ctx.GetSession().User(id)
				if err == nil {
					targetUser = u
				}
			} else {
				// Try ID
				u, err := ctx.GetSession().User(arg)
				if err == nil {
					targetUser = u
				}
			}
		}
	}

	config, err := service.GetConfig(ctx.GetGuildID())
	if err != nil {
		// Fallback
		config = &models.EconomyConfig{CurrencyEmoji: "<:Cash:1443554334670327848>"}
	}

	balance, err := service.GetUserBalance(ctx.GetGuildID(), targetUser.ID)
	if err != nil {
		ctx.ReplyEphemeral(fmt.Sprintf("%s %s", utils.EmojiCross, err.Error()))
		return
	}

	ctx.Reply(fmt.Sprintf("**%s** Have %s **__%d__**", targetUser.Username, config.CurrencyEmoji, balance))
}

func CoinsHandler(s *discordgo.Session, i *discordgo.InteractionCreate, service *services.EconomyService) {
	ctx := framework.NewSlashContext(s, i)
	CoinsCmd(ctx, service)
}
