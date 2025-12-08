package bot

import (
	"discord-giveaway-bot/internal/commands"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) HandleGCreate(i *discordgo.InteractionCreate) {
	commands.HandleGCreate(b.Session, i, b.DB)
}

func (b *Bot) HandleGEnd(i *discordgo.InteractionCreate) {
	commands.HandleGEnd(b.Session, i, b.DB, b.Service)
}

func (b *Bot) HandleGReroll(i *discordgo.InteractionCreate) {
	commands.HandleGReroll(b.Session, i, b.Service)
}

func (b *Bot) HandleGList(i *discordgo.InteractionCreate) {
	commands.HandleGList(b.Session, i, b.DB)
}

func (b *Bot) HandleGCancel(i *discordgo.InteractionCreate) {
	commands.HandleGCancel(b.Session, i, b.Service)
}
