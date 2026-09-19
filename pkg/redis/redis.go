package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gin-starter-pack/config"

	"github.com/redis/go-redis/v9"
)

// Client wraps *redis.Client to provide safe optional caching
type Client struct {
	rdb     *redis.Client
	enabled bool
}

// InitRedis initializes Redis connection conditionally
func InitRedis(cfg *config.RedisConfig) (*Client, error) {
	if !cfg.Enabled {
		slog.Info("Redis is disabled via configuration")
		return &Client{rdb: nil, enabled: false}, nil
	}

	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", addr, err)
	}

	slog.Info("Redis connected successfully", "addr", addr, "db", cfg.DB)
	return &Client{rdb: rdb, enabled: true}, nil
}

// IsEnabled returns true if Redis is active
func (c *Client) IsEnabled() bool {
	return c != nil && c.enabled && c.rdb != nil
}

// Underlying returns the raw go-redis client (can be nil)
func (c *Client) Underlying() *redis.Client {
	if c == nil {
		return nil
	}
	return c.rdb
}

// Get gets string value safely (no-op if disabled)
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	if !c.IsEnabled() {
		return "", nil
	}
	return c.rdb.Get(ctx, key).Result()
}

// Set sets string/raw value safely (no-op if disabled)
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if !c.IsEnabled() {
		return nil
	}
	return c.rdb.Set(ctx, key, value, expiration).Err()
}

// Del deletes keys safely (no-op if disabled)
func (c *Client) Del(ctx context.Context, keys ...string) error {
	if !c.IsEnabled() {
		return nil
	}
	return c.rdb.Del(ctx, keys...).Err()
}

// Invalidate safely deletes one or more cache keys
func (c *Client) Invalidate(ctx context.Context, keys ...string) error {
	return c.Del(ctx, keys...)
}

// GetJSON retrieves and unmarshals cached JSON into dest.
// Returns found (bool) and error if any occurred during retrieval or decoding.
func (c *Client) GetJSON(ctx context.Context, key string, dest interface{}) (bool, error) {
	if !c.IsEnabled() {
		return false, nil
	}

	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}

	if val == "" {
		return false, nil
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		slog.Warn("Failed to decode cached JSON", "key", key, "error", err)
		return false, err
	}

	return true, nil
}

// SetJSON marshals value to JSON and caches it in Redis with expiration
func (c *Client) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if !c.IsEnabled() {
		return nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to encode cache value to JSON: %w", err)
	}

	return c.rdb.Set(ctx, key, string(data), expiration).Err()
}

// Ping checks redis liveness
func (c *Client) Ping(ctx context.Context) error {
	if !c.IsEnabled() {
		return nil
	}
	return c.rdb.Ping(ctx).Err()
}

// Close gracefully closes connection
func (c *Client) Close() error {
	if c != nil && c.rdb != nil {
		return c.rdb.Close()
	}
	return nil
}

// Remember attempts to retrieve a cached value by key, or executes fallback and caches the result
func Remember[T any](ctx context.Context, c *Client, key string, expiration time.Duration, fallback func() (*T, error)) (*T, error) {
	if c.IsEnabled() {
		var cached T
		found, err := c.GetJSON(ctx, key, &cached)
		if err == nil && found {
			slog.Debug("Cache hit", "key", key)
			return &cached, nil
		}
	}

	// Cache miss or disabled, run fallback
	result, err := fallback()
	if err != nil {
		return nil, err
	}

	// Store retrieved result in cache
	if c.IsEnabled() && result != nil {
		if err := c.SetJSON(ctx, key, result, expiration); err != nil {
			slog.Warn("Failed to cache result", "key", key, "error", err)
		}
	}

	return result, nil
}

// RememberSlice attempts to retrieve a cached slice by key, or executes fallback and caches the result
func RememberSlice[T any](ctx context.Context, c *Client, key string, expiration time.Duration, fallback func() ([]T, error)) ([]T, error) {
	if c.IsEnabled() {
		var cached []T
		found, err := c.GetJSON(ctx, key, &cached)
		if err == nil && found {
			slog.Debug("Cache hit", "key", key)
			return cached, nil
		}
	}

	result, err := fallback()
	if err != nil {
		return nil, err
	}

	if c.IsEnabled() && result != nil {
		if err := c.SetJSON(ctx, key, result, expiration); err != nil {
			slog.Warn("Failed to cache slice result", "key", key, "error", err)
		}
	}

	return result, nil
}
