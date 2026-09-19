package worker_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"gin-starter-pack/pkg/worker"
)

type testJob struct {
	counter *int32
}

func (j *testJob) Name() string {
	return "testJob"
}

func (j *testJob) Execute(ctx context.Context) error {
	atomic.AddInt32(j.counter, 1)
	return nil
}

func TestWorkerPool_DispatchAndExecute(t *testing.T) {
	pool := worker.NewPool(3, 10)
	pool.Start()

	var counter int32
	for i := 0; i < 5; i++ {
		ok := pool.Dispatch(&testJob{counter: &counter})
		if !ok {
			t.Errorf("failed to dispatch job %d", i)
		}
	}

	// Give workers time to process
	time.Sleep(100 * time.Millisecond)

	pool.Stop(1 * time.Second)

	if atomic.LoadInt32(&counter) != 5 {
		t.Errorf("expected 5 jobs processed, got %d", counter)
	}
}
