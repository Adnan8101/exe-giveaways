package economy

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/services"
	"discord-giveaway-bot/internal/utils"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var Leaderboard = &discordgo.ApplicationCommand{
	Name:        "leaderboard",
	Description: "Show the economy leaderboard",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "type",
			Description: "Type of leaderboard",
			Required:    true,
			Choices: []*discordgo.ApplicationCommandOptionChoice{
				{
					Name:  "Coins",
					Value: "coins",
				},
			},
		},
	},
}

func LeaderboardCmd(ctx framework.Context, service *services.EconomyService) {
	// Currently only coins supported, so we ignore args/options for now as it's the only choice
	users, err := service.GetLeaderboard(ctx.GetGuildID(), 10)
	if err != nil {
		ctx.ReplyEphemeral(fmt.Sprintf("%s %s", utils.EmojiCross, err.Error()))
		return
	}

	config, err := service.GetConfig(ctx.GetGuildID())
	emoji := "<:Cash:1443554334670327848>"
	if err == nil {
		emoji = config.CurrencyEmoji
	}

	var sb strings.Builder
	sb.WriteString("**🏆 Economy Leaderboard**\n\n")
	for idx, u := range users {
		member, err := ctx.GetSession().GuildMember(ctx.GetGuildID(), u.UserID)
		username := "Unknown User"
		if err == nil {
			username = member.User.Username
		}
		sb.WriteString(fmt.Sprintf("**%d.** %s - %d %s\n", idx+1, username, u.Balance, emoji))
	}

	ctx.Reply(sb.String())
}

func LeaderboardHandler(s *discordgo.Session, i *discordgo.InteractionCreate, service *services.EconomyService) {
	ctx := framework.NewSlashContext(s, i)
	LeaderboardCmd(ctx, service)
}
