package utils

import (
	"discord-giveaway-bot/internal/database"
	"discord-giveaway-bot/internal/models"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

type RequirementResult struct {
	Passed bool
	Reason string
}

func CheckAllRequirements(s *discordgo.Session, db *database.Database, guildID, userID string, g *models.Giveaway) (*RequirementResult, error) {
	member, err := s.GuildMember(guildID, userID)
	if err != nil {
		return nil, err
	}

	// Role Requirement
	if g.RoleRequirement != "" {
		hasRole := false
		for _, roleID := range member.Roles {
			if roleID == g.RoleRequirement {
				hasRole = true
				break
			}
		}
		if !hasRole {
			return &RequirementResult{
				Passed: false,
				Reason: fmt.Sprintf("You need the <@&%s> role to enter", g.RoleRequirement),
			}, nil
		}
	}

	// Invite Requirement
	if g.InviteRequirement > 0 {
		invites, err := s.GuildInvites(guildID)
		if err != nil {
			// Fail safely if we can't check invites
			fmt.Printf("Error checking invites: %v\n", err)
		} else {
			userInvites := 0
			for _, inv := range invites {
				if inv.Inviter != nil && inv.Inviter.ID == userID {
					userInvites += inv.Uses
				}
			}
			if userInvites < g.InviteRequirement {
				return &RequirementResult{
					Passed: false,
					Reason: fmt.Sprintf("You need at least %d invites (you have %d)", g.InviteRequirement, userInvites),
				}, nil
			}
		}
	}

	// Account Age Requirement
	if g.AccountAgeRequirement > 0 {
		// Discord IDs are snowflakes. We can extract creation time.
		// But discordgo User struct doesn't have CreatedAt helper directly on Member.User?
		// Actually we can calculate it from ID.
		creationTime, err := discordgo.SnowflakeTimestamp(userID)
		if err == nil {
			age := time.Since(creationTime)
			ageDays := int(age.Hours() / 24)
			if ageDays < g.AccountAgeRequirement {
				return &RequirementResult{
					Passed: false,
					Reason: fmt.Sprintf("Your account must be at least %d days old (yours is %d days)", g.AccountAgeRequirement, ageDays),
				}, nil
			}
		}
	}

	// Server Age Requirement
	if g.ServerAgeRequirement > 0 {
		age := time.Since(member.JoinedAt)
		ageDays := int(age.Hours() / 24)
		if ageDays < g.ServerAgeRequirement {
			return &RequirementResult{
				Passed: false,
				Reason: fmt.Sprintf("You must be a member for at least %d days (you've been here %d days)", g.ServerAgeRequirement, ageDays),
			}, nil
		}
	}

	// Message Requirement
	if g.MessageRequired > 0 {
		stats, err := db.GetUserStats(guildID, userID)
		if err == nil {
			if stats.MessageCount < g.MessageRequired {
				return &RequirementResult{
					Passed: false,
					Reason: fmt.Sprintf("You need at least %d messages (you have %d)", g.MessageRequired, stats.MessageCount),
				}, nil
			}
		}
	}

	// Voice Requirement
	if g.VoiceRequirement > 0 {
		stats, err := db.GetUserStats(guildID, userID)
		if err == nil {
			if stats.VoiceMinutes < g.VoiceRequirement {
				return &RequirementResult{
					Passed: false,
					Reason: fmt.Sprintf("You need at least %d minutes in voice chat (you have %d)", g.VoiceRequirement, stats.VoiceMinutes),
				}, nil
			}
		}
	}

	return &RequirementResult{Passed: true}, nil
}
