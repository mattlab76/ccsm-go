package styles

import "github.com/charmbracelet/lipgloss"

// Color constants matching the bash version.
var (
	Violet = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))
	Teal   = lipgloss.NewStyle().Foreground(lipgloss.Color("36"))
	Green  = lipgloss.NewStyle().Foreground(lipgloss.Color("35"))
	Amber  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	Red    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	Dim    = lipgloss.NewStyle().Faint(true)
	Bold   = lipgloss.NewStyle().Bold(true)

	// Combined styles.
	TealBold   = lipgloss.NewStyle().Foreground(lipgloss.Color("36")).Bold(true)
	VioletBold = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)

	// Selected row highlight: reverse video (white on violet).
	SelectedRow = lipgloss.NewStyle().Background(lipgloss.Color("99")).Foreground(lipgloss.Color("255")).Bold(true)

	// Status bar styles.
	StatusBarStyle  = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("252"))
	StatusBarKey    = lipgloss.NewStyle().Background(lipgloss.Color("99")).Foreground(lipgloss.Color("255")).Bold(true)
	StatusBarAction = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("252"))
)

// 6-line block letters spelling "CCSM" — bigger and more recognisable
// than the previous compact 3-line variant.
const (
	LogoLine1 = ` ██████╗ ██████╗ ███████╗███╗   ███╗`
	LogoLine2 = `██╔════╝██╔════╝██╔════╝████╗ ████║`
	LogoLine3 = `██║     ██║     ███████╗██╔████╔██║`
	LogoLine4 = `██║     ██║     ╚════██║██║╚██╔╝██║`
	LogoLine5 = `╚██████╗╚██████╗███████║██║ ╚═╝ ██║`
	LogoLine6 = ` ╚═════╝ ╚═════╝╚══════╝╚═╝     ╚═╝`
)

// RenderLogo returns the colored block logo with title + session count.
//
// Layout: title sits to the right of line 3 (vertically centered),
// count one line below. Callers prepend a 2-space indent to the
// returned string; this function continues that indent on lines 2–6
// so the whole block lines up.
//
//	   ██████╗ ██████╗ ███████╗███╗   ███╗
//	  ██╔════╝██╔════╝██╔════╝████╗ ████║
//	  ██║     ██║     ███████╗██╔████╔██║   Title here
//	  ██║     ██║     ╚════██║██║╚██╔╝██║   N session(s)
//	  ╚██████╗╚██████╗███████║██║ ╚═╝ ██║
//	   ╚═════╝ ╚═════╝╚══════╝╚═╝     ╚═╝
func RenderLogo(title string, count int) string {
	countStr := Dim.Render(formatCount(count))
	const gap = "   "
	const indent = "  " // matches the 2-space prefix every caller prepends
	return VioletBold.Render(LogoLine1) + "\n" +
		indent + VioletBold.Render(LogoLine2) + "\n" +
		indent + VioletBold.Render(LogoLine3) + gap + Bold.Render(title) + "\n" +
		indent + VioletBold.Render(LogoLine4) + gap + countStr + "\n" +
		indent + VioletBold.Render(LogoLine5) + "\n" +
		indent + VioletBold.Render(LogoLine6)
}

func formatCount(n int) string {
	if n == 1 {
		return "1 session"
	}
	return lipgloss.NewStyle().Render(intToStr(n) + " session(s)")
}

func intToStr(n int) string {
	// Simple int to string without importing strconv.
	if n == 0 {
		return "0"
	}
	s := ""
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

// DoubleLine returns a double-line separator of the given width.
func DoubleLine(width int) string {
	s := ""
	for range width {
		s += "═"
	}
	return s
}
