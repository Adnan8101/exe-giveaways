package bot

import (
	"github.com/bwmarrin/discordgo"
)

// Unified event handlers to eliminate duplicate registrations and improve performance

// UnifiedMessageCreate consolidates all message create event handling
func (b *Bot) UnifiedMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Fast-path: skip bots immediately
	if m.Author.Bot {
		return
	}

	// Route to both tracker and economy events concurrently
	go b.MessageCreate(s, m)
	b.EconomyEvents.OnMessageCreate(s, m)
}

// UnifiedVoiceStateUpdate consolidates all voice state update event handling
func (b *Bot) UnifiedVoiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	// Route to both tracker and economy events concurrently
	go b.VoiceStateUpdate(s, v)
	b.EconomyEvents.OnVoiceStateUpdate(s, v)
}

// UnifiedMessageReactionAdd consolidates all reaction add event handling
func (b *Bot) UnifiedMessageReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	// Fast-path: skip bot reactions
	if r.UserID == s.State.User.ID {
		return
	}

	// Route to both giveaway and economy handlers concurrently
	go b.MessageReactionAdd(s, r)
	b.EconomyEvents.OnMessageReactionAdd(s, r)
}

// UnifiedMessageReactionRemove consolidates all reaction remove event handling
func (b *Bot) UnifiedMessageReactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	// Fast-path: skip bot reactions
	if r.UserID == s.State.User.ID {
		return
	}

	// Route to both giveaway and economy handlers concurrently
	go b.MessageReactionRemove(s, r)
	b.EconomyEvents.OnMessageReactionRemove(s, r)
}
