package voice

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/database"
	"discord-giveaway-bot/internal/utils"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var AutoAFK = &discordgo.ApplicationCommand{
	Name:        "autoafk",
	Description: "Automatically move idle users to AFK channel after specified minutes",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionInteger,
			Name:        "minutes",
			Description: "Minutes of inactivity before moving to AFK (default: 10)",
			Required:    false,
		},
		{
			Type:        discordgo.ApplicationCommandOptionBoolean,
			Name:        "enabled",
			Description: "Enable or disable auto-AFK (default: true)",
			Required:    false,
		},
	},
	DefaultMemberPermissions: ptrInt64(PermissionVoiceMoveMembers | discordgo.PermissionAdministrator),
}

func AutoAFKCmd(ctx framework.Context, db *database.Database) {
	// Check permissions (admins and server owners only)
	if !hasAdminPermissions(ctx) {
		ctx.ReplyEphemeral(fmt.Sprintf("%s You need Administrator permission to use this command.", utils.EmojiCross))
		return
	}

	minutes := 10
	enabled := true

	// Parse arguments
	if slashCtx, ok := ctx.(*framework.SlashContext); ok {
		options := slashCtx.Interaction.ApplicationCommandData().Options
		optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption)
		for _, opt := range options {
			optionMap[opt.Name] = opt
		}

		if opt, ok := optionMap["minutes"]; ok {
			minutes = int(opt.IntValue())
		}
		if opt, ok := optionMap["enabled"]; ok {
			enabled = opt.BoolValue()
		}
	} else if prefixCtx, ok := ctx.(*framework.PrefixContext); ok {
		if len(prefixCtx.Args) > 0 {
			if strings.ToLower(prefixCtx.Args[0]) == "disable" || strings.ToLower(prefixCtx.Args[0]) == "off" {
				enabled = false
			} else {
				parsed, err := strconv.Atoi(prefixCtx.Args[0])
				if err == nil && parsed > 0 {
					minutes = parsed
				}
			}
		}
	}

	// Get guild's AFK channel
	guild, err := ctx.GetSession().Guild(ctx.GetGuildID())
	if err != nil {
		ctx.Reply(fmt.Sprintf("%s Failed to get guild information.", utils.EmojiCross))
		return
	}

	afkChannelID := guild.AfkChannelID

	// Save settings
	err = db.SetAutoAFKSettings(ctx.GetGuildID(), enabled, minutes, afkChannelID)
	if err != nil {
		ctx.Reply(fmt.Sprintf("%s Failed to save auto-AFK settings: %s", utils.EmojiCross, err.Error()))
		return
	}

	if !enabled {
		ctx.Reply(fmt.Sprintf("%s Auto-AFK has been **disabled** for this server.", utils.EmojiTick))
		return
	}

	afkInfo := "disconnected"
	if afkChannelID != "" {
		channel, err := ctx.GetSession().Channel(afkChannelID)
		if err == nil && channel != nil {
			afkInfo = fmt.Sprintf("moved to **%s**", channel.Name)
		}
	}

	ctx.Reply(fmt.Sprintf("%s Auto-AFK enabled! Idle users (muted/deafened for **%d minutes**) will be %s.", utils.EmojiTick, minutes, afkInfo))
}

func AutoAFKHandler(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database) {
	ctx := framework.NewSlashContext(s, i)
	AutoAFKCmd(ctx, db)
}
