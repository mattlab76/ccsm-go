// Package startup renders the startup-time check for invalid sessions.
// When ccsm boots, it scans for sessions that can no longer be used:
//   - expired sessions whose JSONL transcript has been deleted at the
//     Claude Code side
//   - sessions whose working directory has been deleted locally
// Sessions the user has previously dismissed are skipped. If any remain,
// this view is shown before the main menu so the user can purge or
// dismiss them in one go.
package startup

import (
	"fmt"
	"os"
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

// reason classifies why a session is unusable.
type reason int

const (
	reasonExpired    reason = iota // JSONL transcript gone at Claude Code
	reasonMissingDir               // working directory deleted locally
)

// invalidEntry pairs a session with the reason it can't be used anymore.
type invalidEntry struct {
	session model.Session
	reason  reason
}

// Model holds the invalid sessions and the user's pending choice.
type Model struct {
	db      *db.DB
	invalid []invalidEntry
	cursor  int // 0=purge, 1=dismiss, 2=skip
	width   int
	height  int
}

// Action positions for the cursor.
const (
	actionPurge   = 0
	actionDismiss = 1
	actionSkip    = 2
	actionCount   = 3
)

// New builds a startup model and loads invalid, non-dismissed sessions.
// Cursor defaults to "skip" — the safest action if the user just hits Enter.
func New(database *db.DB) Model {
	m := Model{db: database, width: 100, cursor: actionSkip}
	m.invalid = loadInvalid(database)
	return m
}

// HasWork reports whether the startup view has anything to show.
// The app uses this to decide whether to open the view at all.
func (m Model) HasWork() bool {
	return len(m.invalid) > 0
}

// loadInvalid returns every non-dismissed session that is either expired
// at Claude Code or whose working directory no longer exists locally.
// Expired wins when both conditions apply, because that's the harder
// failure (can't resume regardless of where you launch from).
func loadInvalid(database *db.DB) []invalidEntry {
	sessions, err := db.ListSessions(database, 0)
	if err != nil {
		return nil
	}
	dismissed, _ := db.GetDismissed(database)
	dismissedSet := make(map[string]bool, len(dismissed))
	for _, sid := range dismissed {
		dismissedSet[sid] = true
	}
	var out []invalidEntry
	for _, s := range sessions {
		if dismissedSet[s.SID] {
			continue
		}
		switch {
		case !claude.IsSessionValid(s.SID):
			out = append(out, invalidEntry{s, reasonExpired})
		case !dirExists(s.CWD):
			out = append(out, invalidEntry{s, reasonMissingDir})
		}
	}
	return out
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
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

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "down", "j":
		if m.cursor < actionCount-1 {
			m.cursor++
		}
		return m, nil

	case "home", "g":
		m.cursor = 0
		return m, nil

	case "end", "G":
		m.cursor = actionCount - 1
		return m, nil

	case "enter":
		return m, m.activate(m.cursor)

	case "p":
		return m, m.activate(actionPurge)

	case "d":
		return m, m.activate(actionDismiss)

	case "s", "esc", "q":
		return m, m.activate(actionSkip)
	}
	return m, nil
}

// activate runs the action at the given position.
func (m Model) activate(action int) tea.Cmd {
	switch action {
	case actionPurge:
		return m.purge()
	case actionDismiss:
		return m.dismiss()
	}
	return finish()
}

// purge deletes all invalid sessions from the database.
func (m Model) purge() tea.Cmd {
	sids := make([]string, 0, len(m.invalid))
	for _, e := range m.invalid {
		sids = append(sids, e.session.SID)
	}
	if err := db.DeleteSessions(m.db, sids); err == nil {
		db.LogAction(m.db, model.ActionDelete,
			fmt.Sprintf("startup purge: %d invalid session(s)", len(sids)))
	}
	return finish()
}

// dismiss marks all invalid sessions so they won't be reported on next start.
func (m Model) dismiss() tea.Cmd {
	for _, e := range m.invalid {
		_ = db.AddDismissed(m.db, e.session.SID)
	}
	db.LogAction(m.db, model.ActionSettings,
		fmt.Sprintf("startup dismiss: %d invalid session(s)", len(m.invalid)))
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

	count := len(m.invalid)
	expiredCount, missingCount := m.countByReason()

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
	if expiredCount > 0 {
		b.WriteString(fmt.Sprintf("    %s %s\n",
			styles.Red.Render("[!]"),
			styles.Dim.Render(fmt.Sprintf(i18n.T("startup_breakdown_expired"), expiredCount))))
	}
	if missingCount > 0 {
		b.WriteString(fmt.Sprintf("    %s %s\n",
			styles.Amber.Render("[?]"),
			styles.Dim.Render(fmt.Sprintf(i18n.T("startup_breakdown_missing_dir"), missingCount))))
	}
	b.WriteString("\n")

	// Show up to 10 sessions inline; collapse the rest into a counter.
	const maxShow = 10
	shown := count
	if shown > maxShow {
		shown = maxShow
	}
	for i := 0; i < shown; i++ {
		e := m.invalid[i]
		date := e.session.CreatedAt.Format("2006-01-02")
		subj := e.session.Subject
		if len([]rune(subj)) > 50 {
			subj = string([]rune(subj)[:48]) + ".."
		}
		var marker, line string
		if e.reason == reasonExpired {
			marker = styles.Red.Render("[!]")
			line = styles.Red.Render(subj)
		} else {
			marker = styles.Amber.Render("[?]")
			line = styles.Amber.Render(subj)
		}
		b.WriteString(fmt.Sprintf("    %s %s  %s\n",
			marker,
			styles.Dim.Render(date),
			line))
	}
	if count > maxShow {
		b.WriteString(fmt.Sprintf("    %s\n",
			styles.Dim.Render(fmt.Sprintf(i18n.T("startup_more"), count-maxShow))))
	}
	b.WriteString("\n")

	b.WriteString(actionLine("p", i18n.T("startup_purge"), styles.Red, m.cursor == actionPurge))
	b.WriteString(actionLine("d", i18n.T("startup_dismiss"), styles.Amber, m.cursor == actionDismiss))
	b.WriteString(actionLine("s", i18n.T("startup_skip"), styles.Green, m.cursor == actionSkip))
	b.WriteString("\n")
	b.WriteString("  " + styles.Dim.Render(i18n.T("startup_hint")) + "\n")

	return b.String()
}

func (m Model) countByReason() (expired, missing int) {
	for _, e := range m.invalid {
		switch e.reason {
		case reasonExpired:
			expired++
		case reasonMissingDir:
			missing++
		}
	}
	return
}

func actionLine(key, label string, color lipgloss.Style, selected bool) string {
	prefix := "  "
	if selected {
		prefix = styles.Violet.Render("▶ ")
	}
	return fmt.Sprintf("%s%s %s\n", prefix, color.Render("["+key+"]"), label)
}
