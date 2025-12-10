package antinuke

import (
	"discord-giveaway-bot/internal/database"
	"discord-giveaway-bot/internal/utils"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// HandleAntiNuke handles enable/disable/status
func HandleAntiNuke(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database) {
	options := i.ApplicationCommandData().Options
	subCmd := options[0].Name

	guildID := i.GuildID

	switch subCmd {
	case "enable":
		err := db.EnableAntiNuke(guildID)
		if err != nil {
			utils.SendError(s, i, "Failed to enable AntiNuke: "+err.Error())
			return
		}
		utils.SendSuccess(s, i, "✅ AntiNuke System **ENABLED**\n\nThe engine is now monitoring events in real-time.")
		// TODO: Sync to CDE Memory

	case "disable":
		err := db.DisableAntiNuke(guildID)
		if err != nil {
			utils.SendError(s, i, "Failed to disable AntiNuke: "+err.Error())
			return
		}
		utils.SendSuccess(s, i, "⚠️ AntiNuke System **DISABLED**\n\nYour server is no longer protected.")
		// TODO: Sync to CDE Memory

	case "status":
		config, err := db.GetAntiNukeConfig(guildID)
		if err != nil {
			utils.SendError(s, i, "Failed to get config: "+err.Error())
			return
		}

		status := "DISABLED"
		if config.Enabled {
			status = "ENABLED"
		}

		embed := &discordgo.MessageEmbed{
			Title:       "🛡️ AntiNuke Status",
			Description: fmt.Sprintf("**System Status:** %s\n**Panic Mode:** %v\n**Log Channel:** <#%s>", status, config.PanicMode, config.LogsChannel),
			Color:       0x00FF00,
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
			},
		})
	}
}

// HandleSetLimit handles /setlimit
func HandleSetLimit(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database) {
	options := i.ApplicationCommandData().Options
	action := options[0].StringValue()
	limit := int(options[1].IntValue())
	seconds := int(options[2].IntValue())

	// Use SetActionConfig which handles insertion/updates
	// Default punishment to 'ban' if new, or preserve existing?
	// The simple SetActionConfig helper in DB requires all params.
	// For now, default punishment to "ban" if creating new.
	err := db.SetActionConfig(i.GuildID, action, limit, seconds, "ban")
	if err != nil {
		utils.SendError(s, i, "Failed to set limit")
		return
	}

	utils.SendSuccess(s, i, fmt.Sprintf("✅ Limit updated for **%s**\nThreshold: **%d** events in **%d** seconds", action, limit, seconds))
	// TODO: Sync CDE Rules
}

// HandlePunishment handles /punishment
func HandlePunishment(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database) {
	options := i.ApplicationCommandData().Options
	action := options[0].StringValue()
	punishType := options[1].StringValue()

	err := db.UpdateActionPunishment(i.GuildID, action, punishType)
	if err != nil {
		utils.SendError(s, i, "Failed to update punishment. Make sure the limit is set first!")
		return
	}

	utils.SendSuccess(s, i, fmt.Sprintf("✅ Punishment for **%s** set to **%s**", action, punishType))
}

// HandleWhitelist handles /whitelist
func HandleWhitelist(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database) {
	options := i.ApplicationCommandData().Options
	subCmd := options[0].Name

	switch subCmd {
	case "add":
		var targetID, targetType string
		if len(options[0].Options) > 0 {
			opt := options[0].Options[0]
			if opt.Type == discordgo.ApplicationCommandOptionUser {
				targetID = opt.UserValue(s).ID
				targetType = "user"
			} else {
				targetID = opt.RoleValue(s, i.GuildID).ID
				targetType = "role"
			}
		}

		// Correct method: AddWhitelistEntry(guildID, targetID, targetType, addedBy)
		err := db.AddWhitelistEntry(i.GuildID, targetID, targetType, i.Member.User.ID)
		if err != nil {
			utils.SendError(s, i, "Failed to add whitelist: "+err.Error())
			return
		}
		utils.SendSuccess(s, i, fmt.Sprintf("✅ Added <@%s> to whitelist.", targetID))

	case "remove":
		// Logic similar to add
		var targetID string
		if len(options[0].Options) > 0 {
			opt := options[0].Options[0]
			if opt.Type == discordgo.ApplicationCommandOptionUser {
				targetID = opt.UserValue(s).ID
			} else {
				targetID = opt.RoleValue(s, i.GuildID).ID
			}
		}
		// Correct method: RemoveWhitelistEntry(guildID, targetID)
		err := db.RemoveWhitelistEntry(i.GuildID, targetID)
		if err != nil {
			utils.SendError(s, i, "Failed to remove whitelist")
			return
		}
		utils.SendSuccess(s, i, "✅ Removed from whitelist.")

	case "list":
		// Fetch list
		utils.SendSuccess(s, i, "📜 Whitelist: (Not implemented in display yet)")
	}
}

// HandlePanicMode handles /panic
func HandlePanicMode(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database) {
	// Toggle logic - retrieve current state then flip
	config, err := db.GetAntiNukeConfig(i.GuildID)
	if err != nil {
		utils.SendError(s, i, "Failed to get status")
		return
	}

	newState := !config.PanicMode
	err = db.SetPanicMode(i.GuildID, newState)
	if err != nil {
		utils.SendError(s, i, "Failed to set panic mode")
		return
	}

	stateStr := "OFF"
	if newState {
		stateStr = "ON (All actions blocked)"
	}
	utils.SendSuccess(s, i, fmt.Sprintf("🚨 Panic Mode is now **%s**", stateStr))
}

// HandleLogs handles /logs
func HandleLogs(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database) {
	channelID := i.ApplicationCommandData().Options[0].ChannelValue(s).ID
	// Correct method: SetAntiNukeLogsChannel
	err := db.SetAntiNukeLogsChannel(i.GuildID, channelID)
	if err != nil {
		utils.SendError(s, i, "Failed to set log channel")
		return
	}
	utils.SendSuccess(s, i, fmt.Sprintf("✅ Security logs will be sent to <#%s>", channelID))
}
