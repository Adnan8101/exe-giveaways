package commands

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/database"
	"discord-giveaway-bot/internal/services"
	"discord-giveaway-bot/internal/utils"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var GEnd = &discordgo.ApplicationCommand{
	Name:        "gend",
	Description: "End a giveaway immediately",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "message_id",
			Description: "Message ID of the giveaway",
			Required:    true,
		},
	},
}

func GEndCmd(ctx framework.Context, db *database.Database, service *services.GiveawayService) {
	if ctx.GetMember().Permissions&discordgo.PermissionManageGuild == 0 {
		ctx.ReplyEphemeral(utils.EmojiCross + " You need Manage Server permissions.")
		return
	}

	var messageID string

	if slashCtx, ok := ctx.(*framework.SlashContext); ok {
		messageID = slashCtx.Interaction.ApplicationCommandData().Options[0].StringValue()
	} else if prefixCtx, ok := ctx.(*framework.PrefixContext); ok {
		if len(prefixCtx.Args) < 1 {
			ctx.Reply("Usage: `!gend <message_id>`")
			return
		}
		messageID = prefixCtx.Args[0]
	}

	g, err := db.GetGiveaway(messageID)
	if err != nil {
		ctx.ReplyEphemeral(utils.EmojiCross + " Giveaway not found.")
		return
	}

	if g.Ended {
		ctx.ReplyEphemeral(utils.EmojiCross + " Giveaway already ended.")
		return
	}

	err = service.EndGiveaway(g.MessageID)
	if err != nil {
		ctx.ReplyEphemeral(fmt.Sprintf("%s Failed to end giveaway: %s", utils.EmojiCross, err.Error()))
		return
	}

	ctx.Reply(utils.EmojiTick + " Giveaway ended.")
}

func HandleGEnd(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database, service *services.GiveawayService) {
	ctx := framework.NewSlashContext(s, i)
	GEndCmd(ctx, db, service)
}
