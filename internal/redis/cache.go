package redis

import (
	"discord-giveaway-bot/internal/models"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
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

// Prefix Caching

func (c *Client) GetPrefix(guildID string) (string, bool) {
	key := fmt.Sprintf("prefix:%s", guildID)
	val, err := c.Get(key)
	if err != nil {
		return "", false
	}
	return val, true
}

func (c *Client) SetPrefix(guildID, prefix string) error {
	key := fmt.Sprintf("prefix:%s", guildID)
	// Cache for 24 hours, as prefixes rarely change
	return c.Set(key, prefix, 24*time.Hour)
}

func (c *Client) InvalidatePrefix(guildID string) error {
	key := fmt.Sprintf("prefix:%s", guildID)
	return c.Del(key)
}

// Giveaway Caching

func (c *Client) GetActiveGiveaways(guildID string) ([]*models.Giveaway, bool) {
	key := fmt.Sprintf("giveaways:active:%s", guildID)
	val, err := c.Get(key)
	if err != nil {
		return nil, false
	}

	var giveaways []*models.Giveaway
	if err := json.Unmarshal([]byte(val), &giveaways); err != nil {
		return nil, false
	}
	return giveaways, true
}

func (c *Client) SetActiveGiveaways(guildID string, giveaways []*models.Giveaway) error {
	key := fmt.Sprintf("giveaways:active:%s", guildID)
	data, err := json.Marshal(giveaways)
	if err != nil {
		return err
	}
	// Cache for 5 minutes, invalidate on create/end
	return c.Set(key, data, 5*time.Minute)
}

func (c *Client) InvalidateActiveGiveaways(guildID string) error {
	key := fmt.Sprintf("giveaways:active:%s", guildID)
	return c.Del(key)
}

// Giveaway Ending Queue (ZSET)

func (c *Client) AddToEndingQueue(messageID string, endTime int64) error {
	return c.ZAdd("giveaways:ending", float64(endTime), messageID)
}

func (c *Client) RemoveFromEndingQueue(messageID string) error {
	return c.ZRem("giveaways:ending", messageID)
}

func (c *Client) GetDueGiveaways(now int64) ([]string, error) {
	// Get all giveaways with score <= now
	results, err := c.ZRangeByScore("giveaways:ending", "-inf", fmt.Sprintf("%d", now))
	if err != nil {
		return nil, err
	}
	return results, nil
}

// Message Counting

func (c *Client) IncrementMessageCount(guildID, userID string) error {
	key := fmt.Sprintf("msg_count:%s:%s", guildID, userID)
	_, err := c.Incr(key)
	return err
}

func (c *Client) GetAndClearMessageCounts(guildID string) (map[string]int64, error) {
	// This is tricky with simple keys.
	// Better to use a Hash per guild: msg_counts:{guildID} field: {userID}
	// But Incr works on keys.
	// Let's use HINCRBY.
	return nil, nil
}

func (c *Client) IncrementMessageCountHash(guildID, userID string) error {
	key := fmt.Sprintf("msg_counts:%s", guildID)
	return c.client.HIncrBy(ctx, key, userID, 1).Err()
}

func (c *Client) GetAndClearGuildMessageCounts(guildID string) (map[string]int64, error) {
	key := fmt.Sprintf("msg_counts:%s", guildID)

	// Rename key to process it safely (atomic)
	tempKey := fmt.Sprintf("msg_counts_temp:%s:%d", guildID, time.Now().UnixNano())
	if err := c.client.Rename(ctx, key, tempKey).Err(); err != nil {
		if err == redis.Nil {
			return nil, nil // No key
		}
		// If key doesn't exist (Rename returns error), check if it's that error
		// redis.Nil is usually for Get. Rename returns "ERR no such key"
		return nil, nil
	}

	results, err := c.client.HGetAll(ctx, tempKey).Result()
	if err != nil {
		return nil, err
	}

	// Delete temp key
	c.client.Del(ctx, tempKey)

	counts := make(map[string]int64)
	for userID, countStr := range results {
		count, _ := strconv.ParseInt(countStr, 10, 64)
		counts[userID] = count
	}
	return counts, nil
}
