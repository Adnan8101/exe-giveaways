package fdl

// FastEvent is the normalized event structure that fits in a cache line
// It is 32-40 bytes depending on alignment
type FastEvent struct {
	ReqType        uint8  // 1 byte: Internal Enum (e.g., EvtChannelDelete)
	GuildID        uint64 // 8 bytes: Snowflake
	UserID         uint64 // 8 bytes: Snowflake
	EntityID       uint64 // 8 bytes: Target ID (Role/Channel/User)
	Timestamp      int64  // 8 bytes: Monotonic nanoseconds
	DetectionStart int64  // 8 bytes: Start time for detection speed measurement
}

// Event Types (uint8) - Comprehensive coverage for all antinuke events
const (
	EvtUnknown uint8 = iota
	EvtChannelCreate
	EvtChannelDelete
	EvtChannelUpdate
	EvtRoleCreate
	EvtRoleDelete
	EvtRoleUpdate
	EvtGuildBanAdd
	EvtGuildUnban
	EvtGuildMemberRemove // Kick/Leave
	EvtGuildUpdate
	EvtWebhookCreate
	EvtWebhookUpdate
	EvtWebhookDelete
	EvtMessageCreate // For spam detection
	EvtEmojiCreate
	EvtEmojiDelete
	EvtEmojiUpdate
	EvtMemberUpdate
	EvtIntegrationCreate
	EvtIntegrationUpdate
	EvtIntegrationDelete
	EvtAutomodCreate
	EvtAutomodUpdate
	EvtAutomodDelete
	EvtEventCreate
	EvtEventUpdate
	EvtEventDelete
	EvtMemberPrune
	EvtBotAdd
	EvtRolePing
	EvtEveryonePing
)
