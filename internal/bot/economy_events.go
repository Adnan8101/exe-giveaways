package bot

import (
	"context"
	"discord-giveaway-bot/internal/commands"
	"discord-giveaway-bot/internal/commands/economy"
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/commands/voice"
	"discord-giveaway-bot/internal/database"
	"discord-giveaway-bot/internal/services"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// EconomyEvents handles economy-related events
type EconomyEvents struct {
	Service          *services.EconomyService
	GiveawayService  *services.GiveawayService
	DB               *database.Database
	BlackjackCommand *economy.BlackjackCommand
	// Cache for voice states to track duration
	voiceJoinTimes sync.Map // lock-free concurrent access

	// Rate limiting with lock-free map
	lastMessageTime sync.Map // lock-free concurrent access
}

func NewEconomyEvents(service *services.EconomyService, giveawayService *services.GiveawayService, db *database.Database, bj *economy.BlackjackCommand) *EconomyEvents {
	return &EconomyEvents{
		Service:          service,
		GiveawayService:  giveawayService,
		DB:               db,
		BlackjackCommand: bj,
		// sync.Map doesn't need initialization
	}
}

func (e *EconomyEvents) OnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	// Prefix Command Handling
	prefix, err := e.Service.GetGuildPrefix(m.GuildID)
	if err != nil {
		prefix = "!"
	}

	// Parse command
	if strings.HasPrefix(m.Content, prefix) {
		args := strings.Fields(m.Content[len(prefix):])
		if len(args) == 0 {
			return
		}

		cmdName := strings.ToLower(args[0])
		args = args[1:]

		ctx := framework.NewPrefixContext(s, m, args)

		switch cmdName {
		// User Commands
		case "daily":
			economy.DailyCmd(ctx, e.Service)
		case "weekly":
			economy.WeeklyCmd(ctx, e.Service)
		case "hourly":
			economy.HourlyCmd(ctx, e.Service)
		case "coins", "balance":
			economy.CoinsCmd(ctx, e.Service)
		case "leaderboard":
			economy.LeaderboardCmd(ctx, e.Service)
		case "invites":
			economy.InvitesCmd(ctx, e.Service)
		case "coinflip", "cf":
			economy.CoinflipCmd(ctx, e.Service)
		case "give":
			economy.GiveCmd(ctx, e.Service)
		case "bj", "blackjack":
			e.BlackjackCommand.Handle(s, m, args)

		// Admin Commands - RESTRICTED TO SLASH ONLY
		case "economy", "admin-coins", "set-prefix":
			// Silent ignore as per requirements
			return

		// Giveaway Commands
		case "gcreate":
			commands.GCreateCmd(ctx, e.DB)
		case "gend":
			commands.GEndCmd(ctx, e.DB, e.GiveawayService)
		case "greroll":
			commands.GRerollCmd(ctx, e.GiveawayService)
		case "glist":
			commands.GListCmd(ctx, e.DB)
		case "gcancel":
			commands.GCancelCmd(ctx, e.GiveawayService)

		// Voice Commands
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
		case "autodrag":
			voice.AutoDragCmd(ctx, e.DB)
		case "autoafk":
			voice.AutoAFKCmd(ctx, e.DB)

		case "help":
			ctx.Reply("**Commands:**\n`!daily`, `!weekly`, `!hourly` - Claim rewards\n`!balance` - Check coins\n`!cf <amount> [h/t]` - Gamble\n`!leaderboard` - View top users\n`!invites` - Check invites\n`!gcreate`, `!gend`, `!greroll`, `!glist`, `!gcancel` - Giveaway commands\n`!wv`, `!drag`, `!to`, `!muteall`, `!unmuteall`, `!deafenall`, `!undeafenall`, `!vcclear` - Voice commands")
		}
	} else {
		// Message Reward Logic with Rate Limiting
		// Check allowed channels first
		config, err := e.Service.GetConfig(m.GuildID)
		if err == nil {
			// Check if channel is allowed
			if config.AllowedChannels != "" {
				allowed := false
				channels := strings.Split(config.AllowedChannels, ",")
				for _, chID := range channels {
					if chID == m.ChannelID {
						allowed = true
						break
					}
				}
				if !allowed {
					return
				}
			}

			// Check rate limit (e.g., 1 message per minute) using lock-free sync.Map
			now := time.Now()
			lastTimeVal, exists := e.lastMessageTime.Load(m.Author.ID)
			if !exists || now.Sub(lastTimeVal.(time.Time)) >= time.Minute {
				e.lastMessageTime.Store(m.Author.ID, now)

				// Award coins and increment message count using fast prepared statements with context
				if config.MessageReward > 0 {
					ctx := context.Background()
					e.Service.AddCoins(m.GuildID, m.Author.ID, int64(config.MessageReward))
					e.DB.IncrementMessageCountFast(ctx, m.GuildID, m.Author.ID)
				}
			}
		}
	}
}

func (e *EconomyEvents) OnVoiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	userID := v.UserID
	guildID := v.GuildID

	// User joined a channel
	if v.BeforeUpdate == nil && v.ChannelID != "" {
		e.voiceJoinTimes.Store(userID, time.Now().Unix())
		return
	}

	// User left a channel
	if v.BeforeUpdate != nil && v.ChannelID == "" {
		joinTimeVal, ok := e.voiceJoinTimes.Load(userID)
		if !ok {
			return
		}
		joinTime := joinTimeVal.(int64)
		e.voiceJoinTimes.Delete(userID)

		duration := time.Now().Unix() - joinTime
		minutes := duration / 60

		if minutes > 0 {
			config, err := e.Service.GetConfig(guildID)
			if err != nil || config.VCRewardPerMin == 0 {
				return
			}
			reward := int64(minutes) * int64(config.VCRewardPerMin)
			_ = e.Service.AddCoins(guildID, userID, reward)
		}
		return
	}

	// User switched channels - treat as leave and join
	if v.BeforeUpdate != nil && v.ChannelID != "" && v.BeforeUpdate.ChannelID != v.ChannelID {
		joinTimeVal, ok := e.voiceJoinTimes.Load(userID)
		if ok {
			joinTime := joinTimeVal.(int64)
			duration := time.Now().Unix() - joinTime
			minutes := duration / 60
			if minutes > 0 {
				config, err := e.Service.GetConfig(guildID)
				if err == nil && config.VCRewardPerMin > 0 {
					reward := int64(minutes) * int64(config.VCRewardPerMin)
					_ = e.Service.AddCoins(guildID, userID, reward)
				}
			}
		}
		e.voiceJoinTimes.Store(userID, time.Now().Unix())
	}
}

func (e *EconomyEvents) OnMessageReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r.UserID == s.State.User.ID {
		return
	}

	// Blackjack Handling
	e.BlackjackCommand.HandleReaction(s, &discordgo.MessageReaction{
		UserID:    r.UserID,
		MessageID: r.MessageID,
		ChannelID: r.ChannelID,
		GuildID:   r.GuildID,
		Emoji:     r.Emoji,
	}, true)

	config, err := e.Service.GetConfig(r.GuildID)
	if err != nil {
		return
	}

	// React Reward (for reacting to announcements/events)
	// This is tricky to distinguish from random reactions.
	// Usually this is done by checking if the message is in a specific channel or has a specific role mention.
	// For simplicity, we'll skip "React Reward" implementation as generic "any reaction" would be abusable.
	// Or we can implement "Upvote Reward" where the message author gets coins.

	if config.UpvoteReward > 0 {
		// Get message to find author
		msg, err := s.ChannelMessage(r.ChannelID, r.MessageID)
		if err != nil || msg.Author.Bot || msg.Author.ID == r.UserID {
			return
		}
		_ = e.Service.AddCoins(r.GuildID, msg.Author.ID, int64(config.UpvoteReward))
	}
}

func (e *EconomyEvents) OnMessageReactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	if r.UserID == s.State.User.ID {
		return
	}

	// Blackjack Handling
	e.BlackjackCommand.HandleReaction(s, &discordgo.MessageReaction{
		UserID:    r.UserID,
		MessageID: r.MessageID,
		ChannelID: r.ChannelID,
		GuildID:   r.GuildID,
		Emoji:     r.Emoji,
	}, false)
}
