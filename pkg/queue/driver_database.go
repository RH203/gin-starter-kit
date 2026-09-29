package queue

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gin-starter-pack/internal/domain"

	"gorm.io/gorm"
)

// DatabaseDriver implements a persistent Laravel-style SQL database queue
type DatabaseDriver struct {
	db           *gorm.DB
	queueName    string
	concurrency  int
	maxAttempts  int
	pollInterval time.Duration
	registry     *Registry
	quit         chan struct{}
	wg           sync.WaitGroup
	running      bool
	mu           sync.Mutex
}

// DatabaseConfig holds settings for the DatabaseDriver
type DatabaseConfig struct {
	QueueName    string
	Concurrency  int
	MaxAttempts  int
	PollInterval time.Duration
}

// NewDatabaseDriver creates a database-backed queue driver
func NewDatabaseDriver(db *gorm.DB, cfg DatabaseConfig, reg *Registry) *DatabaseDriver {
	if cfg.QueueName == "" {
		cfg.QueueName = "default"
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 5
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 1000 * time.Millisecond
	}
	if reg == nil {
		reg = NewRegistry()
	}

	return &DatabaseDriver{
		db:           db,
		queueName:    cfg.QueueName,
		concurrency:  cfg.Concurrency,
		maxAttempts:  cfg.MaxAttempts,
		pollInterval: cfg.PollInterval,
		registry:     reg,
		quit:         make(chan struct{}),
	}
}

func (d *DatabaseDriver) Dispatch(ctx context.Context, name string, payload interface{}) error {
	return d.DispatchDelayed(ctx, name, payload, 0)
}

func (d *DatabaseDriver) DispatchDelayed(ctx context.Context, name string, payload interface{}, delay time.Duration) error {
	raw, err := MarshalPayload(payload)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	availableAt := now
	if delay > 0 {
		availableAt = now.Add(delay)
	}

	record := domain.JobRecord{
		Queue:       d.queueName,
		Name:        name,
		Payload:     string(raw),
		Attempts:    0,
		MaxAttempts: d.maxAttempts,
		AvailableAt: availableAt,
		Status:      "pending",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := d.db.WithContext(ctx).Create(&record).Error; err != nil {
		return fmt.Errorf("queue: failed to insert database job: %w", err)
	}

	slog.Debug("Job dispatched to database queue", "id", record.ID, "job", name, "queue", d.queueName)
	return nil
}

func (d *DatabaseDriver) RegisterHandler(name string, handler Handler) {
	d.registry.Register(name, handler)
}

func (d *DatabaseDriver) Start(ctx context.Context) error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return nil
	}
	d.running = true
	d.mu.Unlock()

	slog.Info("Starting database queue worker pool", "queue", d.queueName, "workers", d.concurrency)

	for i := 1; i <= d.concurrency; i++ {
		d.wg.Add(1)
		go d.workerLoop(i)
	}

	return nil
}

func (d *DatabaseDriver) workerLoop(workerID int) {
	defer d.wg.Done()
	slog.Debug("Database worker loop started", "worker_id", workerID)

	for {
		select {
		case <-d.quit:
			slog.Debug("Database worker stopping", "worker_id", workerID)
			return
		default:
			processed, err := d.processNextJob()
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				slog.Error("Database worker query error", "worker_id", workerID, "error", err)
			}

			// If no job was available to claim, back off according to pollInterval
			if !processed {
				select {
				case <-time.After(d.pollInterval):
				case <-d.quit:
					return
				}
			}
		}
	}
}

// processNextJob atomically claims and executes the next pending job
func (d *DatabaseDriver) processNextJob() (bool, error) {
	var job domain.JobRecord

	// Transaction to claim one eligible job atomically
	err := d.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		staleThreshold := now.Add(-10 * time.Minute) // Re-claim jobs if worker crashed 10+ mins ago

		res := tx.Where(
			"queue = ? AND ((status = ? AND available_at <= ?) OR (status = ? AND reserved_at < ?))",
			d.queueName, "pending", now, "processing", staleThreshold,
		).
			Order("available_at ASC, id ASC").
			Limit(1).
			Find(&job)

		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		// Reserve the job
		job.Status = "processing"
		job.Attempts++
		job.ReservedAt = &now
		job.UpdatedAt = now

		return tx.Save(&job).Error
	})

	if err != nil {
		return false, err
	}

	// Job claimed successfully, execute handler
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	execErr := ExecuteJob(ctx, d.registry, job.Name, []byte(job.Payload))
	cancel()

	now := time.Now().UTC()

	if execErr == nil {
		// Completed successfully, delete record to keep table lean (Laravel convention)
		if delErr := d.db.Delete(&domain.JobRecord{}, job.ID).Error; delErr != nil {
			slog.Warn("Failed to delete completed job record", "job_id", job.ID, "error", delErr)
		}
	} else {
		// Job execution failed, handle retry backoff
		if job.Attempts < job.MaxAttempts {
			// Exponential backoff: 2s, 4s, 8s...
			backoffSec := int(1 << job.Attempts) * 2
			availableAt := now.Add(time.Duration(backoffSec) * time.Second)

			slog.Warn("Retrying failed job", "job_id", job.ID, "job", job.Name, "attempt", job.Attempts, "retry_in_sec", backoffSec)

			d.db.Model(&domain.JobRecord{}).Where("id = ?", job.ID).Updates(map[string]interface{}{
				"status":       "pending",
				"available_at": availableAt,
				"reserved_at":  nil,
				"last_error":   execErr.Error(),
				"updated_at":   now,
			})
		} else {
			// Max attempts reached, mark as permanently failed
			slog.Error("Job failed permanently (exhausted attempts)", "job_id", job.ID, "job", job.Name, "attempts", job.Attempts)

			d.db.Model(&domain.JobRecord{}).Where("id = ?", job.ID).Updates(map[string]interface{}{
				"status":     "failed",
				"last_error": execErr.Error(),
				"updated_at": now,
			})
		}
	}

	return true, nil
}

func (d *DatabaseDriver) Stop(timeout time.Duration) error {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return nil
	}
	d.running = false
	close(d.quit)
	d.mu.Unlock()

	slog.Info("Shutting down database queue worker pool...")

	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("Database queue worker pool stopped cleanly")
		return nil
	case <-time.After(timeout):
		slog.Warn("Database queue worker pool shutdown timed out")
		return errors.New("queue: worker pool shutdown timed out")
	}
}

func (d *DatabaseDriver) Close() error {
	return d.Stop(5 * time.Second)
}
