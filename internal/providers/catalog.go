// Package providers is the single explicit catalog of quota providers.
//
// Do not use init() self-registration, directory scanning, or Go plugins —
// every provider is listed once here.
package providers

import (
	"github.com/McKean/aiquokka/internal/provider"
	"github.com/McKean/aiquokka/internal/providers/antigravity"
	"github.com/McKean/aiquokka/internal/providers/claude"
	"github.com/McKean/aiquokka/internal/providers/codex"
	"github.com/McKean/aiquokka/internal/providers/copilot"
	"github.com/McKean/aiquokka/internal/providers/deepseek"
	"github.com/McKean/aiquokka/internal/providers/grok"
	"github.com/McKean/aiquokka/internal/providers/kimi"
	"github.com/McKean/aiquokka/internal/providers/kiro"
	"github.com/McKean/aiquokka/internal/providers/zai"
)

// All returns every built-in provider in display / CLI order.
func All() []provider.Provider {
	return []provider.Provider{
		claude.New(),
		codex.New(),
		kimi.New(),
		grok.New(),
		copilot.New(),
		deepseek.New(),
		kiro.New(),
		antigravity.New(),
		zai.New(),
	}
}

// Registry returns the process-wide provider registry built from All().
func Registry() *provider.Registry {
	return provider.Must(All()...)
}
