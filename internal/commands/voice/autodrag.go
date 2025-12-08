package voice

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/database"
	"discord-giveaway-bot/internal/utils"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var AutoDrag = &discordgo.ApplicationCommand{
	Name:        "autodrag",
	Description: "Automatically drag a user to a specific VC when they join ANY VC",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionUser,
			Name:        "user",
			Description: "User to auto-drag",
			Required:    true,
		},
		{
			Type:        discordgo.ApplicationCommandOptionChannel,
			Name:        "target-vc",
			Description: "Target voice channel (leave empty to remove auto-drag)",
			ChannelTypes: []discordgo.ChannelType{
				discordgo.ChannelTypeGuildVoice,
			},
			Required: false,
		},
	},
	DefaultMemberPermissions: ptrInt64(PermissionVoiceMoveMembers | discordgo.PermissionAdministrator),
}

func AutoDragCmd(ctx framework.Context, db *database.Database) {
	// Check permissions (admins and server owners only)
	if !hasAdminPermissions(ctx) {
		ctx.ReplyEphemeral(fmt.Sprintf("%s You need Administrator permission to use this command.", utils.EmojiCross))
		return
	}

	var targetUserID, targetChannelID string

	// Parse arguments
	if slashCtx, ok := ctx.(*framework.SlashContext); ok {
		options := slashCtx.Interaction.ApplicationCommandData().Options
		targetUserID = options[0].UserValue(slashCtx.Session).ID

		// Check if target-vc is provided
		if len(options) > 1 {
			targetChannelID = options[1].ChannelValue(slashCtx.Session).ID
		}
	} else if prefixCtx, ok := ctx.(*framework.PrefixContext); ok {
		if len(prefixCtx.Args) < 1 {
			ctx.Reply(fmt.Sprintf("%s Usage: `!autodrag <user> [channel-id]` (leave channel empty to remove)", utils.EmojiCross))
			return
		}

		// Get user from mention or ID
		if len(prefixCtx.Message.Mentions) > 0 {
			targetUserID = prefixCtx.Message.Mentions[0].ID
		} else {
			targetUserID = strings.Trim(prefixCtx.Args[0], "<@!>")
		}

		if len(prefixCtx.Args) > 1 {
			targetChannelID = strings.Trim(prefixCtx.Args[1], "<#>")
		}
	}

	user, _ := ctx.GetSession().User(targetUserID)
	username := "User"
	if user != nil {
		username = user.Username
	}

	// If no target channel, remove the autodrag rule
	if targetChannelID == "" {
		err := db.DeleteAutoDragRule(ctx.GetGuildID(), targetUserID)
		if err != nil {
			ctx.Reply(fmt.Sprintf("%s Failed to remove auto-drag rule: %s", utils.EmojiCross, err.Error()))
			return
		}
		ctx.Reply(fmt.Sprintf("%s Removed auto-drag rule for **%s**.", utils.EmojiTick, username))
		return
	}

	// Check if command author has Connect and Speak permissions in target VC (unless admin/owner)
	if !hasAdminPermissions(ctx) {
		hasAccess, err := hasVoiceChannelAccess(ctx.GetSession(), ctx.GetGuildID(), targetChannelID, ctx.GetAuthor().ID)
		if err != nil || !hasAccess {
			ctx.ReplyEphemeral(fmt.Sprintf("%s You need Connect and Speak permissions in the target voice channel to set up autodrag.", utils.EmojiCross))
			return
		}
	}

	// Create autodrag rule
	err := db.CreateAutoDragRule(ctx.GetGuildID(), targetUserID, targetChannelID, ctx.GetAuthor().ID)
	if err != nil {
		ctx.Reply(fmt.Sprintf("%s Failed to create auto-drag rule: %s", utils.EmojiCross, err.Error()))
		return
	}

	channel, err := ctx.GetSession().Channel(targetChannelID)
	channelName := "voice channel"
	if err == nil && channel != nil {
		channelName = channel.Name
	}

	ctx.Reply(fmt.Sprintf("%s **%s** will now be automatically dragged to **%s** whenever they join any voice channel.", utils.EmojiTick, username, channelName))
}

func AutoDragHandler(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database) {
	ctx := framework.NewSlashContext(s, i)
	AutoDragCmd(ctx, db)
}
