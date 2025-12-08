package redis

import (
	"fmt"
	"strconv"
	"time"
)

// Economy Caching

func (c *Client) GetBalance(guildID, userID string) (int64, bool) {
	key := fmt.Sprintf("balance:%s:%s", guildID, userID)
	val, err := c.Get(key)
	if err != nil {
		return 0, false
	}
	balance, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, false
	}
	return balance, true
}

func (c *Client) SetBalance(guildID, userID string, balance int64) error {
	key := fmt.Sprintf("balance:%s:%s", guildID, userID)
	// Cache for 1 hour, write-through pattern will update DB
	return c.Set(key, balance, time.Hour)
}

func (c *Client) InvalidateBalance(guildID, userID string) error {
	key := fmt.Sprintf("balance:%s:%s", guildID, userID)
	return c.Del(key)
}

// Cooldowns

func (c *Client) SetCooldown(key string, duration time.Duration) error {
	return c.Set(key, 1, duration)
}

func (c *Client) CheckCooldown(key string) (time.Duration, bool) {
	ttl := c.client.TTL(ctx, key).Val()
	if ttl <= 0 {
		return 0, false
	}
	return ttl, true
}

// Leaderboards

func (c *Client) UpdateLeaderboard(guildID, userID string, balance float64) error {
	key := fmt.Sprintf("leaderboard:%s", guildID)
	return c.ZAdd(key, balance, userID)
}

func (c *Client) GetLeaderboard(guildID string, limit int) ([]string, error) {
	key := fmt.Sprintf("leaderboard:%s", guildID)
	results, err := c.ZRevRangeWithScores(key, 0, int64(limit-1))
	if err != nil {
		return nil, err
	}

	users := make([]string, len(results))
	for i, z := range results {
		users[i] = z.Member.(string)
	}
	return users, nil
}
