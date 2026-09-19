package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gin-starter-pack/pkg/mail"
)

// Job interface that all background tasks must implement
type Job interface {
	Name() string
	Execute(ctx context.Context) error
}

// Pool represents a worker pool for concurrent background jobs
type Pool struct {
	concurrency int
	jobQueue    chan Job
	wg          sync.WaitGroup
	quit        chan struct{}
	running     bool
	mu          sync.Mutex
}

// NewPool creates a new worker pool
func NewPool(concurrency, queueSize int) *Pool {
	if concurrency <= 0 {
		concurrency = 5
	}
	if queueSize <= 0 {
		queueSize = 100
	}

	return &Pool{
		concurrency: concurrency,
		jobQueue:    make(chan Job, queueSize),
		quit:        make(chan struct{}),
	}
}

// Start spawns worker goroutines to process jobs from the queue
func (p *Pool) Start() {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}
	p.running = true
	p.mu.Unlock()

	slog.Info("Starting background worker pool", "workers", p.concurrency)

	for i := 1; i <= p.concurrency; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

func (p *Pool) worker(id int) {
	defer p.wg.Done()
	slog.Debug("Worker started", "worker_id", id)

	for {
		select {
		case job, ok := <-p.jobQueue:
			if !ok {
				slog.Debug("Worker stopped (queue closed)", "worker_id", id)
				return
			}
			p.processJob(id, job)

		case <-p.quit:
			// Drain remaining jobs before exiting
			for {
				select {
				case job, ok := <-p.jobQueue:
					if !ok {
						return
					}
					p.processJob(id, job)
				default:
					slog.Debug("Worker finished draining", "worker_id", id)
					return
				}
			}
		}
	}
}

func (p *Pool) processJob(workerID int, job Job) {
	start := time.Now()
	jobName := job.Name()
	slog.Info("Worker processing job", "worker_id", workerID, "job", jobName)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := job.Execute(ctx); err != nil {
		slog.Error("Worker job failed", "worker_id", workerID, "job", jobName, "error", err, "duration_ms", time.Since(start).Milliseconds())
	} else {
		slog.Info("Worker job completed", "worker_id", workerID, "job", jobName, "duration_ms", time.Since(start).Milliseconds())
	}
}

// Dispatch adds a job to the background queue (non-blocking if queue has space)
func (p *Pool) Dispatch(job Job) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		slog.Warn("Attempted to dispatch job to stopped worker pool", "job", job.Name())
		return false
	}

	select {
	case p.jobQueue <- job:
		return true
	default:
		slog.Error("Worker queue is full, job dropped", "job", job.Name())
		return false
	}
}

// Stop gracefully stops the worker pool and waits for active jobs to complete
func (p *Pool) Stop(timeout time.Duration) {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	close(p.quit)
	close(p.jobQueue)
	p.mu.Unlock()

	slog.Info("Shutting down background worker pool...")

	c := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(c)
	}()

	select {
	case <-c:
		slog.Info("Background worker pool stopped cleanly")
	case <-time.After(timeout):
		slog.Warn("Background worker pool shutdown timed out")
	}
}

// ==========================================
// Example Jobs
// ==========================================

// WelcomeEmailJob example background job for sending welcome emails
type WelcomeEmailJob struct {
	UserID   string
	Email    string
	UserName string
	Mailer   mail.Mailer
}

func (j *WelcomeEmailJob) Name() string {
	return fmt.Sprintf("WelcomeEmailJob:%s", j.Email)
}

func (j *WelcomeEmailJob) Execute(ctx context.Context) error {
	if j.Mailer != nil {
		subject := fmt.Sprintf("Welcome to Gin Starter Pack, %s!", j.UserName)
		templateData := map[string]interface{}{
			"AppName":   "Gin Starter Pack",
			"UserName":  j.UserName,
			"Email":     j.Email,
			"ActionURL": "http://localhost:8080",
			"Year":      time.Now().Year(),
		}
		return j.Mailer.SendTemplate(ctx, j.Email, subject, "welcome.html", templateData)
	}

	// Fallback to simulated delivery log if no mailer configured
	time.Sleep(150 * time.Millisecond)
	slog.Info("Welcome email dispatched to user", "email", j.Email, "name", j.UserName, "user_id", j.UserID)
	return nil
}
