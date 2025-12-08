package commands

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/services"
	"discord-giveaway-bot/internal/utils"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var GCancel = &discordgo.ApplicationCommand{
	Name:        "gcancel",
	Description: "Cancel a giveaway (no winners)",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "message_id",
			Description: "Message ID of the giveaway",
			Required:    true,
		},
	},
}

func GCancelCmd(ctx framework.Context, service *services.GiveawayService) {
	if ctx.GetMember().Permissions&discordgo.PermissionManageGuild == 0 {
		ctx.ReplyEphemeral(utils.EmojiCross + " You need Manage Server permissions.")
		return
	}

	var messageID string

	if slashCtx, ok := ctx.(*framework.SlashContext); ok {
		messageID = slashCtx.Interaction.ApplicationCommandData().Options[0].StringValue()
	} else if prefixCtx, ok := ctx.(*framework.PrefixContext); ok {
		if len(prefixCtx.Args) < 1 {
			ctx.Reply("Usage: `!gcancel <message_id>`")
			return
		}
		messageID = prefixCtx.Args[0]
	}

	// Cancel is basically ending without winners, but EndGiveaway picks winners.
	// We need a CancelGiveaway method in service or just delete it.
	// For now, let's assume we want to delete it or mark as cancelled.
	// Let's check service.
	err := service.CancelGiveaway(messageID)
	if err != nil {
		ctx.ReplyEphemeral(fmt.Sprintf("%s Failed to cancel giveaway: %s", utils.EmojiCross, err.Error()))
		return
	}

	ctx.Reply(utils.EmojiTick + " Giveaway cancelled.")
}

func HandleGCancel(s *discordgo.Session, i *discordgo.InteractionCreate, service *services.GiveawayService) {
	ctx := framework.NewSlashContext(s, i)
	GCancelCmd(ctx, service)
}
