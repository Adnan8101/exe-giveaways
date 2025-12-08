package voice

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/utils"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var VCClear = &discordgo.ApplicationCommand{
	Name:                     "vcclear",
	Description:              "Disconnects EVERYONE from your current voice channel",
	DefaultMemberPermissions: ptrInt64(PermissionVoiceMoveMembers | discordgo.PermissionAdministrator),
}

func VCClearCmd(ctx framework.Context) {
	// Check permissions (admins and server owners bypass, otherwise requires Move Members)
	if !hasAdminPermissions(ctx) && ctx.GetMember().Permissions&PermissionVoiceMoveMembers == 0 {
		ctx.ReplyEphemeral(fmt.Sprintf("%s You need Administrator or Move Members permission to use this command.", utils.EmojiCross))
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

	// Disconnect all members from the channel
	disconnectedCount := 0
	for _, vs := range guild.VoiceStates {
		if vs.ChannelID == authorChannelID {
			// Move to null channel = disconnect
			err := ctx.GetSession().GuildMemberMove(ctx.GetGuildID(), vs.UserID, nil)
			if err == nil {
				disconnectedCount++
			}
		}
	}

	ctx.Reply(fmt.Sprintf("%s Successfully disconnected **%d** member(s) from the voice channel.", utils.EmojiDisconnect, disconnectedCount))
}

func VCClearHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := framework.NewSlashContext(s, i)
	VCClearCmd(ctx)
}
