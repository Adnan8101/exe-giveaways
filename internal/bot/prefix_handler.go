package bot

import (
	"discord-giveaway-bot/internal/commands"
	"discord-giveaway-bot/internal/commands/economy"
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/commands/voice"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) HandlePrefixCommand(m *discordgo.MessageCreate) {
	content := strings.TrimPrefix(m.Content, "!")
	parts := strings.Fields(content)
	if len(parts) == 0 {
		return
	}

	command := strings.ToLower(parts[0])
	args := parts[1:]

	ctx := framework.NewPrefixContext(b.Session, m, args)

	// Route to appropriate handler
	switch command {
	// Utility
	case "help":
		commands.HelpCmd(ctx)
	case "ping":
		commands.PingCmd(ctx, b.DB, b.Redis)
	case "stats":
		commands.StatsCmd(ctx, b.StartTime)

	// Economy
	case "daily":
		economy.DailyCmd(ctx, b.EconomyService)
	case "weekly":
		economy.WeeklyCmd(ctx, b.EconomyService)
	case "hourly":
		economy.HourlyCmd(ctx, b.EconomyService)
	case "coins", "balance":
		economy.CoinsCmd(ctx, b.EconomyService)
	case "leaderboard", "lb":
		economy.LeaderboardCmd(ctx, b.EconomyService)
	case "invites":
		economy.InvitesCmd(ctx, b.EconomyService)
	case "coinflip", "cf":
		economy.CoinflipCmd(ctx, b.EconomyService)

	// Giveaways
	case "gcreate":
		commands.GCreateCmd(ctx, b.DB)
	case "gend":
		commands.GEndCmd(ctx, b.DB, b.Service)
	case "greroll":
		commands.GRerollCmd(ctx, b.Service)
	case "glist":
		commands.GListCmd(ctx, b.DB)
	case "gcancel":
		commands.GCancelCmd(ctx, b.Service)

	// Voice
	case "wv":
		voice.WhereVoiceCmd(ctx)
	case "drag":
		voice.DragCmd(ctx)
	case "to":
		voice.ToCmd(ctx)
	case "muteall":
		voice.MuteAllCmd(ctx)
	case "unmuteall":
		voice.UnmuteAllCmd(ctx)
	case "deafenall":
		voice.DeafenAllCmd(ctx)
	case "undeafenall":
		voice.UndeafenAllCmd(ctx)
	case "vcclear":
		voice.VCClearCmd(ctx)
	}
}
