package usage

import (
	"fmt"
	"io"
	"math"
	"strings"
	"time"
)

// Render writes a human-readable view of a Report to w, aligning its bars to
// the longest window label in that report.
func Render(w io.Writer, r *Report, now time.Time) {
	RenderAligned(w, r, now, LabelWidth(r))
}

// LabelWidth returns the width needed to align window labels across reports.
// Eight characters is the minimum used by the standard single-provider view.
func LabelWidth(reports ...*Report) int {
	width := 8
	for _, report := range reports {
		if report == nil {
			continue
		}
		for _, win := range report.Windows {
			if len(win.Label) > width {
				width = len(win.Label)
			}
		}
	}
	return width
}

// RenderAligned writes a report using labelWidth for the window-label column.
// Aggregate callers can pass one shared width so bars line up across providers.
func RenderAligned(w io.Writer, r *Report, now time.Time, labelWidth int) {
	if labelWidth < 8 {
		labelWidth = 8
	}
	title := r.Provider
	if r.Plan != "" {
		title = fmt.Sprintf("%s  (%s)", r.Provider, r.Plan)
	}
	fmt.Fprintln(w, title)
	fmt.Fprintln(w, strings.Repeat("─", len(stripANSI(title))))

	if len(r.Windows) == 0 {
		fmt.Fprintln(w, "  no usage windows reported")
	}
	for _, win := range r.Windows {
		fmt.Fprintln(w, renderWindow(win, now, labelWidth))
	}
	for _, f := range r.Extra {
		fmt.Fprintf(w, "  %-14s %s\n", f.Label+":", f.Value)
	}
	if r.ResetCredits != nil {
		renderResetCredits(w, r.ResetCredits, now)
	}
}

// renderResetCredits keeps the normal terminal view glanceable: the summary
// is always shown, followed by at most the three credits that expire first.
func renderResetCredits(w io.Writer, credits *ResetCredits, now time.Time) {
	value := fmt.Sprintf("%d available", credits.AvailableCount)
	if credits.ApplicableAvailableCount != nil {
		value = fmt.Sprintf("%s · %d applicable now", value, *credits.ApplicableAvailableCount)
	}

	expiring := make([]ResetCredit, 0, len(credits.Credits))
	for _, credit := range credits.Credits {
		if credit.ExpiresAt != nil {
			expiring = append(expiring, credit)
		}
	}
	if len(expiring) == 1 && credits.AvailableCount == 1 {
		value += " · expires " + humanizeReset(*expiring[0].ExpiresAt, now)
		fmt.Fprintf(w, "  %-14s %s\n", "Resets:", value)
		return
	}

	fmt.Fprintf(w, "  %-14s %s\n", "Resets:", value)
	for i, credit := range expiring {
		if i == 3 {
			fmt.Fprintf(w, "    +%d more with expiry\n", len(expiring)-i)
			break
		}
		fmt.Fprintf(w, "    #%d             expires %s\n", i+1, humanizeReset(*credit.ExpiresAt, now))
	}
	if credits.DetailsFetched && int64(len(credits.Credits)) < credits.AvailableCount {
		fmt.Fprintf(w, "    details: %d returned (provider reports %d)\n", len(credits.Credits), credits.AvailableCount)
	}
}

func renderWindow(win Window, now time.Time, maxLabel int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "  %-*s ", maxLabel, win.Label)

	pace := win.Pace(now)
	if win.UsedPercent != nil {
		b.WriteString(bar(*win.UsedPercent, pace, 24))
		fmt.Fprintf(&b, " %5.1f%% left", leftPercent(*win.UsedPercent))
	} else if win.Used != nil && win.Limit != nil && *win.Limit > 0 {
		used := float64(*win.Used) / float64(*win.Limit) * 100
		remain := *win.Limit - *win.Used
		if remain < 0 {
			remain = 0
		}
		b.WriteString(bar(used, pace, 24))
		fmt.Fprintf(&b, " %5.1f%% left (%d/%d)", leftPercent(used), remain, *win.Limit)
	} else if win.Remaining != nil {
		b.WriteString(remainingBar(*win.Remaining, win.Currency, 24))
	} else {
		b.WriteString(strings.Repeat(" ", 24) + "     ?")
	}

	if !win.ResetsAt.IsZero() {
		fmt.Fprintf(&b, "   resets %s", humanizeReset(win.ResetsAt, now))
	}
	return b.String()
}

// FormatPercent renders v as a single-decimal percentage, e.g. "9.3%".
func FormatPercent(v float64) string {
	return fmt.Sprintf("%.1f%%", v)
}

// leftPercent is the remaining share of a used-percent window, clamped to 0–100.
func leftPercent(used float64) float64 {
	left := 100 - used
	if left < 0 {
		return 0
	}
	if left > 100 {
		return 100
	}
	return left
}

// bar renders a remaining-quota bar. usedPct is how much has been consumed
// (0–100); the filled cells are what is left. Remaining cells are dim so a
// mostly-empty used window does not look "full". When pace is in [0,1] a cyan
// marker sits at the expected remaining position under even consumption.
func bar(usedPct, pace float64, width int) string {
	left := leftPercent(usedPct)
	filled := int(math.Round(left / 100 * float64(width)))
	color := colorForLeft(left)

	markerIdx := -1
	if pace >= 0 {
		expectedLeft := 1 - pace
		if expectedLeft < 0 {
			expectedLeft = 0
		}
		if expectedLeft > 1 {
			expectedLeft = 1
		}
		markerIdx = int(math.Round(expectedLeft * float64(width)))
		if markerIdx >= width {
			markerIdx = width - 1
		}
	}

	var cells strings.Builder
	cells.WriteByte('[')
	for i := 0; i < width; i++ {
		if i == markerIdx {
			glyph := "▒"
			if i < filled {
				glyph = "▓"
			}
			fmt.Fprintf(&cells, "%s%s%s", colorPace, glyph, colorReset)
			continue
		}
		if i < filled {
			fmt.Fprintf(&cells, "%s█%s", color, colorReset)
		} else {
			fmt.Fprintf(&cells, "%s░%s", colorDim, colorReset)
		}
	}
	cells.WriteByte(']')
	return cells.String()
}

func colorForLeft(left float64) string {
	switch {
	case left < 15:
		return colorRed
	case left <= 40:
		return colorYellow
	default:
		return colorGreen
	}
}

// remainingBar draws a bar for a prepaid balance. The starting amount is
// unknown, so the bar is full whenever any balance remains and empties at
// zero: the amount printed next to it is the source of truth.
func remainingBar(amount float64, currency string, width int) string {
	filled := width
	if amount <= 0 {
		filled = 0
	}
	fillColor := colorRemaining
	if amount <= 0 {
		fillColor = colorRed
	}
	var cells strings.Builder
	cells.WriteByte('[')
	for i := 0; i < filled; i++ {
		fmt.Fprintf(&cells, "%s█%s", fillColor, colorReset)
	}
	for i := filled; i < width; i++ {
		fmt.Fprintf(&cells, "%s░%s", colorDim, colorReset)
	}
	cells.WriteByte(']')
	return fmt.Sprintf("%s %s", cells.String(), FormatMoney(amount, currency))
}

// FormatMoney renders an amount with a familiar symbol for common currencies.
func FormatMoney(amount float64, currency string) string {
	switch strings.ToUpper(currency) {
	case "CNY":
		return fmt.Sprintf("¥%.2f", amount)
	case "USD":
		return fmt.Sprintf("$%.2f", amount)
	case "EUR":
		return fmt.Sprintf("€%.2f", amount)
	case "GBP":
		return fmt.Sprintf("£%.2f", amount)
	default:
		if currency == "" {
			return fmt.Sprintf("%.2f", amount)
		}
		return fmt.Sprintf("%.2f %s", amount, currency)
	}
}

// humanizeReset formats a reset time relative to now, e.g. "in 3h12m (18:40)".
func humanizeReset(t, now time.Time) string {
	d := t.Sub(now)
	if d <= 0 {
		return "now"
	}
	local := t.Local()
	var rel string
	switch {
	case d >= 24*time.Hour:
		days := int(d.Hours()) / 24
		hrs := int(d.Hours()) % 24
		rel = fmt.Sprintf("in %dd%dh", days, hrs)
	case d >= time.Hour:
		rel = fmt.Sprintf("in %dh%dm", int(d.Hours()), int(d.Minutes())%60)
	default:
		rel = fmt.Sprintf("in %dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%s (%s)", rel, local.Format("Mon 15:04"))
}

const (
	colorReset     = "\033[0m"
	colorRed       = "\033[31m"
	colorYellow    = "\033[33m"
	colorGreen     = "\033[32m"
	colorDim       = "\033[90m" // remaining / unused cells
	colorPace      = "\033[96m" // bright cyan — the on-track pace marker
	colorRemaining = "\033[92m" // bright green — a remaining prepaid balance
)

// stripANSI removes color codes for length calculations.
func stripANSI(s string) string { return s }
