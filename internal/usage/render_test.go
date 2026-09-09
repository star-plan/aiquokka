package usage

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestLabelWidthAcrossReports(t *testing.T) {
	short := &Report{Windows: []Window{{Label: "5h"}}}
	long := &Report{Windows: []Window{{Label: "Premium Interactions"}}}

	if got, want := LabelWidth(short, long), len("Premium Interactions"); got != want {
		t.Fatalf("LabelWidth() = %d, want %d", got, want)
	}
}

func TestLabelWidthHasMinimum(t *testing.T) {
	if got := LabelWidth(&Report{Windows: []Window{{Label: "5h"}}}); got != 8 {
		t.Fatalf("LabelWidth() = %d, want 8", got)
	}
}

func TestRenderAlignedUsesSharedBarColumn(t *testing.T) {
	used := 25.0
	now := time.Date(2026, time.August, 12, 12, 0, 0, 0, time.UTC)
	short := &Report{Provider: "Short", Windows: []Window{{Label: "5h", UsedPercent: &used}}}
	long := &Report{Provider: "Long", Windows: []Window{{Label: "Weekly Fable", UsedPercent: &used}}}
	width := LabelWidth(short, long)

	var shortOut, longOut bytes.Buffer
	RenderAligned(&shortOut, short, now, width)
	RenderAligned(&longOut, long, now, width)

	shortBar := strings.Index(shortOut.String(), "[")
	longBar := strings.Index(longOut.String(), "[")
	if shortBar < 0 || longBar < 0 {
		t.Fatalf("bar missing from output:\n%s\n%s", shortOut.String(), longOut.String())
	}
	shortLine := strings.LastIndex(shortOut.String()[:shortBar], "\n")
	longLine := strings.LastIndex(longOut.String()[:longBar], "\n")
	if got, want := shortBar-shortLine, longBar-longLine; got != want {
		t.Fatalf("bar columns differ: short=%d long=%d", got, want)
	}
}

func TestRemainingBarFullThenEmpty(t *testing.T) {
	full := remainingBar(12.34, "USD", 8)
	if !strings.Contains(full, "$12.34") {
		t.Fatalf("remainingBar(12.34) = %q, want amount", full)
	}
	if strings.Count(full, "█") != 8 {
		t.Fatalf("remainingBar(12.34) should be full: %q", full)
	}

	empty := remainingBar(0, "USD", 8)
	if strings.Count(empty, "░") != 8 {
		t.Fatalf("remainingBar(0) should be empty: %q", empty)
	}
	if !strings.Contains(empty, "$0.00") {
		t.Fatalf("remainingBar(0) = %q, want amount", empty)
	}
}

func TestRenderWindowRemaining(t *testing.T) {
	remaining := 5.0
	win := Window{Label: "Balance", Remaining: &remaining, Currency: "USD"}
	out := renderWindow(win, time.Now(), 8)
	if !strings.Contains(out, "$5.00") {
		t.Fatalf("renderWindow remaining = %q, want amount", out)
	}
}

func TestRenderWindowShowsLeftNotUsed(t *testing.T) {
	used := 5.0
	out := renderWindow(Window{Label: "Weekly", UsedPercent: &used}, time.Now(), 8)
	if !strings.Contains(out, "95.0% left") {
		t.Fatalf("renderWindow = %q, want 95.0%% left", out)
	}
	if strings.Contains(out, "  5.0% left") {
		t.Fatalf("renderWindow should not show used percent as left: %q", out)
	}
	if !strings.Contains(out, colorDim) {
		t.Fatalf("remaining cells should be dim: %q", out)
	}
	if !strings.Contains(out, colorGreen) {
		t.Fatalf("remaining quota should be green at 95%% left: %q", out)
	}
}

func TestColorForLeftThresholds(t *testing.T) {
	if colorForLeft(41) != colorGreen {
		t.Fatalf("41%% left should be green")
	}
	if colorForLeft(40) != colorYellow {
		t.Fatalf("40%% left should be yellow")
	}
	if colorForLeft(15) != colorYellow {
		t.Fatalf("15%% left should be yellow")
	}
	if colorForLeft(14.9) != colorRed {
		t.Fatalf("14.9%% left should be red")
	}
}

func TestFormatPercent(t *testing.T) {
	if got := FormatPercent(9.348888888888888); got != "9.3%" {
		t.Fatalf("FormatPercent = %q, want 9.3%%", got)
	}
	if got := FormatPercent(0); got != "0.0%" {
		t.Fatalf("FormatPercent(0) = %q, want 0.0%%", got)
	}
	if got := FormatPercent(70); got != "70.0%" {
		t.Fatalf("FormatPercent(70) = %q, want 70.0%%", got)
	}
}

func TestBarFillsByRemaining(t *testing.T) {
	// 5% used → 95% left → 9.5/10 cells rounds to 10 filled.
	out := bar(5, -1, 10)
	if n := strings.Count(out, "█"); n != 10 {
		t.Fatalf("bar(5%% used) filled=%d, want 10: %q", n, out)
	}

	empty := bar(100, -1, 8)
	if strings.Count(empty, "█") != 0 {
		t.Fatalf("bar(100%% used) should have no filled cells: %q", empty)
	}
	if strings.Count(empty, "░") != 8 {
		t.Fatalf("bar(100%% used) should be all dim empty: %q", empty)
	}
	if !strings.Contains(empty, colorDim) {
		t.Fatalf("empty cells should be dim: %q", empty)
	}
}

func TestFormatMoney(t *testing.T) {
	cases := map[string]string{
		"CNY": "¥1.50",
		"USD": "$1.50",
		"EUR": "€1.50",
		"GBP": "£1.50",
		"JPY": "1.50 JPY",
		"":    "1.50",
	}
	for currency, want := range cases {
		if got := FormatMoney(1.5, currency); got != want {
			t.Errorf("FormatMoney(1.5, %q) = %q, want %q", currency, got, want)
		}
	}
}
