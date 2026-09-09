package claude

import (
	"context"

	"github.com/McKean/aiquokka/internal/usage"
)

// Provider adapts the Claude fetcher to the shared provider.Provider contract.
type Provider struct{}

// New returns the Claude provider.
func New() *Provider { return &Provider{} }

func (*Provider) ID() string          { return "claude" }
func (*Provider) Name() string        { return "Claude" }
func (*Provider) Description() string { return "Claude 5-hour and weekly usage limits" }
func (*Provider) Fetch(ctx context.Context) (*usage.Report, error) {
	return Fetch(ctx)
}
