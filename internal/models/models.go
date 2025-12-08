package models

import "time"

type Giveaway struct {
	ID            int64  `json:"id"`
	MessageID     string `json:"message_id"`
	ChannelID     string `json:"channel_id"`
	GuildID       string `json:"guild_id"`
	HostID        string `json:"host_id"`
	Prize         string `json:"prize"`
	WinnersCount  int    `json:"winners_count"`
	EndTime       int64  `json:"end_time"` // Unix timestamp in milliseconds
	Ended         bool   `json:"ended"`
	CreatedAt     int64  `json:"created_at"`
	CustomMessage string `json:"custom_message"`

	// Requirements
	RoleRequirement       string `json:"role_requirement"`
	InviteRequirement     int    `json:"invite_requirement"`
	AccountAgeRequirement int    `json:"account_age_requirement"`
	ServerAgeRequirement  int    `json:"server_age_requirement"`
	CaptchaRequirement    bool   `json:"captcha_requirement"`
	MessageRequired       int    `json:"message_required"`
	VoiceRequirement      int    `json:"voice_requirement"`
	EntryFee              int    `json:"entry_fee"`

	// New Features
	AssignRole string `json:"assign_role"`
	Thumbnail  string `json:"thumbnail"`
	Emoji      string `json:"emoji"` // Custom emoji for giveaway reactions
}

type Participant struct {
	ID         int64  `json:"id"`
	GiveawayID int64  `json:"giveaway_id"`
	UserID     string `json:"user_id"`
	JoinedAt   int64  `json:"joined_at"`
}

type Winner struct {
	ID         int64  `json:"id"`
	GiveawayID int64  `json:"giveaway_id"`
	UserID     string `json:"user_id"`
	WonAt      int64  `json:"won_at"`
}

type UserStats struct {
	ID           int64  `json:"id"`
	GuildID      string `json:"guild_id"`
	UserID       string `json:"user_id"`
	MessageCount int    `json:"message_count"`
	VoiceMinutes int    `json:"voice_minutes"`
}

type CaptchaSession struct {
	ID         int64  `json:"id"`
	UserID     string `json:"user_id"`
	GiveawayID int64  `json:"giveaway_id"`
	Code       string `json:"code"`
	CreatedAt  int64  `json:"created_at"`
}

type EconomyUser struct {
	UserID      string `json:"user_id"`
	GuildID     string `json:"guild_id"`
	Balance     int64  `json:"balance"`
	LastDaily   int64  `json:"last_daily"`
	LastWeekly  int64  `json:"last_weekly"`
	LastHourly  int64  `json:"last_hourly"`
	TotalEarned int64  `json:"total_earned"`
	TotalSpent  int64  `json:"total_spent"`
}

type EconomyConfig struct {
	GuildID         string `json:"guild_id"`
	MessageReward   int    `json:"message_reward"`
	VCRewardPerMin  int    `json:"vc_reward_per_min"`
	DailyReward     int    `json:"daily_reward"`
	WeeklyReward    int    `json:"weekly_reward"`
	HourlyReward    int    `json:"hourly_reward"`
	InviteReward    int    `json:"invite_reward"`
	ReactReward     int    `json:"react_reward"`
	PollReward      int    `json:"poll_reward"`
	EventReward     int    `json:"event_reward"`
	UpvoteReward    int    `json:"upvote_reward"`
	GambleEnabled   bool   `json:"gamble_enabled"`
	MaxGambleAmount int    `json:"max_gamble_amount"`
	AllowedChannels string `json:"allowed_channels"` // Comma-separated list of channel IDs
	CurrencyEmoji   string `json:"currency_emoji"`
}

type ShopItem struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Price           int    `json:"price"`
	Stock           int    `json:"stock"` // -1 for infinite
	Type            string `json:"type"`  // item, role, boost
	RoleID          string `json:"role_id"`
	Duration        int    `json:"duration"` // seconds
	RequiredBalance int    `json:"required_balance"`
	RoleRequired    string `json:"role_required"`
	ReplyMessage    string `json:"reply_message"`
	ImageURL        string `json:"image_url"`
	Hidden          bool   `json:"hidden"`
	CreatedAt       int64  `json:"created_at"`
}

type InventoryItem struct {
	ID         int64  `json:"id"`
	UserID     string `json:"user_id"`
	GuildID    string `json:"guild_id"`
	ItemID     int64  `json:"item_id"`
	Quantity   int    `json:"quantity"`
	AcquiredAt int64  `json:"acquired_at"`
	ExpiresAt  int64  `json:"expires_at"`

	// Joined fields
	ItemName string `json:"item_name"`
	ItemType string `json:"item_type"`
}

type RedeemCode struct {
	Code      string `json:"code"`
	ItemID    int64  `json:"item_id"`
	UserID    string `json:"user_id"`
	GuildID   string `json:"guild_id"`
	IsClaimed bool   `json:"is_claimed"`
	CreatedAt int64  `json:"created_at"`

	// Joined fields
	ItemName        string `json:"item_name"`
	ItemDescription string `json:"item_description"`
	ItemPrice       int    `json:"item_price"`
}

// Helper to convert bool to int for SQLite
func BoolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Helper to convert int to bool for SQLite
func IntToBool(i int) bool {
	return i != 0
}

// Helper to get current time in milliseconds
func Now() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
