// Package startup renders the startup-time check for invalid sessions.
// When ccsm boots, it scans for sessions whose JSONL transcript has been
// deleted at the Claude Code side (expired) and which the user has not
// previously dismissed. If any are found, this view is shown before the
// main menu so the user can purge or dismiss them in one go.
package startup

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattlab76/ccsm-go/internal/claude"
	"github.com/mattlab76/ccsm-go/internal/db"
	"github.com/mattlab76/ccsm-go/internal/i18n"
	"github.com/mattlab76/ccsm-go/internal/model"
	"github.com/mattlab76/ccsm-go/internal/ui/styles"
)

// SwitchViewMsg is sent to the parent app when the startup check is done.
type SwitchViewMsg struct {
	View int
}

// Model holds the expired sessions and the user's pending choice.
type Model struct {
	db      *db.DB
	expired []model.Session
	width   int
	height  int
	message string
	done    bool
}

// New builds a startup model and loads expired, non-dismissed sessions.
func New(database *db.DB) Model {
	m := Model{db: database, width: 100}
	m.expired = loadExpired(database)
	return m
}

// HasWork reports whether the startup view has anything to show.
// The app uses this to decide whether to open the view at all.
func (m Model) HasWork() bool {
	return len(m.expired) > 0
}

// loadExpired returns sessions whose Claude Code transcript is gone
// and which are not already in the dismissed list.
func loadExpired(database *db.DB) []model.Session {
	sessions, err := db.ListSessions(database, 0)
	if err != nil {
		return nil
	}
	dismissed, _ := db.GetDismissed(database)
	dismissedSet := make(map[string]bool, len(dismissed))
	for _, sid := range dismissed {
		dismissedSet[sid] = true
	}
	var out []model.Session
	for _, s := range sessions {
		if dismissedSet[s.SID] {
			continue
		}
		if !claude.IsSessionValid(s.SID) {
			out = append(out, s)
		}
	}
	return out
}

func (m Model) SetSize(width, height int) Model {
	m.width = width
	m.height = height
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	key := strings.ToLower(keyMsg.String())

	switch key {
	case "ctrl+c":
		return m, tea.Quit

	case "p":
		return m, m.purge()

	case "d":
		return m, m.dismiss()

	case "s", "esc", "enter", "q":
		return m, finish()
	}
	return m, nil
}

// purge deletes all expired sessions from the database.
func (m Model) purge() tea.Cmd {
	sids := make([]string, 0, len(m.expired))
	for _, s := range m.expired {
		sids = append(sids, s.SID)
	}
	if err := db.DeleteSessions(m.db, sids); err == nil {
		db.LogAction(m.db, model.ActionDelete,
			fmt.Sprintf("startup purge: %d expired session(s)", len(sids)))
	}
	return finish()
}

// dismiss marks all expired sessions so they won't be reported on next start.
func (m Model) dismiss() tea.Cmd {
	for _, s := range m.expired {
		_ = db.AddDismissed(m.db, s.SID)
	}
	db.LogAction(m.db, model.ActionSettings,
		fmt.Sprintf("startup dismiss: %d expired session(s)", len(m.expired)))
	return finish()
}

// finish signals the app to switch to the main menu (view 0).
func finish() tea.Cmd {
	return func() tea.Msg {
		return SwitchViewMsg{View: 0}
	}
}

func (m Model) View() string {
	var b strings.Builder

	count := len(m.expired)
	b.WriteString("\n")
	b.WriteString("  " + styles.RenderLogo(i18n.T("startup_title"), count))
	b.WriteString("\n\n")

	w := m.width
	if w < 60 {
		w = 60
	}
	innerW := w - 8
	if innerW < 56 {
		innerW = 56
	}
	b.WriteString(styles.DoubleLine(innerW))
	b.WriteString("\n\n")

	b.WriteString("  " + styles.Amber.Render(fmt.Sprintf(i18n.T("startup_summary"), count)) + "\n")
	b.WriteString("  " + styles.Dim.Render(i18n.T("startup_explain")) + "\n\n")

	// Show up to 10 sessions inline; collapse the rest into a counter.
	const maxShow = 10
	shown := count
	if shown > maxShow {
		shown = maxShow
	}
	for i := 0; i < shown; i++ {
		s := m.expired[i]
		date := s.CreatedAt.Format("2006-01-02")
		subj := s.Subject
		if len([]rune(subj)) > 50 {
			subj = string([]rune(subj)[:48]) + ".."
		}
		b.WriteString(fmt.Sprintf("    %s  %s\n",
			styles.Dim.Render(date),
			styles.Red.Render(subj)))
	}
	if count > maxShow {
		b.WriteString(fmt.Sprintf("    %s\n",
			styles.Dim.Render(fmt.Sprintf(i18n.T("startup_more"), count-maxShow))))
	}
	b.WriteString("\n")

	b.WriteString("  " + actionLine("p", i18n.T("startup_purge"), styles.Red))
	b.WriteString("  " + actionLine("d", i18n.T("startup_dismiss"), styles.Amber))
	b.WriteString("  " + actionLine("s", i18n.T("startup_skip"), styles.Green))
	b.WriteString("\n")
	b.WriteString("  " + styles.Dim.Render(i18n.T("startup_hint")) + "\n")

	return b.String()
}

func actionLine(key, label string, color lipgloss.Style) string {
	return fmt.Sprintf("%s %s\n", color.Render("["+key+"]"), label)
}
