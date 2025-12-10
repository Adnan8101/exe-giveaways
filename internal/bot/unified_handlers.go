package bot

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

// Unified event handlers to eliminate duplicate registrations and improve performance

// UnifiedMessageCreate consolidates all message create event handling
func (b *Bot) UnifiedMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	start := time.Now()
	defer func() { b.PerfMonitor.TrackEvent(time.Since(start)) }()

	// Fast-path: skip bots immediately
	if m.Author.Bot {
		return
	}

	// Route to both tracker and command handler concurrently
	go b.MessageCreate(s, m)
	go b.HandlePrefixCommand(m)
}

// UnifiedVoiceStateUpdate consolidates all voice state update event handling
func (b *Bot) UnifiedVoiceStateUpdate(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
	start := time.Now()
	defer func() { b.PerfMonitor.TrackEvent(time.Since(start)) }()

	// Route to both tracker and economy events concurrently
	go b.VoiceStateUpdate(s, v)
	go b.EconomyEvents.OnVoiceStateUpdate(s, v)
}

// UnifiedMessageReactionAdd consolidates all reaction add event handling
func (b *Bot) UnifiedMessageReactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	start := time.Now()
	defer func() { b.PerfMonitor.TrackEvent(time.Since(start)) }()

	// Fast-path: skip bot reactions
	if r.UserID == s.State.User.ID {
		return
	}

	// Route to both giveaway and economy handlers concurrently
	go b.MessageReactionAdd(s, r)
	go b.EconomyEvents.OnMessageReactionAdd(s, r)
}

// UnifiedMessageReactionRemove consolidates all reaction remove event handling
func (b *Bot) UnifiedMessageReactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	start := time.Now()
	defer func() { b.PerfMonitor.TrackEvent(time.Since(start)) }()

	// Fast-path: skip bot reactions
	if r.UserID == s.State.User.ID {
		return
	}

	// Route to both giveaway and economy handlers concurrently
	go b.MessageReactionRemove(s, r)
	go b.EconomyEvents.OnMessageReactionRemove(s, r)
}
