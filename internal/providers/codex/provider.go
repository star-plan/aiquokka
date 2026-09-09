package codex

import (
	"context"

	"github.com/star-plan/aiquokka/internal/credential"
	"github.com/star-plan/aiquokka/internal/usage"
)

// Provider adapts the Codex fetcher to the shared provider.Provider contract.
type Provider struct{}

// New returns the Codex provider.
func New() *Provider { return &Provider{} }

func (*Provider) ID() string          { return "codex" }
func (*Provider) Name() string        { return "Codex" }
func (*Provider) Description() string { return "Codex weekly usage limit and reset info" }
func (*Provider) Fetch(ctx context.Context) (*usage.Report, error) {
	return Fetch(ctx)
}

// Capabilities declares Codex's credential behaviour. Token refresh may rotate
// the refresh token, so in-memory-only refresh is never safe.
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
