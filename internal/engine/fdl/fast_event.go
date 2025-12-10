package fdl

// FastEvent is the normalized event structure that fits in a cache line (64 bytes)
// Optimized for zero-copy processing and maximum cache efficiency
type FastEvent struct {
	ReqType        uint8  // 1 byte: Internal Enum (e.g., EvtChannelDelete)
	_              [7]byte // Padding for alignment
	GuildID        uint64 // 8 bytes: Snowflake
	UserID         uint64 // 8 bytes: Snowflake
	EntityID       uint64 // 8 bytes: Target ID (Role/Channel/User)
	Timestamp      int64  // 8 bytes: Monotonic nanoseconds
	DetectionStart int64  // 8 bytes: Start time for detection speed measurement
	_              [16]byte // Padding to 64 bytes for cache line alignment
}

// Event Types (uint8) - Ordered by frequency for branch prediction optimization
const (
	EvtUnknown uint8 = iota
	EvtGuildBanAdd
	EvtGuildMemberRemove // Kick/Leave
	EvtChannelDelete
	EvtRoleDelete
	EvtRoleUpdate
	EvtChannelCreate
	EvtChannelUpdate
	EvtRoleCreate
	EvtGuildUpdate
	EvtWebhookCreate
	EvtEmojiCreate
	EvtEmojiDelete
	EvtEmojiUpdate
	EvtStickerCreate
	EvtStickerDelete
	EvtStickerUpdate
	EvtMemberUpdate
	EvtIntegrationCreate
	EvtIntegrationUpdate
	EvtIntegrationDelete
	EvtAutoModRuleCreate
	EvtAutoModRuleUpdate
	EvtAutoModRuleDelete
	EvtGuildEventCreate
	EvtGuildEventUpdate
	EvtGuildEventDelete
	EvtMessageCreate // For spam detection
	EvtWebhookUpdate
	EvtWebhookDelete
	EvtPrune
)
