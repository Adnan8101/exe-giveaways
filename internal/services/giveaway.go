package services

import (
	"discord-giveaway-bot/internal/database"
	"discord-giveaway-bot/internal/models"
	"discord-giveaway-bot/internal/utils"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type GiveawayService struct {
	Session        *discordgo.Session
	DB             *database.Database
	EconomyService *EconomyService
}

func NewGiveawayService(s *discordgo.Session, db *database.Database, economyService *EconomyService) *GiveawayService {
	return &GiveawayService{
		Session:        s,
		DB:             db,
		EconomyService: economyService,
	}
}

func (s *GiveawayService) EndGiveaway(messageID string) error {
	g, err := s.DB.GetGiveaway(messageID)
	if err != nil || g == nil {
		log.Printf("Giveaway not found or error: %v", err)
		return fmt.Errorf("giveaway not found")
	}

	if g.Ended {
		return nil
	}

	// Mark as ended in DB
	if err := s.DB.EndGiveaway(messageID); err != nil {
		log.Printf("Error marking giveaway as ended: %v", err)
		return err
	}

	// Get participants
	participants, err := s.DB.GetParticipants(g.ID)
	if err != nil {
		log.Printf("Error getting participants: %v", err)
		return err
	}

	// Select winners
	winners := s.SelectWinners(participants, g.WinnersCount)

	// Save winners concurrently
	if len(winners) > 0 {
		for _, winnerID := range winners {
			go s.DB.AddWinner(g.ID, winnerID)
		}
		// Small delay to ensure all saves complete
		time.Sleep(100 * time.Millisecond)
	}

	// Check for auto-coin distribution
	var coinReward int64
	// Regex to match "100 coins", "500 exe coins", etc. case insensitive
	// Matches number followed by optional space and "coins" or "exe coins"
	re := regexp.MustCompile(`(?i)(\d+)\s*(?:exe\s*)?coins`)
	matches := re.FindStringSubmatch(g.Prize)
	if len(matches) > 1 {
		amount, err := strconv.ParseInt(matches[1], 10, 64)
		if err == nil && amount > 0 {
			coinReward = amount
		}
	}

	// Update message and announce winners concurrently
	go func() {
		embed := utils.GiveawayEndedEmbed(g, winners)
		if _, err := s.Session.ChannelMessageEditEmbed(g.ChannelID, g.MessageID, embed); err != nil {
			log.Printf("Error updating giveaway message: %v", err)
		}
	}()

	go func() {
		// Announce winners
		if len(winners) > 0 {
			var mentions []string
			for _, id := range winners {
				mentions = append(mentions, fmt.Sprintf("<@%s>", id))

				// Distribute coins if applicable
				if coinReward > 0 {
					err := s.EconomyService.AddCoins(g.GuildID, id, coinReward)
					if err != nil {
						log.Printf("Failed to add auto-coins to winner %s: %v", id, err)
					} else {
						log.Printf("Added %d auto-coins to winner %s", coinReward, id)
					}
				}
			}
			content := fmt.Sprintf("Congrats, %s you have won **%s**\nhosted by <@%s>", strings.Join(mentions, ", "), g.Prize, g.HostID)

			if coinReward > 0 {
				config, _ := s.EconomyService.GetConfig(g.GuildID)
				emoji := "<:Cash:1443554334670327848>"
				if config != nil {
					emoji = config.CurrencyEmoji
				}
				content += fmt.Sprintf("\n\n💰 **%d** %s have been automatically added to your balance!", coinReward, emoji)
			}

			msg := &discordgo.MessageSend{
				Content: content,
				Reference: &discordgo.MessageReference{
					MessageID: g.MessageID,
					ChannelID: g.ChannelID,
					GuildID:   g.GuildID,
				},
			}
			s.Session.ChannelMessageSendComplex(g.ChannelID, msg)
		} else {
			content := fmt.Sprintf("No valid participants for the giveaway: **%s**", g.Prize)
			s.Session.ChannelMessageSend(g.ChannelID, content)
		}
	}()

	return nil
}

func (s *GiveawayService) RerollGiveaway(messageID string) ([]string, error) {
	g, err := s.DB.GetGiveaway(messageID)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, fmt.Errorf("giveaway not found")
	}
	if !g.Ended {
		return nil, fmt.Errorf("giveaway has not ended yet")
	}

	participants, err := s.DB.GetParticipants(g.ID)
	if err != nil {
		return nil, err
	}

	winners := s.SelectWinners(participants, 1)

	if len(winners) > 0 {
		content := fmt.Sprintf("🎉 New winner: <@%s>! You won **%s**!", winners[0], g.Prize)

		// Check for auto-coin distribution on reroll too
		re := regexp.MustCompile(`(?i)(\d+)\s*(?:exe\s*)?coins`)
		matches := re.FindStringSubmatch(g.Prize)
		if len(matches) > 1 {
			amount, err := strconv.ParseInt(matches[1], 10, 64)
			if err == nil && amount > 0 {
				err := s.EconomyService.AddCoins(g.GuildID, winners[0], amount)
				if err == nil {
					config, _ := s.EconomyService.GetConfig(g.GuildID)
					emoji := "<:Cash:1443554334670327848>"
					if config != nil {
						emoji = config.CurrencyEmoji
					}
					content += fmt.Sprintf("\n💰 **%d** %s have been automatically added to your balance!", amount, emoji)
				}
			}
		}

		s.Session.ChannelMessageSend(g.ChannelID, content)
	}

	return winners, nil
}

func (s *GiveawayService) SelectWinners(participants []string, count int) []string {
	if len(participants) == 0 {
		return []string{}
	}

	rand.Seed(time.Now().UnixNano())
	perm := rand.Perm(len(participants))

	limit := count
	if len(participants) < limit {
		limit = len(participants)
	}

	winners := make([]string, limit)
	for i := 0; i < limit; i++ {
		winners[i] = participants[perm[i]]
	}

	return winners
}

func (s *GiveawayService) CancelGiveaway(messageID string) error {
	g, err := s.DB.GetGiveaway(messageID)
	if err != nil {
		return err
	}
	if g == nil {
		return fmt.Errorf("giveaway not found")
	}
	if g.Ended {
		return fmt.Errorf("giveaway already ended")
	}

	// Mark as ended in DB
	if err := s.DB.EndGiveaway(messageID); err != nil {
		return err
	}

	// Update message to show cancelled
	embed := utils.GiveawayCancelledEmbed(g)
	_, err = s.Session.ChannelMessageEditEmbed(g.ChannelID, g.MessageID, embed)
	if err != nil {
		log.Printf("Error updating giveaway message: %v", err)
	}

	return nil
}

func (s *GiveawayService) UpdateGiveawayMessage(g *models.Giveaway) {
	count, _ := s.DB.GetParticipantCount(g.ID)
	embed := utils.GiveawayEmbed(g, count)
	s.Session.ChannelMessageEditEmbed(g.ChannelID, g.MessageID, embed)
}
