package database

import (
	"database/sql"
	"discord-giveaway-bot/internal/models"
	"errors"
)

// Economy User Operations

func (d *Database) GetEconomyUser(guildID, userID string) (*models.EconomyUser, error) {
	// Cache logic moved to service layer
	query := `SELECT * FROM economy_users WHERE guild_id = $1 AND user_id = $2`
	row := d.db.QueryRow(query, guildID, userID)

	var u models.EconomyUser
	err := row.Scan(&u.UserID, &u.GuildID, &u.Balance, &u.LastDaily, &u.LastWeekly, &u.LastHourly, &u.TotalEarned, &u.TotalSpent)
	if err == sql.ErrNoRows {
		// Return a default user if not found
		return &models.EconomyUser{
			UserID:  userID,
			GuildID: guildID,
		}, nil
	}
	if err != nil {
		return nil, err
	}

	return &u, nil
}

func (d *Database) UpdateEconomyUser(u *models.EconomyUser) error {
	query := `
		INSERT INTO economy_users (
			user_id, guild_id, balance, last_daily, last_weekly, last_hourly, total_earned, total_spent
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT(user_id, guild_id) DO UPDATE SET
			balance = EXCLUDED.balance,
			last_daily = EXCLUDED.last_daily,
			last_weekly = EXCLUDED.last_weekly,
			last_hourly = EXCLUDED.last_hourly,
			total_earned = EXCLUDED.total_earned,
			total_spent = EXCLUDED.total_spent
	`
	_, err := d.db.Exec(query,
		u.UserID, u.GuildID, u.Balance, u.LastDaily, u.LastWeekly, u.LastHourly, u.TotalEarned, u.TotalSpent,
	)
	return err
}

// Atomic Operations

func (d *Database) AddUserBalance(guildID, userID string, amount int64) (int64, error) {
	query := `
		INSERT INTO economy_users (user_id, guild_id, balance, total_earned)
		VALUES ($1, $2, $3, $3)
		ON CONFLICT(user_id, guild_id) DO UPDATE SET
			balance = economy_users.balance + $3,
			total_earned = economy_users.total_earned + $3
		RETURNING balance
	`
	var newBalance int64
	err := d.db.QueryRow(query, userID, guildID, amount).Scan(&newBalance)
	return newBalance, err
}

func (d *Database) RemoveUserBalance(guildID, userID string, amount int64) (int64, error) {
	// First check if user exists and has enough balance
	// We can do this in one query with a WHERE clause, but we need to handle "not found" vs "insufficient funds"
	// Actually, for atomic updates, we can try to update and check rows affected or returning.

	query := `
		UPDATE economy_users 
		SET balance = balance - $3, total_spent = total_spent + $3
		WHERE user_id = $1 AND guild_id = $2 AND balance >= $3
		RETURNING balance
	`
	var newBalance int64
	err := d.db.QueryRow(query, userID, guildID, amount).Scan(&newBalance)

	if err == sql.ErrNoRows {
		// Either user doesn't exist or insufficient funds
		// Let's check if user exists
		var balance int64
		errCheck := d.db.QueryRow("SELECT balance FROM economy_users WHERE user_id = $1 AND guild_id = $2", userID, guildID).Scan(&balance)
		if errCheck == sql.ErrNoRows {
			return 0, errors.New("user not found") // Or handle as 0 balance
		}
		if balance < amount {
			return balance, errors.New("insufficient funds")
		}
		return 0, err // Should not happen
	}

	return newBalance, err
}

func (d *Database) GetEconomyLeaderboard(guildID string, limit int) ([]*models.EconomyUser, error) {
	query := `SELECT * FROM economy_users WHERE guild_id = $1 ORDER BY balance DESC LIMIT $2`
	rows, err := d.db.Query(query, guildID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.EconomyUser
	for rows.Next() {
		var u models.EconomyUser
		if err := rows.Scan(&u.UserID, &u.GuildID, &u.Balance, &u.LastDaily, &u.LastWeekly, &u.LastHourly, &u.TotalEarned, &u.TotalSpent); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, nil
}

// Economy Config Operations

func (d *Database) GetEconomyConfig(guildID string) (*models.EconomyConfig, error) {
	// Cache logic moved to service layer
	query := `SELECT guild_id, message_reward, vc_reward_per_min, daily_reward, weekly_reward, hourly_reward, invite_reward, react_reward, poll_reward, event_reward, upvote_reward, gamble_enabled, max_gamble_amount, allowed_channels, currency_emoji FROM economy_config WHERE guild_id = $1`
	row := d.db.QueryRow(query, guildID)

	var c models.EconomyConfig
	var gambleEnabled int
	var currencyEmoji sql.NullString
	err := row.Scan(
		&c.GuildID, &c.MessageReward, &c.VCRewardPerMin, &c.DailyReward, &c.WeeklyReward,
		&c.HourlyReward, &c.InviteReward, &c.ReactReward, &c.PollReward, &c.EventReward,
		&c.UpvoteReward, &gambleEnabled, &c.MaxGambleAmount, &c.AllowedChannels, &currencyEmoji,
	)
	if err == sql.ErrNoRows {
		// Return default config
		return &models.EconomyConfig{GuildID: guildID, MaxGambleAmount: 20000, CurrencyEmoji: "<:Cash:1443554334670327848>"}, nil
	}
	if err != nil {
		return nil, err
	}
	c.GambleEnabled = models.IntToBool(gambleEnabled)
	c.CurrencyEmoji = currencyEmoji.String
	if c.CurrencyEmoji == "" {
		c.CurrencyEmoji = "<:Cash:1443554334670327848>"
	}

	return &c, nil
}

func (d *Database) UpdateEconomyConfig(c *models.EconomyConfig) error {
	query := `
		INSERT INTO economy_config (
			guild_id, message_reward, vc_reward_per_min, daily_reward, weekly_reward, hourly_reward,
			invite_reward, react_reward, poll_reward, event_reward, upvote_reward, gamble_enabled, max_gamble_amount, allowed_channels, currency_emoji
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT(guild_id) DO UPDATE SET
			message_reward = EXCLUDED.message_reward,
			vc_reward_per_min = EXCLUDED.vc_reward_per_min,
			daily_reward = EXCLUDED.daily_reward,
			weekly_reward = EXCLUDED.weekly_reward,
			hourly_reward = EXCLUDED.hourly_reward,
			invite_reward = EXCLUDED.invite_reward,
			react_reward = EXCLUDED.react_reward,
			poll_reward = EXCLUDED.poll_reward,
			event_reward = EXCLUDED.event_reward,
			upvote_reward = EXCLUDED.upvote_reward,
			gamble_enabled = EXCLUDED.gamble_enabled,
			max_gamble_amount = EXCLUDED.max_gamble_amount,
			allowed_channels = EXCLUDED.allowed_channels,
			currency_emoji = EXCLUDED.currency_emoji
	`
	_, err := d.db.Exec(query,
		c.GuildID, c.MessageReward, c.VCRewardPerMin, c.DailyReward, c.WeeklyReward, c.HourlyReward,
		c.InviteReward, c.ReactReward, c.PollReward, c.EventReward, c.UpvoteReward, models.BoolToInt(c.GambleEnabled), c.MaxGambleAmount, c.AllowedChannels, c.CurrencyEmoji,
	)
	return err
}

func (d *Database) ResetEconomy(guildID string) error {
	_, err := d.db.Exec("DELETE FROM economy_users WHERE guild_id = $1", guildID)
	return err
}

func (d *Database) GetTotalEconomyStats() (int64, int64, error) {
	var totalUsers int64
	var totalCoins int64
	err := d.db.QueryRow("SELECT COUNT(*), COALESCE(SUM(balance), 0) FROM economy_users").Scan(&totalUsers, &totalCoins)
	return totalUsers, totalCoins, err
}
