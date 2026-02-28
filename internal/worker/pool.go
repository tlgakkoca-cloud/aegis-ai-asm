package worker

import (
	"context"
	"errors"
	"sync"
	"time"
)

// JobType identifies the nature of the work that should be executed.
type JobType string

// Common job types used by the platform.
const (
	JobTypeRecon     JobType = "recon"
	JobTypeChromium  JobType = "chromium-runtime"
	JobTypeDetection JobType = "ai-detection"
	JobTypeSecurity  JobType = "ai-security-test"
)

// DefaultMaxAttempts defines how many retries a job is allowed to have.
const DefaultMaxAttempts = 3

// Job represents a unit of work emitted by the control plane.
type Job struct {
	ID           string
	TenantID     string
	Domain       string
	Type         JobType
	Payload      map[string]any
	Priority     int
	Attempts     int
	MaxAttempts  int
	CreatedAt    time.Time
	ScheduledFor time.Time
}

// Status represents execution outcome for a job.
type Status string

const (
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusRetrying  Status = "retrying"
)

// Result captures the execution outcome emitted by workers.
type Result struct {
	JobID       string
	Status      Status
	Output      map[string]any
	Error       error
	Attempts    int
	StartedAt   time.Time
	CompletedAt time.Time
}

// Handler executes the actual job logic (Recon, Chromium, etc.).
type Handler interface {
	Handle(ctx context.Context, job Job) (Result, error)
}

// PoolConfig configures the worker pool behavior.
type PoolConfig struct {
	Concurrency int
	Buffer      int
	Backoff     time.Duration
}

// Pool manages a set of goroutines that drain the job queue and invoke the handler.
type Pool struct {
	cfg     PoolConfig
	handler Handler

	ctx    context.Context
	cancel context.CancelFunc

	jobs    chan Job
	results chan Result
	wg      sync.WaitGroup
}

// NewPool constructs a worker pool and starts the goroutines immediately.
func NewPool(parent context.Context, cfg PoolConfig, handler Handler) (*Pool, error) {
	if handler == nil {
		return nil, errors.New("worker: handler required")
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 1
	}
	if cfg.Buffer <= 0 {
		cfg.Buffer = cfg.Concurrency * 2
	}

	if parent == nil {
		parent = context.Background()
	}

	ctx, cancel := context.WithCancel(parent)

	pool := &Pool{
		cfg:     cfg,
		handler: handler,
		ctx:     ctx,
		cancel:  cancel,
		jobs:    make(chan Job, cfg.Buffer),
		results: make(chan Result, cfg.Buffer),
	}

	for i := 0; i < cfg.Concurrency; i++ {
		pool.wg.Add(1)
		go pool.loop()
	}

	return pool, nil
}

// Submit enqueues a job for execution.
func (p *Pool) Submit(ctx context.Context, job Job) error {
	if job.MaxAttempts == 0 {
		job.MaxAttempts = DefaultMaxAttempts
	}

	if ctx == nil {
		ctx = context.Background()
	}

	select {
	case <-p.ctx.Done():
		return errors.New("worker: pool stopped")
	case <-ctx.Done():
		return ctx.Err()
	case p.jobs <- job:
		return nil
	}
}

// Results exposes a read-only channel for downstream consumers (eg. Control Plane sink).
func (p *Pool) Results() <-chan Result {
	return p.results
}

// Pending returns the number of jobs currently buffered (best effort).
func (p *Pool) Pending() int {
	return len(p.jobs)
}

// Shutdown attempts to gracefully stop the pool within the provided context timeout.
func (p *Pool) Shutdown(ctx context.Context) error {
	p.cancel()

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
		close(p.results)
	}()

	if ctx == nil {
		ctx = context.Background()
	}

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Pool) loop() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case job := <-p.jobs:
			res := p.execute(job)
			select {
			case p.results <- res:
			case <-p.ctx.Done():
				return
			}
		}
	}
}

func (p *Pool) execute(job Job) Result {
	attempts := job.Attempts + 1
	start := time.Now()

	ctx := p.ctx
	if p.cfg.Backoff > 0 && job.Attempts > 0 {
		timer := time.NewTimer(p.cfg.Backoff)
		select {
		case <-timer.C:
		case <-p.ctx.Done():
			timer.Stop()
			return Result{JobID: job.ID, Status: StatusFailed, Error: errors.New("worker: pool stopped"), Attempts: attempts}
		}
	}

	res, err := p.handler.Handle(ctx, job)
	res.JobID = job.ID
	res.Attempts = attempts
	res.StartedAt = start
	res.CompletedAt = time.Now()

	if err != nil {
		res.Error = err
		if attempts < job.MaxAttempts {
			res.Status = StatusRetrying
			job.Attempts = attempts
			select {
			case <-p.ctx.Done():
			case p.jobs <- job:
			}
		} else {
			res.Status = StatusFailed
		}
		return res
	}

	if res.Status == "" {
		res.Status = StatusSucceeded
	}

	if res.Output == nil {
		res.Output = make(map[string]any)
	}

	return res
}
