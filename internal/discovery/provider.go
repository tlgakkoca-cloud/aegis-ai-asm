package discovery

import "context"

// Asset represents a discovered resource in the attack surface map.
type Asset struct {
	ID        string
	Domain    string
	Kind      string
	Owner     string
	RiskScore float64
	Metadata  map[string]string
}

// Provider is implemented by components capable of discovering assets from a
// specific source (cloud API, DNS, SaaS inventory, etc.).
type Provider interface {
	Discover(ctx context.Context, domain string) ([]Asset, error)
	Name() string
}
