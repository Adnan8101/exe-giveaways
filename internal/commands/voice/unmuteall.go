package voice

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/utils"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var UnmuteAll = &discordgo.ApplicationCommand{
	Name:                     "unmuteall",
	Description:              "Server unmutes everyone in your current voice channel",
	DefaultMemberPermissions: ptrInt64(PermissionVoiceMuteMembers),
}

func UnmuteAllCmd(ctx framework.Context) {
	// Check permissions (admins and server owners bypass)
	if !hasAdminPermissions(ctx) && ctx.GetMember().Permissions&PermissionVoiceMuteMembers == 0 {
		ctx.ReplyEphemeral(fmt.Sprintf("%s You need Mute Members permission to use this command.", utils.EmojiCross))
		return
	}

	// Get author's voice channel
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
		ctx.ReplyEphemeral(fmt.Sprintf("%s You must be in a voice channel to use this command.", utils.EmojiCross))
		return
	}

	// Collect muted members to unmute
	var toUnmute []string
	for _, vs := range guild.VoiceStates {
		if vs.ChannelID == authorChannelID && vs.Mute {
			toUnmute = append(toUnmute, vs.UserID)
		}
	}

	// Use worker pool for concurrent unmuting
	unmutedCount := processMembersInParallel(ctx.GetSession(), ctx.GetGuildID(), toUnmute, false, func(s *discordgo.Session, guildID, userID string, muted bool) error {
		return s.GuildMemberMute(guildID, userID, muted)
	})

	ctx.Reply(fmt.Sprintf("%s Successfully unmuted **%d** member(s) in your voice channel.", utils.EmojiTick, unmutedCount))
}

func UnmuteAllHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := framework.NewSlashContext(s, i)
	UnmuteAllCmd(ctx)
}
