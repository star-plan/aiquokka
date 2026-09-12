// Package credential defines the credential ownership policy that controls
// whether aiquokka may refresh or persist provider credentials.
//
// Default posture: aiquokka is a credential consumer. Official CLIs / provider
// clients own identity; aiquokka only reads credentials unless the user
// explicitly opts into a more permissive policy that the provider supports.
package credential

import (
	"fmt"
	"strings"
)

// Policy controls how aiquokka may handle expired or missing credentials.
type Policy uint8

const (
	// Auto uses the safest refresh mechanism the provider explicitly supports.
	// It is intended for normal interactive use: a provider with rotating
	// refresh tokens will select RefreshAndPersist, never an unsafe memory-only
	// refresh.
	Auto Policy = iota
	// ReadOnly means aiquokka must not refresh or write credentials itself.
	// The command layer may still tell the user how to reauthenticate through
	// the provider's official CLI.
	ReadOnly Policy = iota
	// RefreshInMemory means aiquokka may refresh tokens in process memory but
	// must not write them back to the credential store.
	RefreshInMemory
	// RefreshAndPersist means aiquokka may refresh tokens and atomically write
	// them back to the credential store.
	RefreshAndPersist
)

// DefaultPolicy silently recovers a credential only when the provider has
// declared that recovery safe.
const DefaultPolicy = Auto

// String returns the canonical CLI name for the policy.
func (p Policy) String() string {
	switch p {
	case Auto:
		return "auto"
	case ReadOnly:
		return "readonly"
	case RefreshInMemory:
		return "memory"
	case RefreshAndPersist:
		return "persist"
	default:
		return fmt.Sprintf("policy(%d)", uint8(p))
	}
}

// ParsePolicy parses a CLI --credential-policy value.
func ParsePolicy(s string) (Policy, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "auto":
		return Auto, nil
	case "readonly", "read-only", "read":
		return ReadOnly, nil
	case "memory", "in-memory", "inmemory", "refresh-in-memory":
		return RefreshInMemory, nil
	case "persist", "refresh-and-persist", "write":
		return RefreshAndPersist, nil
	default:
		return 0, fmt.Errorf("unknown credential policy %q (want auto, readonly, memory, or persist)", s)
	}
}
