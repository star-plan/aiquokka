package provider

import (
	"fmt"
	"strings"
)

// Registry is an ordered, explicit list of providers. It is the single source
// of truth for CLI subcommands, aggregate queries, and --list.
type Registry struct {
	providers []Provider
	byID      map[string]Provider
}

// New builds a registry from the given providers. Duplicate IDs are rejected.
func New(providers ...Provider) (*Registry, error) {
	r := &Registry{
		providers: make([]Provider, 0, len(providers)),
		byID:      make(map[string]Provider, len(providers)),
	}
	for _, p := range providers {
		if p == nil {
			return nil, fmt.Errorf("provider: nil entry")
		}
		id := strings.TrimSpace(p.ID())
		if id == "" {
			return nil, fmt.Errorf("provider %q: empty ID", p.Name())
		}
		if _, exists := r.byID[id]; exists {
			return nil, fmt.Errorf("provider: duplicate ID %q", id)
		}
		r.providers = append(r.providers, p)
		r.byID[id] = p
	}
	return r, nil
}

// Must is like New but panics on error. Intended for the static catalog.
func Must(providers ...Provider) *Registry {
	r, err := New(providers...)
	if err != nil {
		panic(err)
	}
	return r
}

// All returns providers in registration order. The returned slice must not be
// modified by callers.
func (r *Registry) All() []Provider {
	return r.providers
}

// Len returns the number of registered providers.
func (r *Registry) Len() int {
	return len(r.providers)
}

// ByID looks up a provider by its stable ID (subcommand name).
func (r *Registry) ByID(id string) (Provider, bool) {
	p, ok := r.byID[id]
	return p, ok
}

// Lookup finds a provider by ID or alias (case-sensitive).
func (r *Registry) Lookup(name string) (Provider, bool) {
	if p, ok := r.byID[name]; ok {
		return p, true
	}
	for _, p := range r.providers {
		for _, alias := range AliasesOf(p) {
			if alias == name {
				return p, true
			}
		}
	}
	return nil, false
}
