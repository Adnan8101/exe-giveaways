package pool

import (
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

// Global object pools for frequently allocated objects

var (
	// EmbedPool for MessageEmbed objects
	EmbedPool = sync.Pool{
		New: func() interface{} {
			return &discordgo.MessageEmbed{}
		},
	}

	// EmbedFieldPool for MessageEmbedField slices
	EmbedFieldPool = sync.Pool{
		New: func() interface{} {
			return make([]*discordgo.MessageEmbedField, 0, 8)
		},
	}

	// StringBuilderPool for efficient string concatenation
	StringBuilderPool = sync.Pool{
		New: func() interface{} {
			return new(strings.Builder)
		},
	}

	// OptionMapPool for command option maps
	OptionMapPool = sync.Pool{
		New: func() interface{} {
			return make(map[string]*discordgo.ApplicationCommandInteractionDataOption, 16)
		},
	}

	// StringSlicePool for string slices
	StringSlicePool = sync.Pool{
		New: func() interface{} {
			s := make([]string, 0, 16)
			return &s
		},
	}
)

// GetEmbed retrieves a MessageEmbed from the pool
func GetEmbed() *discordgo.MessageEmbed {
	return EmbedPool.Get().(*discordgo.MessageEmbed)
}

// PutEmbed returns a MessageEmbed to the pool after resetting it
func PutEmbed(e *discordgo.MessageEmbed) {
	// Reset the embed
	*e = discordgo.MessageEmbed{}
	EmbedPool.Put(e)
}

// GetEmbedFields retrieves an embed field slice from the pool
func GetEmbedFields() []*discordgo.MessageEmbedField {
	fields := EmbedFieldPool.Get().([]*discordgo.MessageEmbedField)
	return fields[:0] // Reset length but keep capacity
}

// PutEmbedFields returns embed fields to the pool
func PutEmbedFields(fields []*discordgo.MessageEmbedField) {
	// Clear references to prevent memory leaks
	for i := range fields {
		fields[i] = nil
	}
	EmbedFieldPool.Put(fields[:0])
}

// GetStringSlice retrieves a string slice from the pool
func GetStringSlice() *[]string {
	slice := StringSlicePool.Get().(*[]string)
	*slice = (*slice)[:0] // Reset length
	return slice
}

// PutStringSlice returns a string slice to the pool
func PutStringSlice(s *[]string) {
	if cap(*s) > 1024 { // Don't pool very large slices
		return
	}
	*s = (*s)[:0]
	StringSlicePool.Put(s)
}

// GetOptionMap retrieves an option map from the pool
func GetOptionMap() map[string]*discordgo.ApplicationCommandInteractionDataOption {
	m := OptionMapPool.Get().(map[string]*discordgo.ApplicationCommandInteractionDataOption)
	// Clear the map
	for k := range m {
		delete(m, k)
	}
	return m
}

// PutOptionMap returns an option map to the pool
func PutOptionMap(m map[string]*discordgo.ApplicationCommandInteractionDataOption) {
	if len(m) > 64 { // Don't pool very large maps
		return
	}
	for k := range m {
		delete(m, k)
	}
	OptionMapPool.Put(m)
}
