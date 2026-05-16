package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// stripANSI removes ANSI escape sequences so we can assert on
// rendered text without color noise.
func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == 0x1b {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func TestRenderKVTable_EmptyColsReturnsEmpty(t *testing.T) {
	if got := RenderKVTable(nil, nil); got != "" {
		t.Errorf("expected empty string for nil cols, got %q", got)
	}
}

func TestRenderKVTable_RendersHeadersAndRows(t *testing.T) {
	cols := []Col{
		{Header: "Name", Width: 8},
		{Header: "Age", Width: 4, Right: true},
	}
	rows := [][]string{
		{"Alice", "30"},
		{"Bob", "7"},
	}
	out := stripANSI(RenderKVTable(cols, rows))

	// Header row should contain both labels.
	if !strings.Contains(out, "Name") {
		t.Errorf("missing Name header in: %s", out)
	}
	if !strings.Contains(out, "Age") {
		t.Errorf("missing Age header in: %s", out)
	}
	// Data rows present.
	if !strings.Contains(out, "Alice") || !strings.Contains(out, "Bob") {
		t.Errorf("missing data rows in: %s", out)
	}
	// Top, mid, and bottom borders use box-drawing chars.
	for _, want := range []string{"┌", "┬", "┐", "├", "┼", "┤", "└", "┴", "┘"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing border glyph %q in output", want)
		}
	}
}

func TestRenderKVTable_RightAlignment(t *testing.T) {
	cols := []Col{
		{Header: "Qty", Width: 6, Right: true},
	}
	rows := [][]string{{"5"}}
	out := stripANSI(RenderKVTable(cols, rows))

	// Find the data line — it's the only one containing "5".
	var line string
	for _, l := range strings.Split(out, "\n") {
		if strings.Contains(l, " 5 ") || strings.HasSuffix(strings.TrimRight(l, " │"), "5") {
			line = l
			break
		}
	}
	if line == "" {
		t.Fatalf("could not find data line in: %s", out)
	}
	// Right-aligned: the "5" should be near the right edge of the cell.
	// Cell content is " " + padded + " ", so for width=6, right-aligned
	// "5" sits as "     5".
	if !strings.Contains(line, "     5") {
		t.Errorf("expected right-aligned cell, got: %q", line)
	}
}

func TestRenderKVTable_HandlesFewerCellsThanCols(t *testing.T) {
	cols := []Col{
		{Header: "A", Width: 4},
		{Header: "B", Width: 4},
		{Header: "C", Width: 4},
	}
	rows := [][]string{
		{"a", "b"}, // only 2 of 3 cells supplied
	}
	out := stripANSI(RenderKVTable(cols, rows))
	if !strings.Contains(out, "a") || !strings.Contains(out, "b") {
		t.Errorf("data missing in: %s", out)
	}
	// Missing cell should render as empty padding (no panic, no overflow).
	lines := strings.Split(out, "\n")
	for _, l := range lines {
		// Each border-bounded row should have exactly 3 inner cells
		// (4 vertical bars).
		if strings.HasPrefix(l, "│") {
			if c := strings.Count(l, "│"); c != 4 {
				t.Errorf("row should have 4 vertical bars (3 cells), got %d: %q", c, l)
			}
		}
	}
}

func TestPadCell_RespectsANSIWidth(t *testing.T) {
	// padCell uses lipgloss.Width to measure visible width, so styled
	// text shouldn't be over-padded.
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Render("hi")
	visible := lipgloss.Width(styled)
	if visible != 2 {
		t.Fatalf("setup: expected visible width 2, got %d", visible)
	}
	got := padCell(styled, 6, false)
	if lipgloss.Width(got) != 6 {
		t.Errorf("padded cell width = %d, want 6 (input: %q)",
			lipgloss.Width(got), got)
	}
}

func TestPadCell_OverflowReturnsUnchanged(t *testing.T) {
	// When the cell content is wider than the column, the helper just
	// returns it unchanged — the caller is responsible for truncation.
	overflow := "much-too-wide-for-cell"
	got := padCell(overflow, 5, false)
	if got != overflow {
		t.Errorf("overflowing cell should be returned unchanged, got %q", got)
	}
}
