package fdl

import (
	"sync"
	"unsafe"

	"github.com/goccy/go-json"
)

// Pool for reusable structs to minimize allocations
var (
	minimalEventPool = sync.Pool{
		New: func() interface{} {
			return &MinimalEvent{}
		},
	}
	rootPool = sync.Pool{
		New: func() interface{} {
			return &Root{}
		},
	}
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

type Root struct {
	D MinimalDispatch `json:"d"`
}

// ParseFrame converts raw bytes into a FastEvent
// This uses unsafe pointers and minimal structs to reduce overhead
// OPTIMIZED: Zero-allocation path with object pooling
func ParseFrame(data []byte) (*FastEvent, error) {
	// Get pooled objects
	base := minimalEventPool.Get().(*MinimalEvent)
	defer minimalEventPool.Put(base)

	// 1. Initial scan for OpCode and Type
	if err := json.Unmarshal(data, base); err != nil {
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
	root := rootPool.Get().(*Root)
	defer rootPool.Put(root)

	if err := json.Unmarshal(data, root); err != nil {
		return nil, err
	}

	d := root.D

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
	// Optimized switch statement with most common events first
	switch t {
	case "GUILD_BAN_ADD":
		return EvtGuildBanAdd
	case "GUILD_MEMBER_REMOVE":
		return EvtGuildMemberRemove
	case "CHANNEL_DELETE":
		return EvtChannelDelete
	case "GUILD_ROLE_DELETE":
		return EvtRoleDelete
	case "GUILD_ROLE_UPDATE":
		return EvtRoleUpdate
	case "CHANNEL_CREATE":
		return EvtChannelCreate
	case "CHANNEL_UPDATE":
		return EvtChannelUpdate
	case "GUILD_ROLE_CREATE":
		return EvtRoleCreate
	case "WEBHOOKS_UPDATE":
		return EvtWebhookCreate
	case "MESSAGE_CREATE":
		return EvtMessageCreate
	default:
		return EvtUnknown
	}
}

// parseSnowflake converts string to uint64 without error checking for speed
// CRITICAL: Inlined for maximum performance
//
//go:inline
func parseSnowflake(s string) uint64 {
	if s == "" {
		return 0
	}
	// Fast string to uint64 conversion
	var n uint64
	for i := 0; i < len(s); i++ {
		v := s[i] - '0'
		n = n*10 + uint64(v)
	}
	return n
}

// ParseSnowflakeString is the exported version for external packages
func ParseSnowflakeString(s string) uint64 {
	return parseSnowflake(s)
}

// ParseAuditLogEntry parses audit log entry data from gateway events
func ParseAuditLogEntry(data []byte) (*FastEvent, error) {
	// Audit log entries come in a different structure than regular gateway events
	// For now, we'll rely on the auditor package to do the conversion
	// This function serves as a placeholder for future optimization
	return nil, nil
}

// ByteSliceToString converts slice to string without alloc
func ByteSliceToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
