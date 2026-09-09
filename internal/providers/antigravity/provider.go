package antigravity

import (
	"context"

	"github.com/star-plan/aiquokka/internal/usage"
)

// Provider adapts the Antigravity fetcher to the shared provider.Provider contract.
type Provider struct{}

// New returns the Antigravity provider.
func New() *Provider { return &Provider{} }

func (*Provider) ID() string          { return "antigravity" }
func (*Provider) Name() string        { return "Antigravity" }
func (*Provider) Description() string { return "Check Google Antigravity usage limits" }
func (*Provider) Aliases() []string   { return []string{"agy"} }
func (*Provider) Fetch(ctx context.Context) (*usage.Report, error) {
	return Fetch(ctx)
}
