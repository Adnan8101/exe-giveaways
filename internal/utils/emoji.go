package utils

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// ParseAndStealEmoji parses a custom emoji, steals it if not in server, and returns the emoji string to use
func ParseAndStealEmoji(s *discordgo.Session, guildID string, emojiInput string) (string, error) {
	// If it's a standard unicode emoji (no angle brackets), just return it
	if !strings.Contains(emojiInput, "<") {
		return emojiInput, nil
	}

	// Parse custom emoji format: <:name:id> or <a:name:id> for animated
	re := regexp.MustCompile(`<(a?):([^:]+):(\d+)>`)
	matches := re.FindStringSubmatch(emojiInput)

	if len(matches) != 4 {
		return "🎉", fmt.Errorf("invalid emoji format, using default")
	}

	animated := matches[1] == "a"
	name := matches[2]
	id := matches[3]

	return StealEmoji(s, guildID, name, id, animated)
}

// StealEmoji checks if emoji exists in server, if not downloads and adds it
func StealEmoji(s *discordgo.Session, guildID string, name string, emojiID string, animated bool) (string, error) {
	// Check if emoji already exists in the guild
	guild, err := s.Guild(guildID)
	if err == nil {
		for _, emoji := range guild.Emojis {
			if emoji.ID == emojiID {
				// Emoji already exists, return in reaction format (name:id)
				if animated {
					return fmt.Sprintf("a:%s:%s", emoji.Name, emoji.ID), nil
				}
				return fmt.Sprintf("%s:%s", emoji.Name, emoji.ID), nil
			}
		}
	}

	// Emoji doesn't exist, steal it
	// Download emoji image
	extension := "png"
	if animated {
		extension = "gif"
	}
	emojiURL := fmt.Sprintf("https://cdn.discordapp.com/emojis/%s.%s", emojiID, extension)

	resp, err := http.Get(emojiURL)
	if err != nil {
		return "🎉", fmt.Errorf("failed to download emoji: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "🎉", fmt.Errorf("emoji not found (status %d)", resp.StatusCode)
	}

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "🎉", fmt.Errorf("failed to read emoji data: %w", err)
	}

	// Validate image data
	if len(imageData) == 0 {
		return "🎉", fmt.Errorf("empty image data")
	}

	// Create emoji in the guild with proper base64 encoding
	// Discord expects: "data:image/png;base64,BASE64_DATA"
	mimeType := "image/png"
	if animated {
		mimeType = "image/gif"
	}

	base64Data := base64.StdEncoding.EncodeToString(imageData)
	dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)

	newEmoji, err := s.GuildEmojiCreate(guildID, &discordgo.EmojiParams{
		Name:  name,
		Image: dataURI,
	})
	if err != nil {
		return "🎉", fmt.Errorf("failed to create emoji: %w", err)
	}

	// Return formatted emoji string for reactions (name:id format)
	if animated {
		return fmt.Sprintf("a:%s:%s", newEmoji.Name, newEmoji.ID), nil
	}
	return fmt.Sprintf("%s:%s", newEmoji.Name, newEmoji.ID), nil
}
