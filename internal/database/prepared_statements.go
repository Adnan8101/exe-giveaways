package database

import (
	"context"
	"database/sql"
	"discord-giveaway-bot/internal/models"
	"fmt"
	"strings"
	"sync"
	"time"
)

// PreparedStatements holds all commonly used prepared statements for ultra-low latency
type PreparedStatements struct {
	mu sync.RWMutex
	db *sql.DB

	// Giveaway queries
	getGiveaway         *sql.Stmt
	getGiveawayByID     *sql.Stmt
	addParticipant      *sql.Stmt
	removeParticipant   *sql.Stmt
	getParticipantCount *sql.Stmt
	isParticipant       *sql.Stmt

	// User stats queries
	getUserStats          *sql.Stmt
	incrementMessageCount *sql.Stmt
	addVoiceMinutes       *sql.Stmt

	// Economy queries
	getEconomyUser   *sql.Stmt
	updateBalance    *sql.Stmt
	getEconomyConfig *sql.Stmt

	// Guild settings
	getGuildPrefix *sql.Stmt
}

// InitPreparedStatements pre-compiles all frequently used SQL statements
func (d *Database) InitPreparedStatements() error {
	d.PreparedStmts = &PreparedStatements{db: d.db}

	var err error

	// Giveaway queries
	d.PreparedStmts.getGiveaway, err = d.db.Prepare(`
		SELECT id, message_id, channel_id, guild_id, host_id, prize, winners_count,
			end_time, ended, created_at, custom_message,
			role_requirement, invite_requirement, account_age_requirement, 
			server_age_requirement, captcha_requirement, message_required, 
			voice_requirement, entry_fee, assign_role, thumbnail
		FROM giveaways WHERE message_id = $1
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare getGiveaway: %w", err)
	}

	d.PreparedStmts.getGiveawayByID, err = d.db.Prepare(`
		SELECT id, message_id, channel_id, guild_id, host_id, prize, winners_count,
			end_time, ended, created_at, custom_message,
			role_requirement, invite_requirement, account_age_requirement, 
			server_age_requirement, captcha_requirement, message_required, 
			voice_requirement, entry_fee, assign_role, thumbnail
		FROM giveaways WHERE id = $1
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare getGiveawayByID: %w", err)
	}

	d.PreparedStmts.addParticipant, err = d.db.Prepare(`
		INSERT INTO participants (giveaway_id, user_id, joined_at) 
		VALUES ($1, $2, $3) 
		ON CONFLICT DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare addParticipant: %w", err)
	}

	d.PreparedStmts.removeParticipant, err = d.db.Prepare(`
		DELETE FROM participants WHERE giveaway_id = $1 AND user_id = $2
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare removeParticipant: %w", err)
	}

	d.PreparedStmts.getParticipantCount, err = d.db.Prepare(`
		SELECT COUNT(*) FROM participants WHERE giveaway_id = $1
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare getParticipantCount: %w", err)
	}

	d.PreparedStmts.isParticipant, err = d.db.Prepare(`
		SELECT 1 FROM participants WHERE giveaway_id = $1 AND user_id = $2
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare isParticipant: %w", err)
	}

	// User stats queries
	d.PreparedStmts.getUserStats, err = d.db.Prepare(`
		SELECT message_count, voice_minutes 
		FROM user_stats 
		WHERE guild_id = $1 AND user_id = $2
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare getUserStats: %w", err)
	}

	d.PreparedStmts.incrementMessageCount, err = d.db.Prepare(`
		INSERT INTO user_stats (guild_id, user_id, message_count, voice_minutes)
		VALUES ($1, $2, 1, 0)
		ON CONFLICT(guild_id, user_id)
		DO UPDATE SET message_count = user_stats.message_count + 1
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare incrementMessageCount: %w", err)
	}

	d.PreparedStmts.addVoiceMinutes, err = d.db.Prepare(`
		INSERT INTO user_stats (guild_id, user_id, message_count, voice_minutes)
		VALUES ($1, $2, 0, $3)
		ON CONFLICT(guild_id, user_id)
		DO UPDATE SET voice_minutes = user_stats.voice_minutes + $4
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare addVoiceMinutes: %w", err)
	}

	// Economy queries
	d.PreparedStmts.getEconomyUser, err = d.db.Prepare(`
		SELECT user_id, guild_id, balance, last_daily, last_weekly, last_hourly, total_earned, total_spent
		FROM economy_users
		WHERE user_id = $1 AND guild_id = $2
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare getEconomyUser: %w", err)
	}

	d.PreparedStmts.updateBalance, err = d.db.Prepare(`
		INSERT INTO economy_users (user_id, guild_id, balance, last_daily, last_weekly, last_hourly, total_earned, total_spent)
		VALUES ($1, $2, $3, 0, 0, 0, 0, 0)
		ON CONFLICT(user_id, guild_id)
		DO UPDATE SET balance = $3
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare updateBalance: %w", err)
	}

	d.PreparedStmts.getEconomyConfig, err = d.db.Prepare(`
		SELECT guild_id, message_reward, vc_reward_per_min, daily_reward, weekly_reward, hourly_reward,
			invite_reward, react_reward, poll_reward, event_reward, upvote_reward,
			gamble_enabled, max_gamble_amount, allowed_channels, currency_emoji
		FROM economy_config
		WHERE guild_id = $1
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare getEconomyConfig: %w", err)
	}

	// Guild settings
	d.PreparedStmts.getGuildPrefix, err = d.db.Prepare(`
		SELECT prefix FROM guild_settings WHERE guild_id = $1
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare getGuildPrefix: %w", err)
	}

	return nil
}

// StartPreparedStatementRefresher automatically re-prepares statements on DB reconnect
func (d *Database) StartPreparedStatementRefresher(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := d.db.Ping(); err != nil {
					// DB probably restarted → reprepare
					d.ClosePreparedStatements()
					_ = d.InitPreparedStatements()
				}
			}
		}
	}()
}

// ClosePreparedStatements closes all prepared statements
func (d *Database) ClosePreparedStatements() {
	if d.PreparedStmts == nil {
		return
	}

	d.PreparedStmts.mu.Lock()
	defer d.PreparedStmts.mu.Unlock()

	stmts := []*sql.Stmt{
		d.PreparedStmts.getGiveaway,
		d.PreparedStmts.getGiveawayByID,
		d.PreparedStmts.addParticipant,
		d.PreparedStmts.removeParticipant,
		d.PreparedStmts.getParticipantCount,
		d.PreparedStmts.isParticipant,
		d.PreparedStmts.getUserStats,
		d.PreparedStmts.incrementMessageCount,
		d.PreparedStmts.addVoiceMinutes,
		d.PreparedStmts.getEconomyUser,
		d.PreparedStmts.updateBalance,
		d.PreparedStmts.getEconomyConfig,
		d.PreparedStmts.getGuildPrefix,
	}

	for _, stmt := range stmts {
		if stmt != nil {
			stmt.Close()
		}
	}
}

// isBadPreparedStatement checks if error indicates invalid prepared statement
func isBadPreparedStatement(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "cached plan") ||
		strings.Contains(errStr, "closed the connection") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "bad connection")
}

// Fast prepared statement versions of common queries with production-grade improvements

func (d *Database) GetGiveawayFast(ctx context.Context, messageID string) (*models.Giveaway, error) {
	ps := d.PreparedStmts
	if ps == nil {
		return d.GetGiveaway(messageID)
	}

	ps.mu.RLock()
	stmt := ps.getGiveaway
	ps.mu.RUnlock()

	if stmt == nil {
		return d.GetGiveaway(messageID)
	}

	row := stmt.QueryRowContext(ctx, messageID)
	g, err := d.scanGiveaway(row)

	if isBadPreparedStatement(err) {
		// Auto recover
		_ = d.InitPreparedStatements()
		return d.GetGiveawayFast(ctx, messageID)
	}

	return g, err
}

func (d *Database) GetGiveawayByIDFast(ctx context.Context, id int64) (*models.Giveaway, error) {
	ps := d.PreparedStmts
	if ps == nil {
		return d.GetGiveawayByID(id)
	}

	ps.mu.RLock()
	stmt := ps.getGiveawayByID
	ps.mu.RUnlock()

	if stmt == nil {
		return d.GetGiveawayByID(id)
	}

	row := stmt.QueryRowContext(ctx, id)
	g, err := d.scanGiveaway(row)

	if isBadPreparedStatement(err) {
		_ = d.InitPreparedStatements()
		return d.GetGiveawayByIDFast(ctx, id)
	}

	return g, err
}

func (d *Database) AddParticipantFast(ctx context.Context, giveawayID int64, userID string) error {
	ps := d.PreparedStmts
	if ps == nil {
		return d.AddParticipant(giveawayID, userID)
	}

	ps.mu.RLock()
	stmt := ps.addParticipant
	ps.mu.RUnlock()

	if stmt == nil {
		return d.AddParticipant(giveawayID, userID)
	}

	_, err := stmt.ExecContext(ctx, giveawayID, userID, models.Now())

	if isBadPreparedStatement(err) {
		_ = d.InitPreparedStatements()
		return d.AddParticipantFast(ctx, giveawayID, userID)
	}

	return err
}

func (d *Database) RemoveParticipantFast(ctx context.Context, giveawayID int64, userID string) error {
	ps := d.PreparedStmts
	if ps == nil {
		return d.RemoveParticipant(giveawayID, userID)
	}

	ps.mu.RLock()
	stmt := ps.removeParticipant
	ps.mu.RUnlock()

	if stmt == nil {
		return d.RemoveParticipant(giveawayID, userID)
	}

	_, err := stmt.ExecContext(ctx, giveawayID, userID)

	if isBadPreparedStatement(err) {
		_ = d.InitPreparedStatements()
		return d.RemoveParticipantFast(ctx, giveawayID, userID)
	}

	return err
}

func (d *Database) GetParticipantCountFast(ctx context.Context, giveawayID int64) (int, error) {
	ps := d.PreparedStmts
	if ps == nil {
		return d.GetParticipantCount(giveawayID)
	}

	ps.mu.RLock()
	stmt := ps.getParticipantCount
	ps.mu.RUnlock()

	if stmt == nil {
		return d.GetParticipantCount(giveawayID)
	}

	var count int
	err := stmt.QueryRowContext(ctx, giveawayID).Scan(&count)

	if isBadPreparedStatement(err) {
		_ = d.InitPreparedStatements()
		return d.GetParticipantCountFast(ctx, giveawayID)
	}

	return count, err
}

func (d *Database) IsParticipantFast(ctx context.Context, giveawayID int64, userID string) (bool, error) {
	ps := d.PreparedStmts
	if ps == nil {
		return d.IsParticipant(giveawayID, userID)
	}

	ps.mu.RLock()
	stmt := ps.isParticipant
	ps.mu.RUnlock()

	if stmt == nil {
		return d.IsParticipant(giveawayID, userID)
	}

	var exists int
	err := stmt.QueryRowContext(ctx, giveawayID, userID).Scan(&exists)

	if err == sql.ErrNoRows {
		return false, nil
	}

	if isBadPreparedStatement(err) {
		_ = d.InitPreparedStatements()
		return d.IsParticipantFast(ctx, giveawayID, userID)
	}

	return exists == 1, err
}

func (d *Database) GetUserStatsFast(ctx context.Context, guildID, userID string) (*models.UserStats, error) {
	ps := d.PreparedStmts
	if ps == nil {
		return d.GetUserStats(guildID, userID)
	}

	ps.mu.RLock()
	stmt := ps.getUserStats
	ps.mu.RUnlock()

	if stmt == nil {
		return d.GetUserStats(guildID, userID)
	}

	stats := &models.UserStats{GuildID: guildID, UserID: userID}
	err := stmt.QueryRowContext(ctx, guildID, userID).Scan(&stats.MessageCount, &stats.VoiceMinutes)

	if err == sql.ErrNoRows {
		return stats, nil
	}

	if isBadPreparedStatement(err) {
		_ = d.InitPreparedStatements()
		return d.GetUserStatsFast(ctx, guildID, userID)
	}

	return stats, err
}

func (d *Database) IncrementMessageCountFast(ctx context.Context, guildID, userID string) error {
	ps := d.PreparedStmts
	if ps == nil {
		return d.IncrementMessageCount(guildID, userID)
	}

	ps.mu.RLock()
	stmt := ps.incrementMessageCount
	ps.mu.RUnlock()

	if stmt == nil {
		return d.IncrementMessageCount(guildID, userID)
	}

	_, err := stmt.ExecContext(ctx, guildID, userID)

	if isBadPreparedStatement(err) {
		_ = d.InitPreparedStatements()
		return d.IncrementMessageCountFast(ctx, guildID, userID)
	}

	return err
}

func (d *Database) AddVoiceMinutesFast(ctx context.Context, guildID, userID string, minutes int) error {
	ps := d.PreparedStmts
	if ps == nil {
		return d.AddVoiceMinutes(guildID, userID, minutes)
	}

	ps.mu.RLock()
	stmt := ps.addVoiceMinutes
	ps.mu.RUnlock()

	if stmt == nil {
		return d.AddVoiceMinutes(guildID, userID, minutes)
	}

	_, err := stmt.ExecContext(ctx, guildID, userID, minutes, minutes)

	if isBadPreparedStatement(err) {
		_ = d.InitPreparedStatements()
		return d.AddVoiceMinutesFast(ctx, guildID, userID, minutes)
	}

	return err
}
