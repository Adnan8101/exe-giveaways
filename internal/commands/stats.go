package commands

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/utils"
	"fmt"
	"runtime"
	"time"

	"github.com/bwmarrin/discordgo"
)

var Stats = &discordgo.ApplicationCommand{
	Name:        "stats",
	Description: "Show bot statistics",
}

func StatsCmd(ctx framework.Context, startTime time.Time) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(startTime)

	// Format uptime
	days := int(uptime.Hours()) / 24
	hours := int(uptime.Hours()) % 24
	minutes := int(uptime.Minutes()) % 60
	seconds := int(uptime.Seconds()) % 60
	uptimeStr := fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)

	embed := &discordgo.MessageEmbed{
		Title:       "Bot Statistics",
		Color:       utils.ColorDark,
		Description: "Detailed statistics about the bot and its runtime environment.",
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: ctx.GetSession().State.User.AvatarURL(""),
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Bot Info",
				Value:  fmt.Sprintf("**Uptime:** %s\n**Goroutines:** %d\n**Go Version:** %s", uptimeStr, runtime.NumGoroutine(), runtime.Version()),
				Inline: false,
			},
			{
				Name:   "System",
				Value:  fmt.Sprintf("**OS/Arch:** %s/%s\n**CPUs:** %d", runtime.GOOS, runtime.GOARCH, runtime.NumCPU()),
				Inline: false,
			},
			{
				Name:   "Memory",
				Value:  fmt.Sprintf("**Alloc:** %v MB\n**Total Alloc:** %v MB\n**Sys:** %v MB\n**NumGC:** %v", bToMb(m.Alloc), bToMb(m.TotalAlloc), bToMb(m.Sys), m.NumGC),
				Inline: false,
			},
			{
				Name:   "Registry",
				Value:  fmt.Sprintf("**Guilds:** %d\n**Users:** %d (approx)", len(ctx.GetSession().State.Guilds), calculateTotalUsers(ctx.GetSession())),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    fmt.Sprintf("Requested by %s", ctx.GetAuthor().Username),
			IconURL: ctx.GetAuthor().AvatarURL(""),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	ctx.ReplyEmbed(embed)
}

func calculateTotalUsers(s *discordgo.Session) int {
	var count int
	for _, g := range s.State.Guilds {
		count += g.MemberCount
	}
	return count
}

func HandleStats(s *discordgo.Session, i *discordgo.InteractionCreate, startTime time.Time) {
	ctx := framework.NewSlashContext(s, i)
	StatsCmd(ctx, startTime)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
