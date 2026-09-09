package cmd

import "testing"

func TestProviderRegistryBuildsAggregateAndCommands(t *testing.T) {
	providers := allProviders()
	if len(providers) != len(providerCommands) {
		t.Fatalf("allProviders length = %d, want %d", len(providers), len(providerCommands))
	}

	root := newRootCmd()
	for i, entry := range providerCommands {
		if providers[i].name != entry.name {
			t.Errorf("provider %d name = %q, want %q", i, providers[i].name, entry.name)
		}
		if providers[i].fetch == nil {
			t.Errorf("provider %q has no fetcher", entry.name)
		}

		command, _, err := root.Find([]string{entry.use})
		if err != nil {
			t.Fatalf("find command %q: %v", entry.use, err)
		}
		if command.Name() != entry.use {
			t.Errorf("command name = %q, want %q", command.Name(), entry.use)
		}
		if command.Short != entry.short {
			t.Errorf("command %q short = %q, want %q", entry.use, command.Short, entry.short)
		}
	}

	command, _, err := root.Find([]string{"agy"})
	if err != nil {
		t.Fatalf("find antigravity alias: %v", err)
	}
	if command.Name() != "antigravity" {
		t.Errorf("alias resolves to %q, want antigravity", command.Name())
	}
}
