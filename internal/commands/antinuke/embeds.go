package antinuke

import (
	"discord-giveaway-bot/internal/models"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Embed colors
const (
	ColorSuccess = 0x2b2d31 // Dark gray for success
	ColorError   = 0xed4245 // Red for errors
	ColorInfo    = 0x5865f2 // Blurple for info
)

// CreateSuccessEmbed creates a clean success embed
func CreateSuccessEmbed(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "✓ " + title,
		Description: description,
		Color:       ColorSuccess,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "AntiNuke System",
		},
	}
}

// CreateErrorEmbed creates a clean error embed
func CreateErrorEmbed(title, description string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "✗ " + title,
		Description: description,
		Color:       ColorError,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "AntiNuke System",
		},
	}
}

// CreateInfoEmbed creates a clean info embed
func CreateInfoEmbed(title string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:     title,
		Color:     ColorInfo,
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "AntiNuke System",
		},
	}
}

// CreateConfigEmbed creates an embed showing action configuration
func CreateConfigEmbed(actionType string, config *models.ActionConfig) *discordgo.MessageEmbed {
	embed := CreateSuccessEmbed(
		"AntiNuke Configuration Updated",
		fmt.Sprintf("**Action:** %s", models.GetActionDisplayName(actionType)),
	)

	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:   "Limit",
			Value:  fmt.Sprintf("%d actions", config.LimitCount),
			Inline: true,
		},
		{
			Name:   "Time Window",
			Value:  models.FormatWindowTime(config.WindowSeconds),
			Inline: true,
		},
		{
			Name:   "Punishment",
			Value:  strings.Title(config.Punishment),
			Inline: true,
		},
	}

	return embed
}

// CreateStatusEmbed creates an embed showing all configurations
func CreateStatusEmbed(configs []*models.ActionConfig) *discordgo.MessageEmbed {
	embed := CreateInfoEmbed("AntiNuke Status")

	if len(configs) == 0 {
		embed.Description = "No actions are currently configured.\nUse `/antinuke enable` to configure actions."
		return embed
	}

	// Group configs into a table format
	var description strings.Builder
	description.WriteString("```\n")
	description.WriteString(fmt.Sprintf("%-25s %8s %10s %12s\n", "Action", "Limit", "Window", "Punishment"))
	description.WriteString(strings.Repeat("-", 60) + "\n")

	for _, config := range configs {
		actionName := models.GetActionDisplayName(config.ActionType)
		if len(actionName) > 24 {
			actionName = actionName[:21] + "..."
		}
		description.WriteString(fmt.Sprintf(
			"%-25s %8d %10s %12s\n",
			actionName,
			config.LimitCount,
			models.FormatWindowTime(config.WindowSeconds),
			strings.Title(config.Punishment),
		))
	}

	description.WriteString("```")
	embed.Description = description.String()

	return embed
}

// GetAllActionChoices returns all action choices for slash commands
func GetAllActionChoices() []*discordgo.ApplicationCommandOptionChoice {
	actions := models.GetAllActionTypes()
	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, len(actions)+1)

	// Add "all" option first
	choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
		Name:  "All Actions",
		Value: models.ActionAll,
	})

	// Add individual actions
	for _, action := range actions {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  models.GetActionDisplayName(action),
			Value: action,
		})
	}

	return choices
}

// GetPunishmentChoices returns punishment type choices
func GetPunishmentChoices() []*discordgo.ApplicationCommandOptionChoice {
	return []*discordgo.ApplicationCommandOptionChoice{
		{Name: "Ban", Value: models.PunishmentBan},
		{Name: "Kick", Value: models.PunishmentKick},
		{Name: "Timeout", Value: models.PunishmentTimeout},
		{Name: "Quarantine", Value: models.PunishmentQuarantine},
	}
}

// GetWindowChoices returns time window choices
func GetWindowChoices() []*discordgo.ApplicationCommandOptionChoice {
	return []*discordgo.ApplicationCommandOptionChoice{
		{Name: "10 seconds", Value: "10s"},
		{Name: "30 seconds", Value: "30s"},
		{Name: "1 minute", Value: "1m"},
		{Name: "5 minutes", Value: "5m"},
		{Name: "10 minutes", Value: "10m"},
		{Name: "30 minutes", Value: "30m"},
		{Name: "1 hour", Value: "1h"},
	}
}
