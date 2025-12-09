package database

import (
	"database/sql"
	"discord-giveaway-bot/internal/models"
	"log"
	"time"

	"github.com/lib/pq"
)

// AntiNuke Configuration Operations

// GetAntiNukeConfig retrieves the antinuke configuration for a guild
func (d *Database) GetAntiNukeConfig(guildID string) (*models.AntiNukeConfig, error) {
	config := &models.AntiNukeConfig{GuildID: guildID}
	err := d.db.QueryRow(`
		SELECT enabled, logs_channel, panic_mode, created_at, updated_at 
		FROM antinuke_config 
		WHERE guild_id = $1
	`, guildID).Scan(&config.Enabled, &config.LogsChannel, &config.PanicMode, &config.CreatedAt, &config.UpdatedAt)

	if err == sql.ErrNoRows {
		log.Printf("⚠️  [DB] No antinuke_config record for guild %s (returning disabled default)", guildID)
		return config, nil // Return default config with Enabled=false
	}
	if err != nil {
		log.Printf("❌ [DB] Error querying antinuke_config for guild %s: %v", guildID, err)
		return config, err
	}

	log.Printf("✅ [DB] Found antinuke_config for guild %s: Enabled=%v, PanicMode=%v", guildID, config.Enabled, config.PanicMode)
	return config, nil
}

// EnableAntiNuke enables antinuke for a guild
func (d *Database) EnableAntiNuke(guildID string) error {
	now := time.Now().Unix()

	log.Printf("💾 [DB] Enabling AntiNuke for guild %s...", guildID)

	result, err := d.db.Exec(`
		INSERT INTO antinuke_config (guild_id, enabled, panic_mode, created_at, updated_at)
		VALUES ($1, true, false, $2, $3)
		ON CONFLICT (guild_id) DO UPDATE 
		SET enabled = true, updated_at = $4
	`, guildID, now, now, now)

	if err != nil {
		log.Printf("❌ [DB] Failed to enable AntiNuke for guild %s: %v", guildID, err)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("✅ [DB] Successfully enabled AntiNuke for guild %s (rows affected: %d)", guildID, rowsAffected)

	return nil
}

// DisableAntiNuke disables antinuke for a guild
func (d *Database) DisableAntiNuke(guildID string) error {
	now := time.Now().Unix()
	_, err := d.db.Exec(`
		UPDATE antinuke_config 
		SET enabled = false, updated_at = $1 
		WHERE guild_id = $2
	`, now, guildID)
	return err
}

// SetAntiNukeLogsChannel sets the logs channel for antinuke
func (d *Database) SetAntiNukeLogsChannel(guildID, channelID string) error {
	now := time.Now().Unix()
	_, err := d.db.Exec(`
		INSERT INTO antinuke_config (guild_id, enabled, logs_channel, created_at, updated_at)
		VALUES ($1, true, $2, $3, $4)
		ON CONFLICT (guild_id) DO UPDATE 
		SET logs_channel = $5, updated_at = $6
	`, guildID, channelID, now, now, channelID, now)
	return err
}

// SetPanicMode updates the panic mode status for a guild
func (d *Database) SetPanicMode(guildID string, enabled bool) error {
	now := time.Now().Unix()
	_, err := d.db.Exec(`
		UPDATE antinuke_config 
		SET panic_mode = $1, updated_at = $2 
		WHERE guild_id = $3
	`, enabled, now, guildID)
	return err
}

// AntiNuke Action Operations

// GetActionConfig retrieves configuration for a specific action
func (d *Database) GetActionConfig(guildID, actionType string) (*models.ActionConfig, error) {
	config := &models.ActionConfig{
		GuildID:       guildID,
		ActionType:    actionType,
		Enabled:       true,
		LimitCount:    3,
		WindowSeconds: 10,
		Punishment:    "ban",
	}

	err := d.db.QueryRow(`
		SELECT id, enabled, limit_count, window_seconds, punishment, created_at, updated_at
		FROM antinuke_actions
		WHERE guild_id = $1 AND action_type = $2
	`, guildID, actionType).Scan(
		&config.ID, &config.Enabled, &config.LimitCount,
		&config.WindowSeconds, &config.Punishment,
		&config.CreatedAt, &config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return config, nil // Return defaults
	}
	return config, err
}

// GetAllActionConfigs retrieves all action configurations for a guild
func (d *Database) GetAllActionConfigs(guildID string) ([]*models.ActionConfig, error) {
	rows, err := d.db.Query(`
		SELECT id, action_type, enabled, limit_count, window_seconds, punishment, created_at, updated_at
		FROM antinuke_actions
		WHERE guild_id = $1 AND enabled = true
		ORDER BY action_type
	`, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*models.ActionConfig
	for rows.Next() {
		config := &models.ActionConfig{GuildID: guildID}
		err := rows.Scan(
			&config.ID, &config.ActionType, &config.Enabled,
			&config.LimitCount, &config.WindowSeconds, &config.Punishment,
			&config.CreatedAt, &config.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}
	return configs, nil
}

// SetActionConfig creates or updates action configuration
func (d *Database) SetActionConfig(guildID, actionType string, limitCount, windowSeconds int, punishment string) error {
	now := time.Now().Unix()

	// If action type is "all", apply to all action types
	if actionType == models.ActionAll {
		for _, at := range models.GetAllActionTypes() {
			err := d.setActionConfigSingle(guildID, at, limitCount, windowSeconds, punishment, now)
			if err != nil {
				return err
			}
		}
		return nil
	}

	return d.setActionConfigSingle(guildID, actionType, limitCount, windowSeconds, punishment, now)
}

func (d *Database) setActionConfigSingle(guildID, actionType string, limitCount, windowSeconds int, punishment string, now int64) error {
	_, err := d.db.Exec(`
		INSERT INTO antinuke_actions 
		(guild_id, action_type, enabled, limit_count, window_seconds, punishment, created_at, updated_at)
		VALUES ($1, $2, true, $3, $4, $5, $6, $7)
		ON CONFLICT (guild_id, action_type) DO UPDATE 
		SET limit_count = $8, window_seconds = $9, punishment = $10, updated_at = $11, enabled = true
	`, guildID, actionType, limitCount, windowSeconds, punishment, now, now,
		limitCount, windowSeconds, punishment, now)
	return err
}

// UpdateActionLimit updates only the limit for an action
func (d *Database) UpdateActionLimit(guildID, actionType string, limitCount int) error {
	now := time.Now().Unix()
	_, err := d.db.Exec(`
		UPDATE antinuke_actions 
		SET limit_count = $1, updated_at = $2 
		WHERE guild_id = $3 AND action_type = $4
	`, limitCount, now, guildID, actionType)
	return err
}

// UpdateActionPunishment updates only the punishment for an action
func (d *Database) UpdateActionPunishment(guildID, actionType, punishment string) error {
	now := time.Now().Unix()
	_, err := d.db.Exec(`
		UPDATE antinuke_actions 
		SET punishment = $1, updated_at = $2 
		WHERE guild_id = $3 AND action_type = $4
	`, punishment, now, guildID, actionType)
	return err
}

// DisableAction disables a specific action
func (d *Database) DisableAction(guildID, actionType string) error {
	now := time.Now().Unix()
	_, err := d.db.Exec(`
		UPDATE antinuke_actions 
		SET enabled = false, updated_at = $1 
		WHERE guild_id = $2 AND action_type = $3
	`, now, guildID, actionType)
	return err
}

// AntiNuke Whitelist Operations

// IsWhitelisted checks if a user or role is whitelisted
func (d *Database) IsWhitelisted(guildID, targetID string) (bool, error) {
	var exists int
	err := d.db.QueryRow(`
		SELECT 1 FROM antinuke_whitelist 
		WHERE guild_id = $1 AND target_id = $2
	`, guildID, targetID).Scan(&exists)

	if err == sql.ErrNoRows {
		return false, nil
	}
	return exists == 1, err
}

// AddWhitelistEntry adds a user or role to the whitelist
func (d *Database) AddWhitelistEntry(guildID, targetID, targetType, addedBy string) error {
	now := time.Now().Unix()
	_, err := d.db.Exec(`
		INSERT INTO antinuke_whitelist (guild_id, target_id, target_type, added_by, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (guild_id, target_id) DO NOTHING
	`, guildID, targetID, targetType, addedBy, now)
	return err
}

// RemoveWhitelistEntry removes a user or role from the whitelist
func (d *Database) RemoveWhitelistEntry(guildID, targetID string) error {
	_, err := d.db.Exec(`
		DELETE FROM antinuke_whitelist 
		WHERE guild_id = $1 AND target_id = $2
	`, guildID, targetID)
	return err
}

// GetWhitelistEntries retrieves all whitelist entries for a guild
func (d *Database) GetWhitelistEntries(guildID string) ([]*models.WhitelistEntry, error) {
	rows, err := d.db.Query(`
		SELECT id, target_id, target_type, added_by, created_at
		FROM antinuke_whitelist
		WHERE guild_id = $1
		ORDER BY created_at DESC
	`, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*models.WhitelistEntry
	for rows.Next() {
		entry := &models.WhitelistEntry{GuildID: guildID}
		err := rows.Scan(&entry.ID, &entry.TargetID, &entry.TargetType, &entry.AddedBy, &entry.CreatedAt)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// AntiNuke Event Tracking Operations

// TrackActionEvent records an action event for rate limiting
func (d *Database) TrackActionEvent(guildID, actionType, executorID, targetID string) error {
	now := time.Now().Unix()
	_, err := d.db.Exec(`
		INSERT INTO antinuke_events 
		(guild_id, action_type, executor_id, target_id, timestamp, revoked)
		VALUES ($1, $2, $3, $4, $5, false)
	`, guildID, actionType, executorID, targetID, now)
	return err
}

// GetRecentEvents retrieves events within a time window
func (d *Database) GetRecentEvents(guildID, actionType, executorID string, windowSeconds int) ([]*models.ActionEvent, error) {
	cutoff := time.Now().Unix() - int64(windowSeconds)

	rows, err := d.db.Query(`
		SELECT id, action_type, executor_id, target_id, timestamp, revoked
		FROM antinuke_events
		WHERE guild_id = $1 
		  AND action_type = $2 
		  AND executor_id = $3
		  AND timestamp >= $4
		  AND revoked = false
		ORDER BY timestamp DESC
	`, guildID, actionType, executorID, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*models.ActionEvent
	for rows.Next() {
		event := &models.ActionEvent{GuildID: guildID}
		err := rows.Scan(&event.ID, &event.ActionType, &event.ExecutorID, &event.TargetID, &event.Timestamp, &event.Revoked)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

// CountRecentEvents counts events within a time window
func (d *Database) CountRecentEvents(guildID, actionType, executorID string, windowSeconds int) (int, error) {
	cutoff := time.Now().Unix() - int64(windowSeconds)

	var count int
	err := d.db.QueryRow(`
		SELECT COUNT(*) 
		FROM antinuke_events
		WHERE guild_id = $1 
		  AND action_type = $2 
		  AND executor_id = $3
		  AND timestamp >= $4
		  AND revoked = false
	`, guildID, actionType, executorID, cutoff).Scan(&count)

	return count, err
}

// MarkEventsAsRevoked marks events as revoked (actions were undone)
func (d *Database) MarkEventsAsRevoked(eventIDs []int64) error {
	if len(eventIDs) == 0 {
		return nil
	}

	// Use PostgreSQL array notation with pq.Array
	_, err := d.db.Exec("UPDATE antinuke_events SET revoked = true WHERE id = ANY($1)", pq.Array(eventIDs))
	return err
}

// CleanupOldEvents removes events older than 1 hour to prevent database bloat
func (d *Database) CleanupOldEvents() error {
	cutoff := time.Now().Unix() - 3600 // 1 hour ago
	_, err := d.db.Exec(`
		DELETE FROM antinuke_events 
		WHERE timestamp < $1
	`, cutoff)
	return err
}
