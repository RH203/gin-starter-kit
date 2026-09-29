package queue

import (
	"context"
	"time"
)

// SyncDriver executes jobs immediately in the calling goroutine (ideal for unit testing)
type SyncDriver struct {
	registry *Registry
}

// NewSyncDriver creates a synchronous queue driver
func NewSyncDriver(reg *Registry) *SyncDriver {
	if reg == nil {
		reg = NewRegistry()
	}
	return &SyncDriver{registry: reg}
}

func (s *SyncDriver) Dispatch(ctx context.Context, name string, payload interface{}) error {
	raw, err := MarshalPayload(payload)
	if err != nil {
		return err
	}
	return ExecuteJob(ctx, s.registry, name, raw)
}

func (s *SyncDriver) DispatchDelayed(ctx context.Context, name string, payload interface{}, delay time.Duration) error {
	// For sync driver, delay is simulated with time.Sleep if non-zero
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return s.Dispatch(ctx, name, payload)
}

func (s *SyncDriver) RegisterHandler(name string, handler Handler) {
	s.registry.Register(name, handler)
}

func (s *SyncDriver) Close() error {
	return nil
}

// Start for SyncDriver is a no-op since jobs are executed upon dispatch
func (s *SyncDriver) Start(ctx context.Context) error {
	return nil
}

// Stop for SyncDriver is a no-op
func (s *SyncDriver) Stop(timeout time.Duration) error {
	return nil
}
