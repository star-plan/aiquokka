package credential

import (
	"context"
	"errors"
	"testing"
)

func TestParsePolicy(t *testing.T) {
	cases := []struct {
		in   string
		want Policy
	}{
		{"", Auto},
		{"auto", Auto},
		{"readonly", ReadOnly},
		{"read-only", ReadOnly},
		{"memory", RefreshInMemory},
		{"in-memory", RefreshInMemory},
		{"persist", RefreshAndPersist},
		{"write", RefreshAndPersist},
	}
	for _, tc := range cases {
		got, err := ParsePolicy(tc.in)
		if err != nil {
			t.Fatalf("ParsePolicy(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("ParsePolicy(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
	if _, err := ParsePolicy("nope"); err == nil {
		t.Fatal("ParsePolicy(nope) expected error")
	}
}

func TestRotatingRefreshTokenRejectsInMemory(t *testing.T) {
	caps := Capabilities{
		RefreshInMemory:     true, // misconfigured advertisement — Allows must still reject
		RefreshAndPersist:   true,
		RotatesRefreshToken: true,
	}
	err := caps.Allows(RefreshInMemory)
	if err == nil {
		t.Fatal("expected RefreshInMemory to be rejected when RotatesRefreshToken is set")
	}
	if !errors.Is(err, ErrPolicyNotSupported) {
		t.Fatalf("error = %v, want ErrPolicyNotSupported", err)
	}

	if err := caps.Allows(RefreshAndPersist); err != nil {
		t.Fatalf("persist should be allowed: %v", err)
	}
	if err := caps.Allows(ReadOnly); err != nil {
		t.Fatalf("readonly should be allowed: %v", err)
	}
}

func TestApplyReadOnlyDoesNotInvokeRefresh(t *testing.T) {
	called := false
	err := Apply("Grok", ReadOnly, Capabilities{RefreshAndPersist: true, RotatesRefreshToken: true}, RefreshFuncs{
		InMemory: func() error { called = true; return nil },
		Persist:  func() error { called = true; return nil },
	})
	if called {
		t.Fatal("ReadOnly must not call refresh funcs")
	}
	if !errors.Is(err, ErrRefreshRequired) {
		t.Fatalf("error = %v, want ErrRefreshRequired", err)
	}
}

func TestApplyAutoPrefersPersistForRotatingToken(t *testing.T) {
	called := false
	err := Apply("Grok", Auto, Capabilities{
		RefreshAndPersist:   true,
		RotatesRefreshToken: true,
	}, RefreshFuncs{
		Persist: func() error { called = true; return nil },
	})
	if err != nil {
		t.Fatalf("Apply(auto) = %v", err)
	}
	if !called {
		t.Fatal("Auto must persist a rotating refresh token")
	}
}

func TestApplyRejectsUnsupportedPersist(t *testing.T) {
	err := Apply("Demo", RefreshAndPersist, Capabilities{}, RefreshFuncs{
		Persist: func() error { return nil },
	})
	if !errors.Is(err, ErrPolicyNotSupported) {
		t.Fatalf("error = %v, want ErrPolicyNotSupported", err)
	}
	var pe *PolicyError
	if !errors.As(err, &pe) {
		t.Fatalf("error type %T, want *PolicyError", err)
	}
	if pe.Provider != "Demo" {
		t.Fatalf("Provider = %q, want Demo", pe.Provider)
	}
}

func TestPolicyContextRoundTrip(t *testing.T) {
	ctx := WithPolicy(context.Background(), RefreshAndPersist)
	if got := PolicyFrom(ctx); got != RefreshAndPersist {
		t.Fatalf("PolicyFrom = %v, want persist", got)
	}
	if got := PolicyFrom(context.Background()); got != DefaultPolicy {
		t.Fatalf("default PolicyFrom = %v, want %v", got, DefaultPolicy)
	}
}
