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

// Logo returns the ASCII art logo lines (without color).
const (
	LogoLine1 = "╔═╗ ╔═╗ ╔═╗ ╔╦╗"
	LogoLine2 = "║   ║   ╚═╗ ║║║"
	LogoLine3 = "╚═╝ ╚═╝ ╚═╝ ╩ ╩"
)

// RenderLogo returns the colored logo with title and session count.
func RenderLogo(title string, count int) string {
	countStr := Dim.Render(formatCount(count))
	return Violet.Render(LogoLine1) + "   " + Bold.Render(title) + "        " + countStr + "\n" +
		Violet.Render(LogoLine2) + "\n" +
		Violet.Render(LogoLine3)
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
