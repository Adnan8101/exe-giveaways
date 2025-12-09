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
	Reply(content string) (*discordgo.Message, error)
	ReplyEphemeral(content string) error
	ReplyEmbed(embed *discordgo.MessageEmbed) (*discordgo.Message, error)
	ReplyComponent(embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) error
	EditReplyEmbed(msg *discordgo.Message, embed *discordgo.MessageEmbed) error
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

func (c *SlashContext) Reply(content string) (*discordgo.Message, error) {
	err := c.Session.InteractionRespond(c.Interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
	return nil, err
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

func (c *SlashContext) ReplyEmbed(embed *discordgo.MessageEmbed) (*discordgo.Message, error) {
	err := c.Session.InteractionRespond(c.Interaction.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	return nil, err
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

func (c *SlashContext) EditReplyEmbed(msg *discordgo.Message, embed *discordgo.MessageEmbed) error {
	_, err := c.Session.InteractionResponseEdit(c.Interaction.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
	return err
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

func (c *PrefixContext) Reply(content string) (*discordgo.Message, error) {
	return c.Session.ChannelMessageSend(c.Message.ChannelID, content)
}

func (c *PrefixContext) ReplyEphemeral(content string) error {
	// Ephemeral messages don't exist for normal messages, so we just DM or reply normally.
	// For now, reply normally but maybe delete after some time?
	// Or just reply normally.
	_, err := c.Session.ChannelMessageSend(c.Message.ChannelID, content)
	return err
}

func (c *PrefixContext) ReplyEmbed(embed *discordgo.MessageEmbed) (*discordgo.Message, error) {
	return c.Session.ChannelMessageSendEmbed(c.Message.ChannelID, embed)
}

func (c *PrefixContext) ReplyComponent(embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) error {
	msg := &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	}
	_, err := c.Session.ChannelMessageSendComplex(c.Message.ChannelID, msg)
	return err
}

func (c *PrefixContext) EditReplyEmbed(msg *discordgo.Message, embed *discordgo.MessageEmbed) error {
	if msg == nil {
		return nil // Should not happen for PrefixContext unless Reply failed
	}
	_, err := c.Session.ChannelMessageEditEmbed(c.Message.ChannelID, msg.ID, embed)
	return err
}
