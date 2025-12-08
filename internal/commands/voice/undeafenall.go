package voice

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/utils"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var UndeafenAll = &discordgo.ApplicationCommand{
	Name:                     "undeafenall",
	Description:              "Undeafens everyone in your current voice channel",
	DefaultMemberPermissions: ptrInt64(PermissionVoiceDeafenMembers),
}

func UndeafenAllCmd(ctx framework.Context) {
	// Check permissions (admins and server owners bypass)
	if !hasAdminPermissions(ctx) && ctx.GetMember().Permissions&PermissionVoiceDeafenMembers == 0 {
		ctx.ReplyEphemeral(fmt.Sprintf("%s You need Deafen Members permission to use this command.", utils.EmojiCross))
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

	// Collect deafened members to undeafen
	var toUndeafen []string
	for _, vs := range guild.VoiceStates {
		if vs.ChannelID == authorChannelID && vs.Deaf {
			toUndeafen = append(toUndeafen, vs.UserID)
		}
	}

	// Use worker pool for concurrent undeafening
	undeafenedCount := processMembersInParallel(ctx.GetSession(), ctx.GetGuildID(), toUndeafen, false, func(s *discordgo.Session, guildID, userID string, state bool) error {
		return s.GuildMemberDeafen(guildID, userID, state)
	})

	ctx.Reply(fmt.Sprintf("%s Successfully undeafened **%d** member(s) in your voice channel.", utils.EmojiTick, undeafenedCount))
}

func UndeafenAllHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := framework.NewSlashContext(s, i)
	UndeafenAllCmd(ctx)
}
