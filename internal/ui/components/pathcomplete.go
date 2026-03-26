package components

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mattlab76/ccsm-go/internal/ui/styles"
)

// PathComplete provides filesystem path completion.
type PathComplete struct {
	Suggestions []string
	Selected    int // currently highlighted suggestion
	Active      bool
}

// Complete generates path suggestions for the given input.
// Returns the common prefix and list of completions.
func (p *PathComplete) Complete(input string) {
	p.Suggestions = nil
	p.Selected = 0

	if input == "" {
		p.Active = false
		return
	}

	// Expand ~
	expanded := input
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(expanded, "~") {
		expanded = filepath.Join(home, expanded[1:])
	}

	// Get directory and prefix to match.
	dir := filepath.Dir(expanded)
	prefix := filepath.Base(expanded)

	// If input ends with /, list contents of that directory.
	if strings.HasSuffix(input, "/") || strings.HasSuffix(input, string(os.PathSeparator)) {
		dir = expanded
		prefix = ""
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		p.Active = false
		return
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue // Only complete directories.
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(prefix, ".") {
			continue // Skip hidden dirs unless user typed a dot.
		}
		if prefix == "" || strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			fullPath := filepath.Join(dir, name)
			// Show with ~ for home directory.
			display := fullPath
			if home != "" && strings.HasPrefix(display, home) {
				display = "~" + display[len(home):]
			}
			p.Suggestions = append(p.Suggestions, display)
		}
	}

	p.Active = len(p.Suggestions) > 0

	// Limit suggestions shown.
	if len(p.Suggestions) > 8 {
		p.Suggestions = p.Suggestions[:8]
	}
}

// Accept returns the currently selected suggestion (expanded) and resets state.
func (p *PathComplete) Accept() string {
	if !p.Active || p.Selected >= len(p.Suggestions) {
		return ""
	}
	result := p.Suggestions[p.Selected]
	// Expand ~ back to actual path.
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(result, "~") {
		result = filepath.Join(home, result[1:])
	}
	p.Active = false
	p.Suggestions = nil
	return result + "/"
}

// Next moves to the next suggestion.
func (p *PathComplete) Next() {
	if len(p.Suggestions) == 0 {
		return
	}
	p.Selected = (p.Selected + 1) % len(p.Suggestions)
}

// Prev moves to the previous suggestion.
func (p *PathComplete) Prev() {
	if len(p.Suggestions) == 0 {
		return
	}
	p.Selected--
	if p.Selected < 0 {
		p.Selected = len(p.Suggestions) - 1
	}
}

// Reset clears all suggestions.
func (p *PathComplete) Reset() {
	p.Suggestions = nil
	p.Selected = 0
	p.Active = false
}

// View renders the suggestion list.
func (p *PathComplete) View() string {
	if !p.Active || len(p.Suggestions) == 0 {
		return ""
	}
	var b strings.Builder
	for i, s := range p.Suggestions {
		if i == p.Selected {
			b.WriteString("  " + styles.SelectedRow.Render(" "+s+" ") + "\n")
		} else {
			b.WriteString("  " + styles.Dim.Render(" "+s) + "\n")
		}
	}
	return b.String()
}
