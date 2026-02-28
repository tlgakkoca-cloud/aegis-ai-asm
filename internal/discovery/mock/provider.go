package mock

import (
	"context"
	"math/rand"
	"time"

	"github.com/tlgakkoca-cloud/aegis-ai-asm/internal/discovery"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Provider is a mock implementation returning static assets useful for smoke tests.
type Provider struct {
	Assets []discovery.Asset
}

func NewProvider() *Provider {
	return &Provider{
		Assets: []discovery.Asset{
			{
				ID:        "asset-web-001",
				Domain:    "example.com",
				Kind:      "web",
				Owner:     "team-alpha",
				RiskScore: 7.5,
				Metadata: map[string]string{
					"region": "us-east-1",
				},
			},
			{
				ID:        "asset-db-001",
				Domain:    "example.com",
				Kind:      "database",
				Owner:     "team-beta",
				RiskScore: 4.2,
				Metadata: map[string]string{
					"engine": "aurora",
				},
			},
		},
	}
}

func (p *Provider) Discover(ctx context.Context, domain string) ([]discovery.Asset, error) {
	select {
	case <-time.After(100 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	assets := make([]discovery.Asset, len(p.Assets))
	copy(assets, p.Assets)

	for i := range assets {
		assets[i].Domain = domain
		assets[i].RiskScore = assets[i].RiskScore + randFloatDelta()
	}

	return assets, nil
}

func (p *Provider) Name() string {
	return "mock-provider"
}

func randFloatDelta() float64 {
	return rand.Float64()*0.4 - 0.2
}
