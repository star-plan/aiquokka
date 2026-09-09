package claude

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
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o700); err != nil {
		t.Fatal(err)
	}

	auth := []byte(`{
  "claudeAiOauth": {
    "accessToken": "old-access",
    "refreshToken": "old-refresh",
    "expiresAt": 1
  }
}
`)
	path := filepath.Join(home, ".claude", ".credentials.json")
	if err := os.WriteFile(path, auth, 0o600); err != nil {
		t.Fatal(err)
	}

	o, err := loadCredentials(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !o.expired(time.Now()) {
		t.Fatal("fixture should be expired")
	}

	ctx := credential.WithPolicy(context.Background(), credential.ReadOnly)
	err = ensureFresh(ctx, o, false)
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
	o := &oauth{AccessToken: "a", RefreshToken: "r", ExpiresAt: 1}
	ctx := credential.WithPolicy(context.Background(), credential.RefreshInMemory)
	err := ensureFresh(ctx, o, true)
	if !errors.Is(err, credential.ErrPolicyNotSupported) {
		t.Fatalf("ensureFresh = %v, want ErrPolicyNotSupported", err)
	}
}
