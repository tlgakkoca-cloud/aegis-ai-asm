package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type mockHandler struct {
	fail bool
}

func (m *mockHandler) Handle(ctx context.Context, job Job) (Result, error) {
	if m.fail {
		return Result{}, errors.New("boom")
	}
	return Result{Output: map[string]any{"domain": job.Domain}}, nil
}

func TestPoolExecutesJob(t *testing.T) {
	handler := &mockHandler{}
	pool, err := NewPool(context.Background(), PoolConfig{Concurrency: 2}, handler)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	defer pool.Shutdown(context.Background())

	job := Job{ID: "job-1", TenantID: "t1", Domain: "example.com", Type: JobTypeRecon}
	if err := pool.Submit(context.Background(), job); err != nil {
		t.Fatalf("submit: %v", err)
	}

	select {
	case res := <-pool.Results():
		if res.Status != StatusSucceeded {
			t.Fatalf("expected success, got %s", res.Status)
		}
		if res.JobID != job.ID {
			t.Fatalf("unexpected job id %s", res.JobID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for result")
	}
}

func TestPoolRetriesOnFailure(t *testing.T) {
	var attempts atomic.Int32
	handler := HandlerFunc(func(ctx context.Context, job Job) (Result, error) {
		if attempts.Add(1) < 2 {
			return Result{}, errors.New("fail once")
		}
		return Result{}, nil
	})

	pool, err := NewPool(context.Background(), PoolConfig{Concurrency: 1}, handler)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	defer pool.Shutdown(context.Background())

	job := Job{ID: "job-retry", TenantID: "t", Domain: "retry.com", Type: JobTypeRecon}
	if err := pool.Submit(context.Background(), job); err != nil {
		t.Fatalf("submit: %v", err)
	}

	timeout := time.After(3 * time.Second)
	for {
		select {
		case res := <-pool.Results():
			if res.Status == StatusRetrying {
				continue
			}
			if attempts.Load() != 2 {
				t.Fatalf("expected 2 attempts, got %d", attempts.Load())
			}
			if res.Status != StatusSucceeded {
				t.Fatalf("expected success after retry, got %s", res.Status)
			}
			return
		case <-timeout:
			t.Fatal("timeout waiting for retry result")
		}
	}
}

func TestPoolSubmitContextCancellation(t *testing.T) {
	handler := &mockHandler{}
	pool, _ := NewPool(context.Background(), PoolConfig{Concurrency: 1, Buffer: 1}, handler)
	defer pool.Shutdown(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := pool.Submit(ctx, Job{ID: "job"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

// HandlerFunc helps define inline handlers in tests.
type HandlerFunc func(ctx context.Context, job Job) (Result, error)

// Handle allows HandlerFunc to satisfy the Handler interface.
func (f HandlerFunc) Handle(ctx context.Context, job Job) (Result, error) {
	return f(ctx, job)
}
