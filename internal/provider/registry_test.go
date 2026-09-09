package provider

import (
	"context"
	"testing"

	"github.com/McKean/aiquokka/internal/usage"
)

type stub struct {
	id, name, desc string
	aliases        []string
}

func (s stub) ID() string          { return s.id }
func (s stub) Name() string        { return s.name }
func (s stub) Description() string { return s.desc }
func (s stub) Aliases() []string   { return s.aliases }
func (s stub) Fetch(context.Context) (*usage.Report, error) {
	return &usage.Report{Provider: s.name}, nil
}

func TestRegistryRejectsDuplicates(t *testing.T) {
	_, err := New(stub{id: "a", name: "A", desc: "a"}, stub{id: "a", name: "A2", desc: "a2"})
	if err == nil {
		t.Fatal("expected duplicate ID error")
	}
}

func TestRegistryLookupAlias(t *testing.T) {
	r := Must(
		stub{id: "antigravity", name: "Antigravity", desc: "agy", aliases: []string{"agy"}},
		stub{id: "grok", name: "Grok", desc: "grok"},
	)
	p, ok := r.Lookup("agy")
	if !ok || p.ID() != "antigravity" {
		t.Fatalf("Lookup(agy) = %v, %v", p, ok)
	}
	if _, ok := r.ByID("missing"); ok {
		t.Fatal("ByID(missing) should fail")
	}
}
