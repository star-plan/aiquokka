// Package cmd wires up the aiquokka cobra command tree.
package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/star-plan/aiquokka/internal/buildinfo"
	"github.com/star-plan/aiquokka/internal/credential"
)

// Global output-format flags: emit raw structured output instead of the
// rendered bars.
var (
	jsonOut          bool
	yamlOut          bool
	watch            bool
	listProvidersFlg bool
	credentialPolicy = credential.DefaultPolicy
)

// watchInterval is how long --watch waits between refreshes.
const watchInterval = 60 * time.Second

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "aiquokka",
		Short:   "Check AI coding-assistant usage limits",
		Version: buildinfo.Version,
		Long: `aiquokka reports how much of each AI coding subscription is still left.

  aiquokka          all providers at once
  aiquokka --list   list available providers
  aiquokka <name>   one provider (see --list)

  --watch                  refresh every 60s; press r to refresh now, q/Ctrl+C to stop
  --credential-policy      auto (default) | readonly | memory | persist

  bars:
    █ remaining quota (green / yellow / red)
    ░ used (dim)
    cyan marker = where even pace would leave you now
    percentages are remaining (left); --json/--yaml still report used_percent`,
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if listProvidersFlg {
				return listProviders(cmd.OutOrStdout())
			}
			return runAll(allProviders())
		},
	}
	root.SetVersionTemplate(versionOutput())

	root.Flags().BoolVar(&listProvidersFlg, "list", false, "list available providers")
	root.PersistentFlags().BoolVar(&jsonOut, "json", false, "emit raw JSON instead of rendered output")
	root.PersistentFlags().BoolVar(&yamlOut, "yaml", false, "emit raw YAML instead of rendered output")
	root.PersistentFlags().BoolVar(&yamlOut, "yml", false, "alias for --yaml")
	root.PersistentFlags().BoolVarP(&watch, "watch", "w", false, "refresh every 60s (r refresh, q close)")
	root.PersistentFlags().Var(newPolicyValue(&credentialPolicy), "credential-policy", "credential refresh policy: auto, readonly, memory, or persist")
	root.MarkFlagsMutuallyExclusive("json", "yaml")
	root.MarkFlagsMutuallyExclusive("json", "yml")

	root.AddCommand(newListCmd())
	for _, p := range registry.All() {
		root.AddCommand(newProviderCmd(p))
	}
	return root
}

func versionOutput() string {
	if buildinfo.Version == "dev" {
		return "aiquokka dev\n"
	}
	return "aiquokka {{.Version}}\ncommit " + buildinfo.Commit + "\nbuilt " + buildDate() + "\n"
}

func buildDate() string {
	date := buildinfo.Date
	if len(date) >= len("2006-01-02") && date[4] == '-' && date[7] == '-' {
		return date[:len("2006-01-02")]
	}
	return date
}

// Execute runs the root command.
func Execute() error {
	return newRootCmd().Execute()
}

// policyValue adapts credential.Policy to pflag.Value.
type policyValue struct {
	dst *credential.Policy
}

func newPolicyValue(dst *credential.Policy) *policyValue {
	return &policyValue{dst: dst}
}

func (v *policyValue) String() string {
	if v.dst == nil {
		return credential.DefaultPolicy.String()
	}
	return v.dst.String()
}

func (v *policyValue) Set(s string) error {
	p, err := credential.ParsePolicy(s)
	if err != nil {
		return err
	}
	*v.dst = p
	return nil
}

func (v *policyValue) Type() string { return "policy" }
