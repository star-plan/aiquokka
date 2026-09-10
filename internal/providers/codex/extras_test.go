package codex

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFetchResetCreditDetailsOverrideSummaryAndSortByExpiry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	if err := os.WriteFile(filepath.Join(dir, "auth.json"), []byte(`{"tokens":{"access_token":"token","account_id":"acct-1"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("ChatGPT-Account-Id"); got != "acct-1" {
			t.Errorf("ChatGPT-Account-Id = %q, want acct-1", got)
		}
		switch r.URL.Path {
		case "/usage":
			_, _ = w.Write([]byte(`{
				"plan_type":"plus",
				"rate_limit_reset_credits":{"available_count":3,"applicable_available_count":0}
			}`))
		case "/reset-credits":
			_, _ = w.Write([]byte(`{
				"available_count":2,
				"credits":[
					{"id":"later","status":"available","granted_at":"2026-09-01T00:00:00Z","expires_at":"2026-09-30T00:00:00Z","title":"Later"},
					{"id":"sooner","reset_type":"codex_rate_limits","status":"available","granted_at":"2026-09-02T00:00:00Z","expires_at":"2026-09-20T00:00:00Z","description":"Redeem me"}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	overrideCodexEndpoints(t, srv.URL+"/usage", srv.URL+"/reset-credits")

	report, err := Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.ResetCredits == nil {
		t.Fatal("ResetCredits = nil")
	}
	credits := report.ResetCredits
	if credits.AvailableCount != 2 {
		t.Fatalf("AvailableCount = %d, want detail endpoint's 2", credits.AvailableCount)
	}
	if !credits.DetailsFetched {
		t.Fatal("DetailsFetched = false, want true")
	}
	if credits.ApplicableAvailableCount == nil || *credits.ApplicableAvailableCount != 0 {
		t.Fatalf("ApplicableAvailableCount = %v, want 0", credits.ApplicableAvailableCount)
	}
	if len(credits.Credits) != 2 || credits.Credits[0].ID != "sooner" || credits.Credits[1].ID != "later" {
		t.Fatalf("Credits = %+v, want expiry-sorted credits", credits.Credits)
	}
	if credits.Credits[0].ExpiresAt == nil || !credits.Credits[0].ExpiresAt.Equal(time.Date(2026, time.September, 20, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("first expiry = %v, want 2026-09-20", credits.Credits[0].ExpiresAt)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var output struct {
		ResetCredits struct {
			AvailableCount int64 `json:"available_count"`
			Credits        []struct {
				ID string `json:"id"`
			} `json:"credits"`
		} `json:"reset_credits"`
	}
	if err := json.Unmarshal(encoded, &output); err != nil {
		t.Fatal(err)
	}
	if output.ResetCredits.AvailableCount != 2 || len(output.ResetCredits.Credits) != 2 || output.ResetCredits.Credits[0].ID != "sooner" {
		t.Fatalf("structured reset credits = %s", encoded)
	}
}

func TestFetchResetCreditDetailFailureKeepsUsageSummary(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	if err := os.WriteFile(filepath.Join(dir, "auth.json"), []byte(`{"tokens":{"access_token":"token"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/usage":
			_, _ = w.Write([]byte(`{"rate_limit_reset_credits":{"available_count":3}}`))
		case "/reset-credits":
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	overrideCodexEndpoints(t, srv.URL+"/usage", srv.URL+"/reset-credits")

	report, err := Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch = %v, want successful primary report", err)
	}
	if report.ResetCredits == nil || report.ResetCredits.AvailableCount != 3 || report.ResetCredits.DetailsFetched || len(report.ResetCredits.Credits) != 0 {
		t.Fatalf("ResetCredits = %+v, want summary count with no details", report.ResetCredits)
	}
}

func TestFetchSkipsResetCreditDetailsWhenSummaryIsZero(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	if err := os.WriteFile(filepath.Join(dir, "auth.json"), []byte(`{"tokens":{"access_token":"token"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	resetCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/usage":
			_, _ = w.Write([]byte(`{"rate_limit_reset_credits":{"available_count":0}}`))
		case "/reset-credits":
			resetCalls++
			http.Error(w, "should not be requested", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	overrideCodexEndpoints(t, srv.URL+"/usage", srv.URL+"/reset-credits")

	report, err := Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.ResetCredits == nil || report.ResetCredits.AvailableCount != 0 {
		t.Fatalf("ResetCredits = %+v, want zero summary", report.ResetCredits)
	}
	if resetCalls != 0 {
		t.Fatalf("detail endpoint called %d times, want 0", resetCalls)
	}
}

func overrideCodexEndpoints(t *testing.T, usage, resetCredits string) {
	t.Helper()
	originalUsage, originalResetCredits := usageEndpoint, resetCreditsEndpoint
	usageEndpoint, resetCreditsEndpoint = usage, resetCredits
	t.Cleanup(func() { usageEndpoint, resetCreditsEndpoint = originalUsage, originalResetCredits })
}
