package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/textinput"
	lg "github.com/charmbracelet/lipgloss"
)

type formModel struct {
	inputs       []textinput.Model
	focusIndex   int
	submitting   bool
	scrollOffset int // Track scroll position for large forms
}

const (
	nameInput = iota
	hostInput
	portInput
	userInput
	sshAgentInput
	identityFileInput
	identityPassphraseInput
	keyringServiceInput
	keyringAccountInput
	passwordInput
)

var inputLabels = []string{
	"Name",
	"Host/IP",
	"Port",
	"User",
	"Use SSH Agent (true/false)",
	"Identity File Path",
	"Identity Passphrase",
	"Keyring Service",
	"Keyring Account",
	"Password",
}

// Renders the help view and subtracts its height from available height
func (m Model) renderFormHelp(keys help.KeyMap) (string, int) {
	helpStyle := lg.NewStyle().
		Padding(1, 0, 0, 2)

	availHeight := m.height
	if availHeight == 0 {
		availHeight = 40 // fallback
	}

	// Subtract docStyle margins
	_, v := docStyle.GetFrameSize()
	availHeight -= v

	// Build help view and subtract its height
	helpModel := help.New()
	helpView := helpModel.View(keys)
	helpRendered := helpStyle.Render(helpView)
	availHeight -= lg.Height(helpRendered)

	return helpRendered, availHeight
}

// Calculates the visible content of a form based on available height
func (m Model) calculateVisibleFormContent(
	availHeight int,
	b string,
	title string,
	helpRendered string,
	getFormLines func(lines []string, availHeight int) []string,
) string {
	lines := strings.Split(b, "\n")
	visibleLines := getFormLines(lines, availHeight)
	content := lg.NewStyle().Height(availHeight).Render(strings.Join(visibleLines, "\n"))
	return docStyle.Render(lg.JoinVertical(lg.Left, title, content, helpRendered))
}

// Saves a new host to the config file
func saveHostToConfig(configPath string, newHost Host) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var config Configuration
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	config.Hosts = append(config.Hosts, newHost)

	prettyJSON, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, prettyJSON, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Deletes a host from the config file
func deleteHostFromConfig(configPath string, hostIndex int) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	var config Configuration
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if hostIndex < 0 || hostIndex >= len(config.Hosts) {
		return fmt.Errorf("invalid host index")
	}
	config.Hosts = append(config.Hosts[:hostIndex], config.Hosts[hostIndex+1:]...)

	prettyJSON, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, prettyJSON, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
