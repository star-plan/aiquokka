package grok

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/McKean/aiquokka/internal/credential"
)

func TestReadOnlyLeavesCredentialStoreUnchanged(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GROK_HOME", dir)

	auth := []byte(`{
  "https://auth.x.ai::user-1": {
    "key": "old-access",
    "refresh_token": "old-refresh",
    "expires_at": "2000-01-01T00:00:00Z",
    "oidc_client_id": "client-1",
    "oidc_issuer": "https://auth.x.ai"
  }
}
`)
	path := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(path, auth, 0o600); err != nil {
		t.Fatal(err)
	}

	acc, storeKey, err := loadAccount()
	if err != nil {
		t.Fatal(err)
	}
	if !acc.expired(time.Now()) {
		t.Fatal("fixture should be expired")
	}

	ctx := credential.WithPolicy(context.Background(), credential.ReadOnly)
	err = ensureFresh(ctx, acc, storeKey, false)
	if !errors.Is(err, credential.ErrRefreshRequired) {
		t.Fatalf("ensureFresh = %v, want ErrRefreshRequired", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(auth) {
		t.Fatalf("credential store changed under ReadOnly:\n got: %s\nwant: %s", got, auth)
	}
}

func TestRotatingRefreshTokenRejectsInMemoryPolicy(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GROK_HOME", dir)

	auth := []byte(`{
  "https://auth.x.ai::user-1": {
    "key": "old-access",
    "refresh_token": "old-refresh",
    "expires_at": "2000-01-01T00:00:00Z",
    "oidc_client_id": "client-1",
    "oidc_issuer": "https://auth.x.ai"
  }
}
`)
	if err := os.WriteFile(filepath.Join(dir, "auth.json"), auth, 0o600); err != nil {
		t.Fatal(err)
	}

	acc, storeKey, err := loadAccount()
	if err != nil {
		t.Fatal(err)
	}

	ctx := credential.WithPolicy(context.Background(), credential.RefreshInMemory)
	err = ensureFresh(ctx, acc, storeKey, false)
	if !errors.Is(err, credential.ErrPolicyNotSupported) {
		t.Fatalf("ensureFresh = %v, want ErrPolicyNotSupported", err)
	}

	var pe *credential.PolicyError
	if !errors.As(err, &pe) {
		t.Fatalf("error type %T, want *PolicyError", err)
	}
}
