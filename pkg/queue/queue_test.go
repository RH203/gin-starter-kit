package queue_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gin-starter-pack/config"
	"gin-starter-pack/internal/domain"
	"gin-starter-pack/pkg/database"
	"gin-starter-pack/pkg/queue"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestPayload struct {
	Message string `json:"message"`
	Counter int    `json:"counter"`
}

func TestSyncDriver(t *testing.T) {
	reg := queue.NewRegistry()
	var executed bool
	var received TestPayload

	reg.Register("test_sync_job", func(ctx context.Context, payload []byte) error {
		executed = true
		return json.Unmarshal(payload, &received)
	})

	driver := queue.NewSyncDriver(reg)
	err := driver.Dispatch(context.Background(), "test_sync_job", TestPayload{
		Message: "hello sync",
		Counter: 42,
	})

	assert.NoError(t, err)
	assert.True(t, executed)
	assert.Equal(t, "hello sync", received.Message)
	assert.Equal(t, 42, received.Counter)
	assert.NoError(t, driver.Close())
}

func TestMemoryDriver(t *testing.T) {
	reg := queue.NewRegistry()
	var wg sync.WaitGroup
	wg.Add(1)

	var received TestPayload
	reg.Register("test_memory_job", func(ctx context.Context, payload []byte) error {
		defer wg.Done()
		return json.Unmarshal(payload, &received)
	})

	driver := queue.NewMemoryDriver(2, 10, reg)
	require.NoError(t, driver.Start(context.Background()))

	err := driver.Dispatch(context.Background(), "test_memory_job", TestPayload{
		Message: "hello memory",
		Counter: 100,
	})
	require.NoError(t, err)

	wg.Wait()
	assert.Equal(t, "hello memory", received.Message)
	assert.Equal(t, 100, received.Counter)

	assert.NoError(t, driver.Stop(2*time.Second))
}

func TestDatabaseDriver_SuccessAndCleanup(t *testing.T) {
	// Setup in-memory SQLite DB
	dbCfg := &config.DBConfig{
		Driver: "sqlite",
		Name:   ":memory:",
	}
	db, err := database.InitDB(dbCfg, "test")
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(domain.Entities()...))

	reg := queue.NewRegistry()
	var wg sync.WaitGroup
	wg.Add(1)

	var received TestPayload
	reg.Register("test_db_job", func(ctx context.Context, payload []byte) error {
		defer wg.Done()
		return json.Unmarshal(payload, &received)
	})

	driver := queue.NewDatabaseDriver(db, queue.DatabaseConfig{
		QueueName:    "test-queue",
		Concurrency:  2,
		MaxAttempts:  3,
		PollInterval: 50 * time.Millisecond,
	}, reg)

	// Dispatch job before starting worker
	err = driver.Dispatch(context.Background(), "test_db_job", TestPayload{
		Message: "hello db queue",
		Counter: 999,
	})
	require.NoError(t, err)

	// Verify job record is created in pending state
	var count int64
	db.Model(&domain.JobRecord{}).Where("queue = ? AND status = ?", "test-queue", "pending").Count(&count)
	assert.Equal(t, int64(1), count)

	// Start worker to process
	require.NoError(t, driver.Start(context.Background()))

	wg.Wait()
	assert.Equal(t, "hello db queue", received.Message)
	assert.Equal(t, 999, received.Counter)

	// Wait briefly for deletion after success
	time.Sleep(100 * time.Millisecond)
	db.Model(&domain.JobRecord{}).Where("queue = ?", "test-queue").Count(&count)
	assert.Equal(t, int64(0), count, "Successful job should be deleted to keep table lean")

	assert.NoError(t, driver.Stop(2*time.Second))
}

func TestDatabaseDriver_RetryAndFailure(t *testing.T) {
	dbCfg := &config.DBConfig{
		Driver: "sqlite",
		Name:   ":memory:",
	}
	db, err := database.InitDB(dbCfg, "test")
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(domain.Entities()...))

	reg := queue.NewRegistry()
	var attempts atomic.Int32

	reg.Register("failing_job", func(ctx context.Context, payload []byte) error {
		attempts.Add(1)
		return errors.New("simulated error")
	})

	driver := queue.NewDatabaseDriver(db, queue.DatabaseConfig{
		QueueName:    "retry-queue",
		Concurrency:  1,
		MaxAttempts:  2,
		PollInterval: 20 * time.Millisecond,
	}, reg)

	// Dispatch failing job
	err = driver.Dispatch(context.Background(), "failing_job", map[string]string{"foo": "bar"})
	require.NoError(t, err)

	// Trigger worker loop once
	require.NoError(t, driver.Start(context.Background()))

	// Wait for attempt 1
	time.Sleep(100 * time.Millisecond)
	assert.GreaterOrEqual(t, attempts.Load(), int32(1))

	// Stop worker cleanly
	assert.NoError(t, driver.Stop(2*time.Second))

	// Verify database record has last_error and attempts incremented
	var record domain.JobRecord
	err = db.Where("queue = ?", "retry-queue").First(&record).Error
	require.NoError(t, err)
	assert.Contains(t, record.LastError, "simulated error")
	assert.GreaterOrEqual(t, record.Attempts, 1)
}

func TestInitQueue_Factory(t *testing.T) {
	reg := queue.NewRegistry()

	// Test Memory Fallback / Selection
	q, w, err := queue.InitQueue(&config.QueueConfig{
		Driver:      "memory",
		Concurrency: 2,
	}, reg, nil, nil)
	require.NoError(t, err)
	assert.NotNil(t, q)
	assert.NotNil(t, w)
	assert.NoError(t, w.Stop(1*time.Second))

	// Test Sync Driver Selection
	qSync, wSync, err := queue.InitQueue(&config.QueueConfig{
		Driver: "sync",
	}, reg, nil, nil)
	require.NoError(t, err)
	assert.NotNil(t, qSync)
	assert.NotNil(t, wSync)

	// Test Unknown Driver Fallback
	qUnknown, wUnknown, err := queue.InitQueue(&config.QueueConfig{
		Driver: "unknown-driver-xyz",
	}, reg, nil, nil)
	require.NoError(t, err)
	assert.NotNil(t, qUnknown)
	assert.NotNil(t, wUnknown)
	assert.NoError(t, wUnknown.Stop(1*time.Second))
}
