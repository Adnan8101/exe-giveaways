package economy

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/services"
	"discord-giveaway-bot/internal/utils"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var Daily = &discordgo.ApplicationCommand{
	Name:        "daily",
	Description: "Claim your daily reward",
}

func DailyCmd(ctx framework.Context, service *services.EconomyService) {
	amount, err := service.ClaimDaily(ctx.GetGuildID(), ctx.GetAuthor().ID)
	if err != nil {
		ctx.ReplyEphemeral(fmt.Sprintf("%s %s", utils.EmojiCross, err.Error()))
		return
	}

	config, err := service.GetConfig(ctx.GetGuildID())
	emoji := "<:Cash:1443554334670327848>"
	if err == nil {
		emoji = config.CurrencyEmoji
	}

	ctx.Reply(fmt.Sprintf("%s You claimed your daily reward of **%d %s**!", utils.EmojiTick, amount, emoji))
}

func DailyHandler(s *discordgo.Session, i *discordgo.InteractionCreate, service *services.EconomyService) {
	ctx := framework.NewSlashContext(s, i)
	DailyCmd(ctx, service)
}
