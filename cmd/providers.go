package cmd

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/McKean/aiquokka/internal/provider"
	"github.com/McKean/aiquokka/internal/providers"
	"github.com/spf13/cobra"
)

// registry is the process-wide provider catalog. Built once; CLI commands and
// the aggregate view both read from it.
var registry = providers.Registry()

// allProviders returns the set rendered when aiquokka is run without a
// subcommand. Build a new slice so callers cannot mutate the registry.
func allProviders() []provider.Provider {
	src := registry.All()
	out := make([]provider.Provider, len(src))
	copy(out, src)
	return out
}

func newProviderCmd(p provider.Provider) *cobra.Command {
	return &cobra.Command{
		Use:     p.ID(),
		Aliases: provider.AliasesOf(p),
		Short:   p.Description(),
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(p)
		},
	}
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available providers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listProviders(cmd.OutOrStdout())
		},
	}
}

func listProviders(w io.Writer) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, p := range registry.All() {
		id := p.ID()
		if aliases := provider.AliasesOf(p); len(aliases) > 0 {
			id = fmt.Sprintf("%s (%s)", id, aliases[0])
		}
		fmt.Fprintf(tw, "%s\t%s\n", id, p.Description())
	}
	return tw.Flush()
}
