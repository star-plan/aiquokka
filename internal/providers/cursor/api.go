package cursor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/McKean/aiquokka/internal/httpx"
	"github.com/McKean/aiquokka/internal/usage"
)

var apiBase = "https://api2.cursor.sh"

const (
	usagePath = "/aiserver.v1.DashboardService/GetCurrentPeriodUsage"
	planPath  = "/aiserver.v1.DashboardService/GetPlanInfo"
)

type periodUsage struct {
	DisplayMessage    string          `json:"displayMessage"`
	BillingCycleStart json.RawMessage `json:"billingCycleStart"`
	BillingCycleEnd   json.RawMessage `json:"billingCycleEnd"`
	PlanUsage         *planUsage      `json:"planUsage"`
	SpendLimitUsage   *spendLimit     `json:"spendLimitUsage"`
}

type planUsage struct {
	TotalPercentUsed num `json:"totalPercentUsed"`
	AutoPercentUsed  num `json:"autoPercentUsed"`
	APIPercentUsed   num `json:"apiPercentUsed"`
	IncludedSpend    num `json:"includedSpend"`
	Limit            num `json:"limit"`
	Remaining        num `json:"remaining"`
}

type spendLimit struct {
	IndividualUsed  num `json:"individualUsed"`
	IndividualLimit num `json:"individualLimit"`
}

type planInfoResponse struct {
	PlanInfo struct {
		PlanName string `json:"planName"`
	} `json:"planInfo"`
}

// num is a JSON number that may arrive as a number or a protobuf string, and
// records whether it was present. Missing is distinct from zero.
type num struct {
	set   bool
	value float64
}

func (n *num) UnmarshalJSON(data []byte) error {
	text := strings.Trim(strings.TrimSpace(string(data)), `"`)
	if text == "" || text == "null" {
		return nil
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return nil
	}
	n.set, n.value = true, value
	return nil
}

func fetchUsage(ctx context.Context, accessToken string) (*usage.Report, int, error) {
	body, status, err := postJSON(ctx, apiBase+usagePath, accessToken)
	if err != nil {
		return nil, status, err
	}
	var period periodUsage
	if err := json.Unmarshal(body, &period); err != nil {
		return nil, status, fmt.Errorf("decoding Cursor usage: %w", err)
	}

	planName := ""
	if planBody, _, err := postJSON(ctx, apiBase+planPath, accessToken); err == nil {
		var plan planInfoResponse
		if json.Unmarshal(planBody, &plan) == nil {
			planName = strings.TrimSpace(plan.PlanInfo.PlanName)
		}
	}
	return buildReport(period, planName), status, nil
}

func postJSON(ctx context.Context, url, token string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connect-Protocol-Version", "1")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "aiquokka")

	resp, err := httpx.Client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return body, resp.StatusCode, fmt.Errorf("Cursor API %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return body, resp.StatusCode, nil
}

func buildReport(period periodUsage, planName string) *usage.Report {
	report := &usage.Report{Provider: "Cursor", Plan: planName}

	start := parseTimestamp(period.BillingCycleStart)
	end := parseTimestamp(period.BillingCycleEnd)
	window := usage.Window{Label: "Billing", ResetsAt: end}
	if !start.IsZero() && !end.IsZero() && end.After(start) {
		window.Duration = end.Sub(start)
	}

	// Authoritative percentage only. includedSpend/limit is a different scale
	// and must never be used to invent UsedPercent.
	if u := period.PlanUsage; u != nil && u.TotalPercentUsed.set {
		pct := u.TotalPercentUsed.value
		window.UsedPercent = &pct
	}
	report.Windows = append(report.Windows, window)

	if u := period.PlanUsage; u != nil {
		if u.AutoPercentUsed.set {
			report.Extra = append(report.Extra, usage.Fact{
				Label: "Auto", Value: usage.FormatPercent(u.AutoPercentUsed.value) + " used",
			})
		}
		if u.APIPercentUsed.set {
			report.Extra = append(report.Extra, usage.Fact{
				Label: "API", Value: usage.FormatPercent(u.APIPercentUsed.value) + " used",
			})
		}
		debugf("planUsage total=%v auto=%v api=%v displayMessage=%q",
			numDebug(u.TotalPercentUsed), numDebug(u.AutoPercentUsed), numDebug(u.APIPercentUsed),
			strings.TrimSpace(period.DisplayMessage))
	}
	if s := period.SpendLimitUsage; s != nil && s.IndividualUsed.set {
		used := formatCents(s.IndividualUsed.value)
		switch {
		case s.IndividualLimit.set && s.IndividualLimit.value > 0:
			report.Extra = append(report.Extra, usage.Fact{
				Label: "On-Demand", Value: used + " / " + formatCents(s.IndividualLimit.value),
			})
		case s.IndividualLimit.set && s.IndividualLimit.value == 0:
			report.Extra = append(report.Extra, usage.Fact{Label: "On-Demand", Value: "off"})
		default:
			report.Extra = append(report.Extra, usage.Fact{Label: "On-Demand", Value: used})
		}
	}
	if msg := strings.TrimSpace(period.DisplayMessage); msg != "" {
		report.Extra = append(report.Extra, usage.Fact{Label: "⚠ Provider", Value: msg})
	}
	return report
}

func numDebug(n num) string {
	if !n.set {
		return "<missing>"
	}
	return strconv.FormatFloat(n.value, 'f', -1, 64)
}

func formatCents(cents float64) string {
	return fmt.Sprintf("$%.2f", cents/100)
}

func parseTimestamp(raw json.RawMessage) time.Time {
	text := strings.Trim(strings.TrimSpace(string(raw)), `"`)
	if text == "" || text == "null" {
		return time.Time{}
	}
	if ms, err := strconv.ParseInt(text, 10, 64); err == nil {
		if ms <= 0 {
			return time.Time{}
		}
		return time.UnixMilli(ms)
	}
	if parsed, err := time.Parse(time.RFC3339, text); err == nil {
		return parsed
	}
	return time.Time{}
}
