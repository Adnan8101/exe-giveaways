package voice

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/utils"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var WhereVoice = &discordgo.ApplicationCommand{
	Name:        "wv",
	Description: "Shows which voice channel a user is in",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionUser,
			Name:        "user",
			Description: "User to check (mention, ID, or reply)",
			Required:    false,
		},
	},
}

func WhereVoiceCmd(ctx framework.Context) {
	var targetUserID string

	// Get target user from interaction or prefix command
	if slashCtx, ok := ctx.(*framework.SlashContext); ok {
		if len(slashCtx.Interaction.ApplicationCommandData().Options) > 0 {
			targetUserID = slashCtx.Interaction.ApplicationCommandData().Options[0].UserValue(slashCtx.Session).ID
		} else {
			targetUserID = ctx.GetAuthor().ID
		}
	} else if prefixCtx, ok := ctx.(*framework.PrefixContext); ok {
		// Check for mentioned user
		if len(prefixCtx.Message.Mentions) > 0 {
			targetUserID = prefixCtx.Message.Mentions[0].ID
		} else if len(prefixCtx.Args) > 0 {
			// Try to use first arg as user ID
			targetUserID = strings.Trim(prefixCtx.Args[0], "<@!>")
		} else if prefixCtx.Message.ReferencedMessage != nil {
			targetUserID = prefixCtx.Message.ReferencedMessage.Author.ID
		} else {
			targetUserID = ctx.GetAuthor().ID
		}
	}

	// Get guild state
	guild, err := ctx.GetSession().State.Guild(ctx.GetGuildID())
	if err != nil {
		ctx.Reply(fmt.Sprintf("%s Failed to get guild information.", utils.EmojiCross))
		return
	}

	// Check voice states
	var voiceChannel *discordgo.Channel
	for _, vs := range guild.VoiceStates {
		if vs.UserID == targetUserID && vs.ChannelID != "" {
			voiceChannel, err = ctx.GetSession().Channel(vs.ChannelID)
			if err != nil {
				ctx.Reply(fmt.Sprintf("%s User is in a voice channel but I couldn't fetch the details.", utils.EmojiCross))
				return
			}
			break
		}
	}

	user, err := ctx.GetSession().User(targetUserID)
	if err != nil {
		ctx.Reply(fmt.Sprintf("%s Failed to fetch user information.", utils.EmojiCross))
		return
	}

	if voiceChannel != nil {
		ctx.Reply(fmt.Sprintf("<#%s>", voiceChannel.ID))
	} else {
		ctx.Reply(fmt.Sprintf("%s **%s** is not in a voice channel.", utils.EmojiDisconnect, user.Username))
	}
}

func WhereVoiceHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := framework.NewSlashContext(s, i)
	WhereVoiceCmd(ctx)
}
