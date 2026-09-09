package zai

import (
	"context"

	"github.com/McKean/aiquokka/internal/usage"
)

// Provider adapts the Z.ai fetcher to the shared provider.Provider contract.
type Provider struct{}

// New returns the Z.ai provider.
func New() *Provider { return &Provider{} }

func (*Provider) ID() string          { return "zai" }
func (*Provider) Name() string        { return "Z.ai" }
func (*Provider) Description() string { return "Z.ai usage bundles and cash balance" }
func (*Provider) Fetch(ctx context.Context) (*usage.Report, error) {
	return Fetch(ctx)
}
