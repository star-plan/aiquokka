package cursor

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/McKean/aiquokka/internal/credential"
)

func TestCursorUsesProviderReportedPercent(t *testing.T) {
	var period periodUsage
	body := `{
		"billingCycleStart": "1700000000000",
		"billingCycleEnd": "1702592000000",
		"planUsage": {
			"totalPercentUsed": 2.9,
			"includedSpend": 1008,
			"limit": 2000,
			"autoPercentUsed": 1.1,
			"apiPercentUsed": 0.4
		}
	}`
	if err := json.Unmarshal([]byte(body), &period); err != nil {
		t.Fatal(err)
	}
	report := buildReport(period, "Pro")
	if report.Plan != "Pro" {
		t.Fatalf("Plan = %q, want Pro", report.Plan)
	}
	if len(report.Windows) != 1 {
		t.Fatalf("windows = %d, want 1", len(report.Windows))
	}
	got := report.Windows[0].UsedPercent
	if got == nil {
		t.Fatal("UsedPercent is nil")
	}
	if *got != 2.9 {
		t.Fatalf("UsedPercent = %v, want 2.9 (must not use includedSpend/limit = 50%%)", *got)
	}
}

func TestCursorDoesNotGuessMissingPercentage(t *testing.T) {
	var period periodUsage
	body := `{
		"billingCycleStart": "1700000000000",
		"billingCycleEnd": "1702592000000",
		"planUsage": {
			"includedSpend": 1008,
			"limit": 2000
		}
	}`
	if err := json.Unmarshal([]byte(body), &period); err != nil {
		t.Fatal(err)
	}
	report := buildReport(period, "Pro")
	if report.Windows[0].UsedPercent != nil {
		t.Fatalf("UsedPercent = %v, want nil when totalPercentUsed is missing", *report.Windows[0].UsedPercent)
	}
}

func TestCredentialSourcePrecedence(t *testing.T) {
	t.Cleanup(func() {
		loadAgentCredentials = loadAgentAuthFile
		loadOSCredentials = loadOSStore
		loadDesktopCredentials = loadDesktopDB
	})

	agent := &Credential{AccessToken: "agent-token", RefreshToken: "agent-refresh", Source: SourceAgentFile}
	desktop := &Credential{AccessToken: "desktop-token", RefreshToken: "desktop-refresh", Source: SourceDesktopDB}

	loadAgentCredentials = func() (*Credential, error) { return agent, nil }
	loadOSCredentials = func() (*Credential, error) { return nil, nil }
	loadDesktopCredentials = func() (*Credential, error) { return desktop, nil }

	creds, err := defaultDiscoverCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if len(creds) != 2 {
		t.Fatalf("got %d credentials, want 2", len(creds))
	}
	if creds[0].Source != SourceAgentFile || creds[0].AccessToken != "agent-token" {
		t.Fatalf("first credential = %+v, want agent", creds[0])
	}

	loadAgentCredentials = func() (*Credential, error) { return nil, nil }
	creds, err = defaultDiscoverCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if len(creds) != 1 || creds[0].Source != SourceDesktopDB {
		t.Fatalf("fallback credentials = %+v, want desktop only", creds)
	}
}

func TestCredentialDoesNotSpliceAcrossSources(t *testing.T) {
	t.Cleanup(func() {
		loadAgentCredentials = loadAgentAuthFile
		loadOSCredentials = loadOSStore
		loadDesktopCredentials = loadDesktopDB
	})
	loadAgentCredentials = func() (*Credential, error) {
		return &Credential{AccessToken: "agent-access", Source: SourceAgentFile}, nil
	}
	loadDesktopCredentials = func() (*Credential, error) {
		return &Credential{AccessToken: "desk-access", RefreshToken: "desk-refresh", Source: SourceDesktopDB}, nil
	}
	loadOSCredentials = func() (*Credential, error) { return nil, nil }

	creds, err := defaultDiscoverCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if creds[0].RefreshToken != "" {
		t.Fatal("agent credential must not inherit the desktop refresh token")
	}
	if creds[1].AccessToken != "desk-access" || creds[1].RefreshToken != "desk-refresh" {
		t.Fatalf("desktop pair = %+v", creds[1])
	}
}

func TestReadOnlyLeavesCredentialStoreUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")
	auth := []byte(`{
  "accessToken": "old-access",
  "refreshToken": "old-refresh"
}
`)
	if err := os.WriteFile(path, auth, 0o600); err != nil {
		t.Fatal(err)
	}

	origPath := agentAuthPath
	agentAuthPath = func() (string, error) { return path, nil }
	t.Cleanup(func() { agentAuthPath = origPath })

	origDiscover := discoverCredentials
	discoverCredentials = defaultDiscoverCredentials
	loadAgentCredentials = loadAgentAuthFile
	loadOSCredentials = func() (*Credential, error) { return nil, nil }
	loadDesktopCredentials = func() (*Credential, error) { return nil, nil }
	t.Cleanup(func() {
		discoverCredentials = origDiscover
		loadAgentCredentials = loadAgentAuthFile
		loadOSCredentials = loadOSStore
		loadDesktopCredentials = loadDesktopDB
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	origBase := apiBase
	apiBase = srv.URL
	t.Cleanup(func() { apiBase = origBase })

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
	c := &Credential{AccessToken: "a", RefreshToken: "r", Source: SourceAgentFile}
	ctx := credential.WithPolicy(context.Background(), credential.RefreshInMemory)
	err := ensureFresh(ctx, c, true)
	if !errors.Is(err, credential.ErrPolicyNotSupported) {
		t.Fatalf("ensureFresh = %v, want ErrPolicyNotSupported", err)
	}
}

func TestDesktopDBRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.vscdb")
	if err := writeDesktopTokens(path, "desk-access", "desk-refresh"); err != nil {
		t.Fatal(err)
	}
	orig := desktopDBPath
	desktopDBPath = func() (string, error) { return path, nil }
	t.Cleanup(func() { desktopDBPath = orig })

	c, err := loadDesktopDB()
	if err != nil {
		t.Fatal(err)
	}
	if c == nil || c.AccessToken != "desk-access" || c.RefreshToken != "desk-refresh" || c.Source != SourceDesktopDB {
		t.Fatalf("desktop credential = %+v", c)
	}
}

func TestFetchUsesDashboardAPIs(t *testing.T) {
	origDiscover := discoverCredentials
	discoverCredentials = func() ([]Credential, error) {
		return []Credential{{AccessToken: "tok", Source: SourceAgentFile}}, nil
	}
	t.Cleanup(func() { discoverCredentials = origDiscover })

	mux := http.NewServeMux()
	mux.HandleFunc("/aiserver.v1.DashboardService/GetCurrentPeriodUsage", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Connect-Protocol-Version") != "1" {
			t.Errorf("Connect-Protocol-Version = %q", r.Header.Get("Connect-Protocol-Version"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"planUsage":{"totalPercentUsed":2.9,"includedSpend":1008,"limit":2000},"billingCycleStart":"1700000000000","billingCycleEnd":"1702592000000"}`))
	})
	mux.HandleFunc("/aiserver.v1.DashboardService/GetPlanInfo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"planInfo":{"planName":"Pro+"}}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	origBase := apiBase
	apiBase = srv.URL
	t.Cleanup(func() { apiBase = origBase })

	report, err := Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.Plan != "Pro+" {
		t.Fatalf("Plan = %q, want Pro+", report.Plan)
	}
	if report.Windows[0].UsedPercent == nil || *report.Windows[0].UsedPercent != 2.9 {
		t.Fatalf("UsedPercent = %v, want 2.9", report.Windows[0].UsedPercent)
	}
}

func TestBuildReportFormatsPercentsAndProviderMessage(t *testing.T) {
	var period periodUsage
	body := `{
		"planUsage": {
			"totalPercentUsed": 8.7,
			"autoPercentUsed": 9.348888888888888,
			"apiPercentUsed": 0
		},
		"displayMessage": "You've hit your usage limit"
	}`
	if err := json.Unmarshal([]byte(body), &period); err != nil {
		t.Fatal(err)
	}
	report := buildReport(period, "Pro")
	got := map[string]string{}
	for _, f := range report.Extra {
		got[f.Label] = f.Value
	}
	if got["Auto"] != "9.3% used" {
		t.Fatalf("Auto = %q, want 9.3%% used", got["Auto"])
	}
	if got["API"] != "0.0% used" {
		t.Fatalf("API = %q, want 0.0%% used", got["API"])
	}
	if _, ok := got["⚠ Provider"]; ok {
		t.Fatalf("displayMessage must stay hidden when totalPercentUsed is set: Extra=%+v", report.Extra)
	}
}

func TestBuildReportShowsProviderMessageWithoutPercentage(t *testing.T) {
	var period periodUsage
	body := `{"displayMessage": "Usage is billed to your organization"}`
	if err := json.Unmarshal([]byte(body), &period); err != nil {
		t.Fatal(err)
	}
	report := buildReport(period, "")
	got := map[string]string{}
	for _, f := range report.Extra {
		got[f.Label] = f.Value
	}
	if got["⚠ Provider"] != "Usage is billed to your organization" {
		t.Fatalf("Provider = %q, Extra=%+v", got["⚠ Provider"], report.Extra)
	}
}

func TestBuildReportShowsProviderMessageInDebug(t *testing.T) {
	t.Setenv("AIQUOKKA_DEBUG", "1")
	var period periodUsage
	body := `{
		"planUsage": {"totalPercentUsed": 8.7},
		"displayMessage": "You've hit your usage limit"
	}`
	if err := json.Unmarshal([]byte(body), &period); err != nil {
		t.Fatal(err)
	}
	report := buildReport(period, "Pro")
	found := false
	for _, f := range report.Extra {
		if f.Label == "⚠ Provider" && f.Value == "You've hit your usage limit" {
			found = true
		}
	}
	if !found {
		t.Fatalf("debug mode should surface displayMessage: Extra=%+v", report.Extra)
	}
}

func TestBuildReportPutsOnDemandInExtra(t *testing.T) {
	var period periodUsage
	body := `{
		"planUsage": {"totalPercentUsed": "12"},
		"spendLimitUsage": {"individualUsed": "315", "individualLimit": "5000"}
	}`
	if err := json.Unmarshal([]byte(body), &period); err != nil {
		t.Fatal(err)
	}
	report := buildReport(period, "")
	if report.Windows[0].UsedPercent == nil || *report.Windows[0].UsedPercent != 12 {
		t.Fatalf("UsedPercent = %v, want 12 from string protobuf field", report.Windows[0].UsedPercent)
	}
	found := false
	for _, f := range report.Extra {
		if f.Label == "On-Demand" && f.Value == "$3.15 / $50.00" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Extra = %+v, want On-Demand $3.15 / $50.00", report.Extra)
	}
}
