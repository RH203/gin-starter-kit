package queue

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

type memoryJob struct {
	name    string
	payload []byte
}

// MemoryDriver runs an in-memory buffered channel worker pool
type MemoryDriver struct {
	concurrency int
	jobChan     chan memoryJob
	registry    *Registry
	quit        chan struct{}
	wg          sync.WaitGroup
	running     bool
	mu          sync.Mutex
}

// NewMemoryDriver initializes an in-memory queue driver
func NewMemoryDriver(concurrency, queueSize int, reg *Registry) *MemoryDriver {
	if concurrency <= 0 {
		concurrency = 5
	}
	if queueSize <= 0 {
		queueSize = 100
	}
	if reg == nil {
		reg = NewRegistry()
	}

	return &MemoryDriver{
		concurrency: concurrency,
		jobChan:     make(chan memoryJob, queueSize),
		registry:    reg,
		quit:        make(chan struct{}),
	}
}

func (m *MemoryDriver) Dispatch(ctx context.Context, name string, payload interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	raw, err := MarshalPayload(payload)
	if err != nil {
		return err
	}

	select {
	case m.jobChan <- memoryJob{name: name, payload: raw}:
		return nil
	default:
		slog.Error("Memory queue is full, job dropped", "job", name)
		return errors.New("queue: memory buffer full, job dropped")
	}
}

func (m *MemoryDriver) DispatchDelayed(ctx context.Context, name string, payload interface{}, delay time.Duration) error {
	raw, err := MarshalPayload(payload)
	if err != nil {
		return err
	}

	go func() {
		select {
		case <-time.After(delay):
			_ = m.Dispatch(context.Background(), name, raw)
		case <-m.quit:
			slog.Warn("Queue stopped before delayed job fired", "job", name)
		}
	}()
	return nil
}

func (m *MemoryDriver) RegisterHandler(name string, handler Handler) {
	m.registry.Register(name, handler)
}

func (m *MemoryDriver) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = true
	m.mu.Unlock()

	slog.Info("Starting in-memory queue worker", "workers", m.concurrency)

	for i := 1; i <= m.concurrency; i++ {
		m.wg.Add(1)
		go m.workerLoop(i)
	}

	return nil
}

func (m *MemoryDriver) workerLoop(workerID int) {
	defer m.wg.Done()
	slog.Debug("Memory worker started", "worker_id", workerID)

	for {
		select {
		case job, ok := <-m.jobChan:
			if !ok {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			_ = ExecuteJob(ctx, m.registry, job.name, job.payload)
			cancel()

		case <-m.quit:
			// Drain remaining buffered jobs
			for {
				select {
				case job, ok := <-m.jobChan:
					if !ok {
						return
					}
					ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
					_ = ExecuteJob(ctx, m.registry, job.name, job.payload)
					cancel()
				default:
					return
				}
			}
		}
	}
}

func (m *MemoryDriver) Stop(timeout time.Duration) error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = false
	close(m.quit)
	close(m.jobChan)
	m.mu.Unlock()

	slog.Info("Shutting down memory queue worker pool...")

	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("Memory queue worker pool stopped cleanly")
		return nil
	case <-time.After(timeout):
		slog.Warn("Memory queue worker pool shutdown timed out")
		return errors.New("queue: worker pool shutdown timed out")
	}
}

func (m *MemoryDriver) Close() error {
	return m.Stop(5 * time.Second)
}
