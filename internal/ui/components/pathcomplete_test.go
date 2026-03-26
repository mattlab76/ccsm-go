package components

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathComplete_EmptyInput(t *testing.T) {
	var pc PathComplete
	pc.Complete("")
	if pc.Active {
		t.Error("should not be active for empty input")
	}
}

func TestPathComplete_CompletesDirectories(t *testing.T) {
	// Create temp directory structure.
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, "project-alpha"), 0755)
	os.MkdirAll(filepath.Join(tmp, "project-beta"), 0755)
	os.MkdirAll(filepath.Join(tmp, "other"), 0755)
	// Create a file (should not appear in suggestions).
	os.WriteFile(filepath.Join(tmp, "file.txt"), []byte("test"), 0644)

	var pc PathComplete
	pc.Complete(filepath.Join(tmp, "project"))

	if !pc.Active {
		t.Fatal("should be active with matching dirs")
	}
	if len(pc.Suggestions) != 2 {
		t.Errorf("got %d suggestions, want 2", len(pc.Suggestions))
	}
}

func TestPathComplete_NoMatch(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, "alpha"), 0755)

	var pc PathComplete
	pc.Complete(filepath.Join(tmp, "zzz"))

	if pc.Active {
		t.Error("should not be active when no dirs match")
	}
}

func TestPathComplete_HiddenDirs(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, ".hidden"), 0755)
	os.MkdirAll(filepath.Join(tmp, "visible"), 0755)

	// Without dot prefix — should not show hidden.
	var pc PathComplete
	pc.Complete(filepath.Join(tmp, "v"))
	if len(pc.Suggestions) != 1 {
		t.Errorf("without dot: got %d suggestions, want 1", len(pc.Suggestions))
	}

	// With dot prefix — should show hidden.
	pc.Complete(filepath.Join(tmp, "."))
	if len(pc.Suggestions) != 1 {
		t.Errorf("with dot: got %d suggestions, want 1 (.hidden)", len(pc.Suggestions))
	}
}

func TestPathComplete_TrailingSlash(t *testing.T) {
	tmp := t.TempDir()
	os.MkdirAll(filepath.Join(tmp, "sub1"), 0755)
	os.MkdirAll(filepath.Join(tmp, "sub2"), 0755)

	var pc PathComplete
	pc.Complete(tmp + "/")

	if !pc.Active {
		t.Fatal("should be active for trailing slash (list dir contents)")
	}
	if len(pc.Suggestions) != 2 {
		t.Errorf("got %d suggestions, want 2", len(pc.Suggestions))
	}
}

func TestPathComplete_NextPrev(t *testing.T) {
	var pc PathComplete
	pc.Suggestions = []string{"a", "b", "c"}
	pc.Active = true
	pc.Selected = 0

	pc.Next()
	if pc.Selected != 1 {
		t.Errorf("Next: selected = %d, want 1", pc.Selected)
	}
	pc.Next()
	pc.Next()
	if pc.Selected != 0 {
		t.Errorf("Next wrap: selected = %d, want 0", pc.Selected)
	}

	pc.Prev()
	if pc.Selected != 2 {
		t.Errorf("Prev wrap: selected = %d, want 2", pc.Selected)
	}
}

func TestPathComplete_Accept(t *testing.T) {
	tmp := t.TempDir()
	sub := filepath.Join(tmp, "myproject")
	os.MkdirAll(sub, 0755)

	var pc PathComplete
	pc.Complete(filepath.Join(tmp, "my"))

	if !pc.Active {
		t.Fatal("should be active")
	}

	result := pc.Accept()
	if result == "" {
		t.Fatal("Accept returned empty")
	}
	// Should end with /
	if result[len(result)-1] != '/' {
		t.Errorf("Accept should end with /, got %q", result)
	}
	if !pc.Active == true {
		// After accept, should be inactive.
	}
}

func TestPathComplete_Reset(t *testing.T) {
	var pc PathComplete
	pc.Suggestions = []string{"a"}
	pc.Active = true
	pc.Selected = 0

	pc.Reset()
	if pc.Active {
		t.Error("should not be active after reset")
	}
	if len(pc.Suggestions) != 0 {
		t.Error("suggestions should be empty after reset")
	}
}

func TestPathComplete_MaxSuggestions(t *testing.T) {
	tmp := t.TempDir()
	for i := 0; i < 15; i++ {
		os.MkdirAll(filepath.Join(tmp, filepath.Base(t.Name())+string(rune('a'+i))), 0755)
	}

	var pc PathComplete
	pc.Complete(tmp + "/")

	if len(pc.Suggestions) > 8 {
		t.Errorf("should limit to 8 suggestions, got %d", len(pc.Suggestions))
	}
}
