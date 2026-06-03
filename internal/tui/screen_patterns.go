package tui

import (
	"fmt"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"Galileu/internal/guardian"
)

// patternSection identifies which list section the cursor is in.
type patternSection int

const (
	sectionBuiltIn patternSection = iota
	sectionCustom
)

// patternFormField identifies a field in the new/edit pattern form.
type patternFormField int

const (
	fieldName patternFormField = iota
	fieldType
	fieldPattern
	fieldLabel
	fieldActive
	fieldSave
)

// patternForm holds transient state for the add/edit form.
type patternForm struct {
	active      bool
	editIdx     int // -1 = new pattern
	focusedField patternFormField

	name    string
	patType string // "regex" or "literal"
	pattern string
	label   string
	enabled bool

	// derived: regex validation state
	regexValid   bool
	regexChecked bool // true once user has typed something in pattern field
}

// patternScreenState holds all state for screen 2.
type patternScreenState struct {
	loaded  bool
	cfg     guardian.GalileuConfig
	section patternSection
	cursor  int // index within the current section
	form    patternForm
}

var builtInNames = [8]string{
	"openai_key", "openai_project_key", "anthropic_key", "google_key",
	"github_token", "slack_token", "discord_token", "aws_key",
}

func newPatternScreenState(cfg guardian.GalileuConfig) patternScreenState {
	return patternScreenState{
		loaded:  true,
		cfg:     cfg,
		section: sectionBuiltIn,
		cursor:  0,
	}
}

// builtInEnabled returns whether a built-in pattern is enabled by index.
func builtInEnabled(b guardian.BuiltInConfig, idx int) bool {
	switch idx {
	case 0:
		return b.OpenAIKey
	case 1:
		return b.OpenAIProject
	case 2:
		return b.AnthropicKey
	case 3:
		return b.GoogleKey
	case 4:
		return b.GitHubToken
	case 5:
		return b.SlackToken
	case 6:
		return b.DiscordToken
	case 7:
		return b.AWSKey
	}
	return false
}

// toggleBuiltIn flips the built-in flag at idx.
func toggleBuiltIn(b guardian.BuiltInConfig, idx int) guardian.BuiltInConfig {
	switch idx {
	case 0:
		b.OpenAIKey = !b.OpenAIKey
	case 1:
		b.OpenAIProject = !b.OpenAIProject
	case 2:
		b.AnthropicKey = !b.AnthropicKey
	case 3:
		b.GoogleKey = !b.GoogleKey
	case 4:
		b.GitHubToken = !b.GitHubToken
	case 5:
		b.SlackToken = !b.SlackToken
	case 6:
		b.DiscordToken = !b.DiscordToken
	case 7:
		b.AWSKey = !b.AWSKey
	}
	return b
}

// updatePatterns handles all key input for the patterns screen.
func (m model) updatePatterns(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Form is open — route to form handler.
	if m.pat.form.active {
		return m.updatePatternForm(key)
	}

	switch key {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "b", "B", "esc":
		m.currentScreen = screenDashboard
		return m, nil
	case "n", "N":
		m.pat.form = patternForm{
			active:  true,
			editIdx: -1,
			patType: "regex",
			enabled: true,
		}
		return m, nil
	case "up", "k":
		m = m.patMoveCursor(-1)
	case "down", "j":
		m = m.patMoveCursor(1)
	case " ", "enter":
		m = m.patToggleOrEdit()
		return m.savePatternConfig()
	case "d", "D":
		if m.pat.section == sectionCustom && len(m.pat.cfg.Analyzer.CustomPatterns) > 0 {
			idx := m.pat.cursor
			if idx < len(m.pat.cfg.Analyzer.CustomPatterns) {
				name := m.pat.cfg.Analyzer.CustomPatterns[idx].Name
				m.confirm = &confirmMsg{
					prompt: fmt.Sprintf("Apagar padrão '%s'?", name),
					action: func(mm model) (model, tea.Cmd) {
						return mm.deleteCustomPattern(idx)
					},
				}
			}
		}
		return m, nil
	}

	return m, nil
}

func (m model) patMoveCursor(delta int) model {
	switch m.pat.section {
	case sectionBuiltIn:
		next := m.pat.cursor + delta
		if next < 0 {
			// wrap to custom section
			if len(m.pat.cfg.Analyzer.CustomPatterns) > 0 {
				m.pat.section = sectionCustom
				m.pat.cursor = len(m.pat.cfg.Analyzer.CustomPatterns) - 1
			}
		} else if next >= len(builtInNames) {
			if len(m.pat.cfg.Analyzer.CustomPatterns) > 0 {
				m.pat.section = sectionCustom
				m.pat.cursor = 0
			}
		} else {
			m.pat.cursor = next
		}
	case sectionCustom:
		next := m.pat.cursor + delta
		if next < 0 {
			m.pat.section = sectionBuiltIn
			m.pat.cursor = len(builtInNames) - 1
		} else if next >= len(m.pat.cfg.Analyzer.CustomPatterns) {
			// stay at last
		} else {
			m.pat.cursor = next
		}
	}
	return m
}

func (m model) patToggleOrEdit() model {
	switch m.pat.section {
	case sectionBuiltIn:
		m.pat.cfg.Analyzer.BuiltIn = toggleBuiltIn(m.pat.cfg.Analyzer.BuiltIn, m.pat.cursor)
	case sectionCustom:
		idx := m.pat.cursor
		if idx < len(m.pat.cfg.Analyzer.CustomPatterns) {
			cp := m.pat.cfg.Analyzer.CustomPatterns[idx]
			m.pat.form = patternForm{
				active:  true,
				editIdx: idx,
				name:    cp.Name,
				patType: cp.Type,
				pattern: cp.Pattern,
				label:   cp.Label,
				enabled: cp.Enabled,
			}
			if cp.Type == "literal" && len(cp.Values) > 0 {
				m.pat.form.pattern = strings.Join(cp.Values, ", ")
			}
			m = m.validateFormRegex()
		}
	}
	return m
}

func (m model) savePatternConfig() (model, tea.Cmd) {
	if err := guardian.SaveConfig(m.configPath, m.pat.cfg); err != nil {
		m.statusMsg = "Erro ao guardar: " + err.Error()
		m.statusIsErr = true
		return m, scheduleStatusClear()
	}
	m.statusMsg = "✓ Guardado"
	m.statusIsErr = false
	return m, scheduleStatusClear()
}

func (m model) deleteCustomPattern(idx int) (model, tea.Cmd) {
	patterns := m.pat.cfg.Analyzer.CustomPatterns
	m.pat.cfg.Analyzer.CustomPatterns = append(patterns[:idx], patterns[idx+1:]...)
	if m.pat.cursor >= len(m.pat.cfg.Analyzer.CustomPatterns) && m.pat.cursor > 0 {
		m.pat.cursor--
	}
	return m.savePatternConfig()
}

// updatePatternForm handles key input while the inline form is open.
func (m model) updatePatternForm(key string) (tea.Model, tea.Cmd) {
	f := &m.pat.form

	switch key {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.pat.form = patternForm{}
		return m, nil
	case "tab", "down":
		if f.focusedField < fieldSave {
			f.focusedField++
		}
	case "shift+tab", "up":
		if f.focusedField > fieldName {
			f.focusedField--
		}
	case "enter":
		if f.focusedField == fieldSave {
			return m.commitPatternForm()
		}
		if f.focusedField < fieldSave {
			f.focusedField++
		}
	case " ":
		switch f.focusedField {
		case fieldType:
			if f.patType == "regex" {
				f.patType = "literal"
			} else {
				f.patType = "regex"
			}
			m = m.validateFormRegex()
		case fieldActive:
			f.enabled = !f.enabled
		}
	case "backspace":
		switch f.focusedField {
		case fieldName:
			if len(f.name) > 0 {
				f.name = f.name[:len(f.name)-1]
			}
		case fieldPattern:
			if len(f.pattern) > 0 {
				f.pattern = f.pattern[:len(f.pattern)-1]
				m = m.validateFormRegex()
			}
		case fieldLabel:
			if len(f.label) > 0 {
				f.label = f.label[:len(f.label)-1]
			}
		}
	default:
		if len(key) == 1 {
			switch f.focusedField {
			case fieldName:
				f.name += key
			case fieldPattern:
				f.pattern += key
				m = m.validateFormRegex()
			case fieldLabel:
				f.label += key
			}
		}
	}

	return m, nil
}

func (m model) validateFormRegex() model {
	f := &m.pat.form
	if f.patType != "regex" || f.pattern == "" {
		f.regexChecked = false
		f.regexValid = false
		return m
	}
	f.regexChecked = true
	_, err := regexp.Compile(f.pattern)
	f.regexValid = (err == nil)
	return m
}

func (m model) commitPatternForm() (model, tea.Cmd) {
	f := m.pat.form
	if f.name == "" {
		m.statusMsg = "Nome não pode estar vazio"
		m.statusIsErr = true
		m.pat.form = patternForm{}
		return m, scheduleStatusClear()
	}

	cp := guardian.CustomPattern{
		Name:    f.name,
		Type:    f.patType,
		Label:   f.label,
		Enabled: f.enabled,
	}
	if f.patType == "literal" {
		for _, v := range strings.Split(f.pattern, ",") {
			v = strings.TrimSpace(v)
			if v != "" {
				cp.Values = append(cp.Values, v)
			}
		}
	} else {
		cp.Pattern = f.pattern
	}

	if f.editIdx >= 0 && f.editIdx < len(m.pat.cfg.Analyzer.CustomPatterns) {
		m.pat.cfg.Analyzer.CustomPatterns[f.editIdx] = cp
	} else {
		m.pat.cfg.Analyzer.CustomPatterns = append(m.pat.cfg.Analyzer.CustomPatterns, cp)
		m.pat.section = sectionCustom
		m.pat.cursor = len(m.pat.cfg.Analyzer.CustomPatterns) - 1
	}

	m.pat.form = patternForm{}
	return m.savePatternConfig()
}

// viewPatterns renders the patterns screen.
func (m model) viewPatterns(width int) string {
	var b strings.Builder

	right := dimStyle.Render("[N] Novo")
	b.WriteString(renderHeader("PADRÕES DE DETECÇÃO", right, width))
	b.WriteString("\n")
	b.WriteString(renderDivider(width))
	b.WriteString("\n")

	listWidth := width/2 - 1
	detailWidth := width - listWidth - 1

	listPanel := m.renderPatternList(listWidth)
	detailPanel := m.renderPatternDetail(detailWidth)

	listLines := strings.Split(listPanel, "\n")
	detailLines := strings.Split(detailPanel, "\n")

	bodyHeight := m.height - 5
	if bodyHeight < 4 {
		bodyHeight = 4
	}
	for len(listLines) < bodyHeight {
		listLines = append(listLines, "")
	}
	for len(detailLines) < bodyHeight {
		detailLines = append(detailLines, "")
	}

	sep := dividerStyle.Render("│")
	for i := 0; i < bodyHeight; i++ {
		lLine := padRight(listLines[i], listWidth)
		rLine := ""
		if i < len(detailLines) {
			rLine = detailLines[i]
		}
		b.WriteString(lLine + sep + rLine + "\n")
	}

	hints := "  [N] Novo  [D] Apagar  [Enter] Editar  [↑↓] Navegar  [B] Voltar"
	b.WriteString(renderFooter(hints, m.statusMsg, m.statusIsErr, width))

	return b.String()
}

func (m model) renderPatternList(width int) string {
	var b strings.Builder
	b.WriteString(sectionStyle.Render("  BUILT-IN") + "\n")

	for i, name := range builtInNames {
		enabled := builtInEnabled(m.pat.cfg.Analyzer.BuiltIn, i)
		cb := checkboxStr(enabled)
		line := fmt.Sprintf("  %s %s", cb, name)
		if m.pat.section == sectionBuiltIn && m.pat.cursor == i {
			b.WriteString(focusedStyle.Render(line) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("  CUSTOM") + "\n")

	if len(m.pat.cfg.Analyzer.CustomPatterns) == 0 {
		b.WriteString(dimStyle.Render("  (nenhum)") + "\n")
	}
	for i, cp := range m.pat.cfg.Analyzer.CustomPatterns {
		cb := checkboxStr(cp.Enabled)
		line := fmt.Sprintf("  ▶ %s  %s", truncate(cp.Name, 18), cb)
		if m.pat.section == sectionCustom && m.pat.cursor == i {
			b.WriteString(focusedStyle.Render(line) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}

	return b.String()
}

func (m model) renderPatternDetail(width int) string {
	if m.pat.form.active {
		return m.renderPatternForm(width)
	}

	// Show detail of selected custom pattern, or empty panel for built-in.
	if m.pat.section == sectionBuiltIn {
		name := ""
		if m.pat.cursor < len(builtInNames) {
			name = builtInNames[m.pat.cursor]
		}
		enabled := builtInEnabled(m.pat.cfg.Analyzer.BuiltIn, m.pat.cursor)
		var b strings.Builder
		b.WriteString(sectionStyle.Render("  DETALHE") + "\n")
		b.WriteString(labelValue("  Nome", name) + "\n")
		b.WriteString(labelValue("  Tipo", "built-in") + "\n")
		b.WriteString(labelValue("  Activo", boolStr(enabled)) + "\n")
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  [Space/Enter] alternar activo") + "\n")
		return b.String()
	}

	if len(m.pat.cfg.Analyzer.CustomPatterns) == 0 || m.pat.cursor >= len(m.pat.cfg.Analyzer.CustomPatterns) {
		return dimStyle.Render("  (sem selecção)") + "\n"
	}

	cp := m.pat.cfg.Analyzer.CustomPatterns[m.pat.cursor]
	var b strings.Builder
	b.WriteString(sectionStyle.Render("  DETALHE") + "\n")
	b.WriteString(labelValue("  Nome", cp.Name) + "\n")
	b.WriteString(labelValue("  Tipo", cp.Type) + "\n")
	if cp.Type == "literal" {
		b.WriteString(labelValue("  Valores", strings.Join(cp.Values, ", ")) + "\n")
	} else {
		b.WriteString(labelValue("  Padrão", cp.Pattern) + "\n")
	}
	b.WriteString(labelValue("  Label", cp.Label) + "\n")
	b.WriteString(labelValue("  Activo", boolStr(cp.Enabled)) + "\n")
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  [Enter] editar  [D] apagar") + "\n")
	return b.String()
}

func (m model) renderPatternForm(width int) string {
	f := m.pat.form
	var b strings.Builder

	title := "  NOVO PADRÃO"
	if f.editIdx >= 0 {
		title = "  EDITAR PADRÃO"
	}
	b.WriteString(sectionStyle.Render(title) + "\n\n")

	fields := []struct {
		field patternFormField
		label string
		value string
	}{
		{fieldName, "  Nome", f.name},
		{fieldType, "  Tipo", "[ " + f.patType + " ]  (Espaço alterna)"},
		{fieldPattern, "  Padrão / Valores", f.pattern},
		{fieldLabel, "  Label", f.label},
		{fieldActive, "  Activo", checkboxStr(f.enabled) + "  (Espaço alterna)"},
		{fieldSave, "  [Guardar]", ""},
	}

	for _, fd := range fields {
		line := fd.label
		if fd.value != "" {
			line += ":  " + fd.value
		}

		// Regex validation border on the pattern field
		if fd.field == fieldPattern && f.patType == "regex" {
			var style = inputPartialStyle
			if f.regexChecked {
				if f.regexValid {
					style = inputValidStyle
				} else {
					style = inputInvalidStyle
				}
			}
			if fd.field == f.focusedField {
				b.WriteString(focusedStyle.Render(line) + "\n")
				hint := style.Render(" ")
				b.WriteString(hint + "\n")
			} else {
				b.WriteString(line + "\n")
			}
			continue
		}

		if fd.field == f.focusedField {
			b.WriteString(focusedStyle.Render(line) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  Tab/↑↓ navegar  Enter guardar  Esc cancelar") + "\n")
	return b.String()
}

func boolStr(v bool) string {
	if v {
		return "sim"
	}
	return "não"
}
