package crtsh

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tlgakkoca-cloud/aegis-ai-asm/internal/discovery"
)

const endpoint = "https://crt.sh/"

// Provider implements subdomain enumeration using the public crt.sh API.
type Provider struct {
	client *http.Client
}

// NewProvider builds a crt.sh provider with an optional custom http.Client.
func NewProvider(client *http.Client) *Provider {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &Provider{client: client}
}

func (p *Provider) Name() string {
	return "crtsh"
}

// Discover queries crt.sh for subdomains that match the provided domain.
func (p *Provider) Discover(ctx context.Context, domain string) ([]discovery.Asset, error) {
	if strings.TrimSpace(domain) == "" {
		return nil, errors.New("domain is required")
	}

	query := url.Values{}
	query.Set("q", fmt.Sprintf("%%25.%s", domain))
	query.Set("output", "json")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Aegis-ASM/0.1 (+github.com/tlgakkoca-cloud/aegis-ai-asm)")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("crt.sh returned status %d", resp.StatusCode)
	}

	var entries []struct {
		NameValue string `json:"name_value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	var assets []discovery.Asset

	for _, entry := range entries {
		names := strings.Split(entry.NameValue, "\n")
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if !strings.HasSuffix(name, domain) {
				continue
			}
			name = strings.TrimSuffix(name, ".")
			lower := strings.ToLower(name)
			if _, ok := seen[lower]; ok {
				continue
			}
			seen[lower] = struct{}{}

			assets = append(assets, discovery.Asset{
				ID:        fmt.Sprintf("crtsh-%s", lower),
				Domain:    lower,
				Kind:      "dns",
				Owner:     "unknown",
				RiskScore: 0,
				Metadata: map[string]string{
					"source": "crt.sh",
				},
			})
		}
	}

	return assets, nil
}
