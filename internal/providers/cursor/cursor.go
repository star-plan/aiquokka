// Package cursor reads Cursor Agent and Desktop credentials and fetches
// dashboard usage from api2.cursor.sh.
package cursor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/star-plan/aiquokka/internal/credential"
	"github.com/star-plan/aiquokka/internal/usage"
)

// Provider adapts the Cursor fetcher to the shared provider.Provider contract.
type Provider struct{}

// New returns the Cursor provider.
func New() *Provider { return &Provider{} }

func (*Provider) ID() string          { return "cursor" }
func (*Provider) Name() string        { return "Cursor" }
func (*Provider) Description() string { return "Cursor billing-cycle usage" }
func (*Provider) Fetch(ctx context.Context) (*usage.Report, error) {
	return Fetch(ctx)
}

func (*Provider) Capabilities() credential.Capabilities {
	return capabilities()
}

func capabilities() credential.Capabilities {
	return credential.Capabilities{
		RefreshInMemory:     false,
		RefreshAndPersist:   true,
		RotatesRefreshToken: true,
	}
}

// Fetch discovers Cursor credentials in priority order (Agent file, OS store,
// Desktop DB) and queries the dashboard usage APIs. Quota percentage is taken
// only from planUsage.totalPercentUsed — never reconstructed from spend/limit.
func Fetch(ctx context.Context) (*usage.Report, error) {
	creds, err := discoverCredentials()
	if err != nil {
		return nil, err
	}
	if len(creds) == 0 {
		return nil, usage.NotConfigured("no Cursor credentials found — run `agent login` or sign in to Cursor Desktop")
	}

	var lastErr error
	for i := range creds {
		c := creds[i]
		report, status, err := fetchUsage(ctx, c.AccessToken)
		if err == nil {
			annotateSource(report, c)
			return report, nil
		}
		lastErr = err
		if !isAuthStatus(status) {
			return nil, err
		}
		if c.RefreshToken == "" {
			continue
		}
		if rerr := ensureFresh(ctx, &c, true); rerr != nil {
			lastErr = rerr
			if errors.Is(rerr, credential.ErrPolicyNotSupported) {
				return nil, rerr
			}
			continue
		}
		report, _, err = fetchUsage(ctx, c.AccessToken)
		if err == nil {
			annotateSource(report, c)
			return report, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("Cursor credentials were rejected")
	}
	return nil, lastErr
}

func isAuthStatus(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusForbidden
}

func annotateSource(report *usage.Report, c Credential) {
	if report == nil || os.Getenv("AIQUOKKA_DEBUG") == "" {
		return
	}
	report.Extra = append(report.Extra, usage.Fact{Label: "Source", Value: string(c.Source)})
}

func debugEnabled() bool {
	return os.Getenv("AIQUOKKA_DEBUG") != ""
}

func debugf(format string, args ...any) {
	if !debugEnabled() {
		return
	}
	fmt.Fprintf(os.Stderr, "aiquokka cursor: "+format+"\n", args...)
}
