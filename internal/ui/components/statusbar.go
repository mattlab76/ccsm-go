package components

import (
	"strings"

	"github.com/mattlab76/ccsm-go/internal/ui/styles"
)

// StatusBar renders a fixed bottom bar with key binding hints.
// It fills the full terminal width with a dim background.
func StatusBar(hints string, width int) string {
	if width <= 0 {
		width = 80
	}
	// Pad to full width.
	runes := []rune(hints)
	if len(runes) < width {
		hints += strings.Repeat(" ", width-len(runes))
	}
	return styles.StatusBarStyle.Render(hints)
}

// StatusBarItems formats key-action pairs into a hint string.
// Example: StatusBarItems("↑↓", "navigate", "Enter", "select", "q", "back")
func StatusBarItems(pairs ...string) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		key := pairs[i]
		action := pairs[i+1]
		parts = append(parts, styles.StatusBarKey.Render(" "+key+" ")+styles.StatusBarAction.Render(" "+action+" "))
	}
	return strings.Join(parts, " ")
}
