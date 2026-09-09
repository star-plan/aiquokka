package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/McKean/aiquokka/internal/credential"
)

func TestProviderRegistryBuildsAggregateAndCommands(t *testing.T) {
	providers := allProviders()
	if len(providers) != registry.Len() {
		t.Fatalf("allProviders length = %d, want %d", len(providers), registry.Len())
	}

	root := newRootCmd()
	for i, p := range registry.All() {
		if providers[i].Name() != p.Name() {
			t.Errorf("provider %d name = %q, want %q", i, providers[i].Name(), p.Name())
		}
		if providers[i].ID() != p.ID() {
			t.Errorf("provider %d id = %q, want %q", i, providers[i].ID(), p.ID())
		}

		command, _, err := root.Find([]string{p.ID()})
		if err != nil {
			t.Fatalf("find command %q: %v", p.ID(), err)
		}
		if command.Name() != p.ID() {
			t.Errorf("command name = %q, want %q", command.Name(), p.ID())
		}
		if command.Short != p.Description() {
			t.Errorf("command %q short = %q, want %q", p.ID(), command.Short, p.Description())
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

func TestListProviders(t *testing.T) {
	var buf bytes.Buffer
	if err := listProviders(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"claude", "codex", "grok", "antigravity (agy)"} {
		if !strings.Contains(out, want) {
			t.Errorf("list output missing %q:\n%s", want, out)
		}
	}
}

func TestCredentialPolicyFlag(t *testing.T) {
	t.Cleanup(func() { credentialPolicy = credential.DefaultPolicy })
	credentialPolicy = credential.DefaultPolicy

	root := newRootCmd()
	root.SetArgs([]string{"--credential-policy", "persist", "--list"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if credentialPolicy.String() != "persist" {
		t.Fatalf("policy = %s, want persist", credentialPolicy)
	}

	root = newRootCmd()
	root.SetArgs([]string{"--credential-policy", "memory", "--list"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if credentialPolicy.String() != "memory" {
		t.Fatalf("policy = %s, want memory", credentialPolicy)
	}
}
