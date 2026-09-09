package kiro

import (
	"context"

	"github.com/McKean/aiquokka/internal/credential"
	"github.com/McKean/aiquokka/internal/usage"
)

// Provider adapts the Kiro fetcher to the shared provider.Provider contract.
// Kiro keeps official-CLI credential ownership: aiquokka only invokes kiro-cli
// and parses its output.
type Provider struct{}

// New returns the Kiro provider.
func New() *Provider { return &Provider{} }

func (*Provider) ID() string          { return "kiro" }
func (*Provider) Name() string        { return "Kiro" }
func (*Provider) Description() string { return "Check Kiro CLI credits and usage limits" }
func (*Provider) Fetch(ctx context.Context) (*usage.Report, error) {
	return Fetch(ctx)
}

// Capabilities reports that Kiro refreshes only via its official CLI.
func (*Provider) Capabilities() credential.Capabilities {
	return credential.Capabilities{
		OfficialCLIRefresh: true,
	}
}
