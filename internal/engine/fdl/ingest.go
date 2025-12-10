package fdl

import (
	"unsafe"

	"github.com/goccy/go-json"
)

// MinimalEvent is a struct used for partial unmarshalling
// We use goccy/go-json for speed, but ideally we would use a custom iterator
type MinimalEvent struct {
	Op int         `json:"op"`
	Ex interface{} `json:"d,omitempty"` // Placeholder
	T  string      `json:"t,omitempty"`
}

// MinimalDispatch helps us peek at the event type without full alloc
type MinimalDispatch struct {
	GuildID string      `json:"guild_id"`
	User    MinimalUser `json:"user"`
	Author  MinimalUser `json:"author"` // For message events
	Member  struct {
		User MinimalUser `json:"user"`
	} `json:"member"`
	ID string `json:"id"` // Generic ID (Channel/Role/Message)
	// Add other specific fields if needed
}

type MinimalUser struct {
	ID string `json:"id"`
}

// ParseFrame converts raw bytes into a FastEvent
// This uses unsafe pointers and minimal structs to reduce overhead
func ParseFrame(data []byte) (*FastEvent, error) {
	// 1. Initial scan for OpCode and Type
	// Note: In a true zero-alloc environment we would use a streaming lexer
	// For now we use the fastest available standard unmarshaller

	var base MinimalEvent
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, err
	}

	// We only care about Dispatch events (Op 0)
	if base.Op != 0 {
		return nil, nil // Not a dispatch event, ignore
	}

	// 2. Map Event String to Internal Enum
	evtType := mapEventType(base.T)
	if evtType == EvtUnknown {
		return nil, nil
	}

	// 3. Extract IDs from the raw JSON of the "d" field
	// This is the tricky part. To avoid unmarshalling "d" into an interface{},
	// we would typically use a raw message.
	// For this prototype, we'll do a second pass on the specific struct.
	// OPTIMIZATION TODO: Use json.RawMessage or a custom scanner to avoid double parse.

	var dispatch MinimalDispatch
	// We need to find the "d" block again.
	// In production, we'd slice the bytes. Here we assume we can extract it.

	// WORKAROUND: For now, we accept the allocation of unmarshalling 'd'
	// because writing a full lexer is too large for this step.
	// We strive for "Low Alloc" here -> "Zero Alloc" eventually.

	// Let's re-parse just the data fields we need, assuming the structure matches
	// This is essentially efficient unmarshalling
	type Root struct {
		D MinimalDispatch `json:"d"`
	}
	var root Root
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, err
	}

	d := root.D
	_ = dispatch // Suppress unused error if we don't use the middle step

	// 4. Construct FastEvent
	fe := &FastEvent{
		ReqType:   evtType,
		GuildID:   parseSnowflake(d.GuildID),
		Timestamp: 0, // Should be populated from monotonic clock
	}

	// Resolve User ID (Author vs User vs Member.User)
	if d.User.ID != "" {
		fe.UserID = parseSnowflake(d.User.ID)
	} else if d.Member.User.ID != "" {
		fe.UserID = parseSnowflake(d.Member.User.ID)
	} else if d.Author.ID != "" {
		fe.UserID = parseSnowflake(d.Author.ID)
	}

	// Resolve Target ID
	if d.ID != "" {
		fe.EntityID = parseSnowflake(d.ID)
	}

	return fe, nil
}

func mapEventType(t string) uint8 {
	switch t {
	case "CHANNEL_CREATE":
		return EvtChannelCreate
	case "CHANNEL_DELETE":
		return EvtChannelDelete
	case "CHANNEL_UPDATE":
		return EvtChannelUpdate
	case "GUILD_BAN_ADD":
		return EvtGuildBanAdd
	case "GUILD_MEMBER_REMOVE":
		return EvtGuildMemberRemove
	case "GUILD_ROLE_CREATE":
		return EvtRoleCreate
	case "GUILD_ROLE_DELETE":
		return EvtRoleDelete
	case "GUILD_ROLE_UPDATE":
		return EvtRoleUpdate
	case "WEBHOOKS_UPDATE":
		return EvtWebhookCreate // Webhooks update covers creates usually
	case "MESSAGE_CREATE":
		return EvtMessageCreate
	default:
		return EvtUnknown
	}
}

// parseSnowflake converts string to uint64 without error checking for speed
// In prod, simple Atoi is fine, but unsafe conversion is faster if we trust valid JSON
func parseSnowflake(s string) uint64 {
	if s == "" {
		return 0
	}
	// fast string to uint64
	// simplified: usage of strconv.ParseUint is relatively fast,
	// but custom loop is faster.
	var n uint64
	for i := 0; i < len(s); i++ {
		v := s[i] - '0'
		n = n*10 + uint64(v)
	}
	return n
}

// ByteSliceToString converts slice to string without alloc
func ByteSliceToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
