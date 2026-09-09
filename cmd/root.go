// Package cmd wires up the aiquokka cobra command tree.
package cmd

import (
	"time"

	"github.com/spf13/cobra"
)

// Global output-format flags: emit raw structured output instead of the
// rendered bars.
var (
	jsonOut bool
	yamlOut bool
	watch   bool
)

// watchInterval is how long --watch waits between refreshes.
const watchInterval = 60 * time.Second

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "aiquokka",
		Short: "Check AI coding-assistant usage limits",
		Long: `aiquokka reports the usage limits of your AI coding subscriptions.

  aiquokka          all providers at once
  aiquokka claude   5-hour and weekly limits
  aiquokka codex    weekly limit and reset info
  aiquokka kimi     5-hour and weekly limits
  aiquokka grok     weekly usage limit
  aiquokka copilot  copilot chat/completions limits
  aiquokka deepseek  account balance
  aiquokka kiro     Kiro CLI monthly credits and overage status
  aiquokka agy      daily antigravity limits
  aiquokka zai      Z.ai usage bundles and cash balance

  --watch           refresh every 60s; press r to refresh now, q/Ctrl+C to stop`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAll(allProviders())
		},
	}
	root.PersistentFlags().BoolVar(&jsonOut, "json", false, "emit raw JSON instead of rendered output")
	root.PersistentFlags().BoolVar(&yamlOut, "yaml", false, "emit raw YAML instead of rendered output")
	root.PersistentFlags().BoolVar(&yamlOut, "yml", false, "alias for --yaml")
	root.PersistentFlags().BoolVarP(&watch, "watch", "w", false, "refresh every 60s (r refresh, q close)")
	root.MarkFlagsMutuallyExclusive("json", "yaml")
	root.MarkFlagsMutuallyExclusive("json", "yml")

	for _, provider := range providerCommands {
		root.AddCommand(newProviderCmd(provider))
	}
	return root
}

// Execute runs the root command.
func Execute() error {
	return newRootCmd().Execute()
}
