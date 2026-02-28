package discovery

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Engine orchestrates multiple discovery providers concurrently.
type Engine struct {
	providers []Provider
}

// NewEngine constructs an Engine with the provided list of providers.
func NewEngine(providers ...Provider) *Engine {
	return &Engine{providers: providers}
}

// DiscoverAll executes all configured providers concurrently for the given domain
// and merges their results. If one or more providers fail, the combined assets
// are still returned alongside an aggregated error.
func (e *Engine) DiscoverAll(ctx context.Context, domain string) ([]Asset, error) {
	if len(e.providers) == 0 {
		return nil, errors.New("discovery engine has no providers configured")
	}

	var (
		wg    sync.WaitGroup
		resCh = make(chan []Asset, len(e.providers))
		errCh = make(chan error, len(e.providers))
	)

	for _, provider := range e.providers {
		if provider == nil {
			continue
		}
		wg.Add(1)
		go func(p Provider) {
			defer wg.Done()
			assets, err := p.Discover(ctx, domain)
			if err != nil {
				errCh <- fmt.Errorf("%s: %w", p.Name(), err)
				return
			}
			resCh <- assets
		}(provider)
	}

	wg.Wait()
	close(resCh)
	close(errCh)

	var merged []Asset
	for assets := range resCh {
		merged = append(merged, assets...)
	}

	var errs []string
	for err := range errCh {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return merged, errors.New(strings.Join(errs, "; "))
	}

	return merged, nil
}
