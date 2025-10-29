package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	lg "github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

type formKeyMap struct {
	Navigate key.Binding
	Submit   key.Binding
	Cancel   key.Binding
}

func (k formKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Navigate, k.Submit, k.Cancel}
}

func (k formKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Navigate, k.Submit, k.Cancel},
	}
}

var formKeys = formKeyMap{
	Navigate: key.NewBinding(
		key.WithKeys("tab", "shift+tab", "up", "down"),
		key.WithHelp("↑/↓/tab", "navigate"),
	),
	Submit: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("⏎", "submit"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
}

func newFormModel() formModel {
	inputs := make([]textinput.Model, 10)

	for i := range inputs {
		t := textinput.New()
		t.Prompt = "> "
		t.PromptStyle = lg.NewStyle().Foreground(lg.Color("#7D56F4")).Margin(0, 0, 0, 2)
		t.CharLimit = 256

		switch i {
		case nameInput:
			t.Focus()
		case portInput:
			t.CharLimit = 5
		case identityPassphraseInput:
			t.EchoMode = textinput.EchoPassword
		case passwordInput:
			t.EchoMode = textinput.EchoPassword
		}

		inputs[i] = t
	}

	return formModel{
		inputs:     inputs,
		focusIndex: 0,
	}
}

func validateAndCreateHost(f formModel) (Host, error) {
	// Validate required fields
	if f.inputs[nameInput].Value() == "" {
		return Host{}, fmt.Errorf("name is required")
	}
	if f.inputs[hostInput].Value() == "" {
		return Host{}, fmt.Errorf("host/IP is required")
	}
	if f.inputs[userInput].Value() == "" {
		return Host{}, fmt.Errorf("user is required")
	}

	// Parse port
	portStr := f.inputs[portInput].Value()
	if portStr == "" {
		portStr = "22" // Default port
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return Host{}, fmt.Errorf("invalid port number")
	}

	// Parse SSH Agent
	sshAgent := false
	if f.inputs[sshAgentInput].Value() == "true" {
		sshAgent = true
	}

	return Host{
		Name:               f.inputs[nameInput].Value(),
		Host:               f.inputs[hostInput].Value(),
		Port:               port,
		User:               f.inputs[userInput].Value(),
		SSHAgent:           sshAgent,
		IdentityFile:       f.inputs[identityFileInput].Value(),
		IdentityPassphrase: f.inputs[identityPassphraseInput].Value(),
		KeyringService:     f.inputs[keyringServiceInput].Value(),
		KeyringAccount:     f.inputs[keyringAccountInput].Value(),
		Password:           f.inputs[passwordInput].Value(),
	}, nil
}

func (m Model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel and return to list
		m.view = listView
		return m, nil

	case "tab", "shift+tab", "up", "down":
		// Navigate between inputs
		s := msg.String()

		if s == "up" || s == "shift+tab" {
			m.form.focusIndex--
		} else {
			m.form.focusIndex++
		}

		if m.form.focusIndex > len(m.form.inputs)-1 {
			m.form.focusIndex = 0
		} else if m.form.focusIndex < 0 {
			m.form.focusIndex = len(m.form.inputs) - 1
		}

		// Update scroll offset to keep focused input visible
		m.form.scrollOffset = m.calculateScrollOffset()

		cmds := make([]tea.Cmd, len(m.form.inputs))
		for i := 0; i < len(m.form.inputs); i++ {
			if i == m.form.focusIndex {
				cmds[i] = m.form.inputs[i].Focus()
			} else {
				m.form.inputs[i].Blur()
			}
		}

		return m, tea.Batch(cmds...)

	case "enter":
		// Submit form
		newHost, err := validateAndCreateHost(m.form)
		if err != nil {
			m.err = err
			m.showErr = true
			m.view = listView
			return m, nil
		}

		// Save to config
		if err := saveHostToConfig(m.configPath, newHost); err != nil {
			m.err = fmt.Errorf("failed to save host: %w", err)
			m.showErr = true
			m.view = listView
			return m, nil
		}

		// Reload config
		data, err := os.ReadFile(m.configPath)
		if err != nil {
			m.err = fmt.Errorf("failed to reload config: %w", err)
			m.showErr = true
			m.view = listView
			return m, nil
		}

		var config Configuration
		if err := json.Unmarshal(data, &config); err != nil {
			m.err = fmt.Errorf("failed to parse reloaded config: %w", err)
			m.showErr = true
			m.view = listView
			return m, nil
		}

		// Update model with new hosts and return to list
		m.hosts = config.Hosts
		m.list = buildList(m.hosts)
		m.view = listView
		// Trigger window size update to refresh list
		return m, func() tea.Msg {
			w, h, _ := term.GetSize(int(os.Stdout.Fd()))
			return tea.WindowSizeMsg{Width: w, Height: h}
		}
	}

	// Update the focused input
	var cmd tea.Cmd
	m.form.inputs[m.form.focusIndex], cmd = m.form.inputs[m.form.focusIndex].Update(msg)
	return m, cmd
}

func (m Model) renderForm() string {
	titleStyle := lg.NewStyle().
		Bold(true).
		Foreground(lg.Color("#DDDDDD")).
		Background(lg.Color("62")).
		Padding(0, 1).
		Margin(0, 0, 0, 2)

	labelStyle := lg.NewStyle().
		Foreground(lg.Color("#DDDDDD")).
		Bold(true).
		Width(40).
		Margin(0, 0, 0, 2)

	requiredStyle := lg.NewStyle().
		Foreground(lg.Color("#ED5679"))

	optionalStyle := lg.NewStyle().
		Foreground(lg.Color("#888888"))

	helpRendered, availHeight := m.renderFormHelp(formKeys)

	// Title is always visible at the top
	var title string
	title = titleStyle.Render("Add New Host Configuration") + "\n\n"

	// Subtract title height from available height for content
	availHeight -= lg.Height(title)

	// Build form content
	var b string

	// Authentication section header
	authHeaderStyle := lg.NewStyle().
		Foreground(lg.Color("#00FFFF")).
		Bold(true).
		Margin(0, 0, 0, 2)

	authTypeStyle := lg.NewStyle().
		Foreground(lg.Color("#888888")).
		Italic(true).
		Margin(1, 0, 1, 2)

	for i, input := range m.form.inputs {
		// Add section headers
		if i == sshAgentInput {
			b += authHeaderStyle.Render("Authentication (minimum one auth method required):") + "\n"
		}

		// Add auth type labels with separators
		switch i {
		case sshAgentInput:
			b += authTypeStyle.Render("SSH Agent Authentication") + "\n"
		case identityFileInput:
			b += authTypeStyle.Render("Identity File Authentication") + "\n"
		case keyringServiceInput:
			b += authTypeStyle.Render("Keyring Authentication") + "\n"
		case passwordInput:
			b += authTypeStyle.Render("Password Authentication") + "\n"
		}

		label := inputLabels[i]
		isRequired := i < userInput+1 // First 4 fields are required

		var labelText string
		if isRequired {
			labelText = labelStyle.Render(label) + " " + requiredStyle.Render("*")
		} else {
			if i == identityPassphraseInput {
				labelText = labelStyle.Render(label) + " " + optionalStyle.Render("(optional)")
			} else {
				labelText = labelStyle.Render(label)
			}
		}

		b += labelText + "\n"
		b += input.View() + "\n\n"
	}

	return m.calculateVisibleFormContent(availHeight, b, title, helpRendered, m.getVisibleFormLines)
}

// Determines the scroll offset to keep the focused input visible
func (m Model) calculateScrollOffset() int {
	// Calculate the line position of the focused input
	linesPerInput := 4

	// Add extra lines for section headers
	extraLines := 0
	if m.form.focusIndex >= sshAgentInput {
		extraLines += 2 // Auth header
	}
	if m.form.focusIndex >= identityFileInput {
		extraLines += 2 // Identity auth type
	}
	if m.form.focusIndex >= keyringServiceInput {
		extraLines += 2 // Keyring auth type
	}
	if m.form.focusIndex >= passwordInput {
		extraLines += 2 // Password auth type
	}

	focusedLine := m.form.focusIndex*linesPerInput + extraLines

	// Get available height
	_, availHeight := m.renderFormHelp(formKeys)

	// Calculate scroll offset to keep focused input in view
	// Keep some padding (show a few lines above the focused input)
	padding := 3

	if focusedLine < m.form.scrollOffset+padding {
		// Scroll up
		return max(0, focusedLine-padding)
	}

	if focusedLine > m.form.scrollOffset+availHeight-padding {
		// Scroll down
		return focusedLine - availHeight + padding
	}

	return m.form.scrollOffset
}

// Returns the visible portion of form lines based on scroll offset
func (m Model) getVisibleFormLines(lines []string, availHeight int) []string {
	if len(lines) <= availHeight {
		// Form fits on screen, no scrolling needed
		return lines
	}

	start := m.form.scrollOffset
	end := min(start+availHeight, len(lines))

	if start >= len(lines) {
		start = max(0, len(lines)-availHeight)
		end = len(lines)
	}

	return lines[start:end]
}
