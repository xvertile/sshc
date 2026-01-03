package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xvertile/sshc/internal/history"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Input field indices for port forward form
const (
	pfTypeInput = iota
	pfLocalPortInput
	pfRemoteHostInput
	pfRemotePortInput
	pfBindAddressInput
)

type portForwardModel struct {
	inputs         []textinput.Model
	focused        int
	forwardType    PortForwardType
	hostName       string
	err            string
	styles         Styles
	width          int
	height         int
	configFile     string
	historyManager *history.HistoryManager
}

// portForwardSubmitMsg is sent when the port forward form is submitted
type portForwardSubmitMsg struct {
	err     error
	sshArgs []string
}

// portForwardCancelMsg is sent when the port forward form is cancelled
type portForwardCancelMsg struct{}

// NewPortForwardForm creates a new port forward form model
func NewPortForwardForm(hostName string, styles Styles, width, height int, configFile string, historyManager *history.HistoryManager) *portForwardModel {
	inputs := make([]textinput.Model, 5)

	// Forward type input (display only, controlled by arrow keys)
	inputs[pfTypeInput] = textinput.New()
	inputs[pfTypeInput].Placeholder = "Use ←/→ to change forward type"
	inputs[pfTypeInput].Focus()
	inputs[pfTypeInput].Width = 40

	// Local port input
	inputs[pfLocalPortInput] = textinput.New()
	inputs[pfLocalPortInput].Placeholder = "8080"
	inputs[pfLocalPortInput].CharLimit = 5
	inputs[pfLocalPortInput].Width = 20

	// Remote host input
	inputs[pfRemoteHostInput] = textinput.New()
	inputs[pfRemoteHostInput].Placeholder = "localhost"
	inputs[pfRemoteHostInput].CharLimit = 100
	inputs[pfRemoteHostInput].Width = 30
	inputs[pfRemoteHostInput].SetValue("localhost")

	// Remote port input
	inputs[pfRemotePortInput] = textinput.New()
	inputs[pfRemotePortInput].Placeholder = "80"
	inputs[pfRemotePortInput].CharLimit = 5
	inputs[pfRemotePortInput].Width = 20

	// Bind address input (optional)
	inputs[pfBindAddressInput] = textinput.New()
	inputs[pfBindAddressInput].Placeholder = "127.0.0.1 (optional)"
	inputs[pfBindAddressInput].CharLimit = 50
	inputs[pfBindAddressInput].Width = 30

	pf := &portForwardModel{
		inputs:         inputs,
		focused:        0,
		forwardType:    LocalForward,
		hostName:       hostName,
		styles:         styles,
		width:          width,
		height:         height,
		configFile:     configFile,
		historyManager: historyManager,
	}

	// Load previous port forwarding configuration if available
	pf.loadPreviousConfig()

	// Initialize input visibility
	pf.updateInputVisibility()

	return pf
}

func (m *portForwardModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *portForwardModel) Update(msg tea.Msg) (*portForwardModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			return m, func() tea.Msg { return portForwardCancelMsg{} }

		case "enter":
			nextField := m.getNextValidField(m.focused)
			if nextField != -1 {
				// Move to next valid input
				m.inputs[m.focused].Blur()
				m.focused = nextField
				m.inputs[m.focused].Focus()
				return m, textinput.Blink
			} else {
				// Submit form
				return m, m.submitForm()
			}

		case "shift+tab", "up":
			prevField := m.getPrevValidField(m.focused)
			if prevField != -1 {
				m.inputs[m.focused].Blur()
				m.focused = prevField
				m.inputs[m.focused].Focus()
				return m, textinput.Blink
			}

		case "tab", "down":
			nextField := m.getNextValidField(m.focused)
			if nextField != -1 {
				m.inputs[m.focused].Blur()
				m.focused = nextField
				m.inputs[m.focused].Focus()
				return m, textinput.Blink
			}

		case "left", "right":
			if m.focused == pfTypeInput {
				// Change forward type
				if msg.String() == "left" {
					if m.forwardType > 0 {
						m.forwardType--
					} else {
						m.forwardType = DynamicForward
					}
				} else {
					if m.forwardType < DynamicForward {
						m.forwardType++
					} else {
						m.forwardType = LocalForward
					}
				}
				m.inputs[pfTypeInput].SetValue(m.forwardType.String())
				m.updateInputVisibility()

				// Ensure focused field is valid for the new type
				validFields := m.getValidFields()
				validFocus := false
				for _, field := range validFields {
					if field == m.focused {
						validFocus = true
						break
					}
				}
				if !validFocus && len(validFields) > 0 {
					m.inputs[m.focused].Blur()
					m.focused = validFields[0]
					m.inputs[m.focused].Focus()
				}

				return m, nil
			}
		}
	}

	// Update the focused input
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

func (m *portForwardModel) updateInputVisibility() {
	// Reset all inputs visibility
	for i := range m.inputs {
		if i != pfTypeInput {
			m.inputs[i].Placeholder = ""
		}
	}

	switch m.forwardType {
	case LocalForward:
		m.inputs[pfLocalPortInput].Placeholder = "Local port (e.g., 8080)"
		m.inputs[pfRemoteHostInput].Placeholder = "Remote host (e.g., localhost)"
		m.inputs[pfRemotePortInput].Placeholder = "Remote port (e.g., 80)"
		m.inputs[pfBindAddressInput].Placeholder = "Bind address (optional, default: 127.0.0.1)"
	case RemoteForward:
		m.inputs[pfLocalPortInput].Placeholder = "Remote port (e.g., 8080)"
		m.inputs[pfRemoteHostInput].Placeholder = "Local host (e.g., localhost)"
		m.inputs[pfRemotePortInput].Placeholder = "Local port (e.g., 80)"
		m.inputs[pfBindAddressInput].Placeholder = "Bind address (optional)"
	case DynamicForward:
		m.inputs[pfLocalPortInput].Placeholder = "SOCKS port (e.g., 1080)"
		m.inputs[pfRemoteHostInput].Placeholder = ""
		m.inputs[pfRemotePortInput].Placeholder = ""
		m.inputs[pfBindAddressInput].Placeholder = "Bind address (optional, default: 127.0.0.1)"
	}
}

func (m *portForwardModel) View() string {
	theme := GetCurrentTheme()
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.Primary))
	b.WriteString(titleStyle.Render("PORT FORWARDING"))
	b.WriteString("\n\n")

	// Host info
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Accent))
	b.WriteString(infoStyle.Render(fmt.Sprintf("Host: %s", m.hostName)))
	b.WriteString("\n\n")

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Muted)).Width(16)
	focusedLabelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.Primary)).Width(16)
	requiredStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))

	// Helper to render a field
	renderField := func(label string, inputIndex int, required bool) {
		l := label
		if required {
			l += requiredStyle.Render("*")
		}
		if m.focused == inputIndex {
			b.WriteString(focusedLabelStyle.Render(l))
			b.WriteString(" ")
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Primary)).Render("> "))
		} else {
			b.WriteString(labelStyle.Render(l))
			b.WriteString("   ")
		}
		b.WriteString(m.inputs[inputIndex].View())
		b.WriteString("\n")
	}

	// Forward type
	renderField("Type", pfTypeInput, false)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Muted))
	b.WriteString(helpStyle.Render("                    ←/→ to change type"))
	b.WriteString("\n\n")

	switch m.forwardType {
	case LocalForward:
		b.WriteString(helpStyle.Render("ssh -L [bind:]local_port:remote_host:remote_port"))
		b.WriteString("\n\n")
		renderField("Local Port", pfLocalPortInput, true)
		renderField("Remote Host", pfRemoteHostInput, false)
		renderField("Remote Port", pfRemotePortInput, true)

	case RemoteForward:
		b.WriteString(helpStyle.Render("ssh -R [bind:]remote_port:local_host:local_port"))
		b.WriteString("\n\n")
		renderField("Remote Port", pfLocalPortInput, true)
		renderField("Local Host", pfRemoteHostInput, false)
		renderField("Local Port", pfRemotePortInput, true)

	case DynamicForward:
		b.WriteString(helpStyle.Render("ssh -D [bind:]port (SOCKS proxy)"))
		b.WriteString("\n\n")
		renderField("SOCKS Port", pfLocalPortInput, true)
	}

	b.WriteString("\n")
	renderField("Bind Address", pfBindAddressInput, false)

	// Error message
	if m.err != "" {
		b.WriteString("\n")
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		b.WriteString(errorStyle.Render("Error: " + m.err))
	}

	// Help
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("↑/↓: navigate • Enter: connect • Esc: cancel"))

	content := b.String()

	// Container with border
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

func (m *portForwardModel) submitForm() tea.Cmd {
	return func() tea.Msg {
		// Validate inputs
		localPort := strings.TrimSpace(m.inputs[pfLocalPortInput].Value())
		if localPort == "" {
			return portForwardSubmitMsg{err: fmt.Errorf("port is required"), sshArgs: nil}
		}

		// Validate port number
		if _, err := strconv.Atoi(localPort); err != nil {
			return portForwardSubmitMsg{err: fmt.Errorf("invalid port number"), sshArgs: nil}
		}

		// Get form values for saving to history
		remoteHost := strings.TrimSpace(m.inputs[pfRemoteHostInput].Value())
		remotePort := strings.TrimSpace(m.inputs[pfRemotePortInput].Value())
		bindAddress := strings.TrimSpace(m.inputs[pfBindAddressInput].Value())

		// Build SSH command with port forwarding
		var sshArgs []string

		// Add config file if specified
		if m.configFile != "" {
			sshArgs = append(sshArgs, "-F", m.configFile)
		}

		// Add forwarding arguments
		var forwardTypeStr string
		switch m.forwardType {
		case LocalForward:
			forwardTypeStr = "local"
			if remoteHost == "" {
				remoteHost = "localhost"
			}
			if remotePort == "" {
				return portForwardSubmitMsg{err: fmt.Errorf("remote port is required for local forwarding"), sshArgs: nil}
			}

			// Validate remote port
			if _, err := strconv.Atoi(remotePort); err != nil {
				return portForwardSubmitMsg{err: fmt.Errorf("invalid remote port number"), sshArgs: nil}
			}

			// Build -L argument
			var forwardArg string
			if bindAddress != "" {
				forwardArg = fmt.Sprintf("%s:%s:%s:%s", bindAddress, localPort, remoteHost, remotePort)
			} else {
				forwardArg = fmt.Sprintf("%s:%s:%s", localPort, remoteHost, remotePort)
			}
			sshArgs = append(sshArgs, "-L", forwardArg)

		case RemoteForward:
			forwardTypeStr = "remote"
			if remoteHost == "" {
				remoteHost = "localhost"
			}
			if remotePort == "" {
				return portForwardSubmitMsg{err: fmt.Errorf("local port is required for remote forwarding"), sshArgs: nil}
			}

			// Validate local port
			if _, err := strconv.Atoi(remotePort); err != nil {
				return portForwardSubmitMsg{err: fmt.Errorf("invalid local port number"), sshArgs: nil}
			}

			// Build -R argument (note: localPort is actually the remote port in this context)
			var forwardArg string
			if bindAddress != "" {
				forwardArg = fmt.Sprintf("%s:%s:%s:%s", bindAddress, localPort, remoteHost, remotePort)
			} else {
				forwardArg = fmt.Sprintf("%s:%s:%s", localPort, remoteHost, remotePort)
			}
			sshArgs = append(sshArgs, "-R", forwardArg)

		case DynamicForward:
			forwardTypeStr = "dynamic"
			// Build -D argument
			var forwardArg string
			if bindAddress != "" {
				forwardArg = fmt.Sprintf("%s:%s", bindAddress, localPort)
			} else {
				forwardArg = localPort
			}
			sshArgs = append(sshArgs, "-D", forwardArg)
		}

		// Save port forwarding configuration to history
		if m.historyManager != nil {
			if err := m.historyManager.RecordPortForwarding(
				m.hostName,
				forwardTypeStr,
				localPort,
				remoteHost,
				remotePort,
				bindAddress,
			); err != nil {
				// Log the error but don't fail the connection
				// In a production environment, you might want to handle this differently
			}
		}

		// Add hostname
		sshArgs = append(sshArgs, m.hostName)

		// Return success with the SSH command to execute
		return portForwardSubmitMsg{err: nil, sshArgs: sshArgs}
	}
}

// getValidFields returns the list of valid field indices for the current forward type
func (m *portForwardModel) getValidFields() []int {
	switch m.forwardType {
	case LocalForward:
		return []int{pfTypeInput, pfLocalPortInput, pfRemoteHostInput, pfRemotePortInput, pfBindAddressInput}
	case RemoteForward:
		return []int{pfTypeInput, pfLocalPortInput, pfRemoteHostInput, pfRemotePortInput, pfBindAddressInput}
	case DynamicForward:
		return []int{pfTypeInput, pfLocalPortInput, pfBindAddressInput}
	default:
		return []int{pfTypeInput, pfLocalPortInput, pfRemoteHostInput, pfRemotePortInput, pfBindAddressInput}
	}
}

// getNextValidField returns the next valid field index, or -1 if none
func (m *portForwardModel) getNextValidField(currentField int) int {
	validFields := m.getValidFields()

	for i, field := range validFields {
		if field == currentField && i < len(validFields)-1 {
			return validFields[i+1]
		}
	}
	return -1
}

// getPrevValidField returns the previous valid field index, or -1 if none
func (m *portForwardModel) getPrevValidField(currentField int) int {
	validFields := m.getValidFields()

	for i, field := range validFields {
		if field == currentField && i > 0 {
			return validFields[i-1]
		}
	}
	return -1
}

// loadPreviousConfig loads the previous port forwarding configuration for this host
func (m *portForwardModel) loadPreviousConfig() {
	if m.historyManager == nil {
		m.inputs[pfTypeInput].SetValue("Local (-L)")
		return
	}

	config := m.historyManager.GetPortForwardingConfig(m.hostName)
	if config == nil {
		m.inputs[pfTypeInput].SetValue("Local (-L)")
		return
	}

	// Set forward type based on saved configuration
	switch config.Type {
	case "local":
		m.forwardType = LocalForward
	case "remote":
		m.forwardType = RemoteForward
	case "dynamic":
		m.forwardType = DynamicForward
	default:
		m.forwardType = LocalForward
	}
	m.inputs[pfTypeInput].SetValue(m.forwardType.String())

	// Set values from saved configuration
	if config.LocalPort != "" {
		m.inputs[pfLocalPortInput].SetValue(config.LocalPort)
	}
	if config.RemoteHost != "" {
		m.inputs[pfRemoteHostInput].SetValue(config.RemoteHost)
	} else if m.forwardType != DynamicForward {
		// Default to localhost for local and remote forwarding if not set
		m.inputs[pfRemoteHostInput].SetValue("localhost")
	}
	if config.RemotePort != "" {
		m.inputs[pfRemotePortInput].SetValue(config.RemotePort)
	}
	if config.BindAddress != "" {
		m.inputs[pfBindAddressInput].SetValue(config.BindAddress)
	}
}
