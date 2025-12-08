package economy

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/services"

	"github.com/bwmarrin/discordgo"
)

var Invites = &discordgo.ApplicationCommand{
	Name:        "invites",
	Description: "Check your invite count and rewards",
}

func InvitesCmd(ctx framework.Context, service *services.EconomyService) {
	ctx.ReplyEphemeral("ℹ️ Invite tracking is not fully implemented yet. Check back later!")
}

func InvitesHandler(s *discordgo.Session, i *discordgo.InteractionCreate, service *services.EconomyService) {
	ctx := framework.NewSlashContext(s, i)
	InvitesCmd(ctx, service)
}
