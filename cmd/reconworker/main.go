package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tlgakkoca-cloud/aegis-ai-asm/internal/config"
	"github.com/tlgakkoca-cloud/aegis-ai-asm/internal/discovery"
	crtshprovider "github.com/tlgakkoca-cloud/aegis-ai-asm/internal/discovery/crtsh"
	"github.com/tlgakkoca-cloud/aegis-ai-asm/internal/logger"
	"github.com/tlgakkoca-cloud/aegis-ai-asm/internal/worker"
	"go.uber.org/zap"
)

type reconHandler struct {
	engine *discovery.Engine
	log    *zap.Logger
}

func (h *reconHandler) Handle(ctx context.Context, job worker.Job) (worker.Result, error) {
	assets, err := h.engine.DiscoverAll(ctx, job.Domain)
	if err != nil {
		return worker.Result{}, err
	}

	output := map[string]any{
		"asset_count": len(assets),
		"assets":      assets,
	}
	if len(assets) > 0 {
		output["first_asset"] = assets[0]
	}
	if sub := firstSubdomain(assets, job.Domain); sub != nil {
		output["first_subdomain"] = *sub
	}

	h.log.Info("job completed",
		zap.String("job_id", job.ID),
		zap.Int("asset_count", len(assets)),
	)

	return worker.Result{Output: output, Status: worker.StatusSucceeded}, nil
}

func firstSubdomain(assets []discovery.Asset, root string) *discovery.Asset {
	root = strings.TrimSuffix(strings.ToLower(root), ".")
	if root == "" {
		return nil
	}
	for _, asset := range assets {
		domain := strings.TrimSuffix(strings.ToLower(asset.Domain), ".")
		if domain == root {
			continue
		}
		if strings.HasSuffix(domain, "."+root) {
			copy := asset
			return &copy
		}
	}
	return nil
}

func main() {
	cfg := config.MustLoad(".env")

	log, err := logger.New(logger.Options{Level: cfg.LogLevel})
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	handler := &reconHandler{
		engine: discovery.NewEngine(
			crtshprovider.NewProvider(nil),
		),
		log: log,
	}

	ctx := context.Background()
	pool, err := worker.NewPool(ctx, worker.PoolConfig{Concurrency: 1}, handler)
	if err != nil {
		log.Fatal("failed to start worker pool", zap.Error(err))
	}
	defer pool.Shutdown(context.Background())

	job := worker.Job{
		ID:        fmt.Sprintf("job-%d", time.Now().UnixNano()),
		TenantID:  "demo",
		Domain:    cfg.TargetDomain,
		Type:      worker.JobTypeRecon,
		CreatedAt: time.Now().UTC(),
	}

	if err := pool.Submit(ctx, job); err != nil {
		log.Fatal("failed to submit job", zap.Error(err))
	}

	select {
	case res := <-pool.Results():
		if res.Error != nil {
			log.Fatal("worker failed", zap.Error(res.Error))
		}

		if sub, ok := res.Output["first_subdomain"].(discovery.Asset); ok {
			log.Info("first subdomain detected", zap.String("domain", sub.Domain))
			fmt.Printf("FIRST_SUBDOMAIN=%s\n", sub.Domain)
			return
		}

		if first, ok := res.Output["first_asset"].(discovery.Asset); ok {
			log.Info("only apex detected", zap.String("domain", first.Domain))
			fmt.Printf("FIRST_ASSET=%s\n", first.Domain)
		} else {
			log.Warn("no assets discovered")
		}
	case <-time.After(30 * time.Second):
		log.Fatal("timeout waiting for worker result")
	}
}
