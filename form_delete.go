package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	lg "github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Key map for delete confirmation view
type deleteKeyMap struct {
	Confirm key.Binding
	Cancel  key.Binding
}

func (k deleteKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Confirm, k.Cancel}
}

func (k deleteKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Confirm, k.Cancel},
	}
}

var deleteKeys = deleteKeyMap{
	Confirm: key.NewBinding(
		key.WithKeys("y", "Y"),
		key.WithHelp("y", "confirm"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("n", "N", "esc"),
		key.WithHelp("n/esc", "cancel"),
	),
}

func (m Model) updateDeleteConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// Confirm deletion
		if err := deleteHostFromConfig(m.configPath, m.hostToDeleteIndex); err != nil {
			m.err = fmt.Errorf("failed to delete host: %w", err)
			m.showErr = true
			m.view = listView
			m.hostToDelete = nil
			return m, nil
		}

		// Reload config
		data, err := os.ReadFile(m.configPath)
		if err != nil {
			m.err = fmt.Errorf("failed to reload config: %w", err)
			m.showErr = true
			m.view = listView
			m.hostToDelete = nil
			return m, nil
		}

		var config Configuration
		if err := json.Unmarshal(data, &config); err != nil {
			m.err = fmt.Errorf("failed to parse reloaded config: %w", err)
			m.showErr = true
			m.view = listView
			m.hostToDelete = nil
			return m, nil
		}

		// Update model with new hosts and return to list
		m.hosts = config.Hosts
		m.list = buildList(m.hosts)
		m.view = listView
		m.hostToDelete = nil
		// Trigger window size update to refresh list
		return m, func() tea.Msg {
			w, h, _ := term.GetSize(int(os.Stdout.Fd()))
			return tea.WindowSizeMsg{Width: w, Height: h}
		}

	case "n", "N", "esc":
		// Cancel deletion
		m.view = listView
		m.hostToDelete = nil
		return m, nil
	}

	return m, nil
}

func (m Model) renderDeleteConfirm() string {
	titleStyle := lg.NewStyle().
		Bold(true).
		Foreground(lg.Color("#DDDDDD")).
		Background(lg.Color("62")).
		Padding(0, 1).
		Margin(0, 0, 0, 2)

	hostDescriptionStyle := lg.NewStyle().
		Foreground(lg.Color("#DDDDDD")).
		Padding(0, 1)

	hostStyle := lg.NewStyle().
		Foreground(lg.Color("#EE6FF8")).
		Bold(true).
		Margin(0, 2)

	infoStyle := lg.NewStyle().
		Foreground(lg.Color("#ED5679")).
		Padding(0, 2)

	helpRendered, availHeight := m.renderFormHelp(deleteKeys)

	var title string
	title = titleStyle.Render("Delete Host") + "\n\n"
	availHeight -= lg.Height(title)
	var b string

	if m.hostToDelete != nil {
		b += infoStyle.Render("Are you sure you want to delete this host?") + "\n\n"
		b += hostStyle.Render("Name") + hostDescriptionStyle.Render(m.hostToDelete.Name) + "\n"
		b += hostStyle.Render("Host") + hostDescriptionStyle.Render(m.hostToDelete.Host) + "\n"
		b += hostStyle.Render("User") + hostDescriptionStyle.Render(m.hostToDelete.User) + "\n\n"
		b += infoStyle.Render("This action cannot be undone.") + "\n\n"
	}

	return m.calculateVisibleFormContent(availHeight, b, title, helpRendered, m.getVisibleDeleteLines)
}

// Returns the visible portion of delete form lines
// TODO: Does not resize properly, but the delete form is small so not a big deal for now
func (m Model) getVisibleDeleteLines(lines []string, availHeight int) []string {
	if len(lines) <= availHeight {
		return lines
	}

	end := min(availHeight, len(lines))
	return lines[0:end]
}
