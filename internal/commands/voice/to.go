package voice

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/utils"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

var To = &discordgo.ApplicationCommand{
	Name:        "to",
	Description: "Server mute a user for 2 minutes",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionUser,
			Name:        "user",
			Description: "User to server mute",
			Required:    true,
		},
	},
	DefaultMemberPermissions: ptrInt64(PermissionVoiceMuteMembers),
}

func ToCmd(ctx framework.Context) {
	var targetUserID string

	// Check permissions (admins and server owners bypass)
	if !hasAdminPermissions(ctx) && ctx.GetMember().Permissions&PermissionVoiceMuteMembers == 0 {
		ctx.ReplyEphemeral(fmt.Sprintf("%s You need Mute Members permission to use this command.", utils.EmojiCross))
		return
	}

	// Parse arguments
	if slashCtx, ok := ctx.(*framework.SlashContext); ok {
		options := slashCtx.Interaction.ApplicationCommandData().Options
		targetUserID = options[0].UserValue(slashCtx.Session).ID
	} else if prefixCtx, ok := ctx.(*framework.PrefixContext); ok {
		if len(prefixCtx.Args) < 1 {
			ctx.Reply(fmt.Sprintf("%s Usage: `!to <user>`", utils.EmojiCross))
			return
		}

		// Get user from mention, ID, or reply
		if len(prefixCtx.Message.Mentions) > 0 {
			targetUserID = prefixCtx.Message.Mentions[0].ID
		} else {
			targetUserID = strings.Trim(prefixCtx.Args[0], "<@!>")
		}
	}

	user, _ := ctx.GetSession().User(targetUserID)
	username := "User"
	if user != nil {
		username = user.Username
	}

	// Server mute the user
	err := ctx.GetSession().GuildMemberMute(ctx.GetGuildID(), targetUserID, true)
	if err != nil {
		ctx.Reply(fmt.Sprintf("%s Failed to mute user: %s", utils.EmojiCross, err.Error()))
		return
	}

	ctx.Reply(fmt.Sprintf("%s **%s** has been server muted ", utils.EmojiMuted, username))

	// Add success reaction if prefix command
	if prefixCtx, ok := ctx.(*framework.PrefixContext); ok {
		ctx.GetSession().MessageReactionAdd(ctx.GetChannelID(), prefixCtx.Message.ID, "✅")
	}

	// Unmute after 2 minutes
	go func() {
		time.Sleep(2 * time.Minute)
		ctx.GetSession().GuildMemberMute(ctx.GetGuildID(), targetUserID, false)
	}()
}

func ToHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := framework.NewSlashContext(s, i)
	ToCmd(ctx)
}
