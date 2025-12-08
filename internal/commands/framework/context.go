package framework

import (
	"github.com/bwmarrin/discordgo"
)

type Context interface {
	GetSession() *discordgo.Session
	GetGuildID() string
	GetChannelID() string
	GetAuthor() *discordgo.User
	GetMember() *discordgo.Member
	GetArgs() []string
	Reply(content string) error
	ReplyEphemeral(content string) error
	ReplyEmbed(embed *discordgo.MessageEmbed) error
	ReplyComponent(embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) error
}

// SlashContext implements Context for Slash Commands
type SlashContext struct {
	Session     *discordgo.Session
	Interaction *discordgo.InteractionCreate
	Args        []string
}

func NewSlashContext(s *discordgo.Session, i *discordgo.InteractionCreate) *SlashContext {
	return &SlashContext{Session: s, Interaction: i}
}

func NewSlashContextWithArgs(s *discordgo.Session, i *discordgo.InteractionCreate, args []string) *SlashContext {
	return &SlashContext{Session: s, Interaction: i, Args: args}
}

func (c *SlashContext) GetSession() *discordgo.Session {
	return c.Session
}

func (c *SlashContext) GetGuildID() string {
	return c.Interaction.GuildID
}

func (c *SlashContext) GetChannelID() string {
	return c.Interaction.ChannelID
}

func (c *SlashContext) GetAuthor() *discordgo.User {
	if c.Interaction.Member != nil {
		return c.Interaction.Member.User
	}
	return c.Interaction.User
}

func (c *SlashContext) GetMember() *discordgo.Member {
	return c.Interaction.Member
}

func (c *SlashContext) GetArgs() []string {
	return c.Args
}

func (c *SlashContext) Reply(content string) error {
	return c.Session.InteractionRespond(c.Interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
}

func (c *SlashContext) ReplyEphemeral(content string) error {
	return c.Session.InteractionRespond(c.Interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (c *SlashContext) ReplyEmbed(embed *discordgo.MessageEmbed) error {
	return c.Session.InteractionRespond(c.Interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (c *SlashContext) ReplyComponent(embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) error {
	return c.Session.InteractionRespond(c.Interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
}

// PrefixContext implements Context for Prefix Commands
type PrefixContext struct {
	Session *discordgo.Session
	Message *discordgo.MessageCreate
	Args    []string
}

func NewPrefixContext(s *discordgo.Session, m *discordgo.MessageCreate, args []string) *PrefixContext {
	return &PrefixContext{Session: s, Message: m, Args: args}
}

func (c *PrefixContext) GetSession() *discordgo.Session {
	return c.Session
}

func (c *PrefixContext) GetGuildID() string {
	return c.Message.GuildID
}

func (c *PrefixContext) GetChannelID() string {
	return c.Message.ChannelID
}

func (c *PrefixContext) GetAuthor() *discordgo.User {
	return c.Message.Author
}

func (c *PrefixContext) GetMember() *discordgo.Member {
	return c.Message.Member
}

func (c *PrefixContext) GetArgs() []string {
	return c.Args
}

func (c *PrefixContext) Reply(content string) error {
	_, err := c.Session.ChannelMessageSend(c.Message.ChannelID, content)
	return err
}

func (c *PrefixContext) ReplyEphemeral(content string) error {
	// Ephemeral messages don't exist for normal messages, so we just DM or reply normally.
	// For now, reply normally but maybe delete after some time?
	// Or just reply normally.
	_, err := c.Session.ChannelMessageSend(c.Message.ChannelID, content)
	return err
}

func (c *PrefixContext) ReplyEmbed(embed *discordgo.MessageEmbed) error {
	_, err := c.Session.ChannelMessageSendEmbed(c.Message.ChannelID, embed)
	return err
}

func (c *PrefixContext) ReplyComponent(embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) error {
	msg := &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	}
	_, err := c.Session.ChannelMessageSendComplex(c.Message.ChannelID, msg)
	return err
}
