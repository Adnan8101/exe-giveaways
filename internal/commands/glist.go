package commands

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/database"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var GList = &discordgo.ApplicationCommand{
	Name:        "glist",
	Description: "List active giveaways",
}

func GListCmd(ctx framework.Context, db *database.Database) {
	giveaways, err := db.GetActiveGiveaways(ctx.GetGuildID())
	if err != nil {
		ctx.ReplyEphemeral(fmt.Sprintf("❌ Failed to fetch giveaways: %s", err.Error()))
		return
	}

	if len(giveaways) == 0 {
		ctx.Reply("No active giveaways in this server.")
		return
	}

	var sb strings.Builder
	sb.WriteString("**🎉 Active Giveaways**\n\n")

	for _, g := range giveaways {
		timeLeft := time.Until(time.Unix(0, g.EndTime*int64(time.Millisecond)))
		sb.WriteString(fmt.Sprintf("• **%s**\n   ID: `%s` | Ends in: %s\n", g.Prize, g.MessageID, timeLeft.Round(time.Second)))
	}

	ctx.Reply(sb.String())
}

func HandleGList(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database) {
	ctx := framework.NewSlashContext(s, i)
	GListCmd(ctx, db)
}
