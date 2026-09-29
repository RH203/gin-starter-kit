package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

var (
	ErrHandlerNotFound = errors.New("queue: handler not registered for job")
	ErrQueueClosed      = errors.New("queue: queue has been closed")
)

// Handler represents a worker execution function for a specific job payload
type Handler func(ctx context.Context, payload []byte) error

// Queue defines the unified contract for dispatching background jobs
type Queue interface {
	// Dispatch queues a job immediately for background execution
	Dispatch(ctx context.Context, name string, payload interface{}) error

	// DispatchDelayed queues a job to be processed after the specified duration
	DispatchDelayed(ctx context.Context, name string, payload interface{}, delay time.Duration) error

	// RegisterHandler registers a processing handler for a job name
	RegisterHandler(name string, handler Handler)

	// Close gracefully closes the queue dispatcher
	Close() error
}

// Worker defines the contract for processing queued jobs
type Worker interface {
	// Start starts consuming and processing jobs (non-blocking or blocking depending on implementation)
	Start(ctx context.Context) error

	// Stop gracefully stops the worker, waiting for active jobs to complete up to the timeout
	Stop(timeout time.Duration) error
}

// Registry stores handler mappings thread-safely
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

// NewRegistry creates a new handler registry
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]Handler),
	}
}

// Register registers a handler for a job name
func (r *Registry) Register(name string, h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[name] = h
}

// Get retrieves a handler by job name
func (r *Registry) Get(name string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[name]
	return h, ok
}

// MarshalPayload converts arbitrary payload structs or values into raw JSON bytes
func MarshalPayload(payload interface{}) ([]byte, error) {
	if payload == nil {
		return []byte("{}"), nil
	}
	switch v := payload.(type) {
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("queue: failed to marshal payload: %w", err)
		}
		return data, nil
	}
}

// ExecuteJob executes a job with its registered handler, logging duration and errors
func ExecuteJob(ctx context.Context, registry *Registry, jobName string, payload []byte) error {
	handler, exists := registry.Get(jobName)
	if !exists {
		slog.Error("No queue handler registered for job", "job", jobName)
		return fmt.Errorf("%w: %s", ErrHandlerNotFound, jobName)
	}

	start := time.Now()
	slog.Info("Executing background job", "job", jobName, "size_bytes", len(payload))

	err := handler(ctx, payload)
	duration := time.Since(start)

	if err != nil {
		slog.Error("Background job failed", "job", jobName, "duration_ms", duration.Milliseconds(), "error", err)
		return err
	}

	slog.Info("Background job completed", "job", jobName, "duration_ms", duration.Milliseconds())
	return nil
}
