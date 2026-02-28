package controlplane

import (
	"context"
	"testing"
	"time"

	"github.com/tlgakkoca-cloud/aegis-ai-asm/internal/worker"
)

func TestControlPlaneLifecycle(t *testing.T) {
	cp := New(Config{})

	tenant := Tenant{ID: "tenant-1", Name: "Acme", Plan: "trial"}
	if err := cp.RegisterTenant(tenant); err != nil {
		t.Fatalf("register tenant: %v", err)
	}

	token, err := cp.IssueVerificationToken("tenant-1", "example.com")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	if token == "" {
		t.Fatal("expected token")
	}

	if err := cp.VerifyDomain("tenant-1", "example.com", token); err != nil {
		t.Fatalf("verify domain: %v", err)
	}

	ctx := context.Background()
	job, err := cp.ScheduleScan(ctx, ScanRequest{TenantID: "tenant-1", Domain: "example.com", Type: worker.JobTypeRecon})
	if err != nil {
		t.Fatalf("schedule scan: %v", err)
	}

	if job.TenantID != "tenant-1" || job.Domain != "example.com" {
		t.Fatalf("unexpected job: %+v", job)
	}

	next, err := cp.NextJob(ctx)
	if err != nil {
		t.Fatalf("next job: %v", err)
	}

	if next.ID != job.ID {
		t.Fatalf("expected job %s, got %s", job.ID, next.ID)
	}
}

func TestControlPlaneRateLimit(t *testing.T) {
	cp := New(Config{RateLimitWindow: time.Minute})

	tenant := Tenant{ID: "tenant-2", Name: "Rate", Plan: "trial"}
	_ = cp.RegisterTenant(tenant)
	token, _ := cp.IssueVerificationToken("tenant-2", "rate.com")
	_ = cp.VerifyDomain("tenant-2", "rate.com", token)

	ctx := context.Background()
	if _, err := cp.ScheduleScan(ctx, ScanRequest{TenantID: "tenant-2", Domain: "rate.com"}); err != nil {
		t.Fatalf("first schedule: %v", err)
	}

	if _, err := cp.ScheduleScan(ctx, ScanRequest{TenantID: "tenant-2", Domain: "rate.com"}); err != ErrRateLimited {
		t.Fatalf("expected rate limit, got %v", err)
	}
}

func TestVerifyDomainInvalidToken(t *testing.T) {
	cp := New(Config{})
	tenant := Tenant{ID: "tenant-3", Name: "Token", Plan: "trial"}
	_ = cp.RegisterTenant(tenant)
	_, _ = cp.IssueVerificationToken("tenant-3", "token.com")

	if err := cp.VerifyDomain("tenant-3", "token.com", "wrong"); err != ErrInvalidToken {
		t.Fatalf("expected invalid token error, got %v", err)
	}
}
