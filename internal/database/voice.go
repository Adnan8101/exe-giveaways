package database

import "time"

// AutoDragRule represents an auto-drag rule
type AutoDragRule struct {
	ID              int64
	GuildID         string
	UserID          string
	TargetChannelID string
	CreatedAt       int64
	CreatedBy       string
}

// AutoAFKSettings represents auto-AFK settings for a guild
type AutoAFKSettings struct {
	GuildID      string
	Enabled      bool
	Minutes      int
	AFKChannelID string
}

// CreateAutoDragRule creates a new autodrag rule
func (d *Database) CreateAutoDragRule(guildID, userID, targetChannelID, createdBy string) error {
	query := `
		INSERT INTO autodrag_rules (guild_id, user_id, target_channel_id, created_at, created_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT(guild_id, user_id)
		DO UPDATE SET target_channel_id = EXCLUDED.target_channel_id, created_at = EXCLUDED.created_at, created_by = EXCLUDED.created_by
	`
	now := time.Now().Unix()
	_, err := d.db.Exec(query, guildID, userID, targetChannelID, now, createdBy)
	return err
}

// GetAutoDragRule gets the target channel for a user's autodrag rule
func (d *Database) GetAutoDragRule(guildID, userID string) (string, error) {
	var channelID string
	err := d.db.QueryRow("SELECT target_channel_id FROM autodrag_rules WHERE guild_id = $1 AND user_id = $2", guildID, userID).Scan(&channelID)
	return channelID, err
}

// DeleteAutoDragRule removes an autodrag rule
func (d *Database) DeleteAutoDragRule(guildID, userID string) error {
	_, err := d.db.Exec("DELETE FROM autodrag_rules WHERE guild_id = $1 AND user_id = $2", guildID, userID)
	return err
}

// GetAllAutoDragRules gets all autodrag rules for a guild
func (d *Database) GetAllAutoDragRules(guildID string) (map[string]string, error) {
	rows, err := d.db.Query("SELECT user_id, target_channel_id FROM autodrag_rules WHERE guild_id = $1", guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := make(map[string]string)
	for rows.Next() {
		var userID, channelID string
		if err := rows.Scan(&userID, &channelID); err != nil {
			return nil, err
		}
		rules[userID] = channelID
	}
	return rules, nil
}

// SetAutoAFKSettings saves auto-AFK settings for a guild
func (d *Database) SetAutoAFKSettings(guildID string, enabled bool, minutes int, afkChannelID string) error {
	query := `
		INSERT INTO autoafk_settings (guild_id, enabled, minutes, afk_channel_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT(guild_id)
		DO UPDATE SET enabled = EXCLUDED.enabled, minutes = EXCLUDED.minutes, afk_channel_id = EXCLUDED.afk_channel_id
	`
	_, err := d.db.Exec(query, guildID, enabled, minutes, afkChannelID)
	return err
}

// GetAutoAFKSettings retrieves auto-AFK settings for a guild
func (d *Database) GetAutoAFKSettings(guildID string) (*AutoAFKSettings, error) {
	var settings AutoAFKSettings
	err := d.db.QueryRow("SELECT enabled, minutes, afk_channel_id FROM autoafk_settings WHERE guild_id = $1", guildID).
		Scan(&settings.Enabled, &settings.Minutes, &settings.AFKChannelID)
	if err != nil {
		return nil, err
	}
	settings.GuildID = guildID
	return &settings, nil
}
