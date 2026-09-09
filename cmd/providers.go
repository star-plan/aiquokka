package cmd

import (
	"github.com/McKean/aiquokka/internal/providers/antigravity"
	"github.com/McKean/aiquokka/internal/providers/claude"
	"github.com/McKean/aiquokka/internal/providers/codex"
	"github.com/McKean/aiquokka/internal/providers/copilot"
	"github.com/McKean/aiquokka/internal/providers/deepseek"
	"github.com/McKean/aiquokka/internal/providers/grok"
	"github.com/McKean/aiquokka/internal/providers/kimi"
	"github.com/McKean/aiquokka/internal/providers/kiro"
	"github.com/McKean/aiquokka/internal/providers/zai"
	"github.com/spf13/cobra"
)

// providerCommand describes one provider's CLI command and aggregate entry.
// Keeping this registry together ensures both views use the same provider set.
type providerCommand struct {
	name    string
	use     string
	aliases []string
	short   string
	fetch   fetcher
}

var providerCommands = []providerCommand{
	{name: "Claude", use: "claude", short: "Claude 5-hour and weekly usage limits", fetch: claude.Fetch},
	{name: "Codex", use: "codex", short: "Codex weekly usage limit and reset info", fetch: codex.Fetch},
	{name: "Kimi", use: "kimi", short: "Kimi 5-hour and weekly limits", fetch: kimi.Fetch},
	{name: "Grok", use: "grok", short: "Grok usage limits", fetch: grok.Fetch},
	{name: "Copilot", use: "copilot", short: "Check GitHub Copilot usage limits", fetch: copilot.Fetch},
	{name: "DeepSeek", use: "deepseek", short: "DeepSeek account balance", fetch: deepseek.Fetch},
	{name: "Kiro", use: "kiro", short: "Check Kiro CLI credits and usage limits", fetch: kiro.Fetch},
	{name: "Antigravity", use: "antigravity", aliases: []string{"agy"}, short: "Check Google Antigravity usage limits", fetch: antigravity.Fetch},
	{name: "Z.ai", use: "zai", short: "Z.ai usage bundles and cash balance", fetch: zai.Fetch},
}

// allProviders returns the set rendered when aiquokka is run without a
// subcommand. Build a new slice so callers cannot mutate the registry.
func allProviders() []provider {
	providers := make([]provider, len(providerCommands))
	for i, command := range providerCommands {
		providers[i] = provider{name: command.name, fetch: command.fetch}
	}
	return providers
}

func newProviderCmd(provider providerCommand) *cobra.Command {
	return &cobra.Command{
		Use:     provider.use,
		Aliases: provider.aliases,
		Short:   provider.short,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(provider.fetch)
		},
	}
}
