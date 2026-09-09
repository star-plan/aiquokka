// Package provider defines the shared Provider contract and registry used by
// the CLI, aggregate queries, and --list.
package provider

import (
	"context"

	"github.com/star-plan/aiquokka/internal/credential"
	"github.com/star-plan/aiquokka/internal/usage"
)

// Provider is one quota source. Implementations are registered once in
// internal/providers/catalog.go; the CLI builds commands and aggregate views
// from that single list.
type Provider interface {
	// ID is the stable CLI subcommand name, e.g. "claude", "grok".
	ID() string
	// Name is the display name shown in rendered output, e.g. "Claude".
	Name() string
	// Description is the short help text for the provider's subcommand.
	Description() string
	// Fetch returns the current usage report for this provider.
	Fetch(ctx context.Context) (*usage.Report, error)
}

// Aliaser is optionally implemented by providers that expose CLI aliases
// (e.g. "agy" for antigravity).
type Aliaser interface {
	Aliases() []string
}

// CredentialAware is optionally implemented by providers that refresh or
// persist credentials. The registry and CLI use it to validate the user's
// credential policy against what the provider safely supports.
type CredentialAware interface {
	Capabilities() credential.Capabilities
}

// AliasesOf returns p.Aliases() when p implements Aliaser, otherwise nil.
func AliasesOf(p Provider) []string {
	if a, ok := p.(Aliaser); ok {
		return a.Aliases()
	}
	return nil
}

// CapabilitiesOf returns p.Capabilities() when p implements CredentialAware,
// otherwise the zero value (no refresh capabilities).
func CapabilitiesOf(p Provider) credential.Capabilities {
	if c, ok := p.(CredentialAware); ok {
		return c.Capabilities()
	}
	return credential.Capabilities{}
}
