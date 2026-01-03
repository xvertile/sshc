package ui

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/xvertile/sshc/internal/config"
	"github.com/xvertile/sshc/internal/validation"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type addFormModel struct {
	inputs     []textinput.Model
	focused    int
	err        string
	styles     Styles
	success    bool
	width      int
	height     int
	configFile string
}

const (
	addNameInput = iota
	addHostnameInput
	addUserInput
	addPortInput
	addIdentityInput
	addProxyJumpInput
	addTagsInput
)

// Messages for communication with parent model
type addFormSubmitMsg struct {
	hostname string
	err      error
}

type addFormCancelMsg struct{}

// NewAddForm creates a new add form model
func NewAddForm(hostname string, styles Styles, width, height int, configFile string) *addFormModel {
	// Get current user for default
	currentUser, _ := user.Current()
	defaultUser := "root"
	if currentUser != nil {
		defaultUser = currentUser.Username
	}

	// Find default identity file
	homeDir, _ := os.UserHomeDir()
	defaultIdentity := ""

	// Check for common key types
	keyTypes := []string{"id_ed25519", "id_ecdsa", "id_rsa"}
	for _, keyType := range keyTypes {
		keyPath := filepath.Join(homeDir, ".ssh", keyType)
		if _, err := os.Stat(keyPath); err == nil {
			defaultIdentity = keyPath
			break
		}
	}

	inputs := make([]textinput.Model, 7)

	// Name input
	inputs[addNameInput] = textinput.New()
	inputs[addNameInput].Placeholder = "my-server"
	inputs[addNameInput].Focus()
	inputs[addNameInput].CharLimit = 50
	inputs[addNameInput].Width = 40
	if hostname != "" {
		inputs[addNameInput].SetValue(hostname)
	}

	// Hostname input
	inputs[addHostnameInput] = textinput.New()
	inputs[addHostnameInput].Placeholder = "192.168.1.100 or example.com"
	inputs[addHostnameInput].CharLimit = 100
	inputs[addHostnameInput].Width = 40

	// User input
	inputs[addUserInput] = textinput.New()
	inputs[addUserInput].Placeholder = defaultUser
	inputs[addUserInput].CharLimit = 50
	inputs[addUserInput].Width = 40

	// Port input
	inputs[addPortInput] = textinput.New()
	inputs[addPortInput].Placeholder = "22"
	inputs[addPortInput].CharLimit = 5
	inputs[addPortInput].Width = 40

	// Identity input
	inputs[addIdentityInput] = textinput.New()
	if defaultIdentity != "" {
		inputs[addIdentityInput].Placeholder = defaultIdentity
	} else {
		inputs[addIdentityInput].Placeholder = "~/.ssh/id_ed25519"
	}
	inputs[addIdentityInput].CharLimit = 200
	inputs[addIdentityInput].Width = 40

	// ProxyJump input
	inputs[addProxyJumpInput] = textinput.New()
	inputs[addProxyJumpInput].Placeholder = "jump-host or user@host:port"
	inputs[addProxyJumpInput].CharLimit = 200
	inputs[addProxyJumpInput].Width = 40

	// Tags input
	inputs[addTagsInput] = textinput.New()
	inputs[addTagsInput].Placeholder = "web, production"
	inputs[addTagsInput].CharLimit = 200
	inputs[addTagsInput].Width = 40

	return &addFormModel{
		inputs:     inputs,
		focused:    addNameInput,
		styles:     styles,
		width:      width,
		height:     height,
		configFile: configFile,
	}
}

func (m *addFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *addFormModel) Update(msg tea.Msg) (*addFormModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.styles = NewStyles(m.width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, func() tea.Msg { return addFormCancelMsg{} }

		case "ctrl+s":
			return m, m.submitForm()

		case "tab", "down", "enter":
			// Move to next field
			if msg.String() == "enter" && m.focused == addTagsInput {
				// Submit on enter at last field
				return m, m.submitForm()
			}
			m.focused++
			if m.focused >= len(m.inputs) {
				m.focused = 0
			}
			return m, m.updateFocus()

		case "shift+tab", "up":
			// Move to previous field
			m.focused--
			if m.focused < 0 {
				m.focused = len(m.inputs) - 1
			}
			return m, m.updateFocus()
		}

	case addFormSubmitMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.success = true
			m.err = ""
		}
		return m, nil
	}

	// Update focused input
	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *addFormModel) updateFocus() tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.inputs {
		if i == m.focused {
			cmds = append(cmds, m.inputs[i].Focus())
		} else {
			m.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m *addFormModel) View() string {
	if m.success {
		return ""
	}

	theme := GetCurrentTheme()
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.Primary))
	b.WriteString(titleStyle.Render("ADD SSH HOST"))
	b.WriteString("\n\n")

	// Fields
	fields := []struct {
		index    int
		label    string
		required bool
	}{
		{addNameInput, "Name", true},
		{addHostnameInput, "Hostname", true},
		{addUserInput, "User", false},
		{addPortInput, "Port", false},
		{addIdentityInput, "Identity File", false},
		{addProxyJumpInput, "ProxyJump", false},
		{addTagsInput, "Tags", false},
	}

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Muted)).Width(14)
	focusedLabelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.Primary)).Width(14)
	requiredStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))

	for _, field := range fields {
		// Label
		label := field.label
		if field.required {
			label += requiredStyle.Render("*")
		}

		if m.focused == field.index {
			b.WriteString(focusedLabelStyle.Render(label))
			b.WriteString(" ")
			// Show cursor indicator
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Primary)).Render("> "))
		} else {
			b.WriteString(labelStyle.Render(label))
			b.WriteString("   ")
		}

		// Input
		b.WriteString(m.inputs[field.index].View())
		b.WriteString("\n")
	}

	// Error message
	if m.err != "" {
		b.WriteString("\n")
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		b.WriteString(errorStyle.Render("Error: " + m.err))
	}

	// Help
	b.WriteString("\n\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Muted))
	b.WriteString(helpStyle.Render("↑/↓: navigate • Enter: next/submit • Ctrl+S: save • Esc: cancel"))

	content := b.String()

	// Container
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Primary)).
		Padding(1, 2)

	// Logo
	logo := m.styles.Header.Render(asciiTitle)

	// Stack logo and container
	fullContent := lipgloss.JoinVertical(lipgloss.Center, logo, "", box.Render(content))

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		fullContent,
	)
}

func (m *addFormModel) submitForm() tea.Cmd {
	return func() tea.Msg {
		// Get values
		name := strings.TrimSpace(m.inputs[addNameInput].Value())
		hostname := strings.TrimSpace(m.inputs[addHostnameInput].Value())
		user := strings.TrimSpace(m.inputs[addUserInput].Value())
		port := strings.TrimSpace(m.inputs[addPortInput].Value())
		identity := strings.TrimSpace(m.inputs[addIdentityInput].Value())
		proxyJump := strings.TrimSpace(m.inputs[addProxyJumpInput].Value())

		// Set defaults
		if user == "" {
			user = m.inputs[addUserInput].Placeholder
		}
		if port == "" {
			port = "22"
		}

		// Validate required fields
		if err := validation.ValidateHost(name, hostname, port, identity); err != nil {
			return addFormSubmitMsg{err: err}
		}

		// Parse tags
		tagsStr := strings.TrimSpace(m.inputs[addTagsInput].Value())
		var tags []string
		if tagsStr != "" {
			for _, tag := range strings.Split(tagsStr, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					tags = append(tags, tag)
				}
			}
		}

		// Create host configuration
		host := config.SSHHost{
			Name:      name,
			Hostname:  hostname,
			User:      user,
			Port:      port,
			Identity:  identity,
			ProxyJump: proxyJump,
			Tags:      tags,
		}

		// Add to config
		var err error
		if m.configFile != "" {
			err = config.AddSSHHostToFile(host, m.configFile)
		} else {
			err = config.AddSSHHost(host)
		}
		return addFormSubmitMsg{hostname: name, err: err}
	}
}

// Standalone wrapper for add form
type standaloneAddForm struct {
	*addFormModel
}

func (m standaloneAddForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case addFormSubmitMsg:
		if msg.err != nil {
			m.addFormModel.err = msg.err.Error()
		} else {
			m.addFormModel.success = true
			return m, tea.Quit
		}
		return m, nil
	case addFormCancelMsg:
		return m, tea.Quit
	}

	newForm, cmd := m.addFormModel.Update(msg)
	m.addFormModel = newForm
	return m, cmd
}

// RunAddForm provides backward compatibility for standalone add form
func RunAddForm(hostname string, configFile string) error {
	styles := NewStyles(80)
	addForm := NewAddForm(hostname, styles, 80, 24, configFile)
	m := standaloneAddForm{addForm}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
