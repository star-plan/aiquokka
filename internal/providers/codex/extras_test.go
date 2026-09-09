package codex

import "testing"

func TestExtrasResetCreditsFormat(t *testing.T) {
	total := int64(3)
	usable := int64(0)
	facts := extras(&usageResponse{
		RateLimitResetCredits: &struct {
			AvailableCount           *int64 `json:"available_count"`
			ApplicableAvailableCount *int64 `json:"applicable_available_count"`
		}{AvailableCount: &total, ApplicableAvailableCount: &usable},
	})
	if len(facts) != 1 || facts[0].Label != "Resets" || facts[0].Value != "0 / 3 available" {
		t.Fatalf("extras = %+v, want Resets: 0 / 3 available", facts)
	}
}
