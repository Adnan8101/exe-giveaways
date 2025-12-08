package economy

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/services"
	"discord-giveaway-bot/internal/utils"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var Weekly = &discordgo.ApplicationCommand{
	Name:        "weekly",
	Description: "Claim your weekly reward",
}

func WeeklyCmd(ctx framework.Context, service *services.EconomyService) {
	amount, err := service.ClaimWeekly(ctx.GetGuildID(), ctx.GetAuthor().ID)
	if err != nil {
		ctx.ReplyEphemeral(fmt.Sprintf("%s %s", utils.EmojiCross, err.Error()))
		return
	}

	config, err := service.GetConfig(ctx.GetGuildID())
	emoji := "<:Cash:1443554334670327848>"
	if err == nil {
		emoji = config.CurrencyEmoji
	}

	ctx.Reply(fmt.Sprintf("%s You claimed your weekly reward of **%d %s**!", utils.EmojiTick, amount, emoji))
}

func WeeklyHandler(s *discordgo.Session, i *discordgo.InteractionCreate, service *services.EconomyService) {
	ctx := framework.NewSlashContext(s, i)
	WeeklyCmd(ctx, service)
}
