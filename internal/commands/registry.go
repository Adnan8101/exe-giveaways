package commands

import (
	"discord-giveaway-bot/internal/commands/antinuke"
	"discord-giveaway-bot/internal/commands/economy"
	"discord-giveaway-bot/internal/commands/shop"
	"discord-giveaway-bot/internal/commands/voice"

	"github.com/bwmarrin/discordgo"
)

// Helper for float pointers
func floatPtr(v float64) *float64 {
	return &v
}

var Commands = []*discordgo.ApplicationCommand{
	GCreate,
	GEnd,
	GReroll,
	GList,
	GCancel,
	// Economy Commands
	economy.Daily,
	economy.Weekly,
	economy.Hourly,
	economy.Coins,
	economy.Leaderboard,
	economy.Invites,
	economy.Coinflip,
	economy.Economy,
	economy.AdminCoins,
	economy.Give,
	Help,
	Ping,
	Stats,
	// Voice Commands
	voice.WhereVoice,
	voice.Drag,
	voice.To,
	voice.MuteAll,
	voice.UnmuteAll,
	voice.DeafenAll,
	voice.UndeafenAll,
	voice.VCClear,
	voice.AutoDrag,
	voice.AutoAFK,
	// Shop Commands
	shop.Shop,
	shop.Buy,
	shop.AdminShop,
	// AntiNuke Commands
	antinuke.PanicModeCmd,
	antinuke.Enable,
	antinuke.SetLimit,
	antinuke.Punishment,
	antinuke.Whitelist,
	antinuke.Logs,
}
