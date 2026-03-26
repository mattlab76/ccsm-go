package components

import (
	"strings"

	"github.com/mattlab76/ccsm-go/internal/ui/styles"
)

// TagComplete provides tag autocompletion from a list of known tags.
type TagComplete struct {
	KnownTags   []string // all known tags from DB
	Suggestions []string
	Selected    int
	Active      bool
}

// Complete generates tag suggestions based on the current word being typed.
func (t *TagComplete) Complete(input string) {
	t.Suggestions = nil
	t.Selected = 0

	// Find the last word being typed (tags are space-separated).
	words := strings.Fields(input)
	if len(words) == 0 || strings.HasSuffix(input, " ") {
		t.Active = false
		return
	}
	current := words[len(words)-1]
	if current == "" {
		t.Active = false
		return
	}

	// Build set of already-used tags in this input.
	used := make(map[string]bool)
	for _, w := range words[:len(words)-1] {
		used[strings.ToLower(w)] = true
	}

	// Find matching tags.
	lower := strings.ToLower(current)
	for _, tag := range t.KnownTags {
		if used[strings.ToLower(tag)] {
			continue // Already used.
		}
		if strings.HasPrefix(strings.ToLower(tag), lower) && strings.ToLower(tag) != lower {
			t.Suggestions = append(t.Suggestions, tag)
		}
	}

	t.Active = len(t.Suggestions) > 0
	if len(t.Suggestions) > 6 {
		t.Suggestions = t.Suggestions[:6]
	}
}

// Accept returns the full input with the current suggestion replacing the last word.
func (t *TagComplete) Accept(input string) string {
	if !t.Active || t.Selected >= len(t.Suggestions) {
		return input
	}
	words := strings.Fields(input)
	if len(words) == 0 {
		return t.Suggestions[t.Selected] + " "
	}
	// Replace last word with suggestion.
	words[len(words)-1] = t.Suggestions[t.Selected]
	t.Active = false
	t.Suggestions = nil
	return strings.Join(words, " ") + " "
}

// Next moves to the next suggestion.
func (t *TagComplete) Next() {
	if len(t.Suggestions) == 0 {
		return
	}
	t.Selected = (t.Selected + 1) % len(t.Suggestions)
}

// Prev moves to the previous suggestion.
func (t *TagComplete) Prev() {
	if len(t.Suggestions) == 0 {
		return
	}
	t.Selected--
	if t.Selected < 0 {
		t.Selected = len(t.Suggestions) - 1
	}
}

// Reset clears suggestions.
func (t *TagComplete) Reset() {
	t.Suggestions = nil
	t.Selected = 0
	t.Active = false
}

// View renders the suggestion list.
func (t *TagComplete) View() string {
	if !t.Active || len(t.Suggestions) == 0 {
		return ""
	}
	var b strings.Builder
	for i, s := range t.Suggestions {
		if i == t.Selected {
			b.WriteString("  " + styles.SelectedRow.Render(" "+s+" ") + "\n")
		} else {
			b.WriteString("  " + styles.Dim.Render(" "+s) + "\n")
		}
	}
	return b.String()
}
