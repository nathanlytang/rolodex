package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	lg "github.com/charmbracelet/lipgloss"
	"github.com/nathanlytang/rolodex/internal/logger"
	"github.com/nathanlytang/rolodex/internal/ssh"
)

type Model struct {
	list    list.Model
	hosts   []Host
	err     error
	showErr bool
}

type Item struct {
	host Host
}

type Host struct {
	Name               string `json:"name"`
	Host               string `json:"host"`
	Port               int    `json:"port"`
	User               string `json:"user"`
	SSHAgent           bool   `json:"ssh_agent,omitempty"`
	IdentityFile       string `json:"identity_file,omitempty"`
	IdentityPassphrase string `json:"identity_passphrase,omitempty"`
	KeyringService     string `json:"keyring_service,omitempty"`
	KeyringAccount     string `json:"keyring_account,omitempty"`
	Password           string `json:"password,omitempty"`
}

type Folder struct {
	Name  string `json:"name"`
	Hosts []Host `json:"hosts"`
}

type Configuration struct {
	Folders []Folder `json:"folders"`
	Hosts   []Host   `json:"hosts"`
}

type resetListMsg struct{}

type errorMsg struct {
	err error
}

var docStyle = lg.NewStyle().Margin(1, 2)
var enter = key.NewBinding(key.WithKeys("enter"), key.WithHelp("⏎", "connect"))
var isGlobalQuit = false

func (i Item) Title() string       { return i.host.Name }
func (i Item) Description() string { return i.host.Host }
func (i Item) FilterValue() string { return i.host.Name }

func buildList(hosts []Host) list.Model {
	items := []list.Item{}
	for _, h := range hosts {
		it := Item{host: h}
		items = append(items, it)
	}
	hostList := list.New(items, list.NewDefaultDelegate(), 0, 0)
	hostList.Title = "Rolodex"
	hostList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{enter}
	}
	return hostList
}

func initialModel(hosts []Host) Model {
	return Model{
		list:  buildList(hosts),
		hosts: hosts,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If showing error, any key dismisses it (except quit)
		if m.showErr {
			if msg.String() == "ctrl+c" || msg.String() == "q" {
				isGlobalQuit = true
				return Quit(m)
			}
			// Any other key dismisses the error
			m.showErr = false
			m.err = nil
			return m, nil
		}

		if msg.String() == "ctrl+c" || msg.String() == "q" {
			isGlobalQuit = true
			return Quit(m)
		}

		if key.Matches(msg, enter) {
			selected := m.list.SelectedItem()
			if selected != nil {
				if it, ok := selected.(Item); ok {
					return m, func() tea.Msg {
						clearScreen()
						authConfig := ssh.AuthConfig{
							SSHAgent:           it.host.SSHAgent,
							IdentityFile:       it.host.IdentityFile,
							IdentityPassphrase: it.host.IdentityPassphrase,
							KeyringService:     it.host.KeyringService,
							KeyringAccount:     it.host.KeyringAccount,
							Password:           it.host.Password,
						}
						err := ssh.StartSession(it.host.Host, it.host.Port, it.host.User, authConfig)
						if err != nil {
							return errorMsg{err: err}
						}
						return resetListMsg{}
					}
				}
			}
		}

	case errorMsg:
		m.err = msg.err
		m.showErr = true
		return m, nil

	case resetListMsg:
		m.list = buildList(m.hosts)
		m.showErr = false
		m.err = nil
		docStyle.Render(m.list.View())
		m.list, _ = m.list.Update(msg)
		return Quit(m)

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.showErr && m.err != nil {
		errorStyle := lg.NewStyle().
			Bold(true).
			Foreground(lg.Color("#EE0000")).
			Padding(1, 2)

		headerStyle := lg.NewStyle().
			Bold(true).
			Foreground(lg.Color("#FFFF00")).
			Padding(0, 2)

		footerStyle := lg.NewStyle().
			Foreground(lg.Color("#888888")).
			Padding(1, 2)

		header := headerStyle.Render("⚠  Connection Error")
		errMsg := errorStyle.Render(m.err.Error() + "\n\nCheck the logs for more details.")
		footer := footerStyle.Render("Press 'q' to quit or any other key to return to the list.")

		return docStyle.Render(header + "\n" + errMsg + "\n" + footer)
	}
	return docStyle.Render(m.list.View())
}

func Quit(m Model) (tea.Model, tea.Cmd) {
	// m.quit = true
	return m, tea.Quit
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

// Returns the directory containing the executable
func getExecutableDir() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exePath), nil
}

func main() {
	if err := logger.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	// Get the directory where the executable is located
	exeDir, err := getExecutableDir()
	if err != nil {
		logger.Fatalf("Failed to get executable directory: %v", err)
		fmt.Fprintf(os.Stderr, "Error: Failed to get executable directory: %v\n", err)
		os.Exit(1)
	}

	// Look for config.json in the executable's directory
	configPath := filepath.Join(exeDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		logger.Fatalf("Failed to read config.json from %s: %v", configPath, err)
		fmt.Fprintf(os.Stderr, "Error: Failed to read config.json from %s: %v\n", configPath, err)
		os.Exit(1)
	}

	configuration := &Configuration{}
	if err := json.Unmarshal(data, &configuration); err != nil {
		logger.Fatalf("Failed to parse config.json: %v", err)
		fmt.Fprintf(os.Stderr, "Error: Failed to parse config.json: %v\n", err)
		os.Exit(1)
	}

	logger.Printf("Loaded configuration with %d hosts", len(configuration.Hosts))

	p := tea.NewProgram(initialModel(configuration.Hosts), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		logger.Fatalf("Application error: %v", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	logger.Printf("Application exited normally")
	os.Exit(0)
}
