package voice

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/utils"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/bwmarrin/discordgo"
)

// processMembersInParallel processes member operations concurrently using a worker pool
func processMembersInParallel(s *discordgo.Session, guildID string, userIDs []string, state bool, operation func(*discordgo.Session, string, string, bool) error) int {
	const workers = 10 // Limit concurrent workers to avoid rate limits

	var (
		successCount int32
		wg           sync.WaitGroup
		jobs         = make(chan string, len(userIDs))
	)

	// Start worker pool
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for userID := range jobs {
				if err := operation(s, guildID, userID, state); err == nil {
					atomic.AddInt32(&successCount, 1)
				}
			}
		}()
	}

	// Send jobs to workers
	for _, userID := range userIDs {
		jobs <- userID
	}
	close(jobs)

	// Wait for all workers to complete
	wg.Wait()

	return int(successCount)
}

var MuteAll = &discordgo.ApplicationCommand{
	Name:                     "muteall",
	Description:              "Server mutes everyone in your current voice channel",
	DefaultMemberPermissions: ptrInt64(PermissionVoiceMuteMembers),
}

func MuteAllCmd(ctx framework.Context) {
	// Check permissions (admins and server owners bypass)
	if !hasAdminPermissions(ctx) && ctx.GetMember().Permissions&PermissionVoiceMuteMembers == 0 {
		ctx.ReplyEphemeral(fmt.Sprintf("%s You need Mute Members permission to use this command.", utils.EmojiCross))
		return
	}

	// Get author's voice channel
	guild, err := ctx.GetSession().State.Guild(ctx.GetGuildID())
	if err != nil {
		// If state cache failed, try requesting guild directly
		guild, err = ctx.GetSession().Guild(ctx.GetGuildID())
		if err != nil {
			ctx.Reply(fmt.Sprintf("%s Failed to get guild information.", utils.EmojiCross))
			return
		}
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

	// Collect members to mute
	var toMute []string
	totalInChannel := 0
	for _, vs := range guild.VoiceStates {
		if vs.ChannelID == authorChannelID {
			totalInChannel++
			if !vs.Mute && vs.UserID != ctx.GetAuthor().ID {
				toMute = append(toMute, vs.UserID)
			}
		}
	}

	// Use worker pool for concurrent muting (10 workers, rate limiting)
	mutedCount := processMembersInParallel(ctx.GetSession(), ctx.GetGuildID(), toMute, true, func(s *discordgo.Session, guildID, userID string, muted bool) error {
		return s.GuildMemberMute(guildID, userID, muted)
	})

	ctx.Reply(fmt.Sprintf("%s Successfully muted **%d** member(s) in your voice channel. (%d total in channel)", utils.EmojiMuted, mutedCount, totalInChannel))
}

func MuteAllHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := framework.NewSlashContext(s, i)
	MuteAllCmd(ctx)
}
