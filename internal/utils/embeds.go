package utils

import (
	"discord-giveaway-bot/internal/models"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func CreateGiveawayEmbed(g *models.Giveaway, participantCount int) *discordgo.MessageEmbed {
	endTime := time.Unix(0, g.EndTime*int64(time.Millisecond))

	description := fmt.Sprintf("**Winners:** %d\n**Hosted By:** <@%s>\n\nEnds in: <t:%d:R> (<t:%d:f>)\n", g.WinnersCount, g.HostID, endTime.Unix(), endTime.Unix())

	var reqs []string
	if g.EntryFee > 0 {
		reqs = append(reqs, fmt.Sprintf("• **Entry Fee:** %d coins", g.EntryFee))
	}
	if g.RoleRequirement != "" {
		reqs = append(reqs, fmt.Sprintf("• **Required Role:** <@&%s>", g.RoleRequirement))
	}
	if g.InviteRequirement > 0 {
		reqs = append(reqs, fmt.Sprintf("• **Invites:** %d+", g.InviteRequirement))
	}
	if g.AccountAgeRequirement > 0 {
		reqs = append(reqs, fmt.Sprintf("• **Account Age:** %d+ days", g.AccountAgeRequirement))
	}
	if g.ServerAgeRequirement > 0 {
		reqs = append(reqs, fmt.Sprintf("• **Server Age:** %d+ days", g.ServerAgeRequirement))
	}
	if g.MessageRequired > 0 {
		reqs = append(reqs, fmt.Sprintf("• **Messages:** %d+", g.MessageRequired))
	}
	if g.VoiceRequirement > 0 {
		reqs = append(reqs, fmt.Sprintf("• **Voice Time:** %d+ mins", g.VoiceRequirement))
	}
	if g.CaptchaRequirement {
		reqs = append(reqs, "• **Captcha Verification**")
	}

	if len(reqs) > 0 {
		description += "\n**Requirements:**\n" + strings.Join(reqs, "\n")
	}

	description += "\n\nReact with 🎉 to enter!"

	embed := &discordgo.MessageEmbed{
		Title:       g.Prize,
		Description: description,
		Color:       0x2f3136, // Dark embed color like Giveaway Boat
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("%d Participants", participantCount),
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if g.Thumbnail != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: g.Thumbnail,
		}
	}

	return embed
}

func CreateGiveawayButton(giveawayID string) *discordgo.Button {
	return &discordgo.Button{
		Label:    "Enter Giveaway",
		Style:    discordgo.SuccessButton,
		CustomID: "enter_giveaway_" + giveawayID,
		Emoji: &discordgo.ComponentEmoji{
			Name: "🎉",
		},
	}
}

func GiveawayEmbed(g *models.Giveaway, participantCount int) *discordgo.MessageEmbed {
	// Same as CreateGiveawayEmbed but maybe for updates?
	// Let's just reuse logic or make it slightly different if needed.
	// For now, identical.
	return CreateGiveawayEmbed(g, participantCount)
}

func GiveawayEndedEmbed(g *models.Giveaway, winners []string) *discordgo.MessageEmbed {
	winnerMentions := "No valid entrants"
	if len(winners) > 0 {
		var mentions []string
		for _, id := range winners {
			mentions = append(mentions, fmt.Sprintf("<@%s>", id))
		}
		winnerMentions = strings.Join(mentions, ", ")
	}

	return &discordgo.MessageEmbed{
		Title:       "Giveaway Ended",
		Description: fmt.Sprintf("**Prize:** %s\n**Winners:** %s\n**Hosted By:** <@%s>", g.Prize, winnerMentions, g.HostID),
		Color:       0x000000, // Black/Dark
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Ended",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func GiveawayCancelledEmbed(g *models.Giveaway) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "Giveaway Cancelled",
		Description: fmt.Sprintf("**Prize:** %s\n\n❌ This giveaway was cancelled by a host.", g.Prize),
		Color:       0xFF0000, // Red
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Cancelled",
		},
	}
}
