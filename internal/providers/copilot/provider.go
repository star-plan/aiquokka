package copilot

import (
	"context"

	"github.com/McKean/aiquokka/internal/usage"
)

// Provider adapts the Copilot fetcher to the shared provider.Provider contract.
type Provider struct{}

// New returns the Copilot provider.
func New() *Provider { return &Provider{} }

func (*Provider) ID() string          { return "copilot" }
func (*Provider) Name() string        { return "Copilot" }
func (*Provider) Description() string { return "Check GitHub Copilot usage limits" }
func (*Provider) Fetch(ctx context.Context) (*usage.Report, error) {
	return Fetch(ctx)
}
