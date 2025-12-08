package bot

import (
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore bot messages
	if m.Author.Bot {
		return
	}

	// Prefix command handling
	prefix := "!"
	if strings.HasPrefix(m.Content, prefix) {
		b.HandlePrefixCommand(m)
		return // Stop further processing if it's a command
	}

	// Track user stats
	if m.GuildID != "" {
		err := b.DB.IncrementMessageCount(m.GuildID, m.Author.ID)
		if err != nil {
			log.Printf("Error incrementing message count: %v", err)
		}
	}
}

func (b *Bot) VoiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	userID := v.UserID
	guildID := v.GuildID

	// Check autodrag rules when user joins a VC
	if v.ChannelID != "" {
		// User joined or switched to a channel
		targetChannelID, err := b.DB.GetAutoDragRule(guildID, userID)
		if err == nil && targetChannelID != "" && targetChannelID != v.ChannelID {
			// Move user to target channel
			err = s.GuildMemberMove(guildID, userID, &targetChannelID)
			if err != nil {
				log.Printf("Error auto-dragging user %s: %v", userID, err)
			} else {
				log.Printf("Auto-dragged user %s to channel %s", userID, targetChannelID)
				// Delete the autodrag rule after successful execution (only works once)
				err = b.DB.DeleteAutoDragRule(guildID, userID)
				if err != nil {
					log.Printf("Error deleting autodrag rule for user %s: %v", userID, err)
				} else {
					log.Printf("Deleted autodrag rule for user %s (auto-drag only works once)", userID)
				}
			}
		}
	}

	b.VoiceMutex.Lock()
	defer b.VoiceMutex.Unlock()

	// Check if user was previously in a channel
	joinTime, wasInVoice := b.VoiceSessions[userID]

	// Determine if user is currently in a channel (and not deafened/muted if we wanted to be strict, but simple presence for now)
	isInVoice := v.ChannelID != ""

	if wasInVoice {
		// User was in voice, calculate duration
		duration := time.Since(joinTime)
		minutes := int(duration.Minutes())

		if minutes > 0 {
			err := b.DB.AddVoiceMinutes(guildID, userID, minutes)
			if err != nil {
				log.Printf("Error adding voice minutes: %v", err)
			}
		}
	}

	if isInVoice {
		// User is joining or switching to a channel, start new session
		// If they were already in voice (switching), we just closed the old session above and start a new one here.
		// This effectively tracks total time correctly.
		b.VoiceSessions[userID] = time.Now()
	} else {
		// User left voice completely
		delete(b.VoiceSessions, userID)
	}
}
