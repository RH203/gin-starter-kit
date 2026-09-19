package redis_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gin-starter-pack/config"
	"gin-starter-pack/pkg/redis"

	"github.com/stretchr/testify/assert"
)

type sampleUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func TestRedis_DisabledGracefulFallback(t *testing.T) {
	cfg := &config.RedisConfig{Enabled: false}
	client, err := redis.InitRedis(cfg)
	assert.NoError(t, err)
	assert.False(t, client.IsEnabled())
	assert.Nil(t, client.Underlying())

	ctx := context.Background()

	// Safe no-op methods
	assert.NoError(t, client.Set(ctx, "key", "val", time.Minute))
	val, err := client.Get(ctx, "key")
	assert.NoError(t, err)
	assert.Empty(t, val)

	assert.NoError(t, client.Del(ctx, "key"))
	assert.NoError(t, client.Invalidate(ctx, "key"))
	assert.NoError(t, client.Ping(ctx))
	assert.NoError(t, client.Close())
}

func TestRedis_Remember_Disabled(t *testing.T) {
	cfg := &config.RedisConfig{Enabled: false}
	client, _ := redis.InitRedis(cfg)
	ctx := context.Background()

	calls := 0
	res, err := redis.Remember(ctx, client, "user:101", 5*time.Minute, func() (*sampleUser, error) {
		calls++
		return &sampleUser{ID: "101", Name: "Tester"}, nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, calls)
	assert.Equal(t, "101", res.ID)
	assert.Equal(t, "Tester", res.Name)
}

func TestRedis_RememberSlice_Disabled(t *testing.T) {
	cfg := &config.RedisConfig{Enabled: false}
	client, _ := redis.InitRedis(cfg)
	ctx := context.Background()

	items, err := redis.RememberSlice(ctx, client, "users:all", 5*time.Minute, func() ([]sampleUser, error) {
		return []sampleUser{{ID: "1", Name: "A"}, {ID: "2", Name: "B"}}, nil
	})

	assert.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestRedis_Remember_FallbackError(t *testing.T) {
	cfg := &config.RedisConfig{Enabled: false}
	client, _ := redis.InitRedis(cfg)
	ctx := context.Background()

	expectedErr := errors.New("database failure")
	res, err := redis.Remember(ctx, client, "user:error", 5*time.Minute, func() (*sampleUser, error) {
		return nil, expectedErr
	})

	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}
