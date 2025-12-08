package economy

import (
	"discord-giveaway-bot/internal/commands/framework"
	"discord-giveaway-bot/internal/database"
	"discord-giveaway-bot/internal/utils"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var SetPrefix = &discordgo.ApplicationCommand{
	Name:        "set-prefix",
	Description: "Set the bot prefix for this server",
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionString,
			Name:        "prefix",
			Description: "New prefix",
			Required:    true,
		},
	},
}

func SetPrefixCmd(ctx framework.Context, db *database.Database) {
	if ctx.GetMember().Permissions&discordgo.PermissionAdministrator == 0 {
		ctx.ReplyEphemeral(utils.EmojiCross + " You need Administrator permissions to use this command.")
		return
	}

	var newPrefix string

	if slashCtx, ok := ctx.(*framework.SlashContext); ok {
		newPrefix = slashCtx.Interaction.ApplicationCommandData().Options[0].StringValue()
	} else if prefixCtx, ok := ctx.(*framework.PrefixContext); ok {
		if len(prefixCtx.Args) < 1 {
			ctx.Reply("Usage: `!set-prefix <new_prefix>`")
			return
		}
		newPrefix = prefixCtx.Args[0]
	}

	if len(newPrefix) > 5 {
		ctx.ReplyEphemeral(utils.EmojiCross + " Prefix is too long (max 5 characters).")
		return
	}

	if err := db.SetGuildPrefix(ctx.GetGuildID(), newPrefix); err != nil {
		ctx.ReplyEphemeral(fmt.Sprintf("%s Failed to set prefix: %s", utils.EmojiCross, err.Error()))
		return
	}

	ctx.Reply(fmt.Sprintf("%s Prefix updated to `%s`", utils.EmojiTick, newPrefix))
}

func SetPrefixHandler(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database) {
	ctx := framework.NewSlashContext(s, i)
	SetPrefixCmd(ctx, db)
}
