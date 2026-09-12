package credential

import "fmt"

// RefreshFuncs are the provider-supplied actions Apply may invoke after
// validating Policy against Capabilities.
type RefreshFuncs struct {
	// InMemory refreshes credentials in process memory only (optional).
	InMemory func() error
	// Persist refreshes credentials and atomically writes them back (optional).
	Persist func() error
}

// Apply selects a silent refresh strategy after validating capabilities.
// Interactive reauthentication is intentionally not represented here: callers
// receive ReauthRequiredError and the command layer decides whether a CLI login
// may safely be started.
func Apply(provider string, policy Policy, caps Capabilities, fns RefreshFuncs) error {
	if err := caps.Allows(policy); err != nil {
		return &PolicyError{Provider: provider, Policy: policy, Err: err}
	}

	switch policy {
	case Auto:
		// Prefer persistence whenever it is available. This is required for
		// rotating refresh tokens and is also the most durable recovery after
		// the process exits.
		if caps.RefreshAndPersist {
			if fns.Persist == nil {
				return &PolicyError{Provider: provider, Policy: policy, Err: fmt.Errorf("%w: persisting refresh not implemented", ErrPolicyNotSupported)}
			}
			return fns.Persist()
		}
		if caps.RefreshInMemory && !caps.RotatesRefreshToken {
			if fns.InMemory == nil {
				return &PolicyError{Provider: provider, Policy: policy, Err: fmt.Errorf("%w: in-memory refresh not implemented", ErrPolicyNotSupported)}
			}
			return fns.InMemory()
		}
		return &ReauthRequiredError{Provider: provider, Cause: ErrRefreshRequired}

	case ReadOnly:
		hint := "re-login with the official CLI, or pass --credential-policy persist"
		if caps.RefreshAndPersist {
			hint = "re-login with the official CLI, or pass --credential-policy persist"
		}
		return &RefreshRequiredError{Provider: provider, Hint: hint, Err: ErrRefreshRequired}

	case RefreshInMemory:
		if fns.InMemory == nil {
			return &PolicyError{
				Provider: provider,
				Policy:   policy,
				Err:      fmt.Errorf("%w: in-memory refresh not implemented", ErrPolicyNotSupported),
			}
		}
		return fns.InMemory()

	case RefreshAndPersist:
		if fns.Persist == nil {
			return &PolicyError{
				Provider: provider,
				Policy:   policy,
				Err:      fmt.Errorf("%w: persisting refresh not implemented", ErrPolicyNotSupported),
			}
		}
		return fns.Persist()

	default:
		return &PolicyError{
			Provider: provider,
			Policy:   policy,
			Err:      fmt.Errorf("%w: unknown policy %v", ErrPolicyNotSupported, policy),
		}
	}
}
