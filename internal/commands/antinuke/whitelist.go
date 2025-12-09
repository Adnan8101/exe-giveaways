package antinuke

import (
	"discord-giveaway-bot/internal/database"
	"discord-giveaway-bot/internal/models"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var Whitelist = &discordgo.ApplicationCommand{
	Name:        "whitelist",
	Description: "Manage AntiNuke whitelist",
	DefaultMemberPermissions: func() *int64 {
		perms := int64(discordgo.PermissionAdministrator)
		return &perms
	}(),
	Options: []*discordgo.ApplicationCommandOption{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "add",
			Description: "Add user or role to whitelist",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "User to whitelist",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionRole,
					Name:        "role",
					Description: "Role to whitelist",
					Required:    false,
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "remove",
			Description: "Remove user or role from whitelist",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "User to remove",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionRole,
					Name:        "role",
					Description: "Role to remove",
					Required:    false,
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "view",
			Description: "View all whitelisted users and roles",
		},
	},
}

// HandleWhitelist handles the /whitelist command
func HandleWhitelist(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database) {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		respondWithError(s, i, "Invalid command", "No subcommand provided")
		return
	}

	subcommand := options[0]

	switch subcommand.Name {
	case "add":
		handleWhitelistAdd(s, i, db, subcommand.Options)
	case "remove":
		handleWhitelistRemove(s, i, db, subcommand.Options)
	case "view":
		handleWhitelistView(s, i, db)
	default:
		respondWithError(s, i, "Invalid subcommand", "Unknown subcommand: "+subcommand.Name)
	}
}

func handleWhitelistAdd(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var targetID string
	var targetType string
	var targetName string

	// Check for user or role
	for _, opt := range options {
		if opt.Name == "user" && opt.UserValue(s) != nil {
			user := opt.UserValue(s)
			targetID = user.ID
			targetType = "user"
			targetName = user.Username
			break
		} else if opt.Name == "role" && opt.RoleValue(s, i.GuildID) != nil {
			role := opt.RoleValue(s, i.GuildID)
			targetID = role.ID
			targetType = "role"
			targetName = role.Name
			break
		}
	}

	if targetID == "" {
		respondWithError(s, i, "Invalid Input", "Please provide either a user or a role to whitelist")
		return
	}

	// Check if already whitelisted
	isWhitelisted, _ := db.IsWhitelisted(i.GuildID, targetID)
	if isWhitelisted {
		respondWithError(s, i, "Already Whitelisted", fmt.Sprintf("%s is already in the whitelist", targetName))
		return
	}

	// Create action select menu for choosing which actions to whitelist
	embed := CreateInfoEmbed("Select Actions to Whitelist")
	embed.Description = fmt.Sprintf("**Target:** %s (%s)\n\nSelect the actions you want to whitelist for this %s:",
		targetName, targetID, targetType)

	// Create select menu with all actions + "All"
	selectMenu := discordgo.SelectMenu{
		CustomID:    fmt.Sprintf("whitelist_add_select_%s", targetID),
		Placeholder: "Choose actions to whitelist",
		MinValues:   func() *int { v := 1; return &v }(),
		MaxValues:   13,                                    // 12 actions + "all"
		Options:     createActionSelectOptions([]string{}), // Empty means nothing selected
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{selectMenu},
				},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	// Store the target info in a temporary map (in production, use Redis or DB)
	// For now, we'll encode it in the custom ID
}

func handleWhitelistRemove(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database, options []*discordgo.ApplicationCommandInteractionDataOption) {
	var targetID string
	var targetName string

	// Check for user or role
	for _, opt := range options {
		if opt.Name == "user" && opt.UserValue(s) != nil {
			user := opt.UserValue(s)
			targetID = user.ID
			targetName = user.Username
			break
		} else if opt.Name == "role" && opt.RoleValue(s, i.GuildID) != nil {
			role := opt.RoleValue(s, i.GuildID)
			targetID = role.ID
			targetName = role.Name
			break
		}
	}

	if targetID == "" {
		respondWithError(s, i, "Invalid Input", "Please provide either a user or a role to remove")
		return
	}

	// Check if whitelisted
	isWhitelisted, _ := db.IsWhitelisted(i.GuildID, targetID)
	if !isWhitelisted {
		respondWithError(s, i, "Not Whitelisted", fmt.Sprintf("%s is not in the whitelist", targetName))
		return
	}

	// Remove from whitelist
	err := db.RemoveWhitelistEntry(i.GuildID, targetID)
	if err != nil {
		respondWithError(s, i, "Database Error", "Failed to remove from whitelist: "+err.Error())
		return
	}

	// Success embed
	embed := CreateSuccessEmbed(
		"Removed from Whitelist",
		fmt.Sprintf("**%s** has been removed from the whitelist", targetName),
	)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func handleWhitelistView(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database) {
	entries, err := db.GetWhitelistEntries(i.GuildID)
	if err != nil {
		respondWithError(s, i, "Database Error", "Failed to retrieve whitelist: "+err.Error())
		return
	}

	embed := CreateInfoEmbed("Whitelist")

	if len(entries) == 0 {
		embed.Description = "No users or roles are currently whitelisted.\nUse `/whitelist add` to add entries."
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
			},
		})
		return
	}

	// Separate users and roles
	var users []string
	var roles []string

	for _, entry := range entries {
		if entry.TargetType == "user" {
			users = append(users, fmt.Sprintf("<@%s>", entry.TargetID))
		} else {
			roles = append(roles, fmt.Sprintf("<@&%s>", entry.TargetID))
		}
	}

	// Add fields
	if len(users) > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Whitelisted Users",
			Value:  strings.Join(users, "\n"),
			Inline: false,
		})
	}

	if len(roles) > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Whitelisted Roles",
			Value:  strings.Join(roles, "\n"),
			Inline: false,
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

// HandleWhitelistSelect handles the select menu interaction for adding to whitelist
func HandleWhitelistSelect(s *discordgo.Session, i *discordgo.InteractionCreate, db *database.Database) {
	customID := i.MessageComponentData().CustomID
	if !strings.HasPrefix(customID, "whitelist_add_select_") {
		return
	}

	// Extract target ID from custom ID
	targetID := strings.TrimPrefix(customID, "whitelist_add_select_")
	selectedActions := i.MessageComponentData().Values

	// Determine if user selected "all"
	hasAll := false
	for _, action := range selectedActions {
		if action == models.ActionAll {
			hasAll = true
			break
		}
	}

	// Get user ID who executed the command
	var executorID string
	if i.Member != nil {
		executorID = i.Member.User.ID
	} else if i.User != nil {
		executorID = i.User.ID
	}

	// Determine target type by checking if it's a user or role
	targetType := "user"
	role, err := s.State.Role(i.GuildID, targetID)
	if err == nil && role != nil {
		targetType = "role"
	}

	// Add to whitelist
	err = db.AddWhitelistEntry(i.GuildID, targetID, targetType, executorID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					CreateErrorEmbed("Error", "Failed to add to whitelist: "+err.Error()),
				},
				Components: []discordgo.MessageComponent{},
			},
		})
		return
	}

	// Create success embed showing selected actions
	actionsText := "All Actions"
	if !hasAll {
		var actionNames []string
		for _, action := range selectedActions {
			actionNames = append(actionNames, models.GetActionDisplayName(action))
		}
		actionsText = strings.Join(actionNames, ", ")
	}

	embed := CreateSuccessEmbed(
		"Added to Whitelist",
		"",
	)
	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:   "Target",
			Value:  fmt.Sprintf("<%s%s>", getTargetPrefix(targetType), targetID),
			Inline: true,
		},
		{
			Name:   "Type",
			Value:  strings.Title(targetType),
			Inline: true,
		},
		{
			Name:   "Actions Whitelisted",
			Value:  actionsText,
			Inline: false,
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{},
		},
	})
}

// Helper functions

func createActionSelectOptions(selectedActions []string) []discordgo.SelectMenuOption {
	options := make([]discordgo.SelectMenuOption, 0, 13)

	// Add "All Actions" option first
	options = append(options, discordgo.SelectMenuOption{
		Label:       "All Actions",
		Value:       models.ActionAll,
		Description: "Whitelist for all protection actions",
		Default:     contains(selectedActions, models.ActionAll),
	})

	// Add individual actions
	for _, action := range models.GetAllActionTypes() {
		options = append(options, discordgo.SelectMenuOption{
			Label:       models.GetActionDisplayName(action),
			Value:       action,
			Description: fmt.Sprintf("Whitelist for %s", strings.ToLower(models.GetActionDisplayName(action))),
			Default:     contains(selectedActions, action),
		})
	}

	return options
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func getTargetPrefix(targetType string) string {
	if targetType == "role" {
		return "@&"
	}
	return "@"
}
