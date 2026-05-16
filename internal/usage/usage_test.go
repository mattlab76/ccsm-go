package usage

import (
	"strings"
	"testing"
	"time"
)

func TestFormatUSD(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{0, "$0.00"},
		{0.1, "$0.10"},
		{9.99, "$9.99"},
		{10, "$10"},
		{99, "$99"},
		{100, "$100"},
		{999, "$999"},
		{1000, "$1.0k"},
		{1234, "$1.2k"},
		{12345, "$12.3k"},
	}
	for _, tt := range tests {
		if got := FormatUSD(tt.in); got != tt.want {
			t.Errorf("FormatUSD(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFormatMoney(t *testing.T) {
	tests := []struct {
		name     string
		usd      float64
		currency string
		rate     float64
		want     string
	}{
		{"USD passthrough", 100, "USD", 1.0, "$100"},
		{"empty currency falls back to USD", 100, "", 1.0, "$100"},
		{"zero rate falls back to USD", 100, "EUR", 0, "$100"},
		{"negative rate falls back to USD", 100, "EUR", -1, "$100"},
		{"EUR conversion", 100, "EUR", 0.92, "€92"},
		{"EUR small amount", 1, "EUR", 0.92, "€0.92"},
		{"EUR large amount", 5000, "EUR", 0.92, "€4.6k"},
		{"GBP", 100, "GBP", 0.79, "£79"},
		{"JPY", 100, "JPY", 155, "¥15.5k"},
		{"CHF", 100, "CHF", 0.88, "CHF 88"},
		{"unknown currency uses code prefix", 100, "PLN", 4.0, "PLN 400"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatMoney(tt.usd, tt.currency, tt.rate); got != tt.want {
				t.Errorf("FormatMoney(%v, %q, %v) = %q, want %q",
					tt.usd, tt.currency, tt.rate, got, tt.want)
			}
		})
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{999, "999"},
		{1_000, "1.0k"},
		{1_500, "1.5k"},
		{999_999, "1000.0k"},
		{1_000_000, "1.0M"},
		{14_500_000, "14.5M"},
	}
	for _, tt := range tests {
		if got := FormatTokens(tt.in); got != tt.want {
			t.Errorf("FormatTokens(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestModelShort(t *testing.T) {
	tests := map[string]string{
		"claude-opus-4-7":            "opus 4.7",
		"claude-opus-4-6":            "opus 4.6",
		"claude-opus-3-5-20240620":   "opus",
		"claude-sonnet-4-6":          "sonnet 4.6",
		"claude-sonnet-4-5-20250929": "sonnet 4.5",
		"claude-sonnet-3-5":          "sonnet",
		"claude-haiku-4-5-20251001":  "haiku 4.5",
		"claude-haiku-3":             "haiku",
		"":                           "",
		"unknown-model":              "unknown-model",
	}
	for in, want := range tests {
		if got := ModelShort(in); got != want {
			t.Errorf("ModelShort(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPricingFor(t *testing.T) {
	// Opus prefix should map to the opus pricing tuple.
	if p := pricingFor("claude-opus-4-7"); p.in != 15.00 || p.out != 75.00 {
		t.Errorf("opus pricing wrong: %+v", p)
	}
	// Sonnet.
	if p := pricingFor("claude-sonnet-4-6"); p.in != 3.00 || p.out != 15.00 {
		t.Errorf("sonnet pricing wrong: %+v", p)
	}
	// Haiku.
	if p := pricingFor("claude-haiku-4-5"); p.in != 1.00 || p.out != 5.00 {
		t.Errorf("haiku pricing wrong: %+v", p)
	}
	// Unknown model returns zero pricing — better to under-report than invent.
	if p := pricingFor("gpt-4"); p.in != 0 || p.out != 0 {
		t.Errorf("unknown model pricing should be zero, got: %+v", p)
	}
}

func TestProgressBar(t *testing.T) {
	tests := []struct {
		name  string
		pct   float64
		width int
		wantPrefix string
		wantLen    int // visible width (not byte count)
	}{
		{"empty bar", 0, 10, "          ", 10},
		{"full bar", 100, 10, "██████████", 10},
		{"half bar", 50, 10, "█████", 10},
		{"over 100 caps at full", 250, 10, "██████████", 10},
		{"negative caps at zero", -10, 10, "          ", 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProgressBar(tt.pct, tt.width)
			if !strings.HasPrefix(got, tt.wantPrefix) {
				t.Errorf("ProgressBar(%v, %d) = %q, want prefix %q",
					tt.pct, tt.width, got, tt.wantPrefix)
			}
			// Visible-width check; half-blocks may add a non-space rune.
			runes := []rune(strings.TrimRight(got, " "))
			if len(runes) > tt.wantLen {
				t.Errorf("ProgressBar wider than width: %d runes, max %d",
					len(runes), tt.wantLen)
			}
		})
	}
}

func TestProgressBarTinyWidth(t *testing.T) {
	// Anything below 4 should be promoted to 4 so the bar stays usable.
	bar := ProgressBar(50, 1)
	if len([]rune(bar)) != 4 {
		t.Errorf("ProgressBar(50, 1) should pad to width 4, got %d runes: %q",
			len([]rune(bar)), bar)
	}
}

func TestFormatReset(t *testing.T) {
	if got := FormatReset(time.Time{}); got != "" {
		t.Errorf("zero time should format to empty, got %q", got)
	}
	// Same-day formatting is "15:04".
	now := time.Now()
	sameDay := time.Date(now.Year(), now.Month(), now.Day(), 10, 30, 0, 0, time.Local)
	got := FormatReset(sameDay)
	if got != "10:30" {
		t.Errorf("same-day reset = %q, want %q", got, "10:30")
	}
	// Different day uses longer format with weekday + month/day + time.
	tomorrow := now.AddDate(0, 0, 1)
	got = FormatReset(tomorrow)
	if !strings.Contains(got, ",") || len(got) < 10 {
		t.Errorf("different-day reset should include date, got %q", got)
	}
}

func TestCurrencySymbol(t *testing.T) {
	tests := map[string]string{
		"EUR": "€",
		"GBP": "£",
		"JPY": "¥",
		"CHF": "CHF ",
		"USD": "$",
		"":    "$",
		"PLN": "PLN ", // fallback adds space
	}
	for in, want := range tests {
		if got := currencySymbol(in); got != want {
			t.Errorf("currencySymbol(%q) = %q, want %q", in, got, want)
		}
	}
}
