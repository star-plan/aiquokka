package grok

import (
	"context"

	"github.com/star-plan/aiquokka/internal/credential"
	"github.com/star-plan/aiquokka/internal/usage"
)

// Provider adapts the Grok fetcher to the shared provider.Provider contract.
type Provider struct{}

// New returns the Grok provider.
func New() *Provider { return &Provider{} }

func (*Provider) ID() string          { return "grok" }
func (*Provider) Name() string        { return "Grok" }
func (*Provider) Description() string { return "Grok usage limits" }
func (*Provider) Fetch(ctx context.Context) (*usage.Report, error) {
	return Fetch(ctx)
}

// Capabilities declares Grok's credential behaviour. xAI refresh responses may
// rotate the refresh token, so in-memory-only refresh is never safe.
func (*Provider) Capabilities() credential.Capabilities {
	return capabilities()
}

func capabilities() credential.Capabilities {
	return credential.Capabilities{
		RefreshInMemory:     false,
		RefreshAndPersist:   true,
		RotatesRefreshToken: true,
	}
}
