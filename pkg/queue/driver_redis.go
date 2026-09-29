package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	appRedis "gin-starter-pack/pkg/redis"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type redisJobMessage struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Payload     string    `json:"payload"`
	Attempts    int       `json:"attempts"`
	MaxAttempts int       `json:"max_attempts"`
	CreatedAt   time.Time `json:"created_at"`
}

// RedisDriver implements a high-throughput Redis List queue
type RedisDriver struct {
	client      *appRedis.Client
	queueKey    string
	concurrency int
	maxAttempts int
	registry    *Registry
	quit        chan struct{}
	wg          sync.WaitGroup
	running     bool
	mu          sync.Mutex
}

// RedisConfig holds settings for RedisDriver
type RedisConfig struct {
	QueueName   string
	Concurrency int
	MaxAttempts int
}

// NewRedisDriver creates a Redis-backed queue driver
func NewRedisDriver(client *appRedis.Client, cfg RedisConfig, reg *Registry) *RedisDriver {
	if cfg.QueueName == "" {
		cfg.QueueName = "default"
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 5
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if reg == nil {
		reg = NewRegistry()
	}

	return &RedisDriver{
		client:      client,
		queueKey:    fmt.Sprintf("queue:%s", cfg.QueueName),
		concurrency: cfg.Concurrency,
		maxAttempts: cfg.MaxAttempts,
		registry:    reg,
		quit:        make(chan struct{}),
	}
}

func (r *RedisDriver) Dispatch(ctx context.Context, name string, payload interface{}) error {
	return r.DispatchDelayed(ctx, name, payload, 0)
}

func (r *RedisDriver) DispatchDelayed(ctx context.Context, name string, payload interface{}, delay time.Duration) error {
	if !r.client.IsEnabled() {
		return errors.New("queue: redis is disabled in configuration")
	}

	raw, err := MarshalPayload(payload)
	if err != nil {
		return err
	}

	msg := redisJobMessage{
		ID:          uuid.NewString(),
		Name:        name,
		Payload:     string(raw),
		Attempts:    0,
		MaxAttempts: r.maxAttempts,
		CreatedAt:   time.Now().UTC(),
	}

	bytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("queue: failed to marshal redis job message: %w", err)
	}

	if delay > 0 {
		// Delayed job dispatch using a goroutine timer before pushing into active queue
		go func() {
			select {
			case <-time.After(delay):
				_ = r.client.Underlying().LPush(context.Background(), r.queueKey, string(bytes)).Err()
			case <-r.quit:
				slog.Warn("Redis queue stopped before delayed job executed", "job", name)
			}
		}()
		return nil
	}

	if err := r.client.Underlying().LPush(ctx, r.queueKey, string(bytes)).Err(); err != nil {
		return fmt.Errorf("queue: failed to push job into redis: %w", err)
	}

	slog.Debug("Job dispatched to redis queue", "id", msg.ID, "job", name, "queue", r.queueKey)
	return nil
}

func (r *RedisDriver) RegisterHandler(name string, handler Handler) {
	r.registry.Register(name, handler)
}

func (r *RedisDriver) Start(ctx context.Context) error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return nil
	}
	r.running = true
	r.mu.Unlock()

	slog.Info("Starting Redis queue worker pool", "queue", r.queueKey, "workers", r.concurrency)

	for i := 1; i <= r.concurrency; i++ {
		r.wg.Add(1)
		go r.workerLoop(i)
	}

	return nil
}

func (r *RedisDriver) workerLoop(workerID int) {
	defer r.wg.Done()
	slog.Debug("Redis worker started", "worker_id", workerID)

	for {
		select {
		case <-r.quit:
			return
		default:
			if !r.client.IsEnabled() {
				time.Sleep(1 * time.Second)
				continue
			}

			// BRPop blocks up to 2 seconds waiting for an item
			res, err := r.client.Underlying().BRPop(context.Background(), 2*time.Second, r.queueKey).Result()
			if err != nil {
				if errors.Is(err, redis.Nil) {
					continue // Timeout with no jobs
				}
				slog.Debug("Redis BRPop error", "worker_id", workerID, "error", err)
				select {
				case <-time.After(500 * time.Millisecond):
				case <-r.quit:
					return
				}
				continue
			}

			if len(res) < 2 {
				continue
			}

			rawJSON := res[1]
			var msg redisJobMessage
			if err := json.Unmarshal([]byte(rawJSON), &msg); err != nil {
				slog.Error("Failed to decode redis job message", "error", err)
				continue
			}

			msg.Attempts++
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			execErr := ExecuteJob(ctx, r.registry, msg.Name, []byte(msg.Payload))
			cancel()

			if execErr != nil {
				if msg.Attempts < msg.MaxAttempts {
					backoff := time.Duration(1<<msg.Attempts) * 2 * time.Second
					slog.Warn("Retrying failed Redis job", "job", msg.Name, "id", msg.ID, "attempt", msg.Attempts, "retry_in", backoff)
					_ = r.DispatchDelayed(context.Background(), msg.Name, msg.Payload, backoff)
				} else {
					slog.Error("Redis job permanently failed (max attempts exhausted)", "job", msg.Name, "id", msg.ID, "attempts", msg.Attempts)
				}
			}
		}
	}
}

func (r *RedisDriver) Stop(timeout time.Duration) error {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return nil
	}
	r.running = false
	close(r.quit)
	r.mu.Unlock()

	slog.Info("Shutting down Redis queue worker pool...")

	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("Redis queue worker pool stopped cleanly")
		return nil
	case <-time.After(timeout):
		slog.Warn("Redis queue worker pool shutdown timed out")
		return errors.New("queue: redis worker pool shutdown timed out")
	}
}

func (r *RedisDriver) Close() error {
	return r.Stop(5 * time.Second)
}
