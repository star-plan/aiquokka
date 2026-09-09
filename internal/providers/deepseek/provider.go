package deepseek

import (
	"context"

	"github.com/star-plan/aiquokka/internal/usage"
)

// Provider adapts the DeepSeek fetcher to the shared provider.Provider contract.
type Provider struct{}

// New returns the DeepSeek provider.
func New() *Provider { return &Provider{} }

func (*Provider) ID() string          { return "deepseek" }
func (*Provider) Name() string        { return "DeepSeek" }
func (*Provider) Description() string { return "DeepSeek account balance" }
func (*Provider) Fetch(ctx context.Context) (*usage.Report, error) {
	return Fetch(ctx)
}
