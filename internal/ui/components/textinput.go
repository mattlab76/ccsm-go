package components

import (
	"unicode"
	"unicode/utf8"

	"github.com/mattlab76/ccsm-go/internal/ui/styles"
)

// TextInput is a reusable text input with cursor navigation.
type TextInput struct {
	Value       string
	Pos         int // cursor position (0 = before first char)
	Placeholder string
}

// NewTextInput creates a new text input.
func NewTextInput(placeholder string) TextInput {
	return TextInput{Placeholder: placeholder}
}

// SetValue replaces the entire value and moves cursor to end.
func (t *TextInput) SetValue(s string) {
	t.Value = s
	t.Pos = len([]rune(s))
}

// Reset clears the input.
func (t *TextInput) Reset() {
	t.Value = ""
	t.Pos = 0
}

// HandleKey processes a key event. Returns true if the key was handled.
func (t *TextInput) HandleKey(key string) bool {
	runes := []rune(t.Value)

	switch key {
	case "left":
		if t.Pos > 0 {
			t.Pos--
		}
	case "right":
		if t.Pos < len(runes) {
			t.Pos++
		}
	case "home", "ctrl+a":
		t.Pos = 0
	case "end", "ctrl+e":
		t.Pos = len(runes)
	case "ctrl+left", "alt+b":
		// Jump word left.
		t.Pos = t.wordLeft(runes)
	case "ctrl+right", "alt+f":
		// Jump word right.
		t.Pos = t.wordRight(runes)
	case "backspace":
		if t.Pos > 0 {
			runes = append(runes[:t.Pos-1], runes[t.Pos:]...)
			t.Pos--
			t.Value = string(runes)
		}
	case "delete", "ctrl+d":
		if t.Pos < len(runes) {
			runes = append(runes[:t.Pos], runes[t.Pos+1:]...)
			t.Value = string(runes)
		}
	case "ctrl+k":
		// Kill to end of line.
		t.Value = string(runes[:t.Pos])
	case "ctrl+u":
		// Kill to start of line.
		t.Value = string(runes[t.Pos:])
		t.Pos = 0
	case "ctrl+w":
		// Kill word backward.
		newPos := t.wordLeft(runes)
		t.Value = string(append(runes[:newPos], runes[t.Pos:]...))
		t.Pos = newPos
	default:
		// Insert a typed character. A typed character arrives as exactly
		// one rune — including multi-byte runes like umlauts (ä, ö, ü) and
		// € — so measure runes, not bytes. The old len(key)==1 byte check
		// silently dropped every non-ASCII character.
		if key == "space" {
			runes = insertRune(runes, t.Pos, ' ')
			t.Pos++
			t.Value = string(runes)
		} else if utf8.RuneCountInString(key) == 1 {
			r := []rune(key)[0]
			runes = insertRune(runes, t.Pos, r)
			t.Pos++
			t.Value = string(runes)
		} else {
			return false // Named key (enter, tab, up, ...) — not handled.
		}
	}
	return true
}

// View renders the input with cursor.
func (t *TextInput) View() string {
	if t.Value == "" && t.Placeholder != "" {
		return styles.Dim.Render(t.Placeholder) + "█"
	}
	runes := []rune(t.Value)
	if t.Pos >= len(runes) {
		return t.Value + "█"
	}
	// Insert cursor block at position.
	before := string(runes[:t.Pos])
	cursor := string(runes[t.Pos])
	after := string(runes[t.Pos+1:])
	return before + styles.SelectedRow.Render(cursor) + after
}

// wordLeft finds the position of the start of the previous word.
func (t *TextInput) wordLeft(runes []rune) int {
	pos := t.Pos
	// Skip spaces.
	for pos > 0 && unicode.IsSpace(runes[pos-1]) {
		pos--
	}
	// Skip word chars.
	for pos > 0 && !unicode.IsSpace(runes[pos-1]) {
		pos--
	}
	return pos
}

// wordRight finds the position after the end of the next word.
func (t *TextInput) wordRight(runes []rune) int {
	pos := t.Pos
	n := len(runes)
	// Skip word chars.
	for pos < n && !unicode.IsSpace(runes[pos]) {
		pos++
	}
	// Skip spaces.
	for pos < n && unicode.IsSpace(runes[pos]) {
		pos++
	}
	return pos
}

func insertRune(runes []rune, pos int, r rune) []rune {
	result := make([]rune, len(runes)+1)
	copy(result, runes[:pos])
	result[pos] = r
	copy(result[pos+1:], runes[pos:])
	return result
}
