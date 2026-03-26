package components

import (
	"testing"
)

func newTagComplete(tags []string) *TagComplete {
	return &TagComplete{KnownTags: tags}
}

func TestTagComplete_EmptyInput(t *testing.T) {
	tc := newTagComplete([]string{"#infra", "#webapp"})
	tc.Complete("")
	if tc.Active {
		t.Error("should not be active for empty input")
	}
}

func TestTagComplete_TrailingSpace(t *testing.T) {
	tc := newTagComplete([]string{"#infra", "#webapp"})
	tc.Complete("#infra ")
	if tc.Active {
		t.Error("should not be active when input ends with space (done typing tag)")
	}
}

func TestTagComplete_MatchesPrefix(t *testing.T) {
	tc := newTagComplete([]string{"#infra", "#infrastructure", "#webapp", "#inferno"})
	tc.Complete("#inf")

	if !tc.Active {
		t.Fatal("should be active with matching tags")
	}
	// #infra, #infrastructure, #inferno match "#inf" prefix.
	if len(tc.Suggestions) != 3 {
		t.Errorf("got %d suggestions, want 3", len(tc.Suggestions))
	}
}

func TestTagComplete_CaseInsensitive(t *testing.T) {
	tc := newTagComplete([]string{"#Infra", "#WEBAPP"})
	tc.Complete("#inf")

	if !tc.Active {
		t.Fatal("should match case-insensitively")
	}
	if len(tc.Suggestions) != 1 {
		t.Errorf("got %d suggestions, want 1", len(tc.Suggestions))
	}
}

func TestTagComplete_ExcludesExactMatch(t *testing.T) {
	tc := newTagComplete([]string{"#infra", "#webapp"})
	tc.Complete("#infra")

	// Exact match should not be suggested (already typed).
	if tc.Active {
		t.Error("should not suggest exact match")
	}
}

func TestTagComplete_ExcludesAlreadyUsed(t *testing.T) {
	tc := newTagComplete([]string{"#infra", "#webapp", "#docker"})
	tc.Complete("#infra #w")

	if !tc.Active {
		t.Fatal("should be active")
	}
	// Only #webapp should match, not #infra (already used).
	if len(tc.Suggestions) != 1 {
		t.Errorf("got %d suggestions, want 1", len(tc.Suggestions))
	}
	if tc.Suggestions[0] != "#webapp" {
		t.Errorf("suggestion = %q, want #webapp", tc.Suggestions[0])
	}
}

func TestTagComplete_NoMatch(t *testing.T) {
	tc := newTagComplete([]string{"#infra", "#webapp"})
	tc.Complete("#zzz")

	if tc.Active {
		t.Error("should not be active when no match")
	}
}

func TestTagComplete_Accept(t *testing.T) {
	tc := newTagComplete([]string{"#infra", "#webapp"})
	tc.Complete("#w")

	if !tc.Active {
		t.Fatal("should be active")
	}

	result := tc.Accept("#w")
	if result != "#webapp " {
		t.Errorf("Accept = %q, want %q", result, "#webapp ")
	}
}

func TestTagComplete_AcceptMultiple(t *testing.T) {
	tc := newTagComplete([]string{"#infra", "#webapp", "#docker"})
	tc.Complete("#infra #d")

	result := tc.Accept("#infra #d")
	if result != "#infra #docker " {
		t.Errorf("Accept = %q, want %q", result, "#infra #docker ")
	}
}

func TestTagComplete_NextPrev(t *testing.T) {
	tc := newTagComplete([]string{"#a", "#ab", "#abc"})
	tc.Complete("#a")

	if len(tc.Suggestions) < 2 {
		t.Fatal("need at least 2 suggestions for Next/Prev test")
	}

	tc.Next()
	if tc.Selected != 1 {
		t.Errorf("Next: selected = %d, want 1", tc.Selected)
	}
	tc.Prev()
	if tc.Selected != 0 {
		t.Errorf("Prev: selected = %d, want 0", tc.Selected)
	}
	tc.Prev()
	if tc.Selected != len(tc.Suggestions)-1 {
		t.Errorf("Prev wrap: selected = %d, want %d", tc.Selected, len(tc.Suggestions)-1)
	}
}

func TestTagComplete_Reset(t *testing.T) {
	tc := newTagComplete([]string{"#infra"})
	tc.Complete("#i")
	tc.Reset()

	if tc.Active {
		t.Error("should not be active after reset")
	}
	if len(tc.Suggestions) != 0 {
		t.Error("suggestions should be empty after reset")
	}
}

func TestTagComplete_MaxSuggestions(t *testing.T) {
	tags := make([]string, 20)
	for i := range tags {
		tags[i] = "#tag" + string(rune('a'+i))
	}
	tc := newTagComplete(tags)
	tc.Complete("#tag")

	if len(tc.Suggestions) > 6 {
		t.Errorf("should limit to 6 suggestions, got %d", len(tc.Suggestions))
	}
}
