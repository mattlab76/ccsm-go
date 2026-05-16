package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattlab76/ccsm-go/internal/ui/styles"
)

// Col defines one column in a generic key/value table.
type Col struct {
	Header string
	Width  int  // inner width (excluding the surrounding spaces)
	Right  bool // right-align the cell contents (use for numbers)
}

// RenderKVTable renders an ASCII-bordered table with the given columns and
// rows. Cell strings may contain ANSI escape sequences (styled colors);
// lipgloss.Width is used to measure visible widths so padding stays correct.
//
// If a cell is wider than the column, it overflows — pre-truncate at the
// caller if you need hard limits.
func RenderKVTable(cols []Col, rows [][]string) string {
	if len(cols) == 0 {
		return ""
	}
	var b strings.Builder

	writeBorder(&b, cols, "┌", "┬", "┐")
	b.WriteString("\n│")
	for _, c := range cols {
		b.WriteString(" " + styles.Violet.Render(padCell(c.Header, c.Width, c.Right)) + " │")
	}
	b.WriteString("\n")
	writeBorder(&b, cols, "├", "┼", "┤")
	b.WriteString("\n")

	for _, row := range rows {
		b.WriteString("│")
		for i, c := range cols {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			b.WriteString(" " + padCell(cell, c.Width, c.Right) + " │")
		}
		b.WriteString("\n")
	}
	writeBorder(&b, cols, "└", "┴", "┘")
	return b.String()
}

func writeBorder(b *strings.Builder, cols []Col, left, mid, right string) {
	b.WriteString(left)
	for i, c := range cols {
		b.WriteString(strings.Repeat("─", c.Width+2))
		if i < len(cols)-1 {
			b.WriteString(mid)
		}
	}
	b.WriteString(right)
}

// padCell pads s to width, accounting for ANSI escape codes via lipgloss.Width.
// If s is already wider, returns s unchanged (overflow is the caller's problem).
func padCell(s string, width int, right bool) string {
	visible := lipgloss.Width(s)
	if visible >= width {
		return s
	}
	pad := strings.Repeat(" ", width-visible)
	if right {
		return pad + s
	}
	return s + pad
}
