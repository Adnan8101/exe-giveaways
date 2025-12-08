package voice

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/utils"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var Drag = &discordgo.ApplicationCommand{
	Name:        "drag",
	Description: "Moves a user into your current voice channel",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionUser,
			Name:        "user",
			Description: "User to drag (mention, ID, or reply)",
			Required:    true,
		},
	},
	DefaultMemberPermissions: ptrInt64(PermissionVoiceMoveMembers),
}

func DragCmd(ctx framework.Context) {
	// Check permissions (admins and server owners bypass)
	if !hasAdminPermissions(ctx) && ctx.GetMember().Permissions&PermissionVoiceMoveMembers == 0 {
		ctx.ReplyEphemeral(fmt.Sprintf("%s You need Move Members permission to use this command.", utils.EmojiCross))
		return
	}

	// Get author's voice state
	guild, err := ctx.GetSession().State.Guild(ctx.GetGuildID())
	if err != nil {
		ctx.Reply(fmt.Sprintf("%s Failed to get guild information.", utils.EmojiCross))
		return
	}

	var authorChannelID string
	for _, vs := range guild.VoiceStates {
		if vs.UserID == ctx.GetAuthor().ID {
			authorChannelID = vs.ChannelID
			break
		}
	}

	if authorChannelID == "" {
		ctx.ReplyEphemeral(fmt.Sprintf("%s You must be in a voice channel to drag someone.", utils.EmojiCross))
		return
	}

	// Check if command author has Connect and Speak permissions in their VC (unless admin/owner)
	if !hasAdminPermissions(ctx) {
		hasAccess, err := hasVoiceChannelAccess(ctx.GetSession(), ctx.GetGuildID(), authorChannelID, ctx.GetAuthor().ID)
		if err != nil || !hasAccess {
			ctx.ReplyEphemeral(fmt.Sprintf("%s You need Connect and Speak permissions in your voice channel to drag users.", utils.EmojiCross))
			return
		}
	}

	// Get target user
	var targetUserID string
	if slashCtx, ok := ctx.(*framework.SlashContext); ok {
		targetUserID = slashCtx.Interaction.ApplicationCommandData().Options[0].UserValue(slashCtx.Session).ID
	} else if prefixCtx, ok := ctx.(*framework.PrefixContext); ok {
		if len(prefixCtx.Message.Mentions) > 0 {
			targetUserID = prefixCtx.Message.Mentions[0].ID
		} else if len(prefixCtx.Args) > 0 {
			targetUserID = strings.Trim(prefixCtx.Args[0], "<@!>")
		} else if prefixCtx.Message.ReferencedMessage != nil {
			targetUserID = prefixCtx.Message.ReferencedMessage.Author.ID
		} else {
			ctx.Reply(fmt.Sprintf("%s Please specify a user to drag (mention, ID, or reply to their message).", utils.EmojiCross))
			return
		}
	}

	// Move the user
	err = ctx.GetSession().GuildMemberMove(ctx.GetGuildID(), targetUserID, &authorChannelID)
	if err != nil {
		ctx.Reply(fmt.Sprintf("%s Failed to drag user: %s", utils.EmojiCross, err.Error()))
		return
	}

	user, _ := ctx.GetSession().User(targetUserID)
	username := "User"
	if user != nil {
		username = user.Username
	}

	ctx.Reply(fmt.Sprintf("%s Successfully dragged **%s** to your voice channel!", utils.EmojiTick, username))
}

func DragHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := framework.NewSlashContext(s, i)
	DragCmd(ctx)
}

func ptrInt64(i int64) *int64 {
	return &i
}
