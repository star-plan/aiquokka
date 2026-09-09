package credential

import "fmt"

// RefreshFuncs are the provider-supplied actions Apply may invoke after
// validating Policy against Capabilities.
type RefreshFuncs struct {
	// OfficialCLI runs the official CLI refresh path (optional).
	OfficialCLI func() error
	// InMemory refreshes credentials in process memory only (optional).
	InMemory func() error
	// Persist refreshes credentials and atomically writes them back (optional).
	Persist func() error
}

// Apply selects a refresh strategy for policy after validating capabilities.
//
// Recommended call site after a failed / expired credential check:
//
//  1. If caps.OfficialCLIRefresh and OfficialCLI is set, try it first and
//     reload credentials — even under ReadOnly.
//  2. If still unusable, call Apply with the user Policy.
//
// ReadOnly returns ErrRefreshRequired (wrapped) without calling InMemory/Persist.
func Apply(provider string, policy Policy, caps Capabilities, fns RefreshFuncs) error {
	if err := caps.Allows(policy); err != nil {
		return &PolicyError{Provider: provider, Policy: policy, Err: err}
	}

	switch policy {
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

// TryOfficialRefresh runs OfficialCLI when the capability is declared and the
// func is non-nil. It is safe to call under any Policy, including ReadOnly.
func TryOfficialRefresh(caps Capabilities, officialCLI func() error) error {
	if !caps.OfficialCLIRefresh || officialCLI == nil {
		return fmt.Errorf("official CLI refresh not available")
	}
	return officialCLI()
}
