package credential

import "fmt"

// Capabilities declares which credential behaviours a provider can safely
// support. The user chooses a Policy; the provider only advertises what is
// safe. A Policy outside Capabilities must error — never silently degrade.
type Capabilities struct {
	// OfficialCLIRefresh means the provider's official CLI owns credential
	// recovery. The command layer, not the provider, decides whether an
	// interactive reauthentication is appropriate.
	OfficialCLIRefresh bool
	// RefreshInMemory means the provider can exchange a refresh token without
	// writing the result back. Must be false when refresh rotates the refresh
	// token (see RotatesRefreshToken).
	RefreshInMemory bool
	// RefreshAndPersist means the provider can refresh and atomically persist
	// the new credentials.
	RefreshAndPersist bool
	// RotatesRefreshToken means a successful refresh may invalidate the old
	// refresh token. When true, RefreshInMemory must not be advertised as safe
	// — discarding the new refresh token would lock the user out.
	RotatesRefreshToken bool
}

// Allows reports whether the provider can honour policy. It returns a
// descriptive error when the policy exceeds declared capabilities.
func (c Capabilities) Allows(policy Policy) error {
	switch policy {
	case Auto:
		return nil
	case ReadOnly:
		return nil
	case RefreshInMemory:
		if c.RotatesRefreshToken {
			return fmt.Errorf("%w: provider rotates refresh tokens; in-memory refresh would discard the new token (use persist, or readonly)", ErrPolicyNotSupported)
		}
		if !c.RefreshInMemory {
			return fmt.Errorf("%w: provider does not support in-memory refresh", ErrPolicyNotSupported)
		}
		return nil
	case RefreshAndPersist:
		if !c.RefreshAndPersist {
			return fmt.Errorf("%w: provider does not support persisting refreshed credentials", ErrPolicyNotSupported)
		}
		return nil
	default:
		return fmt.Errorf("%w: unknown policy %v", ErrPolicyNotSupported, policy)
	}
}
