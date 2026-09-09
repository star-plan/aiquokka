package credential

import (
	"errors"
	"fmt"
)

var (
	// ErrPolicyNotSupported means the requested Policy exceeds the provider's
	// Capabilities. Callers must surface this; never silently fall back.
	ErrPolicyNotSupported = errors.New("credential policy not supported")

	// ErrRefreshRequired means credentials are expired/unusable and the active
	// Policy does not allow aiquokka to refresh them.
	ErrRefreshRequired = errors.New("credential refresh required")
)

// PolicyError explains a Policy/Capabilities mismatch for a named provider.
type PolicyError struct {
	Provider string
	Policy   Policy
	Err      error
}

func (e *PolicyError) Error() string {
	if e.Provider == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: %v", e.Provider, e.Err)
}

func (e *PolicyError) Unwrap() error { return e.Err }

// RefreshRequiredError tells the user credentials need attention under the
// active ReadOnly (or otherwise non-refreshing) policy.
type RefreshRequiredError struct {
	Provider string
	Hint     string
	Err      error
}

func (e *RefreshRequiredError) Error() string {
	msg := "credentials expired or unusable"
	if e.Provider != "" {
		msg = e.Provider + ": " + msg
	}
	if e.Hint != "" {
		msg += " — " + e.Hint
	}
	return msg
}

func (e *RefreshRequiredError) Unwrap() error {
	if e.Err != nil {
		return e.Err
	}
	return ErrRefreshRequired
}
