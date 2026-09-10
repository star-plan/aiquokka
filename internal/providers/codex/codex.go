package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/star-plan/aiquokka/internal/httpx"
	"github.com/star-plan/aiquokka/internal/usage"
)

// usageEndpoint is the ChatGPT backend on-demand usage endpoint (the same data
// Codex shows as "5h limit" / "weekly limit"). Overridable in tests.
var usageEndpoint = "https://chatgpt.com/backend-api/wham/usage"

// resetCreditsEndpoint returns per-credit expiry data. It is deliberately a
// separate request: /wham/usage contains only its summary counts.
var resetCreditsEndpoint = "https://chatgpt.com/backend-api/wham/rate-limit-reset-credits"

// window mirrors rate_limit.{primary,secondary}_window from /wham/usage.
type window struct {
	UsedPercent        *float64 `json:"used_percent"`
	LimitWindowSeconds int64    `json:"limit_window_seconds"`
	ResetAt            *int64   `json:"reset_at"`
	ResetAfterSeconds  *int64   `json:"reset_after_seconds"`
}

type credits struct {
	HasCredits bool   `json:"has_credits"`
	Unlimited  bool   `json:"unlimited"`
	Balance    string `json:"balance"`
}

type usageResponse struct {
	PlanType  string `json:"plan_type"`
	RateLimit struct {
		Primary   *window `json:"primary_window"`
		Secondary *window `json:"secondary_window"`
	} `json:"rate_limit"`
	Credits               *credits `json:"credits"`
	RateLimitResetCredits *struct {
		AvailableCount           *int64 `json:"available_count"`
		ApplicableAvailableCount *int64 `json:"applicable_available_count"`
	} `json:"rate_limit_reset_credits"`
}

// resetCreditsResponse mirrors /wham/rate-limit-reset-credits. Timestamps are
// RFC 3339 strings (or null), unlike the epoch timestamps in usage windows.
type resetCreditsResponse struct {
	AvailableCount *int64                `json:"available_count"`
	Credits        []resetCreditResponse `json:"credits"`
}

type resetCreditResponse struct {
	ID          string  `json:"id"`
	ResetType   string  `json:"reset_type"`
	Status      string  `json:"status"`
	GrantedAt   *string `json:"granted_at"`
	ExpiresAt   *string `json:"expires_at"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
}

// Fetch reads local Codex credentials and returns the current usage windows.
// Token refresh on 401 follows the credential policy attached to ctx
// (default: ReadOnly — see --credential-policy).
func Fetch(ctx context.Context) (*usage.Report, error) {
	auth, err := loadAuth()
	if err != nil {
		return nil, err
	}

	resp, err := getUsage(ctx, auth)
	if err != nil {
		return nil, err
	}

	// The window that is "primary" vs "secondary" depends on the plan (a Plus
	// plan reports only the weekly window as primary), so label each by its
	// actual duration rather than by position.
	report := &usage.Report{Provider: "Codex", Plan: resp.PlanType}
	now := time.Now()
	for _, w := range []*window{resp.RateLimit.Primary, resp.RateLimit.Secondary} {
		if win := toWindow(w, now); win != nil {
			report.Windows = append(report.Windows, *win)
		}
	}
	report.Extra = extras(resp)
	report.ResetCredits = resetCreditsSummary(resp)
	if report.ResetCredits != nil && report.ResetCredits.AvailableCount > 0 {
		// The detail endpoint is best-effort. Losing expiry information must not
		// hide quota windows or the summary count from the primary usage read.
		if details, err := getResetCredits(ctx, auth); err == nil {
			if details.AvailableCount != nil {
				report.ResetCredits.AvailableCount = *details.AvailableCount
			}
			report.ResetCredits.DetailsFetched = true
			report.ResetCredits.Credits = toResetCredits(details.Credits)
		}
	}
	return report, nil
}

// getUsage performs the authed GET, refreshing (per credential policy) and
// retrying once on 401.
func getUsage(ctx context.Context, auth *authFile) (*usageResponse, error) {
	resp, status, err := doUsage(ctx, auth.Tokens.AccessToken, auth.Tokens.AccountID)
	if err == nil {
		return resp, nil
	}
	if status == http.StatusUnauthorized && auth.Tokens.RefreshToken != "" {
		if rerr := ensureFresh(ctx, auth, true); rerr != nil {
			return nil, rerr
		}
		resp, _, err = doUsage(ctx, auth.Tokens.AccessToken, auth.Tokens.AccountID)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}
	return nil, err
}

// doUsage issues the raw request and returns the parsed body plus HTTP status.
func doUsage(ctx context.Context, accessToken, accountID string) (*usageResponse, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, usageEndpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if accountID != "" {
		req.Header.Set("ChatGPT-Account-Id", accountID)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "aiquokka")
	req.Header.Set("originator", "codex_cli_rs")

	httpResp, err := httpx.Client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer httpResp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, httpResp.StatusCode, fmt.Errorf("%s: %s", httpResp.Status, string(body))
	}
	var out usageResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, httpResp.StatusCode, fmt.Errorf("decoding usage: %w", err)
	}
	return &out, httpResp.StatusCode, nil
}

// getResetCredits performs the secondary, non-fatal detail request. It uses
// the same one-time refresh behavior as usage so a token expiring between the
// two requests does not unnecessarily discard useful expiry information.
func getResetCredits(ctx context.Context, auth *authFile) (*resetCreditsResponse, error) {
	resp, status, err := doResetCredits(ctx, auth.Tokens.AccessToken, auth.Tokens.AccountID)
	if err == nil {
		return resp, nil
	}
	if status == http.StatusUnauthorized && auth.Tokens.RefreshToken != "" {
		if rerr := ensureFresh(ctx, auth, true); rerr != nil {
			return nil, rerr
		}
		resp, _, err = doResetCredits(ctx, auth.Tokens.AccessToken, auth.Tokens.AccountID)
		if err == nil {
			return resp, nil
		}
	}
	return nil, err
}

func doResetCredits(ctx context.Context, accessToken, accountID string) (*resetCreditsResponse, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resetCreditsEndpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if accountID != "" {
		req.Header.Set("ChatGPT-Account-Id", accountID)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "aiquokka")
	req.Header.Set("originator", "codex_cli_rs")

	httpResp, err := httpx.Client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer httpResp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, httpResp.StatusCode, fmt.Errorf("%s: %s", httpResp.Status, string(body))
	}
	var out resetCreditsResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, httpResp.StatusCode, fmt.Errorf("decoding reset credits: %w", err)
	}
	return &out, httpResp.StatusCode, nil
}

// toWindow converts a /wham/usage window to the shared model, deriving the
// label from the window's duration.
func toWindow(w *window, now time.Time) *usage.Window {
	if w == nil || w.UsedPercent == nil {
		return nil
	}
	out := usage.Window{Label: windowLabel(w.LimitWindowSeconds), UsedPercent: w.UsedPercent}
	if w.LimitWindowSeconds > 0 {
		out.Duration = time.Duration(w.LimitWindowSeconds) * time.Second
	}
	switch {
	case w.ResetAt != nil && *w.ResetAt > 0:
		out.ResetsAt = time.Unix(*w.ResetAt, 0)
	case w.ResetAfterSeconds != nil && *w.ResetAfterSeconds > 0:
		out.ResetsAt = now.Add(time.Duration(*w.ResetAfterSeconds) * time.Second)
	}
	return &out
}

// windowLabel names a window by its length in seconds.
func windowLabel(seconds int64) string {
	switch seconds {
	case 0:
		return "window"
	case 18000:
		return "5h"
	case 604800:
		return "Weekly"
	}
	switch {
	case seconds%86400 == 0:
		return fmt.Sprintf("%dd", seconds/86400)
	case seconds%3600 == 0:
		return fmt.Sprintf("%dh", seconds/3600)
	case seconds%60 == 0:
		return fmt.Sprintf("%dm", seconds/60)
	default:
		return fmt.Sprintf("%ds", seconds)
	}
}

// extras surfaces Codex-specific facts other than reset credits, which have a
// typed representation in usage.Report so their expiry data survives JSON/YAML.
func extras(r *usageResponse) []usage.Fact {
	var facts []usage.Fact
	if c := r.Credits; c != nil {
		switch {
		case c.Unlimited:
			facts = append(facts, usage.Fact{Label: "Credits", Value: "unlimited"})
		case c.HasCredits && c.Balance != "":
			facts = append(facts, usage.Fact{Label: "Credits", Value: c.Balance})
		}
	}
	return facts
}

func resetCreditsSummary(r *usageResponse) *usage.ResetCredits {
	rc := r.RateLimitResetCredits
	if rc == nil || rc.AvailableCount == nil {
		return nil
	}
	return &usage.ResetCredits{
		AvailableCount:           *rc.AvailableCount,
		ApplicableAvailableCount: rc.ApplicableAvailableCount,
	}
}

func toResetCredits(raw []resetCreditResponse) []usage.ResetCredit {
	credits := make([]usage.ResetCredit, 0, len(raw))
	for _, credit := range raw {
		credits = append(credits, usage.ResetCredit{
			ID:          credit.ID,
			ResetType:   credit.ResetType,
			Status:      credit.Status,
			GrantedAt:   parseCreditTime(credit.GrantedAt),
			ExpiresAt:   parseCreditTime(credit.ExpiresAt),
			Title:       credit.Title,
			Description: credit.Description,
		})
	}
	// The API does not promise display order. Sorting lets the default terminal
	// view foreground the credits users need to redeem first.
	sort.SliceStable(credits, func(i, j int) bool {
		a, b := credits[i].ExpiresAt, credits[j].ExpiresAt
		switch {
		case a == nil:
			return false
		case b == nil:
			return true
		default:
			return a.Before(*b)
		}
	})
	return credits
}

func parseCreditTime(raw *string) *time.Time {
	if raw == nil || *raw == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *raw)
	if err != nil {
		return nil
	}
	return &t
}
