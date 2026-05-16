package settings

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattlab76/ccsm-go/internal/db"
	"github.com/mattlab76/ccsm-go/internal/i18n"
)

func setup(t *testing.T) *db.DB {
	t.Helper()
	// Force a deterministic locale so assertions on user-facing strings
	// are stable regardless of the dev's $LANG.
	prev := i18n.Lang()
	i18n.SetLang("en")
	t.Cleanup(func() { i18n.SetLang(prev) })

	database, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

// keyRunes is shorthand for sending a single character keystroke.
func keyRunes(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func typeString(m Model, s string) Model {
	for _, r := range s {
		m, _ = m.Update(keyRunes(r))
	}
	return m
}

// ─── Cursor navigation ──────────────────────────────────────────────

func TestCursorStartsAtZero(t *testing.T) {
	m := New(setup(t))
	if m.cursor != 0 {
		t.Errorf("cursor should start at 0, got %d", m.cursor)
	}
}

func TestCursorDownClampsAtLastField(t *testing.T) {
	m := New(setup(t))
	for i := 0; i < 20; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if m.cursor != fieldCount-1 {
		t.Errorf("cursor should clamp at %d, got %d", fieldCount-1, m.cursor)
	}
}

func TestNumberShortcutsJumpToField(t *testing.T) {
	tests := map[rune]int{
		'1': 0, '2': 1, '3': 2, '4': 3, '5': 4, '6': 5, '7': 6,
	}
	for key, wantCursor := range tests {
		m := New(setup(t))
		m, _ = m.Update(keyRunes(key))
		if m.cursor != wantCursor {
			t.Errorf("key %c: cursor = %d, want %d", key, m.cursor, wantCursor)
		}
		if m.state == stateMenu {
			t.Errorf("key %c: should have entered edit mode", key)
		}
	}
}

func TestEnterOpensEditModeForCurrentField(t *testing.T) {
	m := New(setup(t))
	// Cursor on field 0 (lang) by default → Enter should open lang input.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != stateLangInput {
		t.Errorf("Enter on lang field should open stateLangInput, got %v", m.state)
	}
}

// ─── Input states ───────────────────────────────────────────────────

func TestEditLangAcceptsEnDe(t *testing.T) {
	database := setup(t)
	m := New(database)
	m, _ = m.Update(keyRunes('1')) // open lang edit
	m = typeString(m, "de")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.state != stateMenu {
		t.Errorf("should return to stateMenu, got %v", m.state)
	}
	if m.settings.Lang != "de" {
		t.Errorf("Lang = %q, want %q", m.settings.Lang, "de")
	}
	// Verify it persisted.
	stored, _ := db.GetSettings(database)
	if stored.Lang != "de" {
		t.Errorf("DB Lang = %q, want de", stored.Lang)
	}
}

func TestEditLangRejectsInvalid(t *testing.T) {
	m := New(setup(t))
	m, _ = m.Update(keyRunes('1'))
	m = typeString(m, "fr")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.settings.Lang == "fr" {
		t.Error("invalid lang should not be saved")
	}
	if m.message == "" || m.message == i18n.T("settings_saved") {
		t.Errorf("expected an invalid-input message, got %q", m.message)
	}
}

func TestEditCleanupDaysAcceptsValidNumber(t *testing.T) {
	m := New(setup(t))
	m, _ = m.Update(keyRunes('2'))
	m = typeString(m, "60")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.settings.CleanupDays != 60 {
		t.Errorf("CleanupDays = %d, want 60", m.settings.CleanupDays)
	}
}

func TestEditCleanupDaysRejectsOutOfRange(t *testing.T) {
	m := New(setup(t))
	m.settings.CleanupDays = 30 // baseline
	m, _ = m.Update(keyRunes('2'))
	m = typeString(m, "1000")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.settings.CleanupDays == 1000 {
		t.Error("1000 should be rejected (>999)")
	}
}

func TestEditCurrencyNormalisesToUpper(t *testing.T) {
	database := setup(t)
	m := New(database)
	m, _ = m.Update(keyRunes('4'))
	m = typeString(m, "eur")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.settings.Currency != "EUR" {
		t.Errorf("Currency = %q, want EUR (uppercased)", m.settings.Currency)
	}
}

func TestEditRateAcceptsDecimal(t *testing.T) {
	m := New(setup(t))
	m, _ = m.Update(keyRunes('5'))
	m = typeString(m, "0.92")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.settings.ExchangeRate != 0.92 {
		t.Errorf("ExchangeRate = %v, want 0.92", m.settings.ExchangeRate)
	}
}

func TestEditRateRejectsZeroOrNegative(t *testing.T) {
	m := New(setup(t))
	m.settings.ExchangeRate = 1.0
	m, _ = m.Update(keyRunes('5'))
	m = typeString(m, "0")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.settings.ExchangeRate == 0 {
		t.Error("rate of 0 should be rejected")
	}
}

func TestEditPlanNameAcceptsFreeText(t *testing.T) {
	m := New(setup(t))
	m, _ = m.Update(keyRunes('6'))
	m = typeString(m, "Max 20x")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.settings.PlanName != "Max 20x" {
		t.Errorf("PlanName = %q, want %q", m.settings.PlanName, "Max 20x")
	}
}

func TestEditPlanPriceAcceptsDecimal(t *testing.T) {
	m := New(setup(t))
	m, _ = m.Update(keyRunes('7'))
	m = typeString(m, "184.99")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.settings.PlanMonthlyPrice != 184.99 {
		t.Errorf("PlanMonthlyPrice = %v, want 184.99", m.settings.PlanMonthlyPrice)
	}
}

// ─── Escape and quit ────────────────────────────────────────────────

func TestEscFromEditReturnsToMenu(t *testing.T) {
	m := New(setup(t))
	m, _ = m.Update(keyRunes('1')) // enter lang edit
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != stateMenu {
		t.Errorf("Esc should return to stateMenu, got %v", m.state)
	}
}

func TestBackspaceInEditTrimsInput(t *testing.T) {
	m := New(setup(t))
	m, _ = m.Update(keyRunes('2'))
	m = typeString(m, "300")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.input != "30" {
		t.Errorf("after backspace: input = %q, want %q", m.input, "30")
	}
}

func TestQFromMenuReturnsToMainMenu(t *testing.T) {
	m := New(setup(t))
	_, cmd := m.Update(keyRunes('q'))
	if cmd == nil {
		t.Fatal("q should emit a switch cmd")
	}
	msg := cmd()
	sw, ok := msg.(SwitchViewMsg)
	if !ok {
		t.Fatalf("expected SwitchViewMsg, got %T", msg)
	}
	if sw.View != 0 {
		t.Errorf("expected View=0 (main menu), got %d", sw.View)
	}
}
