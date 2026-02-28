package main

import (
	"context"
	"time"

	"github.com/kullanici-adin/aegis-ai-asm/internal/config"
	"github.com/kullanici-adin/aegis-ai-asm/internal/discovery"
	crtshprovider "github.com/kullanici-adin/aegis-ai-asm/internal/discovery/crtsh"
	discoverymock "github.com/kullanici-adin/aegis-ai-asm/internal/discovery/mock"
	"github.com/kullanici-adin/aegis-ai-asm/internal/logger"
	"github.com/kullanici-adin/aegis-ai-asm/internal/reporter"
	"go.uber.org/zap"
)

func main() {
	cfg := config.MustLoad(".env")

	log, err := logger.New(logger.Options{Level: cfg.LogLevel})
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	log = log.With(
		zap.String("env", cfg.AppEnv),
		zap.String("target_domain", cfg.TargetDomain),
	)

	providers := []discovery.Provider{
		discoverymock.NewProvider(),
		crtshprovider.NewProvider(nil),
	}

	engine := discovery.NewEngine(providers...)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	assets, err := engine.DiscoverAll(ctx, cfg.TargetDomain)
	if err != nil {
		log.Error("one or more discovery providers failed", zap.Error(err))
	}

	log.Info("discovery run complete", zap.Int("asset_count", len(assets)))

	rep, err := reporter.New("outputs")
	if err != nil {
		log.Fatal("failed to init reporter", zap.Error(err))
	}

	jsonPath, err := rep.WriteJSON(assets)
	if err != nil {
		log.Fatal("failed to write json report", zap.Error(err))
	}
	csvPath, err := rep.WriteCSV(assets)
	if err != nil {
		log.Fatal("failed to write csv report", zap.Error(err))
	}

	log.Info("reports written",
		zap.String("json", jsonPath),
		zap.String("csv", csvPath),
	)
}
