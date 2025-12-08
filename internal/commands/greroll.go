package commands

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/services"
	"discord-giveaway-bot/internal/utils"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var GReroll = &discordgo.ApplicationCommand{
	Name:        "greroll",
	Description: "Reroll a giveaway winner",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "message_id",
			Description: "Message ID of the giveaway",
			Required:    true,
		},
	},
}

func GRerollCmd(ctx framework.Context, service *services.GiveawayService) {
	if ctx.GetMember().Permissions&discordgo.PermissionManageGuild == 0 {
		ctx.ReplyEphemeral(utils.EmojiCross + " You need Manage Server permissions.")
		return
	}

	var messageID string

	if slashCtx, ok := ctx.(*framework.SlashContext); ok {
		messageID = slashCtx.Interaction.ApplicationCommandData().Options[0].StringValue()
	} else if prefixCtx, ok := ctx.(*framework.PrefixContext); ok {
		if len(prefixCtx.Args) < 1 {
			ctx.Reply("Usage: `!greroll <message_id>`")
			return
		}
		messageID = prefixCtx.Args[0]
	}

	_, err := service.RerollGiveaway(messageID)
	if err != nil {
		ctx.ReplyEphemeral(fmt.Sprintf("%s Failed to reroll: %s", utils.EmojiCross, err.Error()))
		return
	}

	ctx.Reply(utils.EmojiTick + " Rerolled new winner(s)!")
}

func HandleGReroll(s *discordgo.Session, i *discordgo.InteractionCreate, service *services.GiveawayService) {
	ctx := framework.NewSlashContext(s, i)
	GRerollCmd(ctx, service)
}
