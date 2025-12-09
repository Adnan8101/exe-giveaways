package commands

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/models"
	"discord-giveaway-bot/internal/services"
	"discord-giveaway-bot/internal/utils"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
)

var GCreate = &discordgo.ApplicationCommand{
	Name:        "gcreate",
	Description: "Start a new giveaway",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "prize",
			Description: "The prize to give away",
			Required:    true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionInteger,
			Name:        "winners",
			Description: "Number of winners",
			Required:    true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "duration",
			Description: "Duration (e.g. 10m, 1h, 2d)",
			Required:    true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionChannel,
			Name:        "channel",
			Description: "Channel to host the giveaway in (default: current channel)",
			ChannelTypes: []discordgo.ChannelType{
				discordgo.ChannelTypeGuildText,
			},
			Required: false,
		},
		{
			Type:        discordgo.ApplicationCommandOptionRole,
			Name:        "role_requirement",
			Description: "Required role to enter",
			Required:    false,
		},
		{
			Type:        discordgo.ApplicationCommandOptionInteger,
			Name:        "invite_requirement",
			Description: "Minimum invites required",
			Required:    false,
		},
		{
			Type:        discordgo.ApplicationCommandOptionInteger,
			Name:        "account_age",
			Description: "Minimum account age in days",
			Required:    false,
		},
		{
			Type:        discordgo.ApplicationCommandOptionInteger,
			Name:        "server_age",
			Description: "Minimum days in server",
			Required:    false,
		},
		{
			Type:        discordgo.ApplicationCommandOptionBoolean,
			Name:        "captcha",
			Description: "Require captcha verification",
			Required:    false,
		},
		{
			Type:        discordgo.ApplicationCommandOptionInteger,
			Name:        "message_required",
			Description: "Minimum messages required to enter",
			Required:    false,
		},
		{
			Type:        discordgo.ApplicationCommandOptionInteger,
			Name:        "voice",
			Description: "Minimum voice minutes required",
			Required:    false,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "custom_message",
			Description: "Custom message to display in giveaway",
			Required:    false,
		},
		{
			Type:        discordgo.ApplicationCommandOptionInteger,
			Name:        "required_fees",
			Description: "Entry fee in coins",
			Required:    false,
		},
		{
			Type:        discordgo.ApplicationCommandOptionRole,
			Name:        "assign_role",
			Description: "Role to assign to participants",
			Required:    false,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "thumbnail",
			Description: "URL for giveaway thumbnail",
			Required:    false,
		},
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "custom_emoji",
			Description: "Custom emoji for giveaway reaction (default: 🎉)",
			Required:    false,
		},
	},
}

func GCreateCmd(ctx framework.Context, service *services.GiveawayService) {
	// ...existing code...

	if ctx.GetMember().Permissions&discordgo.PermissionManageGuild == 0 {
		ctx.ReplyEphemeral("❌ You need Manage Server permissions to start giveaways.")
		return
	}

	if slashCtx, ok := ctx.(*framework.SlashContext); ok {
		options := slashCtx.Interaction.ApplicationCommandData().Options
		optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
		for _, opt := range options {
			optionMap[opt.Name] = opt
		}

		prize := optionMap["prize"].StringValue()
		winners := int(optionMap["winners"].IntValue())
		durationStr := optionMap["duration"].StringValue()

		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			ctx.ReplyEphemeral("❌ Invalid duration format. Use 10m, 1h, 2d, etc.")
			return
		}

		if winners < 1 {
			ctx.ReplyEphemeral("❌ Invalid number of winners.")
			return
		}

		channelID := ctx.GetChannelID()
		if opt, ok := optionMap["channel"]; ok {
			channelID = opt.ChannelValue(slashCtx.Session).ID
		}

		// Optional Requirements
		var roleReq string
		if opt, ok := optionMap["role_requirement"]; ok {
			roleReq = opt.RoleValue(slashCtx.Session, slashCtx.GetGuildID()).ID
		}

		var inviteReq int
		if opt, ok := optionMap["invite_requirement"]; ok {
			inviteReq = int(opt.IntValue())
		}

		var accountAgeReq int
		if opt, ok := optionMap["account_age"]; ok {
			accountAgeReq = int(opt.IntValue())
		}

		var serverAgeReq int
		if opt, ok := optionMap["server_age"]; ok {
			serverAgeReq = int(opt.IntValue())
		}

		var captchaReq bool
		if opt, ok := optionMap["captcha"]; ok {
			captchaReq = opt.BoolValue()
		}

		var messageReq int
		if opt, ok := optionMap["message_required"]; ok {
			messageReq = int(opt.IntValue())
		}

		var voiceReq int
		if opt, ok := optionMap["voice"]; ok {
			voiceReq = int(opt.IntValue())
		}

		var customMessage string
		if opt, ok := optionMap["custom_message"]; ok {
			customMessage = opt.StringValue()
		}

		var entryFee int
		if opt, ok := optionMap["required_fees"]; ok {
			entryFee = int(opt.IntValue())
		}

		var assignRole string
		if opt, ok := optionMap["assign_role"]; ok {
			assignRole = opt.RoleValue(slashCtx.Session, slashCtx.GetGuildID()).ID
		}

		var thumbnail string
		if opt, ok := optionMap["thumbnail"]; ok {
			thumbnail = opt.StringValue()
		}

		var customEmoji string
		if opt, ok := optionMap["custom_emoji"]; ok {
			customEmoji = opt.StringValue()
		}
		// Default to party emoji if not specified
		if customEmoji == "" {
			customEmoji = "🎉"
		}

		// Parse and steal emoji if needed (for custom emojis)
		finalEmoji, err := utils.ParseAndStealEmoji(slashCtx.Session, ctx.GetGuildID(), customEmoji)
		if err != nil {
			log.Printf("Failed to parse/steal emoji: %v, using default", err)
			finalEmoji = "🎉"
		}

		endTime := time.Now().Add(duration).UnixNano() / int64(time.Millisecond)

		g := &models.Giveaway{
			ChannelID:             channelID,
			GuildID:               ctx.GetGuildID(),
			HostID:                ctx.GetAuthor().ID,
			Prize:                 prize,
			WinnersCount:          winners,
			EndTime:               endTime,
			CreatedAt:             models.Now(),
			RoleRequirement:       roleReq,
			InviteRequirement:     inviteReq,
			AccountAgeRequirement: accountAgeReq,
			ServerAgeRequirement:  serverAgeReq,
			CaptchaRequirement:    captchaReq,
			MessageRequired:       messageReq,
			VoiceRequirement:      voiceReq,
			CustomMessage:         customMessage,
			EntryFee:              entryFee,
			AssignRole:            assignRole,
			Thumbnail:             thumbnail,
			Emoji:                 finalEmoji, // Store the emoji
		}

		// Send initial message
		embed := utils.CreateGiveawayEmbed(g, 0)

		// We need to send to the target channel, not necessarily the interaction channel
		msg, err := slashCtx.Session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{embed},
		})
		if err != nil {
			ctx.ReplyEphemeral(fmt.Sprintf("❌ Failed to send giveaway message to <#%s>: %s", channelID, err.Error()))
			return
		}

		g.MessageID = msg.ID
		id, err := service.DB.CreateGiveaway(g)
		if err != nil {
			ctx.ReplyEphemeral(fmt.Sprintf("❌ Failed to save giveaway: %s", err.Error()))
			// Try to delete the message if save failed
			slashCtx.Session.ChannelMessageDelete(channelID, msg.ID)
			return
		}
		// Invalidate cache
		service.Redis.InvalidateActiveGiveaways(g.GuildID)
		// Add to ending queue
		service.Redis.AddToEndingQueue(g.MessageID, g.EndTime)

		// Add reaction with custom/stolen emoji
		err = slashCtx.Session.MessageReactionAdd(channelID, msg.ID, g.Emoji)
		if err != nil {
			log.Printf("Failed to add reaction: %v", err)
		}

		// Update giveaway with real ID (if needed for embed footer or something, but usually not needed for just ID if not in footer)
		// We don't need to edit the message just for the button anymore.
		// But if the embed needs the ID (e.g. footer), we might need to update it.
		// utils.CreateGiveawayEmbed uses g.ID? Let's check.
		// g.ID is 0 initially. db.CreateGiveaway returns the new ID.
		// If the embed shows the ID, we should update the embed.

		g.ID = id
		newEmbed := utils.CreateGiveawayEmbed(g, 0)
		slashCtx.Session.ChannelMessageEditEmbed(channelID, msg.ID, newEmbed)

		ctx.ReplyEphemeral("✅ Giveaway created successfully!")

	} else if prefixCtx, ok := ctx.(*framework.PrefixContext); ok {
		// Legacy prefix command support (simplified)
		// !gcreate <duration> <winners> <prize>
		if len(prefixCtx.Args) < 3 {
			ctx.Reply("Usage: `!gcreate <duration> <winners> <prize>`")
			return
		}

		durationStr := prefixCtx.Args[0]
		winnersStr := prefixCtx.Args[1]
		prize := ""
		for i := 2; i < len(prefixCtx.Args); i++ {
			prize += prefixCtx.Args[i] + " "
		}
		prize = prize[:len(prize)-1]

		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			ctx.Reply("❌ Invalid duration format. Use 10m, 1h, 2d, etc.")
			return
		}

		winners, err := strconv.Atoi(winnersStr)
		if err != nil || winners < 1 {
			ctx.Reply("❌ Invalid number of winners.")
			return
		}

		// Create giveaway
		endTime := time.Now().Add(duration).UnixNano() / int64(time.Millisecond)

		g := &models.Giveaway{
			ChannelID:    ctx.GetChannelID(),
			GuildID:      ctx.GetGuildID(),
			HostID:       ctx.GetAuthor().ID,
			Prize:        prize,
			WinnersCount: winners,
			EndTime:      endTime,
			CreatedAt:    models.Now(),
		}

		// Send initial message
		embed := utils.CreateGiveawayEmbed(g, 0)

		msg, err := ctx.GetSession().ChannelMessageSendComplex(ctx.GetChannelID(), &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{embed},
		})
		if err != nil {
			ctx.Reply(fmt.Sprintf("❌ Failed to send giveaway message: %s", err.Error()))
			return
		}

		g.MessageID = msg.ID
		id, err := service.DB.CreateGiveaway(g)
		if err != nil {
			ctx.Reply(fmt.Sprintf("❌ Failed to save giveaway: %s", err.Error()))
			return
		}
		// Invalidate cache
		service.Redis.InvalidateActiveGiveaways(g.GuildID)
		// Add to ending queue
		service.Redis.AddToEndingQueue(g.MessageID, g.EndTime)

		// Add reaction
		err = ctx.GetSession().MessageReactionAdd(ctx.GetChannelID(), msg.ID, "🎉")
		if err != nil {
			log.Printf("Failed to add reaction: %v", err)
		}

		// Update giveaway with real ID
		g.ID = id
		newEmbed := utils.CreateGiveawayEmbed(g, 0)
		ctx.GetSession().ChannelMessageEditEmbed(ctx.GetChannelID(), msg.ID, newEmbed)
	}
}

func HandleGCreate(s *discordgo.Session, i *discordgo.InteractionCreate, service *services.GiveawayService) {
	ctx := framework.NewSlashContext(s, i)
	GCreateCmd(ctx, service)
}
