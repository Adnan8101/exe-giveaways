package fdl

// FastEvent is the normalized event structure that fits in a cache line
// It is 32-40 bytes depending on alignment
type FastEvent struct {
	ReqType   uint8  // 1 byte: Internal Enum (e.g., EvtChannelDelete)
	GuildID   uint64 // 8 bytes: Snowflake
	UserID    uint64 // 8 bytes: Snowflake
	EntityID  uint64 // 8 bytes: Target ID (Role/Channel/User)
	Timestamp int64  // 8 bytes: Monotonic nanoseconds
}

// Event Types (uint8)
const (
	EvtUnknown uint8 = iota
	EvtChannelCreate
	EvtChannelDelete
	EvtChannelUpdate
	EvtRoleCreate
	EvtRoleDelete
	EvtRoleUpdate
	EvtGuildBanAdd
	EvtGuildMemberRemove // Kick/Leave
	EvtGuildUpdate
	EvtWebhookCreate
	EvtMessageCreate // For spam detection
)
