package queue

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"gin-starter-pack/config"
	appRedis "gin-starter-pack/pkg/redis"

	"gorm.io/gorm"
)

// DriverBuilder is a factory function for instantiating custom queue and worker drivers
type DriverBuilder func(cfg *config.QueueConfig, reg *Registry, db *gorm.DB, redisClient *appRedis.Client) (Queue, Worker, error)

var (
	driverMu sync.RWMutex
	drivers  = make(map[string]DriverBuilder)
)

func init() {
	// Register built-in drivers
	RegisterDriver("database", func(cfg *config.QueueConfig, reg *Registry, db *gorm.DB, redisClient *appRedis.Client) (Queue, Worker, error) {
		if db == nil {
			return nil, nil, fmt.Errorf("queue: database driver requires non-nil *gorm.DB")
		}
		driver := NewDatabaseDriver(db, DatabaseConfig{
			QueueName:    cfg.QueueName,
			Concurrency:  cfg.Concurrency,
			MaxAttempts:  cfg.MaxAttempts,
			PollInterval: time.Duration(cfg.PollIntervalMs) * time.Millisecond,
		}, reg)
		return driver, driver, nil
	})

	RegisterDriver("db", func(cfg *config.QueueConfig, reg *Registry, db *gorm.DB, redisClient *appRedis.Client) (Queue, Worker, error) {
		return drivers["database"](cfg, reg, db, redisClient)
	})

	RegisterDriver("memory", func(cfg *config.QueueConfig, reg *Registry, db *gorm.DB, redisClient *appRedis.Client) (Queue, Worker, error) {
		driver := NewMemoryDriver(cfg.Concurrency, 1000, reg)
		return driver, driver, nil
	})

	RegisterDriver("sync", func(cfg *config.QueueConfig, reg *Registry, db *gorm.DB, redisClient *appRedis.Client) (Queue, Worker, error) {
		driver := NewSyncDriver(reg)
		return driver, driver, nil
	})

	RegisterDriver("redis", func(cfg *config.QueueConfig, reg *Registry, db *gorm.DB, redisClient *appRedis.Client) (Queue, Worker, error) {
		if redisClient == nil || !redisClient.IsEnabled() {
			slog.Warn("Redis is disabled or nil, falling back to memory queue driver")
			return drivers["memory"](cfg, reg, db, redisClient)
		}
		driver := NewRedisDriver(redisClient, RedisConfig{
			QueueName:   cfg.QueueName,
			Concurrency: cfg.Concurrency,
			MaxAttempts: cfg.MaxAttempts,
		}, reg)
		return driver, driver, nil
	})
}

// RegisterDriver registers a custom queue driver builder
func RegisterDriver(name string, builder DriverBuilder) {
	driverMu.Lock()
	defer driverMu.Unlock()
	drivers[strings.ToLower(name)] = builder
}

// GetRegisteredDrivers returns all currently registered queue driver names
func GetRegisteredDrivers() []string {
	driverMu.RLock()
	defer driverMu.RUnlock()
	list := make([]string, 0, len(drivers))
	for name := range drivers {
		list = append(list, name)
	}
	return list
}

// InitQueue creates the configured Queue dispatcher and Worker consumer using the shared Registry
func InitQueue(cfg *config.QueueConfig, reg *Registry, db *gorm.DB, redisClient *appRedis.Client) (Queue, Worker, error) {
	if reg == nil {
		reg = NewRegistry()
	}

	driverName := strings.ToLower(cfg.Driver)
	if driverName == "" {
		driverName = "database"
	}

	driverMu.RLock()
	builder, exists := drivers[driverName]
	driverMu.RUnlock()

	if !exists {
		slog.Warn(
			fmt.Sprintf("Unsupported queue driver '%s', falling back to memory. Registered drivers: [%s]",
				cfg.Driver, strings.Join(GetRegisteredDrivers(), ", ")),
		)
		builder = drivers["memory"]
	}

	q, w, err := builder(cfg, reg, db, redisClient)
	if err != nil {
		slog.Error("Failed to initialize queue driver, falling back to memory", "driver", cfg.Driver, "error", err)
		return drivers["memory"](cfg, reg, db, redisClient)
	}

	slog.Info("Queue subsystem initialized successfully",
		"driver", driverName,
		"queue", cfg.QueueName,
		"concurrency", cfg.Concurrency,
		"max_attempts", cfg.MaxAttempts,
	)

	return q, w, nil
}
