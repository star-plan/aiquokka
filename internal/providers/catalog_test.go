package providers

import (
	"testing"

	"github.com/McKean/aiquokka/internal/credential"
	"github.com/McKean/aiquokka/internal/provider"
)

func TestCatalogIDsUniqueAndStable(t *testing.T) {
	reg := Registry()
	if reg.Len() == 0 {
		t.Fatal("catalog is empty")
	}

	seen := map[string]bool{}
	for _, p := range reg.All() {
		id := p.ID()
		if id == "" {
			t.Fatalf("provider %q has empty ID", p.Name())
		}
		if seen[id] {
			t.Fatalf("duplicate provider ID %q", id)
		}
		seen[id] = true
		if p.Name() == "" {
			t.Fatalf("provider %q has empty Name", id)
		}
		if p.Description() == "" {
			t.Fatalf("provider %q has empty Description", id)
		}
	}

	if _, ok := reg.Lookup("agy"); !ok {
		t.Fatal("antigravity alias agy not found")
	}
	if p, ok := reg.ByID("grok"); !ok || p.Name() != "Grok" {
		t.Fatalf("ByID(grok) = %v, %v", p, ok)
	}
	if p, ok := reg.ByID("cursor"); !ok || p.Name() != "Cursor" {
		t.Fatalf("ByID(cursor) = %v, %v", p, ok)
	}
}

func TestGrokRejectsInMemoryPolicy(t *testing.T) {
	p, ok := Registry().ByID("grok")
	if !ok {
		t.Fatal("grok not registered")
	}
	caps := provider.CapabilitiesOf(p)
	if !caps.RotatesRefreshToken {
		t.Fatal("Grok should declare RotatesRefreshToken")
	}
	if err := caps.Allows(credential.ReadOnly); err != nil {
		t.Fatalf("ReadOnly should be allowed: %v", err)
	}
	if err := caps.Allows(credential.RefreshInMemory); err == nil {
		t.Fatal("RefreshInMemory must be rejected when refresh tokens rotate")
	}
	if err := caps.Allows(credential.RefreshAndPersist); err != nil {
		t.Fatalf("RefreshAndPersist should be allowed: %v", err)
	}
}

func TestCursorRejectsInMemoryPolicy(t *testing.T) {
	p, ok := Registry().ByID("cursor")
	if !ok {
		t.Fatal("cursor not registered")
	}
	caps := provider.CapabilitiesOf(p)
	if err := caps.Allows(credential.RefreshInMemory); err == nil {
		t.Fatal("RefreshInMemory must be rejected when refresh tokens rotate")
	}
}
