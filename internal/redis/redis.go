package redis

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
	Network  string `json:"network"` // "tcp" or "unix" for socket path
}

type Client struct {
	client         *redis.Client
	lastPingTime   time.Time
	lastPingError  error
	pingCacheMutex sync.RWMutex
}

var ctx = context.Background()

func New(cfg Config) (*Client, error) {
	// Determine network type - use Unix socket for local Redis (microsecond latency)
	network := "tcp"
	if cfg.Network != "" {
		network = cfg.Network
	}
	
	// If addr looks like a socket path, automatically use unix
	if len(cfg.Addr) > 0 && cfg.Addr[0] == '/' {
		network = "unix"
	}

	opts := &redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		Network:  network,
		// Connection pool settings for high performance
		PoolSize:     100, // Increased from default 10
		MinIdleConns: 20,  // Keep connections warm
		MaxRetries:   3,   // Retry failed commands
		PoolTimeout:  4 * time.Second,
		// Performance optimizations
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	rdb := redis.NewClient(opts)

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	if network == "unix" {
		log.Println("✓ Redis connected via Unix socket (microsecond latency)")
	} else {
		log.Println("✓ Redis connected via TCP")
	}

	return &Client{client: rdb}, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) Ping() error {
	// Do actual ping
	return c.client.Ping(ctx).Err()
}

// Basic operations

func (c *Client) Set(key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

func (c *Client) Get(key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

func (c *Client) Del(key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *Client) Incr(key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

func (c *Client) Decr(key string) (int64, error) {
	return c.client.Decr(ctx, key).Result()
}

func (c *Client) Expire(key string, expiration time.Duration) error {
	return c.client.Expire(ctx, key, expiration).Err()
}

// ZSet operations (for leaderboards)

func (c *Client) ZAdd(key string, score float64, member interface{}) error {
	return c.client.ZAdd(ctx, key, redis.Z{
		Score:  score,
		Member: member,
	}).Err()
}

func (c *Client) ZIncrBy(key string, increment float64, member string) (float64, error) {
	return c.client.ZIncrBy(ctx, key, increment, member).Result()
}

func (c *Client) ZRevRangeWithScores(key string, start, stop int64) ([]redis.Z, error) {
	return c.client.ZRevRangeWithScores(ctx, key, start, stop).Result()
}

func (c *Client) ZScore(key string, member string) (float64, error) {
	return c.client.ZScore(ctx, key, member).Result()
}

func (c *Client) ZRem(key string, members ...interface{}) error {
	return c.client.ZRem(ctx, key, members...).Err()
}

func (c *Client) ZRangeByScore(key string, min, max string) ([]string, error) {
	return c.client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: min,
		Max: max,
	}).Result()
}

// Hash operations

func (c *Client) HSet(key string, values ...interface{}) error {
	return c.client.HSet(ctx, key, values...).Err()
}

func (c *Client) HGet(key, field string) (string, error) {
	return c.client.HGet(ctx, key, field).Result()
}

func (c *Client) HGetAll(key string) (map[string]string, error) {
	return c.client.HGetAll(ctx, key).Result()
}

// Batch operations using pipelining for high performance

// MGet retrieves multiple keys in a single round-trip
func (c *Client) MGet(keys ...string) ([]interface{}, error) {
	return c.client.MGet(ctx, keys...).Result()
}

// MSet sets multiple key-value pairs in a single round-trip
func (c *Client) MSet(pairs ...interface{}) error {
	return c.client.MSet(ctx, pairs...).Err()
}

// Pipeline returns a Redis pipeline for batching commands
func (c *Client) Pipeline() redis.Pipeliner {
	return c.client.Pipeline()
}

// ExecutePipeline executes a pipeline with multiple commands
func (c *Client) ExecutePipeline(fn func(redis.Pipeliner) error) error {
	pipe := c.client.Pipeline()
	if err := fn(pipe); err != nil {
		return err
	}
	_, err := pipe.Exec(ctx)
	return err
}
