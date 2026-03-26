package components

import (
	"testing"
)

func TestStatusBarItems(t *testing.T) {
	result := StatusBarItems("↑↓", "navigate", "Enter", "select")
	if result == "" {
		t.Error("StatusBarItems should not be empty")
	}
}

func TestStatusBarItems_Empty(t *testing.T) {
	result := StatusBarItems()
	if result != "" {
		t.Errorf("StatusBarItems() with no args should be empty, got %q", result)
	}
}

func TestStatusBarItems_OddArgs(t *testing.T) {
	// Odd number of args — last one should be ignored.
	result := StatusBarItems("↑↓", "navigate", "orphan")
	if result == "" {
		t.Error("should still produce output for valid pairs")
	}
}

func TestStatusBar_Width(t *testing.T) {
	result := StatusBar("test", 40)
	if result == "" {
		t.Error("StatusBar should not be empty")
	}
}

func TestStatusBar_ZeroWidth(t *testing.T) {
	result := StatusBar("test", 0)
	if result == "" {
		t.Error("StatusBar should handle zero width")
	}
}
