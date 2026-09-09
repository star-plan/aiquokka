package kimi

import (
	"context"

	"github.com/star-plan/aiquokka/internal/usage"
)

// Provider adapts the Kimi fetcher to the shared provider.Provider contract.
type Provider struct{}

// New returns the Kimi provider.
func New() *Provider { return &Provider{} }

func (*Provider) ID() string          { return "kimi" }
func (*Provider) Name() string        { return "Kimi" }
func (*Provider) Description() string { return "Kimi 5-hour and weekly limits" }
func (*Provider) Fetch(ctx context.Context) (*usage.Report, error) {
	return Fetch(ctx)
}
