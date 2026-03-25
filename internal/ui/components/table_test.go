package components

import (
	"strings"
	"testing"
	"time"

	"github.com/mattlab76/ccsm-go/internal/model"
)

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0"},
		{-1, "0"},
		{102, "102"},
		{999, "999"},
		{1000, "1.0k"},
		{1353, "1.3k"},
		{47567, "47.5k"},
		{999999, "999.9k"},
		{1000000, "1.0M"},
		{1353202, "1.3M"},
		{9473568, "9.4M"},
	}
	for _, tt := range tests {
		got := FormatTokens(tt.input)
		if got != tt.want {
			t.Errorf("FormatTokens(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("Hello World", 20); got != "Hello World" {
		t.Errorf("truncate short = %q", got)
	}
	if got := truncate("Hello World", 7); got != "Hello.." {
		t.Errorf("truncate long = %q, want %q", got, "Hello..")
	}
	if got := truncate("AB", 2); got != "AB" {
		t.Errorf("truncate exact = %q", got)
	}
}

func TestShortenPath(t *testing.T) {
	// Long path should be truncated from the left.
	long := "/very/long/path/to/some/project/directory"
	got := ShortenPath(long, 20)
	if !strings.HasPrefix(got, "..") {
		t.Errorf("ShortenPath should start with .., got %q", got)
	}
	if len([]rune(got)) > 20 {
		t.Errorf("ShortenPath too long: %d runes", len([]rune(got)))
	}
}

func TestRenderTable_Empty(t *testing.T) {
	result := RenderTable(nil, TableCompact, 100)
	if result != "" {
		t.Errorf("empty table should return empty string, got %q", result)
	}
}

func TestRenderTable_Compact(t *testing.T) {
	rows := []TableRow{
		{Num: 1, Session: model.Session{
			SID: "test-1", CWD: "/tmp/test", Subject: "Test session",
			CreatedAt: time.Date(2026, 3, 24, 10, 30, 0, 0, time.UTC),
			TotalInputTokens: 1500, TotalOutputTokens: 200,
		}, Status: StatusOK},
	}
	result := RenderTable(rows, TableCompact, 100)
	if !strings.Contains(result, "Test session") {
		t.Error("table should contain subject")
	}
	if !strings.Contains(result, "┌") && !strings.Contains(result, "└") {
		t.Error("table should have box-drawing borders")
	}
}

func TestRenderTable_Full(t *testing.T) {
	rows := []TableRow{
		{Num: 1, Session: model.Session{
			SID: "test-1", CWD: "/tmp/test", Subject: "Full table test",
			CreatedAt: time.Date(2026, 3, 24, 10, 30, 0, 0, time.UTC),
			TotalInputTokens: 1000000, TotalOutputTokens: 5000,
			LastInputTokens: 500000, LastOutputTokens: 2500,
		}, Status: StatusOK},
	}
	result := RenderTable(rows, TableFull, 120)
	if !strings.Contains(result, "Full table test") {
		t.Error("full table should contain subject")
	}
}

func TestRenderLegend(t *testing.T) {
	// No markers → no legend.
	rows := []TableRow{{Status: StatusOK}}
	if got := RenderLegend(rows); got != "" {
		t.Errorf("no-marker legend should be empty, got %q", got)
	}

	// With expired marker.
	rows = []TableRow{{Status: StatusExpired}}
	got := RenderLegend(rows)
	if got == "" {
		t.Error("expired legend should not be empty")
	}
}
