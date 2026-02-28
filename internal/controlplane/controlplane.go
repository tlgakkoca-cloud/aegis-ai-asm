package controlplane

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tlgakkoca-cloud/aegis-ai-asm/internal/worker"
)

var (
	// ErrTenantExists is returned when a tenant with the same ID already exists.
	ErrTenantExists = errors.New("controlplane: tenant already exists")
	// ErrTenantNotFound is returned when a tenant lookup fails.
	ErrTenantNotFound = errors.New("controlplane: tenant not found")
	// ErrDomainNotFound indicates that a domain has not been registered for the tenant.
	ErrDomainNotFound = errors.New("controlplane: domain not found")
	// ErrDomainNotVerified indicates that the domain has not completed verification.
	ErrDomainNotVerified = errors.New("controlplane: domain verification pending")
	// ErrInvalidToken is returned when the provided verification token is invalid.
	ErrInvalidToken = errors.New("controlplane: invalid verification token")
	// ErrRateLimited indicates the tenant hit their scheduling rate limit.
	ErrRateLimited = errors.New("controlplane: tenant rate limited")
)

// Config describes runtime options for the in-memory control plane.
type Config struct {
	QueueSize       int
	RateLimitWindow time.Duration
}

// Tenant represents a logical customer in the multi-tenant system.
type Tenant struct {
	ID        string
	Name      string
	Plan      string
	Quota     int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DomainStatus enumerates the lifecycle of a tenant domain.
type DomainStatus string

const (
	DomainPending  DomainStatus = "pending"
	DomainVerified DomainStatus = "verified"
	DomainRevoked  DomainStatus = "revoked"
)

// Domain captures ownership verification metadata.
type Domain struct {
	Name              string
	Status            DomainStatus
	VerificationToken string
	VerifiedAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ScanRequest represents an orchestration request originating from API/UI.
type ScanRequest struct {
	TenantID  string
	Domain    string
	Type      worker.JobType
	Payload   map[string]any
	NotBefore time.Time
}

// ControlPlane provides an in-memory scheduler suitable for early-stage MVPs.
type ControlPlane struct {
	cfg       Config
	mu        sync.RWMutex
	tenants   map[string]*Tenant
	domains   map[string]map[string]*Domain
	lastJobAt map[string]time.Time
	jobQueue  chan worker.Job
}

// New creates a ControlPlane with sane defaults for queue length & rate limiting.
func New(cfg Config) *ControlPlane {
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 256
	}
	if cfg.RateLimitWindow == 0 {
		cfg.RateLimitWindow = 3 * time.Second
	}

	return &ControlPlane{
		cfg:       cfg,
		tenants:   make(map[string]*Tenant),
		domains:   make(map[string]map[string]*Domain),
		lastJobAt: make(map[string]time.Time),
		jobQueue:  make(chan worker.Job, cfg.QueueSize),
	}
}

// RegisterTenant adds a tenant to the control plane registry.
func (cp *ControlPlane) RegisterTenant(t Tenant) error {
	if strings.TrimSpace(t.ID) == "" {
		return errors.New("controlplane: tenant id required")
	}
	if strings.TrimSpace(t.Name) == "" {
		return errors.New("controlplane: tenant name required")
	}

	now := time.Now().UTC()

	cp.mu.Lock()
	defer cp.mu.Unlock()

	if _, exists := cp.tenants[t.ID]; exists {
		return ErrTenantExists
	}

	t.CreatedAt = now
	t.UpdatedAt = now
	cp.tenants[t.ID] = &t

	return nil
}

// IssueVerificationToken creates or updates a domain entry with a new verification token.
func (cp *ControlPlane) IssueVerificationToken(tenantID, domain string) (string, error) {
	domain = normalizeDomain(domain)
	if domain == "" {
		return "", errors.New("controlplane: domain required")
	}

	cp.mu.Lock()
	defer cp.mu.Unlock()

	if _, ok := cp.tenants[tenantID]; !ok {
		return "", ErrTenantNotFound
	}

	token := randomToken()
	now := time.Now().UTC()

	if _, ok := cp.domains[tenantID]; !ok {
		cp.domains[tenantID] = make(map[string]*Domain)
	}

	d, exists := cp.domains[tenantID][domain]
	if !exists {
		d = &Domain{Name: domain, CreatedAt: now}
		cp.domains[tenantID][domain] = d
	}

	d.VerificationToken = token
	d.Status = DomainPending
	d.UpdatedAt = now
	d.VerifiedAt = nil

	return token, nil
}

// VerifyDomain transitions the domain into a verified state if the token matches.
func (cp *ControlPlane) VerifyDomain(tenantID, domain, token string) error {
	domain = normalizeDomain(domain)

	cp.mu.Lock()
	defer cp.mu.Unlock()

	domains, ok := cp.domains[tenantID]
	if !ok {
		return ErrDomainNotFound
	}

	d, ok := domains[domain]
	if !ok {
		return ErrDomainNotFound
	}

	if d.VerificationToken == "" || token != d.VerificationToken {
		return ErrInvalidToken
	}

	now := time.Now().UTC()
	d.Status = DomainVerified
	d.VerificationToken = ""
	d.UpdatedAt = now
	d.VerifiedAt = &now

	return nil
}

// ScheduleScan validates the request and enqueues a worker job.
func (cp *ControlPlane) ScheduleScan(ctx context.Context, req ScanRequest) (worker.Job, error) {
	req.Domain = normalizeDomain(req.Domain)
	if req.Domain == "" {
		return worker.Job{}, errors.New("controlplane: domain required")
	}
	if req.TenantID == "" {
		return worker.Job{}, errors.New("controlplane: tenant id required")
	}

	cp.mu.Lock()
	tenant, ok := cp.tenants[req.TenantID]
	if !ok {
		cp.mu.Unlock()
		return worker.Job{}, ErrTenantNotFound
	}

	domainEntry, ok := cp.domains[req.TenantID][req.Domain]
	if !ok {
		cp.mu.Unlock()
		return worker.Job{}, ErrDomainNotFound
	}
	if domainEntry.Status != DomainVerified {
		cp.mu.Unlock()
		return worker.Job{}, ErrDomainNotVerified
	}

	if window := cp.cfg.RateLimitWindow; window > 0 {
		if last := cp.lastJobAt[req.TenantID]; time.Since(last) < window {
			cp.mu.Unlock()
			return worker.Job{}, ErrRateLimited
		}
		cp.lastJobAt[req.TenantID] = time.Now()
	}
	cp.mu.Unlock()

	job := worker.Job{
		ID:           randomToken(),
		TenantID:     tenant.ID,
		Domain:       req.Domain,
		Type:         req.Type,
		Payload:      clonePayload(req.Payload),
		ScheduledFor: req.NotBefore,
		CreatedAt:    time.Now().UTC(),
		MaxAttempts:  worker.DefaultMaxAttempts,
	}

	if job.Type == "" {
		job.Type = worker.JobTypeRecon
	}
	if job.ScheduledFor.IsZero() {
		job.ScheduledFor = job.CreatedAt
	}

	select {
	case cp.jobQueue <- job:
		return job, nil
	case <-ctx.Done():
		return worker.Job{}, ctx.Err()
	}
}

// NextJob blocks until the next job is available or the context is cancelled.
func (cp *ControlPlane) NextJob(ctx context.Context) (worker.Job, error) {
	select {
	case job := <-cp.jobQueue:
		return job, nil
	case <-ctx.Done():
		return worker.Job{}, ctx.Err()
	}
}

// QueueDepth returns the number of queued jobs (best-effort snapshot).
func (cp *ControlPlane) QueueDepth() int {
	return len(cp.jobQueue)
}

func normalizeDomain(domain string) string {
	domain = strings.TrimSpace(strings.ToLower(domain))
	domain = strings.TrimSuffix(domain, ".")
	return domain
}

func clonePayload(payload map[string]any) map[string]any {
	if payload == nil {
		return make(map[string]any)
	}
	out := make(map[string]any, len(payload))
	for k, v := range payload {
		out[k] = v
	}
	return out
}

func randomToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Errorf("controlplane: token entropy failure: %w", err))
	}
	return hex.EncodeToString(b)
}
