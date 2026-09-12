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

	// ErrReauthRequired means the refresh credential can no longer be used and
	// the user must sign in through the provider's official CLI.
	ErrReauthRequired = errors.New("credential reauthentication required")
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

// ReauthRequiredError is returned only when silent refresh cannot continue
// (for example a missing, revoked, or rotated refresh token). Command is the
// exact official-CLI command the user can run, such as "codex login".
// The CLI layer may decide to run that command interactively in a TTY.
type ReauthRequiredError struct {
	Provider string
	Command  string
	Cause    error
}

func (e *ReauthRequiredError) Error() string {
	provider := e.Provider
	if provider == "" {
		provider = "Authentication"
	}
	if e.Command == "" {
		return provider + " authentication expired"
	}
	return fmt.Sprintf("%s authentication expired — run `%s`", provider, e.Command)
}

func (e *ReauthRequiredError) Unwrap() error {
	if e.Cause != nil {
		return errors.Join(ErrReauthRequired, e.Cause)
	}
	return ErrReauthRequired
}
