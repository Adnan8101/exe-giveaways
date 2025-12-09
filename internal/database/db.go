package database

import (
	"context"
	"database/sql"
	"discord-giveaway-bot/internal/models"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

type Database struct {
	db               *sql.DB
	PreparedPingStmt *sql.Stmt
	PreparedStmts    *PreparedStatements
	// Cache for ping results
	lastPingTime   time.Time
	lastPingError  error
	pingCacheMutex sync.RWMutex
}

type PostgresConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	SSLMode  string `json:"sslmode"`
}

const schema = `
-- Giveaways table
CREATE TABLE IF NOT EXISTS giveaways (
    id SERIAL PRIMARY KEY,
    message_id TEXT UNIQUE NOT NULL,
    channel_id TEXT NOT NULL,
    guild_id TEXT NOT NULL,
    host_id TEXT NOT NULL,
    prize TEXT NOT NULL,
    winners_count INTEGER NOT NULL,
    end_time BIGINT NOT NULL,
    ended INTEGER DEFAULT 0,
    created_at BIGINT NOT NULL,
    custom_message TEXT,
    
    -- Requirements
    role_requirement TEXT,
    invite_requirement INTEGER,
    account_age_requirement INTEGER,
    server_age_requirement INTEGER,
    captcha_requirement INTEGER DEFAULT 0,
	message_required INTEGER,
    voice_requirement INTEGER,
    entry_fee INTEGER DEFAULT 0,
    assign_role TEXT,
    thumbnail TEXT
);

-- Captcha sessions table
CREATE TABLE IF NOT EXISTS captcha_sessions (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    giveaway_id INTEGER NOT NULL,
    code TEXT NOT NULL,
    created_at BIGINT NOT NULL,
    UNIQUE(user_id, giveaway_id),
    FOREIGN KEY (giveaway_id) REFERENCES giveaways(id) ON DELETE CASCADE
);

-- Participants table
CREATE TABLE IF NOT EXISTS participants (
    id SERIAL PRIMARY KEY,
    giveaway_id INTEGER NOT NULL,
    user_id TEXT NOT NULL,
    joined_at BIGINT NOT NULL,
    FOREIGN KEY (giveaway_id) REFERENCES giveaways(id) ON DELETE CASCADE,
    UNIQUE(giveaway_id, user_id)
);

-- Winners table
CREATE TABLE IF NOT EXISTS winners (
    id SERIAL PRIMARY KEY,
    giveaway_id INTEGER NOT NULL,
    user_id TEXT NOT NULL,
    won_at BIGINT NOT NULL,
    FOREIGN KEY (giveaway_id) REFERENCES giveaways(id) ON DELETE CASCADE
);

-- Refund tracking table
CREATE TABLE IF NOT EXISTS giveaway_refunds (
    giveaway_id INTEGER NOT NULL,
    user_id TEXT NOT NULL,
    refund_count INTEGER DEFAULT 0,
    PRIMARY KEY (giveaway_id, user_id),
    FOREIGN KEY (giveaway_id) REFERENCES giveaways(id) ON DELETE CASCADE
);

-- User stats table (for tracking messages and voice time)
CREATE TABLE IF NOT EXISTS user_stats (
    id SERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    message_count INTEGER DEFAULT 0,
    voice_minutes INTEGER DEFAULT 0,
    UNIQUE(guild_id, user_id)
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_giveaways_guild ON giveaways(guild_id);
CREATE INDEX IF NOT EXISTS idx_giveaways_ended ON giveaways(ended);
CREATE INDEX IF NOT EXISTS idx_participants_giveaway ON participants(giveaway_id);
CREATE INDEX IF NOT EXISTS idx_participants_user ON participants(user_id);
CREATE INDEX IF NOT EXISTS idx_winners_giveaway ON winners(giveaway_id);
CREATE INDEX IF NOT EXISTS idx_user_stats_guild_user ON user_stats(guild_id, user_id);
CREATE INDEX IF NOT EXISTS idx_captcha_sessions_user_giveaway ON captcha_sessions(user_id, giveaway_id);

-- Economy Users table
CREATE TABLE IF NOT EXISTS economy_users (
    user_id TEXT NOT NULL,
    guild_id TEXT NOT NULL,
    balance BIGINT DEFAULT 0,
    last_daily BIGINT DEFAULT 0,
    last_weekly BIGINT DEFAULT 0,
    last_hourly BIGINT DEFAULT 0,
    total_earned BIGINT DEFAULT 0,
    total_spent BIGINT DEFAULT 0,
    PRIMARY KEY (user_id, guild_id)
);

-- Economy Config table
CREATE TABLE IF NOT EXISTS economy_config (
    guild_id TEXT PRIMARY KEY,
    message_reward INTEGER DEFAULT 0,
    vc_reward_per_min INTEGER DEFAULT 0,
    daily_reward INTEGER DEFAULT 0,
    weekly_reward INTEGER DEFAULT 0,
    hourly_reward INTEGER DEFAULT 0,
    invite_reward INTEGER DEFAULT 0,
    react_reward INTEGER DEFAULT 0,
    poll_reward INTEGER DEFAULT 0,
    event_reward INTEGER DEFAULT 0,
    upvote_reward INTEGER DEFAULT 0,
    gamble_enabled INTEGER DEFAULT 0,
    max_gamble_amount INTEGER DEFAULT 20000,
    allowed_channels TEXT DEFAULT '',
    currency_emoji TEXT DEFAULT '<:Cash:1443554334670327848>'
);

-- Guild Settings table
CREATE TABLE IF NOT EXISTS guild_settings (
    guild_id TEXT PRIMARY KEY,
    prefix TEXT DEFAULT '!'
);

-- Auto Drag Rules table
CREATE TABLE IF NOT EXISTS autodrag_rules (
    id SERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    target_channel_id TEXT NOT NULL,
    created_at BIGINT NOT NULL,
    created_by TEXT NOT NULL,
    UNIQUE(guild_id, user_id)
);

-- Auto AFK Settings table
CREATE TABLE IF NOT EXISTS autoafk_settings (
    guild_id TEXT PRIMARY KEY,
    enabled BOOLEAN DEFAULT FALSE,
    minutes INTEGER DEFAULT 10,
    afk_channel_id TEXT DEFAULT ''
);
-- Shop Items table
CREATE TABLE IF NOT EXISTS shop_items (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    price INTEGER NOT NULL,
    stock INTEGER DEFAULT -1, -- -1 means infinite
    type TEXT NOT NULL, -- 'item', 'role', 'boost', etc.
    role_id TEXT, -- For role items
    duration INTEGER, -- Duration in seconds for temporary roles
    required_balance INTEGER DEFAULT 0,
    role_required TEXT,
    reply_message TEXT,
    image_url TEXT,
    hidden BOOLEAN DEFAULT FALSE,
    created_at BIGINT NOT NULL
);

-- User Inventory table
CREATE TABLE IF NOT EXISTS user_inventory (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    guild_id TEXT NOT NULL,
    item_id INTEGER NOT NULL,
    quantity INTEGER DEFAULT 1,
    acquired_at BIGINT NOT NULL,
    expires_at BIGINT DEFAULT 0, -- 0 means permanent
    FOREIGN KEY (item_id) REFERENCES shop_items(id) ON DELETE CASCADE,
    UNIQUE(user_id, guild_id, item_id)
);

-- Create indexes for shop
CREATE INDEX IF NOT EXISTS idx_shop_items_name ON shop_items(name);
CREATE INDEX IF NOT EXISTS idx_user_inventory_user ON user_inventory(user_id, guild_id);

-- Redeem Codes table
CREATE TABLE IF NOT EXISTS redeem_codes (
    code TEXT PRIMARY KEY,
    item_id INTEGER NOT NULL,
    user_id TEXT NOT NULL,
    guild_id TEXT NOT NULL,
    is_claimed BOOLEAN DEFAULT FALSE,
    created_at BIGINT NOT NULL,
    FOREIGN KEY (item_id) REFERENCES shop_items(id) ON DELETE CASCADE
);

-- AntiNuke Config table
CREATE TABLE IF NOT EXISTS antinuke_config (
    guild_id TEXT PRIMARY KEY,
    enabled BOOLEAN DEFAULT FALSE,
    logs_channel TEXT DEFAULT '',
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

-- AntiNuke Actions table
CREATE TABLE IF NOT EXISTS antinuke_actions (
    id SERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL,
    action_type TEXT NOT NULL, -- 'ban_members', 'kick_members', 'delete_roles', etc.
    enabled BOOLEAN DEFAULT TRUE,
    limit_count INTEGER DEFAULT 3,
    window_seconds INTEGER DEFAULT 10,
    punishment TEXT DEFAULT 'ban', -- 'ban', 'kick', 'timeout', 'quarantine'
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    UNIQUE(guild_id, action_type)
);

-- AntiNuke Whitelist table
CREATE TABLE IF NOT EXISTS antinuke_whitelist (
    id SERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL,
    target_id TEXT NOT NULL, -- User ID or Role ID
    target_type TEXT NOT NULL, -- 'user' or 'role'
    added_by TEXT NOT NULL,
    created_at BIGINT NOT NULL,
    UNIQUE(guild_id, target_id)
);

-- AntiNuke Events table (for rate limiting tracking)
CREATE TABLE IF NOT EXISTS antinuke_events (
    id SERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL,
    action_type TEXT NOT NULL,
    executor_id TEXT NOT NULL,
    target_id TEXT, -- Channel ID, Role ID, User ID, etc.
    timestamp BIGINT NOT NULL,
    revoked BOOLEAN DEFAULT FALSE
);

-- Create indexes for antinuke
CREATE INDEX IF NOT EXISTS idx_antinuke_config_guild ON antinuke_config(guild_id);
CREATE INDEX IF NOT EXISTS idx_antinuke_actions_guild ON antinuke_actions(guild_id);
CREATE INDEX IF NOT EXISTS idx_antinuke_actions_guild_action ON antinuke_actions(guild_id, action_type);
CREATE INDEX IF NOT EXISTS idx_antinuke_whitelist_guild ON antinuke_whitelist(guild_id);
CREATE INDEX IF NOT EXISTS idx_antinuke_whitelist_target ON antinuke_whitelist(guild_id, target_id);
CREATE INDEX IF NOT EXISTS idx_antinuke_events_guild_action ON antinuke_events(guild_id, action_type);
CREATE INDEX IF NOT EXISTS idx_antinuke_events_timestamp ON antinuke_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_antinuke_events_guild_executor_time ON antinuke_events(guild_id, executor_id, action_type, timestamp);

`

func NewDatabase(cfg PostgresConfig) (*Database, error) {
	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	// Add TCP_NODELAY for ultra-low latency
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s tcp_user_timeout=1000",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, sslMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Configure connection pool for ultra-low latency (increased from 50 to 100)
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(50)
	db.SetConnMaxIdleTime(5 * time.Minute)
	db.SetConnMaxLifetime(1 * time.Hour)

	// Execute schema
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}

	// Migrations
	_, _ = db.Exec("ALTER TABLE economy_config ADD COLUMN IF NOT EXISTS currency_emoji TEXT DEFAULT '<:Cash:1443554334670327848>'")

	// Prepare the ping statement for ultra-low latency
	pingStmt, err := db.Prepare("SELECT 1")
	if err != nil {
		return nil, fmt.Errorf("failed to prepare ping statement: %w", err)
	}

	d := &Database{
		db:               db,
		PreparedPingStmt: pingStmt,
	}

	// Pre-warm connections by executing the prepared statement (increased to 20)
	for i := 0; i < 20; i++ {
		var result int
		pingStmt.QueryRow().Scan(&result)
	}

	// Initialize prepared statements for fast queries
	if err := d.InitPreparedStatements(); err != nil {
		return nil, fmt.Errorf("failed to init prepared statements: %w", err)
	}

	// Start automatic re-preparation on DB reconnect
	d.StartPreparedStatementRefresher(context.Background())

	return d, nil
}

func (d *Database) Close() error {
	if d.PreparedPingStmt != nil {
		d.PreparedPingStmt.Close()
	}
	d.ClosePreparedStatements()
	return d.db.Close()
}

func (d *Database) Ping() error {
	// Use prepared statement for fastest possible ping
	var err error
	if d.PreparedPingStmt != nil {
		var result int
		err = d.PreparedPingStmt.QueryRow().Scan(&result)
	} else {
		err = d.db.Ping()
	}
	return err
}

// Giveaway operations

func (d *Database) CreateGiveaway(g *models.Giveaway) (int64, error) {
	query := `
		INSERT INTO giveaways (
			message_id, channel_id, guild_id, host_id, prize, winners_count,
			end_time, created_at, custom_message, role_requirement, invite_requirement,
			account_age_requirement, server_age_requirement, captcha_requirement,
			message_required, voice_requirement, entry_fee, assign_role, thumbnail
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING id
	`

	var id int64
	err := d.db.QueryRow(query,
		g.MessageID, g.ChannelID, g.GuildID, g.HostID, g.Prize, g.WinnersCount,
		g.EndTime, models.Now(), g.CustomMessage,
		g.RoleRequirement, g.InviteRequirement, g.AccountAgeRequirement, g.ServerAgeRequirement,
		models.BoolToInt(g.CaptchaRequirement), g.MessageRequired, g.VoiceRequirement, g.EntryFee,
		g.AssignRole, g.Thumbnail,
	).Scan(&id)

	if err != nil {
		return 0, err
	}
	return id, nil
}

func (d *Database) GetGiveaway(messageID string) (*models.Giveaway, error) {
	query := `
		SELECT 
			id, message_id, channel_id, guild_id, host_id, prize, winners_count,
			end_time, ended, created_at, custom_message,
			role_requirement, invite_requirement, account_age_requirement, server_age_requirement, 
			captcha_requirement, message_required, voice_requirement, entry_fee, assign_role, thumbnail
		FROM giveaways WHERE message_id = $1
	`
	return d.scanGiveaway(d.db.QueryRow(query, messageID))
}

func (d *Database) GetGiveawayByID(id int64) (*models.Giveaway, error) {
	query := `
		SELECT 
			id, message_id, channel_id, guild_id, host_id, prize, winners_count,
			end_time, ended, created_at, custom_message,
			role_requirement, invite_requirement, account_age_requirement, server_age_requirement, 
			captcha_requirement, message_required, voice_requirement, entry_fee, assign_role, thumbnail
		FROM giveaways WHERE id = $1
	`
	return d.scanGiveaway(d.db.QueryRow(query, id))
}

func (d *Database) GetActiveGiveaways(guildID string) ([]*models.Giveaway, error) {
	query := `
		SELECT 
			id, message_id, channel_id, guild_id, host_id, prize, winners_count,
			end_time, ended, created_at, custom_message,
			role_requirement, invite_requirement, account_age_requirement, server_age_requirement, 
			captcha_requirement, message_required, voice_requirement, entry_fee, assign_role, thumbnail
		FROM giveaways WHERE guild_id = $1 AND ended = 0 ORDER BY end_time ASC
	`
	rows, err := d.db.Query(query, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var giveaways []*models.Giveaway
	for rows.Next() {
		g, err := d.scanGiveawayRows(rows)
		if err != nil {
			return nil, err
		}
		giveaways = append(giveaways, g)
	}
	return giveaways, nil
}

func (d *Database) GetAllActiveGiveaways() ([]*models.Giveaway, error) {
	query := `
		SELECT 
			id, message_id, channel_id, guild_id, host_id, prize, winners_count,
			end_time, ended, created_at, custom_message,
			role_requirement, invite_requirement, account_age_requirement, server_age_requirement, 
			captcha_requirement, message_required, voice_requirement, entry_fee, assign_role, thumbnail
		FROM giveaways WHERE ended = 0
	`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var giveaways []*models.Giveaway
	for rows.Next() {
		g, err := d.scanGiveawayRows(rows)
		if err != nil {
			return nil, err
		}
		giveaways = append(giveaways, g)
	}
	return giveaways, nil
}

func (d *Database) UpdateGiveawayMessageID(tempID, newID string) error {
	_, err := d.db.Exec("UPDATE giveaways SET message_id = $1 WHERE message_id = $2", newID, tempID)
	return err
}

func (d *Database) EndGiveaway(messageID string) error {
	_, err := d.db.Exec("UPDATE giveaways SET ended = 1 WHERE message_id = $1", messageID)
	return err
}

// Participant operations

func (d *Database) AddParticipant(giveawayID int64, userID string) error {
	_, err := d.db.Exec("INSERT INTO participants (giveaway_id, user_id, joined_at) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING",
		giveawayID, userID, models.Now())
	return err
}

func (d *Database) RemoveParticipant(giveawayID int64, userID string) error {
	_, err := d.db.Exec("DELETE FROM participants WHERE giveaway_id = $1 AND user_id = $2", giveawayID, userID)
	return err
}

func (d *Database) GetParticipantCount(giveawayID int64) (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM participants WHERE giveaway_id = $1", giveawayID).Scan(&count)
	return count, err
}

func (d *Database) GetParticipants(giveawayID int64) ([]string, error) {
	rows, err := d.db.Query("SELECT user_id FROM participants WHERE giveaway_id = $1", giveawayID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		users = append(users, userID)
	}
	return users, nil
}

func (d *Database) IsParticipant(giveawayID int64, userID string) (bool, error) {
	var exists int
	err := d.db.QueryRow("SELECT 1 FROM participants WHERE giveaway_id = $1 AND user_id = $2", giveawayID, userID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return exists == 1, err
}

// Winner operations

func (d *Database) AddWinner(giveawayID int64, userID string) error {
	_, err := d.db.Exec("INSERT INTO winners (giveaway_id, user_id, won_at) VALUES ($1, $2, $3)",
		giveawayID, userID, models.Now())
	return err
}

func (d *Database) GetWinners(giveawayID int64) ([]string, error) {
	rows, err := d.db.Query("SELECT user_id FROM winners WHERE giveaway_id = $1", giveawayID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		users = append(users, userID)
	}
	return users, nil
}

// Captcha operations

func (d *Database) CreateCaptchaSession(userID string, giveawayID int64, code string) error {
	query := `
		INSERT INTO captcha_sessions (user_id, giveaway_id, code, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT(user_id, giveaway_id)
		DO UPDATE SET code = $5, created_at = $6
	`
	now := models.Now()
	_, err := d.db.Exec(query, userID, giveawayID, code, now, code, now)
	return err
}

func (d *Database) VerifyCaptcha(userID string, giveawayID int64, inputCode string) (bool, error) {
	var code string
	err := d.db.QueryRow("SELECT code FROM captcha_sessions WHERE user_id = $1 AND giveaway_id = $2", userID, giveawayID).Scan(&code)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	isValid := strings.EqualFold(code, inputCode)
	if isValid {
		_, _ = d.db.Exec("DELETE FROM captcha_sessions WHERE user_id = $1 AND giveaway_id = $2", userID, giveawayID)
	}
	return isValid, nil
}

// User Stats operations

func (d *Database) GetUserStats(guildID, userID string) (*models.UserStats, error) {
	stats := &models.UserStats{GuildID: guildID, UserID: userID}
	err := d.db.QueryRow("SELECT message_count, voice_minutes FROM user_stats WHERE guild_id = $1 AND user_id = $2", guildID, userID).Scan(&stats.MessageCount, &stats.VoiceMinutes)
	if err == sql.ErrNoRows {
		return stats, nil
	}
	return stats, err
}

func (d *Database) IncrementMessageCount(guildID, userID string) error {
	query := `
		INSERT INTO user_stats (guild_id, user_id, message_count, voice_minutes)
		VALUES ($1, $2, 1, 0)
		ON CONFLICT(guild_id, user_id)
		DO UPDATE SET message_count = user_stats.message_count + 1
	`
	_, err := d.db.Exec(query, guildID, userID)
	return err
}

func (d *Database) AddVoiceMinutes(guildID, userID string, minutes int) error {
	query := `
		INSERT INTO user_stats (guild_id, user_id, message_count, voice_minutes)
		VALUES ($1, $2, 0, $3)
		ON CONFLICT(guild_id, user_id)
		DO UPDATE SET voice_minutes = user_stats.voice_minutes + $4
	`
	_, err := d.db.Exec(query, guildID, userID, minutes, minutes)
	return err
}

func (d *Database) AddMessageCount(guildID, userID string, amount int) error {
	query := `
		INSERT INTO user_stats (guild_id, user_id, message_count, voice_minutes)
		VALUES ($1, $2, $3, 0)
		ON CONFLICT(guild_id, user_id)
		DO UPDATE SET message_count = user_stats.message_count + $3
	`
	_, err := d.db.Exec(query, guildID, userID, amount)
	return err
}

// Helpers

func (d *Database) scanGiveaway(row *sql.Row) (*models.Giveaway, error) {
	var g models.Giveaway
	var captchaReq int
	var customMessage sql.NullString
	var roleReq sql.NullString
	var inviteReq sql.NullInt64
	var accountAgeReq sql.NullInt64
	var serverAgeReq sql.NullInt64
	var messageReq sql.NullInt64
	var voiceReq sql.NullInt64
	var entryFee sql.NullInt64
	var assignRole sql.NullString
	var thumbnail sql.NullString

	err := row.Scan(
		&g.ID, &g.MessageID, &g.ChannelID, &g.GuildID, &g.HostID, &g.Prize, &g.WinnersCount,
		&g.EndTime, &g.Ended, &g.CreatedAt, &customMessage,
		&roleReq, &inviteReq, &accountAgeReq, &serverAgeReq, &captchaReq, &messageReq, &voiceReq, &entryFee,
		&assignRole, &thumbnail,
	)
	if err != nil {
		return nil, err
	}

	g.CustomMessage = customMessage.String
	g.RoleRequirement = roleReq.String
	g.InviteRequirement = int(inviteReq.Int64)
	g.AccountAgeRequirement = int(accountAgeReq.Int64)
	g.ServerAgeRequirement = int(serverAgeReq.Int64)
	g.CaptchaRequirement = models.IntToBool(captchaReq)
	g.MessageRequired = int(messageReq.Int64)
	g.VoiceRequirement = int(voiceReq.Int64)
	g.EntryFee = int(entryFee.Int64)
	g.AssignRole = assignRole.String
	g.Thumbnail = thumbnail.String

	return &g, nil
}

func (d *Database) scanGiveawayRows(rows *sql.Rows) (*models.Giveaway, error) {
	var g models.Giveaway
	var captchaReq int
	var customMessage sql.NullString
	var roleReq sql.NullString
	var inviteReq sql.NullInt64
	var accountAgeReq sql.NullInt64
	var serverAgeReq sql.NullInt64
	var messageReq sql.NullInt64
	var voiceReq sql.NullInt64
	var entryFee sql.NullInt64
	var assignRole sql.NullString
	var thumbnail sql.NullString

	err := rows.Scan(
		&g.ID, &g.MessageID, &g.ChannelID, &g.GuildID, &g.HostID, &g.Prize, &g.WinnersCount,
		&g.EndTime, &g.Ended, &g.CreatedAt, &customMessage,
		&roleReq, &inviteReq, &accountAgeReq, &serverAgeReq, &captchaReq, &messageReq, &voiceReq, &entryFee,
		&assignRole, &thumbnail,
	)
	if err != nil {
		return nil, err
	}

	g.CustomMessage = customMessage.String
	g.RoleRequirement = roleReq.String
	g.InviteRequirement = int(inviteReq.Int64)
	g.AccountAgeRequirement = int(accountAgeReq.Int64)
	g.ServerAgeRequirement = int(serverAgeReq.Int64)
	g.CaptchaRequirement = models.IntToBool(captchaReq)
	g.MessageRequired = int(messageReq.Int64)
	g.VoiceRequirement = int(voiceReq.Int64)
	g.EntryFee = int(entryFee.Int64)
	g.AssignRole = assignRole.String
	g.Thumbnail = thumbnail.String

	return &g, nil
}

// Refund operations

func (d *Database) GetRefundCount(giveawayID int64, userID string) (int, error) {
	var count int
	err := d.db.QueryRow("SELECT refund_count FROM giveaway_refunds WHERE giveaway_id = $1 AND user_id = $2", giveawayID, userID).Scan(&count)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return count, err
}

func (d *Database) IncrementRefundCount(giveawayID int64, userID string) error {
	query := `
		INSERT INTO giveaway_refunds (giveaway_id, user_id, refund_count)
		VALUES ($1, $2, 1)
		ON CONFLICT(giveaway_id, user_id)
		DO UPDATE SET refund_count = giveaway_refunds.refund_count + 1
	`
	_, err := d.db.Exec(query, giveawayID, userID)
	return err
}
