package newsession

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattlab76/ccsm-go/internal/claude"
	"github.com/mattlab76/ccsm-go/internal/db"
	"github.com/mattlab76/ccsm-go/internal/hook"
	"github.com/mattlab76/ccsm-go/internal/i18n"
	"github.com/mattlab76/ccsm-go/internal/model"
	"github.com/mattlab76/ccsm-go/internal/ui/components"
	"github.com/mattlab76/ccsm-go/internal/ui/styles"
)

// SwitchViewMsg is sent to the parent app to change views.
type SwitchViewMsg struct {
	View int
}

// StartResumeMsg tells this model to launch claude --resume.
type StartResumeMsg struct {
	SID string
	CWD string
}

type step int

const (
	stepSubject step = iota
	stepDirConfirm
	stepDirInput
	stepDirCreate
	stepAlreadyRunning
	stepLaunching
	stepSaveConfirm
	stepSaveSubjectConfirm
	stepSaveSubjectInput
	stepSaveTags
	stepDone
)

// Model handles the new session flow.
type Model struct {
	db     *db.DB
	step   step
	width  int
	height int

	// Input state
	input         string
	presetSubject string
	targetDir     string
	pathComplete  components.PathComplete
	tagComplete   components.TagComplete

	// Session data from hook
	hookData    *hook.HookData
	isUpdate    bool // true if session already exists in DB
	existingSession *model.Session

	// Save state
	chosenSubject string
	chosenTags    string
	autoSubject   string

	// Messages
	message string
	err     error
}

// New creates a new session flow model.
func New(database *db.DB) Model {
	cwd, _ := os.Getwd()
	return Model{
		db:        database,
		step:      stepSubject,
		targetDir: cwd,
	}
}

// NewAlreadyRunning creates a model that shows the "already running" warning.
func NewAlreadyRunning(database *db.DB, dir string) Model {
	return Model{
		db:        database,
		step:      stepAlreadyRunning,
		targetDir: dir,
	}
}

// NewForResume creates a model that skips straight to save dialog after resume.
func NewForResume(database *db.DB, sid string) Model {
	m := Model{
		db:   database,
		step: stepLaunching, // Will go to save after claude exits
	}
	// Load existing session for update flow.
	if s, err := db.GetSession(database, sid); err == nil {
		m.existingSession = s
		m.targetDir = s.CWD
		m.isUpdate = true
	}
	return m
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
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case StartResumeMsg:
		m.step = stepLaunching
		m.targetDir = msg.CWD
		return m, claude.ResumeSession(msg.SID, msg.CWD)

	case claude.SessionFinishedMsg:
		return m.handleSessionFinished(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()

	// Global escape: q in confirm steps, ctrl+c always
	if key == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.step {
	case stepSubject:
		return m.handleSubjectInput(key, msg)
	case stepDirConfirm:
		return m.handleDirConfirm(key)
	case stepDirInput:
		return m.handleDirInput(key, msg)
	case stepDirCreate:
		return m.handleDirCreate(key)
	case stepAlreadyRunning:
		switch strings.ToLower(key) {
		case "d":
			// Change directory — go to dir input.
			m.step = stepDirInput
			m.input = ""
		case "q", "esc":
			return m, switchView(0)
		}
		return m, nil
	case stepSaveConfirm:
		return m.handleSaveConfirm(key)
	case stepSaveSubjectConfirm:
		return m.handleSaveSubjectConfirm(key)
	case stepSaveSubjectInput:
		return m.handleSaveSubjectInput(key, msg)
	case stepSaveTags:
		return m.handleSaveTagsInput(key, msg)
	case stepDone:
		// Any key returns to main menu.
		return m, switchView(0)
	}
	return m, nil
}

// --- Step handlers ---

func (m Model) handleSubjectInput(key string, msg tea.KeyMsg) (Model, tea.Cmd) {
	switch key {
	case "enter":
		m.presetSubject = strings.TrimSpace(m.input)
		m.input = ""
		m.step = stepDirConfirm
	case "esc":
		return m, switchView(0)
	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
	default:
		if len(key) == 1 {
			m.input += key
		} else if key == "space" {
			m.input += " "
		}
	}
	return m, nil
}

func (m Model) handleDirConfirm(key string) (Model, tea.Cmd) {
	switch strings.ToLower(key) {
	case "y", "j", "enter":
		// Use current directory → launch claude.
		return m.launchClaude()
	case "n":
		m.step = stepDirInput
		m.input = ""
	case "q", "esc":
		return m, switchView(0)
	}
	return m, nil
}

func (m Model) handleDirInput(key string, msg tea.KeyMsg) (Model, tea.Cmd) {
	switch key {
	case "tab":
		if m.pathComplete.Active {
			// Accept selected suggestion.
			accepted := m.pathComplete.Accept()
			if accepted != "" {
				m.input = accepted
			}
		} else {
			// Trigger completion.
			m.pathComplete.Complete(m.input)
		}
		return m, nil
	case "shift+tab", "up":
		if m.pathComplete.Active {
			m.pathComplete.Prev()
			return m, nil
		}
	case "down":
		if m.pathComplete.Active {
			m.pathComplete.Next()
			return m, nil
		}
	case "enter":
		if m.pathComplete.Active {
			accepted := m.pathComplete.Accept()
			if accepted != "" {
				m.input = accepted
			}
			return m, nil
		}
		path := strings.TrimSpace(m.input)
		if path == "" || path == "q" {
			return m, switchView(0)
		}
		// Expand ~
		if strings.HasPrefix(path, "~") {
			home, _ := os.UserHomeDir()
			path = filepath.Join(home, path[1:])
		}
		m.targetDir = path
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			m.input = ""
			m.pathComplete.Reset()
			return m.launchClaude()
		}
		// Dir doesn't exist.
		m.step = stepDirCreate
		m.input = ""
		m.pathComplete.Reset()
	case "esc":
		if m.pathComplete.Active {
			m.pathComplete.Reset()
			return m, nil
		}
		return m, switchView(0)
	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
		m.pathComplete.Reset()
	default:
		if len(key) == 1 {
			m.input += key
		} else if key == "space" {
			m.input += " "
		}
		m.pathComplete.Reset()
	}
	return m, nil
}

func (m Model) handleDirCreate(key string) (Model, tea.Cmd) {
	switch strings.ToLower(key) {
	case "y", "j":
		if err := os.MkdirAll(m.targetDir, 0755); err != nil {
			m.err = err
			m.step = stepDirInput
			m.input = ""
			return m, nil
		}
		m.message = i18n.T("new_dir_created")
		return m.launchClaude()
	case "n", "q", "esc":
		m.step = stepDirInput
		m.input = ""
	}
	return m, nil
}

func (m Model) launchClaude() (Model, tea.Cmd) {
	// Check if claude is already running in this directory.
	if claude.IsClaudeRunningInDir(m.targetDir) {
		m.step = stepAlreadyRunning
		return m, nil
	}

	m.step = stepLaunching
	m.message = ""

	// Create lock file before starting claude.
	claude.CreateSessionLock(m.targetDir, os.Getpid())

	// Log the new session action.
	subject := m.presetSubject
	if subject == "" {
		subject = "new session"
	}
	db.LogAction(m.db, model.ActionNew, fmt.Sprintf("%s (%s)", subject, m.targetDir))

	return m, claude.StartSession(m.targetDir)
}

func (m Model) handleSessionFinished(msg claude.SessionFinishedMsg) (Model, tea.Cmd) {
	// Try to read hook data — prefer matching by directory.
	hookData, err := hook.FindSessionDataForDir(m.targetDir)
	if err != nil {
		m.message = i18n.T("save_no_data")
		m.step = stepDone
		return m, nil
	}
	m.hookData = hookData

	// Check if this is an existing session (resume/update).
	if existing, err := db.GetSession(m.db, hookData.SessionID); err == nil {
		m.existingSession = existing
		m.isUpdate = true
	}

	// Determine auto-subject.
	if m.presetSubject != "" {
		m.autoSubject = m.presetSubject
	} else if hookData.Subject != "" {
		m.autoSubject = hookData.Subject
	} else {
		m.autoSubject = fmt.Sprintf("Session %s %s",
			filepath.Base(hookData.CWD),
			time.Now().Format("2006-01-02 15:04"))
	}

	if m.isUpdate {
		m.step = stepSaveSubjectConfirm
	} else {
		m.step = stepSaveConfirm
	}
	return m, nil
}

// --- Save flow handlers ---

func (m Model) handleSaveConfirm(key string) (Model, tea.Cmd) {
	switch strings.ToLower(key) {
	case "y", "j":
		m.step = stepSaveSubjectConfirm
	case "n", "q", "esc":
		m.message = i18n.T("save_not_saved")
		if m.hookData != nil {
			hook.CleanupTempFile(m.hookData.SessionID)
		}
		m.step = stepDone
	}
	return m, nil
}

func (m Model) handleSaveSubjectConfirm(key string) (Model, tea.Cmd) {
	switch strings.ToLower(key) {
	case "y", "j", "enter":
		if m.isUpdate && m.existingSession != nil {
			m.chosenSubject = m.existingSession.Subject
		} else {
			m.chosenSubject = m.autoSubject
		}
		if m.isUpdate {
			return m.doSave()
		}
		m.step = stepSaveTags
		m.input = ""
	case "n":
		m.step = stepSaveSubjectInput
		if m.isUpdate && m.existingSession != nil {
			m.input = m.existingSession.Subject
		} else {
			m.input = m.autoSubject
		}
	case "q", "esc":
		m.message = i18n.T("save_not_saved")
		m.step = stepDone
	}
	return m, nil
}

func (m Model) handleSaveSubjectInput(key string, msg tea.KeyMsg) (Model, tea.Cmd) {
	switch key {
	case "enter":
		subject := strings.TrimSpace(m.input)
		if subject == "" {
			m.message = i18n.T("save_no_empty")
			return m, nil
		}
		m.chosenSubject = subject
		if m.isUpdate {
			return m.doSave()
		}
		m.step = stepSaveTags
		m.input = ""
	case "esc":
		m.message = i18n.T("save_not_saved")
		m.step = stepDone
	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
	default:
		if len(key) == 1 {
			m.input += key
		} else if key == "space" {
			m.input += " "
		}
	}
	return m, nil
}

func (m Model) handleSaveTagsInput(key string, msg tea.KeyMsg) (Model, tea.Cmd) {
	switch key {
	case "tab":
		if m.tagComplete.Active {
			m.input = m.tagComplete.Accept(m.input)
		} else {
			// Load known tags and trigger completion.
			if m.tagComplete.KnownTags == nil {
				tags, _ := db.GetAllTags(m.db)
				m.tagComplete.KnownTags = tags
			}
			m.tagComplete.Complete(m.input)
		}
		return m, nil
	case "shift+tab":
		if m.tagComplete.Active {
			m.tagComplete.Prev()
			return m, nil
		}
	case "enter":
		if m.tagComplete.Active {
			m.input = m.tagComplete.Accept(m.input)
			return m, nil
		}
		m.chosenTags = strings.TrimSpace(m.input)
		m.tagComplete.Reset()
		return m.doSave()
	case "esc":
		if m.tagComplete.Active {
			m.tagComplete.Reset()
			return m, nil
		}
		m.message = i18n.T("save_not_saved")
		m.step = stepDone
	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
		m.tagComplete.Reset()
	default:
		if len(key) == 1 {
			m.input += key
		} else if key == "space" {
			m.input += " "
		}
		m.tagComplete.Reset()
	}
	return m, nil
}

func (m Model) doSave() (Model, tea.Cmd) {
	h := m.hookData
	if h == nil {
		m.message = i18n.T("save_no_data")
		m.step = stepDone
		return m, nil
	}

	now := time.Now()
	tags := m.chosenTags
	if tags == "" && m.existingSession != nil {
		tags = m.existingSession.Tags
	}

	s := &model.Session{
		SID:       h.SessionID,
		CWD:       h.CWD,
		Subject:   m.chosenSubject,
		CreatedAt: now,
		Tags:      tags,
		LastInputTokens:  h.InputTokens,
		LastOutputTokens: h.OutputTokens,
	}

	// Accumulate tokens for existing sessions.
	if m.isUpdate && m.existingSession != nil {
		s.TotalInputTokens = m.existingSession.TotalInputTokens + h.InputTokens
		s.TotalOutputTokens = m.existingSession.TotalOutputTokens + h.OutputTokens
	} else {
		s.TotalInputTokens = h.InputTokens
		s.TotalOutputTokens = h.OutputTokens
	}

	if err := db.SaveSession(m.db, s); err != nil {
		m.err = err
		m.step = stepDone
		return m, nil
	}

	// Add to lifetime tokens.
	db.AddLifetimeTokens(m.db, h.InputTokens, h.OutputTokens)

	// Log action.
	if m.isUpdate {
		db.LogAction(m.db, model.ActionSave, "updated: "+m.chosenSubject)
		m.message = i18n.T("save_updated") + ": " + m.chosenSubject
	} else {
		db.LogAction(m.db, model.ActionSave, fmt.Sprintf("new: %s (%s)", m.chosenSubject, h.CWD))
		m.message = i18n.T("save_saved") + ": " + m.chosenSubject
	}

	// Cleanup temp file.
	hook.CleanupTempFile(h.SessionID)

	m.step = stepDone
	return m, nil
}

// --- View ---

func (m Model) View() string {
	var b strings.Builder
	b.WriteString("\n")

	switch m.step {
	case stepSubject:
		b.WriteString("  " + styles.TealBold.Render(i18n.T("menu_new")) + "\n\n")
		b.WriteString("  " + i18n.T("new_subject_prompt") + "\n")
		b.WriteString("  > " + m.input + "█\n")

	case stepDirConfirm:
		b.WriteString("  " + styles.TealBold.Render(i18n.T("menu_new")) + "\n\n")
		if m.presetSubject != "" {
			b.WriteString("  Subject: " + styles.Violet.Render(m.presetSubject) + "\n\n")
		}
		b.WriteString("  " + styles.Dim.Render(i18n.T("new_dir_current")+" "+m.targetDir) + "\n")
		b.WriteString("  " + i18n.T("new_dir_question") + " [" + i18n.ConfirmPrompt() + "] ")

	case stepDirInput:
		b.WriteString("  " + i18n.T("new_dir_enter_or_q") + " " + styles.Dim.Render("(Tab: autocomplete)") + "\n")
		b.WriteString("  > " + m.input + "█\n")
		if m.pathComplete.Active {
			b.WriteString(m.pathComplete.View())
		}
		if m.err != nil {
			b.WriteString("  " + styles.Red.Render(m.err.Error()) + "\n")
		}

	case stepDirCreate:
		b.WriteString("  " + styles.Amber.Render(i18n.T("new_dir_not_found")) + "\n")
		b.WriteString("  " + m.targetDir + "\n")
		b.WriteString("  [" + i18n.ConfirmPrompt() + "] ")

	case stepAlreadyRunning:
		b.WriteString("  " + styles.Red.Render(i18n.T("already_running_title")) + "\n\n")
		b.WriteString("  " + i18n.T("already_running_dir") + ": " + styles.Teal.Render(m.targetDir) + "\n\n")
		b.WriteString("  " + i18n.T("already_running_hint") + "\n\n")
		b.WriteString("  " + styles.Green.Render("[d]") + " " + i18n.T("already_running_change_dir") + "\n")
		b.WriteString("  " + styles.Green.Render("[q]") + " " + i18n.T("menu_quit") + "\n")

	case stepLaunching:
		b.WriteString("  " + i18n.T("msg_starting") + "\n")
		b.WriteString("  Dir: " + styles.Teal.Render(m.targetDir) + "\n")
		if m.presetSubject != "" {
			b.WriteString("  Subject: " + styles.Violet.Render(m.presetSubject) + "\n")
		}

	case stepSaveConfirm:
		b.WriteString(m.renderSaveBox())
		b.WriteString("  " + i18n.T("save_want") + " [" + i18n.ConfirmPrompt() + "] ")

	case stepSaveSubjectConfirm:
		b.WriteString(m.renderSaveBox())
		if m.isUpdate && m.existingSession != nil {
			b.WriteString("  " + styles.Dim.Render(i18n.T("save_current_subject")+": "+m.existingSession.Subject) + "\n")
		} else {
			b.WriteString("  " + styles.Dim.Render("Auto: "+m.autoSubject) + "\n")
		}
		b.WriteString("  " + i18n.T("save_accept") + " [" + i18n.ConfirmPrompt() + "] ")

	case stepSaveSubjectInput:
		b.WriteString(m.renderSaveBox())
		b.WriteString("  " + i18n.T("save_enter") + ":\n")
		b.WriteString("  > " + m.input + "█\n")
		if m.message != "" {
			b.WriteString("  " + styles.Amber.Render(m.message) + "\n")
		}

	case stepSaveTags:
		b.WriteString(m.renderSaveBox())
		b.WriteString("  Subject: " + styles.Violet.Render(m.chosenSubject) + "\n\n")
		b.WriteString("  " + i18n.T("save_tags") + " " + styles.Dim.Render("(Tab: autocomplete)") + ":\n")
		b.WriteString("  > " + m.input + "█\n")
		if m.tagComplete.Active {
			b.WriteString(m.tagComplete.View())
		}

	case stepDone:
		if m.err != nil {
			b.WriteString("  " + styles.Red.Render(fmt.Sprintf("Error: %v", m.err)) + "\n")
		} else if m.message != "" {
			b.WriteString("  " + styles.Green.Render("✓ "+m.message) + "\n")
		}
		b.WriteString("\n  " + styles.Dim.Render(i18n.T("press_q")) + "\n")
	}

	return b.String()
}

func (m Model) renderSaveBox() string {
	if m.hookData == nil {
		return ""
	}
	h := m.hookData
	fmtIn := lipgloss.NewStyle().Render(formatTokens(h.InputTokens))
	fmtOut := lipgloss.NewStyle().Render(formatTokens(h.OutputTokens))

	var b strings.Builder
	b.WriteString("  " + styles.TealBold.Render(i18n.T("save_title")) + "\n\n")
	b.WriteString("  Dir:        " + h.CWD + "\n")
	b.WriteString("  Tokens in:  " + fmtIn + "\n")
	b.WriteString("  Tokens out: " + fmtOut + "\n\n")
	return b.String()
}

func formatTokens(n int64) string {
	if n <= 0 {
		return "0"
	}
	if n >= 1_000_000 {
		w := n / 100_000
		return fmt.Sprintf("%d.%dM", w/10, w%10)
	}
	if n >= 1_000 {
		w := n / 100
		return fmt.Sprintf("%d.%dk", w/10, w%10)
	}
	return fmt.Sprintf("%d", n)
}

func switchView(v int) tea.Cmd {
	return func() tea.Msg {
		return SwitchViewMsg{View: v}
	}
}
