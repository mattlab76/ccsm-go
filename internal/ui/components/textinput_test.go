package components

import (
	"testing"
)

func TestTextInput_BasicTyping(t *testing.T) {
	ti := NewTextInput("")
	ti.HandleKey("h")
	ti.HandleKey("e")
	ti.HandleKey("l")
	ti.HandleKey("l")
	ti.HandleKey("o")
	if ti.Value != "hello" {
		t.Errorf("Value = %q, want %q", ti.Value, "hello")
	}
	if ti.Pos != 5 {
		t.Errorf("Pos = %d, want 5", ti.Pos)
	}
}

func TestTextInput_Backspace(t *testing.T) {
	ti := NewTextInput("")
	ti.SetValue("hello")
	ti.HandleKey("backspace")
	if ti.Value != "hell" {
		t.Errorf("Value = %q, want %q", ti.Value, "hell")
	}
}

func TestTextInput_BackspaceAtStart(t *testing.T) {
	ti := NewTextInput("")
	ti.SetValue("hello")
	ti.Pos = 0
	ti.HandleKey("backspace")
	if ti.Value != "hello" {
		t.Error("backspace at start should not change value")
	}
}

func TestTextInput_CursorLeftRight(t *testing.T) {
	ti := NewTextInput("")
	ti.SetValue("hello")
	// Pos is 5 (end)
	ti.HandleKey("left")
	if ti.Pos != 4 {
		t.Errorf("after left: Pos = %d, want 4", ti.Pos)
	}
	ti.HandleKey("left")
	ti.HandleKey("left")
	if ti.Pos != 2 {
		t.Errorf("after 3 lefts: Pos = %d, want 2", ti.Pos)
	}
	ti.HandleKey("right")
	if ti.Pos != 3 {
		t.Errorf("after right: Pos = %d, want 3", ti.Pos)
	}
}

func TestTextInput_HomeEnd(t *testing.T) {
	ti := NewTextInput("")
	ti.SetValue("hello world")
	ti.HandleKey("home")
	if ti.Pos != 0 {
		t.Errorf("home: Pos = %d, want 0", ti.Pos)
	}
	ti.HandleKey("end")
	if ti.Pos != 11 {
		t.Errorf("end: Pos = %d, want 11", ti.Pos)
	}
}

func TestTextInput_CtrlAE(t *testing.T) {
	ti := NewTextInput("")
	ti.SetValue("test")
	ti.HandleKey("ctrl+a")
	if ti.Pos != 0 {
		t.Errorf("ctrl+a: Pos = %d, want 0", ti.Pos)
	}
	ti.HandleKey("ctrl+e")
	if ti.Pos != 4 {
		t.Errorf("ctrl+e: Pos = %d, want 4", ti.Pos)
	}
}

func TestTextInput_InsertMiddle(t *testing.T) {
	ti := NewTextInput("")
	ti.SetValue("helo")
	ti.Pos = 3 // between 'l' and 'o'
	ti.HandleKey("l")
	if ti.Value != "hello" {
		t.Errorf("insert middle: Value = %q, want %q", ti.Value, "hello")
	}
	if ti.Pos != 4 {
		t.Errorf("insert middle: Pos = %d, want 4", ti.Pos)
	}
}

func TestTextInput_Delete(t *testing.T) {
	ti := NewTextInput("")
	ti.SetValue("hello")
	ti.Pos = 2
	ti.HandleKey("delete")
	if ti.Value != "helo" {
		t.Errorf("delete: Value = %q, want %q", ti.Value, "helo")
	}
	if ti.Pos != 2 {
		t.Errorf("delete: Pos = %d, want 2", ti.Pos)
	}
}

func TestTextInput_DeleteAtEnd(t *testing.T) {
	ti := NewTextInput("")
	ti.SetValue("hello")
	ti.HandleKey("delete")
	if ti.Value != "hello" {
		t.Error("delete at end should not change value")
	}
}

func TestTextInput_WordJump(t *testing.T) {
	ti := NewTextInput("")
	ti.SetValue("hello world foo")
	// Pos at end (15)
	ti.HandleKey("ctrl+left")
	if ti.Pos != 12 {
		t.Errorf("ctrl+left from end: Pos = %d, want 12 (start of 'foo')", ti.Pos)
	}
	ti.HandleKey("ctrl+left")
	if ti.Pos != 6 {
		t.Errorf("ctrl+left again: Pos = %d, want 6 (start of 'world')", ti.Pos)
	}
	ti.HandleKey("ctrl+right")
	if ti.Pos != 12 {
		t.Errorf("ctrl+right: Pos = %d, want 12", ti.Pos)
	}
}

func TestTextInput_KillToEnd(t *testing.T) {
	ti := NewTextInput("")
	ti.SetValue("hello world")
	ti.Pos = 5
	ti.HandleKey("ctrl+k")
	if ti.Value != "hello" {
		t.Errorf("ctrl+k: Value = %q, want %q", ti.Value, "hello")
	}
}

func TestTextInput_KillToStart(t *testing.T) {
	ti := NewTextInput("")
	ti.SetValue("hello world")
	ti.Pos = 6
	ti.HandleKey("ctrl+u")
	if ti.Value != "world" {
		t.Errorf("ctrl+u: Value = %q, want %q", ti.Value, "world")
	}
	if ti.Pos != 0 {
		t.Errorf("ctrl+u: Pos = %d, want 0", ti.Pos)
	}
}

func TestTextInput_KillWord(t *testing.T) {
	ti := NewTextInput("")
	ti.SetValue("hello world")
	ti.Pos = 11 // end
	ti.HandleKey("ctrl+w")
	if ti.Value != "hello " {
		t.Errorf("ctrl+w: Value = %q, want %q", ti.Value, "hello ")
	}
}

func TestTextInput_Space(t *testing.T) {
	ti := NewTextInput("")
	ti.HandleKey("a")
	ti.HandleKey("space")
	ti.HandleKey("b")
	if ti.Value != "a b" {
		t.Errorf("space: Value = %q, want %q", ti.Value, "a b")
	}
}

func TestTextInput_SetValue(t *testing.T) {
	ti := NewTextInput("")
	ti.SetValue("preset")
	if ti.Value != "preset" || ti.Pos != 6 {
		t.Errorf("SetValue: Value=%q Pos=%d", ti.Value, ti.Pos)
	}
}

func TestTextInput_Reset(t *testing.T) {
	ti := NewTextInput("")
	ti.SetValue("something")
	ti.Reset()
	if ti.Value != "" || ti.Pos != 0 {
		t.Error("Reset should clear value and pos")
	}
}

func TestTextInput_ViewWithPlaceholder(t *testing.T) {
	ti := NewTextInput("type here...")
	view := ti.View()
	if view == "" {
		t.Error("View with placeholder should not be empty")
	}
}

func TestTextInput_ViewWithCursor(t *testing.T) {
	ti := NewTextInput("")
	ti.SetValue("hello")
	ti.Pos = 2
	view := ti.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}

func TestTextInput_BoundsCheck(t *testing.T) {
	ti := NewTextInput("")
	// Left at start should stay at 0.
	ti.HandleKey("left")
	if ti.Pos != 0 {
		t.Error("left at empty should stay 0")
	}
	// Right at end should stay at 0.
	ti.HandleKey("right")
	if ti.Pos != 0 {
		t.Error("right at empty should stay 0")
	}
}

func TestTextInput_Unicode(t *testing.T) {
	ti := NewTextInput("")
	ti.SetValue("über")
	if ti.Pos != 4 {
		t.Errorf("unicode: Pos = %d, want 4", ti.Pos)
	}
	ti.HandleKey("left")     // Pos 3 (before 'r')
	ti.HandleKey("backspace") // deletes 'e' at pos 2
	if ti.Value != "übr" {
		t.Errorf("unicode backspace: Value = %q, want %q", ti.Value, "übr")
	}
}
