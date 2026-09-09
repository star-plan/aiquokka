package codex

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/star-plan/aiquokka/internal/credential"
)

func TestReadOnlyLeavesCredentialStoreUnchanged(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)

	auth := []byte(`{
  "tokens": {
    "access_token": "old-access",
    "refresh_token": "old-refresh",
    "account_id": "acct-1"
  }
}
`)
	path := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(path, auth, 0o600); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	orig := usageEndpoint
	usageEndpoint = srv.URL
	t.Cleanup(func() { usageEndpoint = orig })

	ctx := credential.WithPolicy(context.Background(), credential.ReadOnly)
	_, err := Fetch(ctx)
	if !errors.Is(err, credential.ErrRefreshRequired) {
		t.Fatalf("Fetch = %v, want ErrRefreshRequired", err)
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
	auth := &authFile{Tokens: tokens{AccessToken: "a", RefreshToken: "r"}}
	ctx := credential.WithPolicy(context.Background(), credential.RefreshInMemory)
	err := ensureFresh(ctx, auth, true)
	if !errors.Is(err, credential.ErrPolicyNotSupported) {
		t.Fatalf("ensureFresh = %v, want ErrPolicyNotSupported", err)
	}
}
