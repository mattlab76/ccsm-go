package components

import (
	"fmt"
	"os"
	"strings"

	"github.com/mattlab76/ccsm-go/internal/i18n"
	"github.com/mattlab76/ccsm-go/internal/model"
	"github.com/mattlab76/ccsm-go/internal/ui/styles"
)

// TableMode determines the number of columns.
type TableMode int

const (
	// TableCompact shows 6 columns: #, Date, Subject, Directory, Total In, Total Out.
	TableCompact TableMode = iota
	// TableFull shows 8 columns: #, Date, Subject, Directory, Total In, Total Out, Last In, Last Out.
	TableFull
)

// SessionStatus represents the validation state of a session.
type SessionStatus int

const (
	StatusOK         SessionStatus = iota
	StatusExpired                  // [!] red — session expired at Claude Code
	StatusMissingDir               // [?] amber — working directory missing
)

// TableRow holds data for a single table row.
type TableRow struct {
	Num     int
	Session model.Session
	Status  SessionStatus
}

// columnWidths holds computed column widths.
type columnWidths struct {
	num, date, subj, dir, ti, to, li, lo int
}

func calcCompactWidths(termWidth int) columnWidths {
	w := columnWidths{num: 3, date: 18}
	avail := termWidth - 16
	if avail < 50 {
		avail = 50
	}
	w.ti = avail * 9 / 100
	if w.ti < 10 {
		w.ti = 10
	}
	w.to = w.ti
	remaining := avail - w.num - w.date - w.ti - w.to - 7
	w.subj = remaining * 55 / 100
	if w.subj < 20 {
		w.subj = 20
	}
	w.dir = remaining - w.subj
	if w.dir < 16 {
		w.dir = 16
	}
	return w
}

func calcFullWidths(termWidth int) columnWidths {
	w := columnWidths{num: 3, date: 18}
	avail := termWidth - 16
	if avail < 60 {
		avail = 60
	}
	tokW := avail * 8 / 100
	if tokW < 9 {
		tokW = 9
	}
	w.ti = tokW
	w.to = tokW
	w.li = tokW
	w.lo = tokW
	remaining := avail - w.num - w.date - w.ti - w.to - w.li - w.lo - 9
	w.subj = remaining * 55 / 100
	if w.subj < 16 {
		w.subj = 16
	}
	w.dir = remaining - w.subj
	if w.dir < 14 {
		w.dir = 14
	}
	return w
}

func hline(width int) string {
	return strings.Repeat("─", width)
}

// RenderTable renders a bordered session table.
func RenderTable(rows []TableRow, mode TableMode, termWidth int) string {
	if len(rows) == 0 {
		return ""
	}

	var w columnWidths
	if mode == TableCompact {
		w = calcCompactWidths(termWidth)
	} else {
		w = calcFullWidths(termWidth)
	}

	var b strings.Builder

	// Top border
	if mode == TableCompact {
		fmt.Fprintf(&b, "┌%s┬%s┬%s┬%s┬%s┬%s┐\n",
			hline(w.num), hline(w.date), hline(w.subj), hline(w.dir), hline(w.ti), hline(w.to))
	} else {
		fmt.Fprintf(&b, "┌%s┬%s┬%s┬%s┬%s┬%s┬%s┬%s┐\n",
			hline(w.num), hline(w.date), hline(w.subj), hline(w.dir),
			hline(w.ti), hline(w.to), hline(w.li), hline(w.lo))
	}

	// Header row
	v := styles.Violet
	if mode == TableCompact {
		fmt.Fprintf(&b, "│%s│%s│%s│%s│%s│%s│\n",
			v.Render(padRight(" "+i18n.T("header_nr"), w.num)),
			v.Render(padRight(" "+i18n.T("header_date"), w.date)),
			v.Render(padRight(" "+i18n.T("header_subject"), w.subj)),
			v.Render(padRight(" "+i18n.T("header_dir"), w.dir)),
			v.Render(padLeft(i18n.T("header_in")+" ", w.ti)),
			v.Render(padLeft(i18n.T("header_out")+" ", w.to)))
	} else {
		fmt.Fprintf(&b, "│%s│%s│%s│%s│%s│%s│%s│%s│\n",
			v.Render(padRight(" "+i18n.T("header_nr"), w.num)),
			v.Render(padRight(" "+i18n.T("header_date"), w.date)),
			v.Render(padRight(" "+i18n.T("header_subject"), w.subj)),
			v.Render(padRight(" "+i18n.T("header_dir"), w.dir)),
			v.Render(padLeft("Total "+i18n.T("header_in")+" ", w.ti)),
			v.Render(padLeft("Total "+i18n.T("header_out")+" ", w.to)),
			v.Render(padLeft("Last "+i18n.T("header_in")+" ", w.li)),
			v.Render(padLeft("Last "+i18n.T("header_out")+" ", w.lo)))
	}

	// Mid border
	if mode == TableCompact {
		fmt.Fprintf(&b, "├%s┼%s┼%s┼%s┼%s┼%s┤\n",
			hline(w.num), hline(w.date), hline(w.subj), hline(w.dir), hline(w.ti), hline(w.to))
	} else {
		fmt.Fprintf(&b, "├%s┼%s┼%s┼%s┼%s┼%s┼%s┼%s┤\n",
			hline(w.num), hline(w.date), hline(w.subj), hline(w.dir),
			hline(w.ti), hline(w.to), hline(w.li), hline(w.lo))
	}

	// Data rows
	for _, row := range rows {
		s := row.Session
		dateStr := s.CreatedAt.Format("2006-01-02 15:04")

		// Subject with status prefix
		subj := s.Subject
		switch row.Status {
		case StatusExpired:
			subj = "[!] " + subj
		case StatusMissingDir:
			subj = "[?] " + subj
		}
		subj = truncate(subj, w.subj-2)

		// Directory: replace HOME with ~, truncate from left
		dir := ShortenPath(s.CWD, w.dir-2)

		numStr := fmt.Sprintf(" %d", row.Num)
		ti := FormatTokens(s.TotalInputTokens)
		to := FormatTokens(s.TotalOutputTokens)

		var line string
		if mode == TableCompact {
			line = fmt.Sprintf("│%s│%s│%s│%s│%s│%s│",
				padRight(numStr, w.num),
				padRight(" "+dateStr, w.date),
				padRight(" "+subj, w.subj),
				padRight(" "+dir, w.dir),
				padLeft(ti+" ", w.ti),
				padLeft(to+" ", w.to))
		} else {
			li := FormatTokens(s.LastInputTokens)
			lo := FormatTokens(s.LastOutputTokens)
			line = fmt.Sprintf("│%s│%s│%s│%s│%s│%s│%s│%s│",
				padRight(numStr, w.num),
				padRight(" "+dateStr, w.date),
				padRight(" "+subj, w.subj),
				padRight(" "+dir, w.dir),
				padLeft(ti+" ", w.ti),
				padLeft(to+" ", w.to),
				padLeft(li+" ", w.li),
				padLeft(lo+" ", w.lo))
		}

		// Apply row color for status
		switch row.Status {
		case StatusExpired:
			line = styles.Red.Render(line)
		case StatusMissingDir:
			line = styles.Amber.Render(line)
		}

		b.WriteString(line + "\n")
	}

	// Bottom border
	if mode == TableCompact {
		fmt.Fprintf(&b, "└%s┴%s┴%s┴%s┴%s┴%s┘",
			hline(w.num), hline(w.date), hline(w.subj), hline(w.dir), hline(w.ti), hline(w.to))
	} else {
		fmt.Fprintf(&b, "└%s┴%s┴%s┴%s┴%s┴%s┴%s┴%s┘",
			hline(w.num), hline(w.date), hline(w.subj), hline(w.dir),
			hline(w.ti), hline(w.to), hline(w.li), hline(w.lo))
	}

	return b.String()
}

// RenderLegend returns the legend text if any rows have non-OK status.
func RenderLegend(rows []TableRow) string {
	hasExpired := false
	hasMissing := false
	for _, r := range rows {
		switch r.Status {
		case StatusExpired:
			hasExpired = true
		case StatusMissingDir:
			hasMissing = true
		}
	}
	if !hasExpired && !hasMissing {
		return ""
	}
	var parts []string
	if hasExpired {
		parts = append(parts, styles.Red.Render(i18n.T("legend_expired")))
	}
	if hasMissing {
		parts = append(parts, styles.Amber.Render(i18n.T("legend_missing_dir")))
	}
	return "  " + strings.Join(parts, "  ")
}

// FormatTokens formats a token count: 1353202 → "1.4M", 47567 → "47.6k", 102 → "102".
func FormatTokens(n int64) string {
	if n <= 0 {
		return "0"
	}
	if n >= 1_000_000 {
		whole := n / 100_000
		return fmt.Sprintf("%d.%dM", whole/10, whole%10)
	}
	if n >= 1_000 {
		whole := n / 100
		return fmt.Sprintf("%d.%dk", whole/10, whole%10)
	}
	return fmt.Sprintf("%d", n)
}

// truncate shortens a string to maxLen, appending ".." if truncated.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 2 {
		return ".."
	}
	return string(runes[:maxLen-2]) + ".."
}

// ShortenPath replaces HOME with ~ and truncates long paths from the left.
func ShortenPath(path string, maxLen int) string {
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(path, home) {
		path = "~" + path[len(home):]
	}
	runes := []rune(path)
	if len(runes) <= maxLen {
		return path
	}
	if maxLen <= 4 {
		return ".."
	}
	// Truncate from left: ".." + last (maxLen-2) chars
	return ".." + string(runes[len(runes)-(maxLen-2):])
}

func padRight(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return string(runes[:width])
	}
	return s + strings.Repeat(" ", width-len(runes))
}

func padLeft(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return string(runes[:width])
	}
	return strings.Repeat(" ", width-len(runes)) + s
}
