package commands

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/database"
	"discord-giveaway-bot/internal/redis"
	"discord-giveaway-bot/internal/utils"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

var Ping = &discordgo.ApplicationCommand{
	Name:        "ping",
	Description: "Check bot latency",
}

func PingCmd(ctx framework.Context, db *database.Database, rdb *redis.Client) {
	// Send initial response to make it feel snappy
	msg, _ := ctx.Reply(utils.EmojiTick + " Pong! Calculating...")

	// Calculate latency from snowflake timestamp
	var timestamp int64
	if slashCtx, ok := ctx.(*framework.SlashContext); ok {
		// Interaction ID is a snowflake
		id, _ := strconv.ParseInt(slashCtx.Interaction.ID, 10, 64)
		timestamp = (id >> 22) + 1420070400000
	} else if prefixCtx, ok := ctx.(*framework.PrefixContext); ok {
		// Message ID is a snowflake
		id, _ := strconv.ParseInt(prefixCtx.Message.ID, 10, 64)
		timestamp = (id >> 22) + 1420070400000
	}

	botLatency := time.Since(time.Unix(0, timestamp*int64(time.Millisecond)))

	// Calculate API latency (heartbeat)
	apiLatency := ctx.GetSession().HeartbeatLatency()

	// Measure Database and Redis Latency concurrently
	var dbLatency, redisLatency time.Duration
	var errDB, errRedis error
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		startDB := time.Now()
		errDB = db.Ping()
		dbLatency = time.Since(startDB)
	}()

	go func() {
		defer wg.Done()
		startRedis := time.Now()
		errRedis = rdb.Ping()
		redisLatency = time.Since(startRedis)
	}()

	wg.Wait()

	dbStatus := fmt.Sprintf("`%dms`", dbLatency.Milliseconds())
	if errDB != nil {
		dbStatus = "`❌ Error`"
	}

	redisStatus := fmt.Sprintf("`%dms`", redisLatency.Milliseconds())
	if errRedis != nil {
		redisStatus = "`❌ Error`"
	}

	embed := &discordgo.MessageEmbed{
		Title: utils.EmojiTick + " Pong!",
		Color: utils.ColorDark,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Bot Latency",
				Value:  fmt.Sprintf("`%dms`", botLatency.Milliseconds()),
				Inline: true,
			},
			{
				Name:   "API Latency",
				Value:  fmt.Sprintf("`%dms`", apiLatency.Milliseconds()),
				Inline: true,
			},
			{
				Name:   "Database",
				Value:  dbStatus,
				Inline: true,
			},
			{
				Name:   "Redis",
				Value:  redisStatus,
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    fmt.Sprintf("Requested by %s", ctx.GetAuthor().Username),
			IconURL: ctx.GetAuthor().AvatarURL(""),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Edit the initial response with the embed
	ctx.EditReplyEmbed(msg, embed)
}

func HandlePing(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database, rdb *redis.Client) {
	ctx := framework.NewSlashContext(s, i)
	PingCmd(ctx, db, rdb)
}
