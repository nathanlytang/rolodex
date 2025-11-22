package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	lg "github.com/charmbracelet/lipgloss"
	"github.com/micmonay/keybd_event"
	"github.com/nathanlytang/rolodex/internal/logger"
	"github.com/nathanlytang/rolodex/internal/ssh"
	"golang.org/x/term"
)

type viewState int

const (
	listView viewState = iota
	formView
	deleteConfirmView
)

type Model struct {
	list              list.Model
	hosts             []Host
	err               error
	showErr           bool
	view              viewState
	form              formModel
	configPath        string
	hostToDelete      *Host
	hostToDeleteIndex int
	width             int
	height            int
	connectHost       *Host
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
var addHost = key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add host"))
var deleteHost = key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete host"))

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
		return []key.Binding{enter, addHost, deleteHost}
	}
	return hostList
}

func initialModel(hosts []Host, configPath string) Model {
	return Model{
		list:       buildList(hosts),
		hosts:      hosts,
		view:       listView,
		configPath: configPath,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	kb, _ := keybd_event.NewKeyBonding()
	kb.SetKeys(keybd_event.VK_SPACE)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global quit
		if msg.String() == "ctrl+c" {
			return Quit(m)
		}

		// Handle based on current view
		switch m.view {
		case formView:
			return m.updateForm(msg)
		case deleteConfirmView:
			return m.updateDeleteConfirm(msg)
		}
		return m.updateList(msg)

	case errorMsg:
		m.err = msg.err
		m.showErr = true
		m.view = listView
		return m, nil

	case resetListMsg:
		return m, func() tea.Msg {
			w, h, _ := term.GetSize(int(os.Stdout.Fd()))
			return tea.WindowSizeMsg{Width: w, Height: h}
		}

	case tea.WindowSizeMsg:
		logger.Printf("Window size: %d x %d", msg.Width, msg.Height)
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		m.width = msg.Width
		m.height = msg.Height

		// HACK: Keyboard event so that arrow keys work immediately
		// TODO: Figure out why an extra initial key press is needed
		kb.Press()
		time.Sleep(10 * time.Millisecond)
		kb.Release()
	}

	// Pass other messages to the list if in list view
	if m.view == listView {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If showing error, any key dismisses it (except quit)
	if m.showErr {
		if msg.String() == "q" {
			return Quit(m)
		}

		m.showErr = false
		m.err = nil
		return m, nil
	}

	if msg.String() == "q" {
		return Quit(m)
	}

	// Only key commands when NOT in filtering mode
	if !m.list.SettingFilter() {
		// Handle 'a' key to add new host
		if key.Matches(msg, addHost) {
			m.view = formView
			m.form = newFormModel()
			return m, textinput.Blink
		}

		// Handle 'd' key to delete host
		if key.Matches(msg, deleteHost) {
			selected := m.list.SelectedItem()
			if selected != nil {
				if it, ok := selected.(Item); ok {
					m.hostToDelete = &it.host
					m.hostToDeleteIndex = m.list.Index()
					m.view = deleteConfirmView
					return m, nil
				}
			}
		}
	}

	// Handle enter to connect
	if key.Matches(msg, enter) {
		selected := m.list.SelectedItem()
		if selected != nil {
			if it, ok := selected.(Item); ok {
				m.connectHost = &it.host
				return Quit(m)
			}
		}
	}

	// Pass all other keys to the list for navigation (arrow keys, etc.)
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

	if m.view == formView {
		return m.renderForm()
	}

	if m.view == deleteConfirmView {
		return m.renderDeleteConfirm()
	}

	return docStyle.Render(m.list.View())
}

func Quit(m Model) (tea.Model, tea.Cmd) {
	return m, tea.Quit
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

// Returns the directory containing the config file
// If running via 'go run', uses current working directory
// Otherwise, uses the directory containing the executable
func getConfigDir() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}

	exeDir := filepath.Dir(exePath)

	if strings.Contains(exeDir, "go-build") || strings.Contains(exeDir, "Temp") {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return cwd, nil
	}

	return exeDir, nil
}

func main() {
	if err := logger.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	// Get the directory where the config file is located
	configDir, err := getConfigDir()
	if err != nil {
		logger.Fatalf("Failed to get config directory: %v", err)
		fmt.Fprintf(os.Stderr, "Error: Failed to get config directory: %v\n", err)
		os.Exit(1)
	}

	// Look for config.json in the config directory
	configPath := filepath.Join(configDir, "config.json")
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

	model := initialModel(configuration.Hosts, configPath)
	for {
		p := tea.NewProgram(model, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			logger.Fatalf("Application error: %v", err)
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		m, ok := finalModel.(Model)
		if !ok {
			logger.Fatalf("Unexpected model type returned from Bubble Tea")
			fmt.Fprintln(os.Stderr, "Error: Unexpected model type returned from Bubble Tea")
			os.Exit(1)
		}

		if m.connectHost == nil {
			logger.Printf("Application exited normally")
			os.Exit(0)
		}

		clearScreen()

		// Run SSH session in the main terminal buffer
		h := m.connectHost
		authConfig := ssh.AuthConfig{
			SSHAgent:           h.SSHAgent,
			IdentityFile:       h.IdentityFile,
			IdentityPassphrase: h.IdentityPassphrase,
			KeyringService:     h.KeyringService,
			KeyringAccount:     h.KeyringAccount,
			Password:           h.Password,
		}
		err = ssh.StartSession(h.Host, h.Port, h.User, authConfig, m.width, m.height)
		if err != nil {
			// Show error when we return to the TUI
			model = initialModel(configuration.Hosts, configPath)
			model.err = err
			model.showErr = true
		} else {
			// Reset the TUI after a successful session
			model = initialModel(configuration.Hosts, configPath)
		}
	}
}
