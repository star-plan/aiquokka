package cursor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/McKean/aiquokka/internal/credential"
	"github.com/McKean/aiquokka/internal/httpx"
)

var tokenEndpoint = "https://api2.cursor.sh/oauth/token"

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func ensureFresh(ctx context.Context, c *Credential, force bool) error {
	if !force {
		return nil
	}
	if c.RefreshToken == "" {
		return fmt.Errorf("Cursor token expired and cannot be refreshed (no refresh token in %s)", c.Source)
	}
	policy := credential.PolicyFrom(ctx)
	return credential.Apply("Cursor", policy, capabilities(), credential.RefreshFuncs{
		Persist: func() error {
			return refreshAndPersist(ctx, c)
		},
	})
}

func refreshAndPersist(ctx context.Context, c *Credential) error {
	if err := refreshInPlace(ctx, c); err != nil {
		return err
	}
	return persistCredential(*c)
}

func refreshInPlace(ctx context.Context, c *Credential) error {
	body, _ := json.Marshal(map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": c.RefreshToken,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "aiquokka")

	resp, err := httpx.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("refreshing Cursor token: %s: %s", resp.Status, string(raw))
	}
	var tr tokenResponse
	if err := json.Unmarshal(raw, &tr); err != nil {
		return err
	}
	if tr.AccessToken == "" {
		return fmt.Errorf("Cursor refresh returned no access token")
	}
	c.AccessToken = tr.AccessToken
	if tr.RefreshToken != "" {
		c.RefreshToken = tr.RefreshToken
	}
	return nil
}

func persistCredential(c Credential) error {
	switch c.Source {
	case SourceAgentFile:
		return persistAgentAuthFile(c)
	case SourceDesktopDB:
		return persistDesktopDB(c)
	case SourceOSStore:
		return fmt.Errorf("persisting Cursor OS-store credentials is not supported")
	default:
		return fmt.Errorf("unknown Cursor credential source %q", c.Source)
	}
}
