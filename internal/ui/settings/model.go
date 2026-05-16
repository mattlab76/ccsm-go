package settings

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattlab76/ccsm-go/internal/db"
	"github.com/mattlab76/ccsm-go/internal/i18n"
	"github.com/mattlab76/ccsm-go/internal/model"
	"github.com/mattlab76/ccsm-go/internal/ui/styles"
)

// SwitchViewMsg is sent to the parent app to change views.
type SwitchViewMsg struct {
	View int
}

type state int

const (
	stateMenu state = iota
	stateLangInput
	stateCleanupInput
	stateLogDaysInput
	stateCurrencyInput
	stateRateInput
	statePlanNameInput
	statePlanPriceInput
)

// Number of fields the cursor can stop on (rows shown in the View).
const fieldCount = 7

// Model handles the settings menu.
type Model struct {
	db       *db.DB
	settings *model.Settings
	width    int
	height   int
	state    state
	cursor   int // 0..2 — selected field in stateMenu
	input    string
	message  string
}

// New creates a new settings model.
func New(database *db.DB) Model {
	s, _ := db.GetSettings(database)
	return Model{db: database, settings: s, width: 100, state: stateMenu}
}

// editStateForCursor maps the cursor position to its input state.
func editStateForCursor(c int) state {
	switch c {
	case 0:
		return stateLangInput
	case 1:
		return stateCleanupInput
	case 2:
		return stateLogDaysInput
	case 3:
		return stateCurrencyInput
	case 4:
		return stateRateInput
	case 5:
		return statePlanNameInput
	case 6:
		return statePlanPriceInput
	}
	return stateMenu
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
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()

	if key == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.state {
	case stateMenu:
		switch key {
		case "1":
			m.cursor = 0
			m.state = stateLangInput
			m.input = ""
			m.message = ""
		case "2":
			m.cursor = 1
			m.state = stateCleanupInput
			m.input = ""
			m.message = ""
		case "3":
			m.cursor = 2
			m.state = stateLogDaysInput
			m.input = ""
			m.message = ""
		case "4":
			m.cursor = 3
			m.state = stateCurrencyInput
			m.input = ""
			m.message = ""
		case "5":
			m.cursor = 4
			m.state = stateRateInput
			m.input = ""
			m.message = ""
		case "6":
			m.cursor = 5
			m.state = statePlanNameInput
			m.input = ""
			m.message = ""
		case "7":
			m.cursor = 6
			m.state = statePlanPriceInput
			m.input = ""
			m.message = ""
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < fieldCount-1 {
				m.cursor++
			}
		case "enter":
			m.state = editStateForCursor(m.cursor)
			m.input = ""
			m.message = ""
		case "q", "esc":
			return m, func() tea.Msg { return SwitchViewMsg{View: 0} }
		}

	case stateLangInput:
		switch key {
		case "enter":
			lang := strings.TrimSpace(m.input)
			if lang == "en" || lang == "de" {
				m.settings.Lang = lang
				db.SaveSettings(m.db, m.settings)
				i18n.SetLang(lang)
				db.LogAction(m.db, model.ActionSettings, "language: "+lang)
				m.message = i18n.T("settings_saved")
			} else {
				m.message = i18n.T("settings_invalid")
			}
			m.state = stateMenu
			m.input = ""
		case "esc":
			m.state = stateMenu
			m.input = ""
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			if len(key) == 1 {
				m.input += key
			}
		}

	case stateCleanupInput:
		switch key {
		case "enter":
			val := strings.TrimSpace(m.input)
			n, err := strconv.Atoi(val)
			if err == nil && n >= 0 && n <= 999 {
				m.settings.CleanupDays = n
				db.SaveSettings(m.db, m.settings)
				db.LogAction(m.db, model.ActionSettings, fmt.Sprintf("cleanup_days: %d", n))
				m.message = i18n.T("settings_saved")
			} else {
				m.message = i18n.T("settings_invalid")
			}
			m.state = stateMenu
			m.input = ""
		case "esc":
			m.state = stateMenu
			m.input = ""
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
				m.input += key
			}
		}

	case stateLogDaysInput:
		switch key {
		case "enter":
			val := strings.TrimSpace(m.input)
			n, err := strconv.Atoi(val)
			if err == nil && n >= 0 && n <= 999 {
				m.settings.LogDays = n
				db.SaveSettings(m.db, m.settings)
				db.LogAction(m.db, model.ActionSettings, fmt.Sprintf("log_days: %d", n))
				m.message = i18n.T("settings_saved")
			} else {
				m.message = i18n.T("settings_invalid")
			}
			m.state = stateMenu
			m.input = ""
		case "esc":
			m.state = stateMenu
			m.input = ""
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
				m.input += key
			}
		}

	case stateCurrencyInput:
		switch key {
		case "enter":
			v := strings.ToUpper(strings.TrimSpace(m.input))
			if len(v) >= 3 && len(v) <= 4 {
				m.settings.Currency = v
				db.SaveSettings(m.db, m.settings)
				db.LogAction(m.db, model.ActionSettings, "currency: "+v)
				m.message = i18n.T("settings_saved")
			} else {
				m.message = i18n.T("settings_invalid")
			}
			m.state = stateMenu
			m.input = ""
		case "esc":
			m.state = stateMenu
			m.input = ""
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			if len(key) == 1 && ((key[0] >= 'a' && key[0] <= 'z') ||
				(key[0] >= 'A' && key[0] <= 'Z')) {
				m.input += key
			}
		}

	case stateRateInput:
		switch key {
		case "enter":
			v, err := strconv.ParseFloat(strings.TrimSpace(m.input), 64)
			if err == nil && v > 0 && v < 1000 {
				m.settings.ExchangeRate = v
				db.SaveSettings(m.db, m.settings)
				db.LogAction(m.db, model.ActionSettings, fmt.Sprintf("exchange_rate: %.4f", v))
				m.message = i18n.T("settings_saved")
			} else {
				m.message = i18n.T("settings_invalid")
			}
			m.state = stateMenu
			m.input = ""
		case "esc":
			m.state = stateMenu
			m.input = ""
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			if len(key) == 1 && ((key[0] >= '0' && key[0] <= '9') ||
				(key[0] == '.' && !strings.Contains(m.input, "."))) {
				m.input += key
			}
		}

	case statePlanNameInput:
		switch key {
		case "enter":
			m.settings.PlanName = strings.TrimSpace(m.input)
			db.SaveSettings(m.db, m.settings)
			db.LogAction(m.db, model.ActionSettings, "plan_name: "+m.settings.PlanName)
			m.message = i18n.T("settings_saved")
			m.state = stateMenu
			m.input = ""
		case "esc":
			m.state = stateMenu
			m.input = ""
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			if len(key) == 1 {
				m.input += key
			}
		}

	case statePlanPriceInput:
		switch key {
		case "enter":
			v, err := strconv.ParseFloat(strings.TrimSpace(m.input), 64)
			if err == nil && v >= 0 && v < 100_000 {
				m.settings.PlanMonthlyPrice = v
				db.SaveSettings(m.db, m.settings)
				db.LogAction(m.db, model.ActionSettings, fmt.Sprintf("plan_monthly_price: %.2f", v))
				m.message = i18n.T("settings_saved")
			} else {
				m.message = i18n.T("settings_invalid")
			}
			m.state = stateMenu
			m.input = ""
		case "esc":
			m.state = stateMenu
			m.input = ""
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			if len(key) == 1 && ((key[0] >= '0' && key[0] <= '9') ||
				(key[0] == '.' && !strings.Contains(m.input, "."))) {
				m.input += key
			}
		}

	}

	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	w := m.width
	if w < 60 {
		w = 60
	}
	innerW := w - 8
	if innerW < 56 {
		innerW = 56
	}

	b.WriteString("\n")
	b.WriteString("  " + styles.RenderLogo(i18n.T("settings_title"), 0))
	b.WriteString("\n\n")
	b.WriteString(styles.DoubleLine(innerW))
	b.WriteString("\n\n")

	// Current values, with cursor marker on the active field.
	b.WriteString(settingRow(m.cursor == 0 && m.state == stateMenu, "1",
		i18n.T("settings_lang"),
		i18n.T("settings_lang_current"),
		m.settings.Lang))

	b.WriteString(settingRow(m.cursor == 1 && m.state == stateMenu, "2",
		i18n.T("settings_cleanup"),
		i18n.T("settings_cleanup_current"),
		fmt.Sprintf("%d", m.settings.CleanupDays)))

	b.WriteString(settingRow(m.cursor == 2 && m.state == stateMenu, "3",
		i18n.T("settings_log_days"),
		i18n.T("settings_cleanup_current"),
		fmt.Sprintf("%d", m.settings.LogDays)))

	currency := m.settings.Currency
	if currency == "" {
		currency = "USD"
	}
	b.WriteString(settingRow(m.cursor == 3 && m.state == stateMenu, "4",
		i18n.T("settings_currency"),
		i18n.T("settings_lang_current"),
		currency))

	rate := m.settings.ExchangeRate
	rateVal := "1.0"
	if rate > 0 {
		rateVal = fmt.Sprintf("%.4f", rate)
	}
	b.WriteString(settingRow(m.cursor == 4 && m.state == stateMenu, "5",
		i18n.T("settings_rate"),
		i18n.T("settings_lang_current"),
		rateVal))

	planName := m.settings.PlanName
	if planName == "" {
		planName = "(not set)"
	}
	b.WriteString(settingRow(m.cursor == 5 && m.state == stateMenu, "6",
		i18n.T("settings_plan_name"),
		i18n.T("settings_lang_current"),
		planName))

	planPrice := "(not set)"
	if m.settings.PlanMonthlyPrice > 0 {
		planPrice = fmt.Sprintf("%.2f %s", m.settings.PlanMonthlyPrice, currency)
	}
	b.WriteString(settingRow(m.cursor == 6 && m.state == stateMenu, "7",
		i18n.T("settings_plan_price"),
		i18n.T("settings_lang_current"),
		planPrice))

	b.WriteString("\n")

	if m.message != "" {
		b.WriteString("  " + styles.Green.Render("✓ "+m.message) + "\n\n")
	}

	switch m.state {
	case stateMenu:
		b.WriteString("  " + styles.Dim.Render("↑/↓ navigate  Enter edit  1-7 jump  q back") + "\n")
	case stateLangInput:
		b.WriteString("  " + i18n.T("settings_lang_prompt") + "\n")
		b.WriteString("  > " + m.input + "█\n")
	case stateCleanupInput:
		b.WriteString("  " + i18n.T("settings_cleanup_prompt") + "\n")
		b.WriteString("  > " + m.input + "█\n")
	case stateLogDaysInput:
		b.WriteString("  " + i18n.T("settings_log_days_prompt") + "\n")
		b.WriteString("  > " + m.input + "█\n")
	case stateCurrencyInput:
		b.WriteString("  " + i18n.T("settings_currency_prompt") + "\n")
		b.WriteString("  > " + m.input + "█\n")
	case stateRateInput:
		b.WriteString("  " + i18n.T("settings_rate_prompt") + "\n")
		b.WriteString("  > " + m.input + "█\n")
	case statePlanNameInput:
		b.WriteString("  " + i18n.T("settings_plan_name_prompt") + "\n")
		b.WriteString("  > " + m.input + "█\n")
	case statePlanPriceInput:
		b.WriteString("  " + i18n.T("settings_plan_price_prompt") + "\n")
		b.WriteString("  > " + m.input + "█\n")
	}

	return b.String()
}

func settingRow(selected bool, key, label, currentLabel, value string) string {
	prefix := "  "
	if selected {
		prefix = styles.Violet.Render("▶ ")
	}
	return fmt.Sprintf("%s%s %s  %s %s\n",
		prefix,
		styles.Green.Render("["+key+"]"),
		label,
		styles.Dim.Render(currentLabel+":"),
		styles.Violet.Render(value))
}
