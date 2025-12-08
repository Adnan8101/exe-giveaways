package voice

import (
	"discord-giveaway-bot/internal/commands/framework"

	"github.com/bwmarrin/discordgo"
)

// Permission constants for voice commands
// Discord.go uses different constant names than standard Discord API
const (
	// PermissionVoiceMuteMembers is the permission to mute members in voice channels
	PermissionVoiceMuteMembers = 0x00400000 // 1 << 22

	// PermissionVoiceDeafenMembers is the permission to deafen members in voice channels
	PermissionVoiceDeafenMembers = 0x00800000 // 1 << 23

	// PermissionVoiceMoveMembers is the permission to move members between voice channels
	PermissionVoiceMoveMembers = 0x01000000 // 1 << 24
)

// hasAdminPermissions checks if user has Administrator permission or is the server owner
func hasAdminPermissions(ctx framework.Context) bool {
	// Check if user has Administrator permission
	if ctx.GetMember().Permissions&discordgo.PermissionAdministrator != 0 {
		return true
	}

	// Check if user is server owner
	guild, err := ctx.GetSession().Guild(ctx.GetGuildID())
	if err == nil && guild.OwnerID == ctx.GetAuthor().ID {
		return true
	}

	return false
}

// hasVoiceChannelAccess checks if user has both Connect and Speak permissions in a voice channel
func hasVoiceChannelAccess(s *discordgo.Session, guildID, channelID, userID string) (bool, error) {
	// Calculate permissions for this user in this channel
	perms, err := s.UserChannelPermissions(userID, channelID)
	if err != nil {
		return false, err
	}

	// Check if user has Connect and Speak permissions
	hasConnect := perms&discordgo.PermissionVoiceConnect != 0
	hasSpeak := perms&discordgo.PermissionVoiceSpeak != 0

	return hasConnect && hasSpeak, nil
}
