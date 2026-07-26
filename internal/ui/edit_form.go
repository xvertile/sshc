package ui

import (
	"fmt"
	"strings"

	"github.com/xvertile/sshc/internal/config"
	"github.com/xvertile/sshc/internal/validation"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	focusAreaHosts = iota
	focusAreaProperties
)

type editFormSubmitMsg struct {
	hostname string
	err      error
}

type editFormCancelMsg struct{}

type editFormModel struct {
	hostInputs       []textinput.Model // Support for multiple hosts
	inputs           []textinput.Model
	focusArea        int // 0=hosts, 1=properties
	focused          int
	err              string
	styles           Styles
	originalName     string
	originalHosts    []string        // Store original host names for multi-host detection
	host             *config.SSHHost // Store the original host with SourceFile
	configFile       string          // Configuration file path passed by user
	actualConfigFile string          // Actual config file to use (either configFile or host.SourceFile)
	width            int
	height           int
}

// NewEditForm creates a new edit form model that supports both single and multi-host editing
func NewEditForm(hostName string, styles Styles, width, height int, configFile string) (*editFormModel, error) {
	// Get the existing host configuration
	var host *config.SSHHost
	var err error

	if configFile != "" {
		host, err = config.GetSSHHostFromFile(hostName, configFile)
	} else {
		host, err = config.GetSSHHost(hostName)
	}

	if err != nil {
		return nil, err
	}

	// Check if this host is part of a multi-host declaration
	var actualConfigFile string
	var hostNames []string
	var isMulti bool

	if configFile != "" {
		actualConfigFile = configFile
	} else {
		actualConfigFile = host.SourceFile
	}

	if actualConfigFile != "" {
		isMulti, hostNames, err = config.IsPartOfMultiHostDeclaration(hostName, actualConfigFile)
		if err != nil {
			// If we can't determine multi-host status, treat as single host
			isMulti = false
			hostNames = []string{hostName}
		}
	}

	if !isMulti {
		hostNames = []string{hostName}
	}

	// Create host inputs
	hostInputs := make([]textinput.Model, len(hostNames))
	for i, name := range hostNames {
		hostInputs[i] = textinput.New()
		hostInputs[i].Placeholder = "host-name"
		hostInputs[i].SetValue(name)
		if i == 0 {
			hostInputs[i].Focus()
		}
	}

	inputs := make([]textinput.Model, 9) // Increased from 8 to 9 for RequestTTY

	// Hostname input
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "192.168.1.100 or example.com"
	inputs[0].CharLimit = 100
	inputs[0].Width = 30
	inputs[0].SetValue(host.Hostname)

	// User input
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "root"
	inputs[1].CharLimit = 50
	inputs[1].Width = 30
	inputs[1].SetValue(host.User)

	// Port input
	inputs[2] = textinput.New()
	inputs[2].Placeholder = "22"
	inputs[2].CharLimit = 5
	inputs[2].Width = 30
	inputs[2].SetValue(host.Port)

	// Identity input
	inputs[3] = textinput.New()
	inputs[3].Placeholder = "~/.ssh/id_rsa"
	inputs[3].CharLimit = 200
	inputs[3].Width = 50
	inputs[3].SetValue(host.Identity)

	// ProxyJump input
	inputs[4] = textinput.New()
	inputs[4].Placeholder = "jump-server"
	inputs[4].CharLimit = 100
	inputs[4].Width = 30
	inputs[4].SetValue(host.ProxyJump)

	// Options input
	inputs[5] = textinput.New()
	inputs[5].Placeholder = "-o StrictHostKeyChecking=no"
	inputs[5].CharLimit = 200
	inputs[5].Width = 50
	if host.Options != "" {
		inputs[5].SetValue(config.FormatSSHOptionsForCommand(host.Options))
	}

	// Tags input
	inputs[6] = textinput.New()
	inputs[6].Placeholder = "production, web, database"
	inputs[6].CharLimit = 200
	inputs[6].Width = 50
	if len(host.Tags) > 0 {
		inputs[6].SetValue(strings.Join(host.Tags, ", "))
	}

	// Remote Command input
	inputs[7] = textinput.New()
	inputs[7].Placeholder = "ls -la, htop, bash"
	inputs[7].CharLimit = 300
	inputs[7].Width = 70
	inputs[7].SetValue(host.RemoteCommand)

	// RequestTTY input
	inputs[8] = textinput.New()
	inputs[8].Placeholder = "yes, no, force, auto"
	inputs[8].CharLimit = 10
	inputs[8].Width = 30
	inputs[8].SetValue(host.RequestTTY)

	return &editFormModel{
		hostInputs:       hostInputs,
		inputs:           inputs,
		focusArea:        focusAreaHosts, // Start with hosts focused for multi-host editing
		focused:          0,
		originalName:     hostName,
		originalHosts:    hostNames,
		host:             host,
		configFile:       configFile,
		actualConfigFile: actualConfigFile,
		styles:           styles,
		width:            width,
		height:           height,
	}, nil
}

func (m *editFormModel) Init() tea.Cmd {
	return textinput.Blink
}

// addHostInput adds a new empty host input
func (m *editFormModel) addHostInput() tea.Cmd {
	newInput := textinput.New()
	newInput.Placeholder = "host-name"
	newInput.Focus()

	// Unfocus current input regardless of which area we're in
	if m.focusArea == focusAreaHosts && m.focused < len(m.hostInputs) {
		m.hostInputs[m.focused].Blur()
	} else if m.focusArea == focusAreaProperties && m.focused < len(m.inputs) {
		m.inputs[m.focused].Blur()
	}

	m.hostInputs = append(m.hostInputs, newInput)

	// Move focus to the new host input
	m.focusArea = focusAreaHosts
	m.focused = len(m.hostInputs) - 1

	return textinput.Blink
}

// deleteHostInput removes the currently focused host input
func (m *editFormModel) deleteHostInput() tea.Cmd {
	if len(m.hostInputs) <= 1 || m.focusArea != focusAreaHosts {
		return nil // Can't delete if only one host or not in host area
	}

	// Remove the focused host input
	m.hostInputs = append(m.hostInputs[:m.focused], m.hostInputs[m.focused+1:]...)

	// Adjust focus
	if m.focused >= len(m.hostInputs) {
		m.focused = len(m.hostInputs) - 1
	}

	// Focus the new current input
	if len(m.hostInputs) > 0 {
		m.hostInputs[m.focused].Focus()
	}

	return nil
}

// updateFocus updates the focus state based on current area and index
func (m *editFormModel) updateFocus() tea.Cmd {
	// Blur all inputs first
	for i := range m.hostInputs {
		m.hostInputs[i].Blur()
	}
	for i := range m.inputs {
		m.inputs[i].Blur()
	}

	// Focus the appropriate input
	if m.focusArea == focusAreaHosts {
		if m.focused < len(m.hostInputs) {
			m.hostInputs[m.focused].Focus()
		}
	} else {
		if m.focused < len(m.inputs) {
			m.inputs[m.focused].Focus()
		}
	}

	return textinput.Blink
}

// handleEditNavigation moves focus through the host names and then every
// property in display order, wrapping back to the host names at the end.
func (m *editFormModel) handleEditNavigation(key string) tea.Cmd {
	back := key == "up" || key == "shift+tab"

	if m.focusArea == focusAreaHosts {
		if back {
			m.focused--
		} else {
			m.focused++
		}

		switch {
		case m.focused >= len(m.hostInputs):
			m.focusArea = focusAreaProperties
			m.focused = editFields[0].index
		case m.focused < 0:
			// Already at the very top: stay put.
			m.focused = 0
		}

		return m.updateFocus()
	}

	position := editFieldPosition(m.focused)

	// Enter on the final field submits, as it does in the other forms.
	if key == "enter" && position == len(editFields)-1 {
		return m.submitEditForm()
	}

	if back {
		position--
	} else {
		position++
	}

	switch {
	case position >= len(editFields):
		// Already at the very bottom: stay put.
		m.focused = editFields[len(editFields)-1].index
	case position < 0:
		m.focusArea = focusAreaHosts
		m.focused = len(m.hostInputs) - 1
	default:
		m.focused = editFields[position].index
	}

	return m.updateFocus()
}

func (m *editFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.styles = NewStyles(m.width)
		fitInputs(m.hostInputs, m.width)
		fitInputs(m.inputs, m.width)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.err = ""
			return m, func() tea.Msg { return editFormCancelMsg{} }

		case "ctrl+s":
			// Allow submission from any field with Ctrl+S (Save)
			return m, m.submitEditForm()

		case "tab", "shift+tab", "enter", "up", "down":
			return m, m.handleEditNavigation(msg.String())

		case "ctrl+a":
			// Add a new host input
			return m, m.addHostInput()

		case "ctrl+d":
			// Delete the currently focused host (if more than one exists)
			if m.focusArea == focusAreaHosts && len(m.hostInputs) > 1 {
				return m, m.deleteHostInput()
			}
		}

	case editFormSubmitMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			// Success: let the wrapper handle this
			// In TUI mode, this will be handled by the parent
			// In standalone mode, the wrapper will quit
		}
		return m, nil
	}

	// Update host inputs
	hostCmd := make([]tea.Cmd, len(m.hostInputs))
	for i := range m.hostInputs {
		m.hostInputs[i], hostCmd[i] = m.hostInputs[i].Update(msg)
	}
	cmds = append(cmds, hostCmd...)

	// Update property inputs
	propCmd := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], propCmd[i] = m.inputs[i].Update(msg)
	}
	cmds = append(cmds, propCmd...)

	return m, tea.Batch(cmds...)
}

func (m *editFormModel) View() string {
	var lines []string
	focusLine := noFocusLine

	// addField records the line index of the focused row as it goes, so the
	// screen can scroll to keep it visible on a short terminal.
	addField := func(label string, required, focused bool, input string) {
		if focused {
			focusLine = len(lines)
		}
		lines = append(lines, formField(label, required, focused, input, formLabelWidth))
	}

	// Config file info
	if m.host != nil && m.host.SourceFile != "" {
		lines = append(lines, muted("config "+formatConfigFile(m.host.SourceFile)), "")
	}

	// Host Names Section
	lines = append(lines, accent("host names"))

	for i, hostInput := range m.hostInputs {
		addField(
			fmt.Sprintf("Name %d", i+1),
			true,
			m.focusArea == focusAreaHosts && m.focused == i,
			hostInput.View(),
		)
	}

	lines = append(lines, "", accent("connection"))

	for _, field := range editFields {
		addField(
			field.label,
			field.required,
			m.focusArea == focusAreaProperties && m.focused == field.index,
			m.inputs[field.index].View(),
		)
	}

	hints := []keyHint{
		{"↑↓", "move"},
		{"ctrl+a", "add name"},
	}
	if len(m.hostInputs) > 1 {
		hints = append(hints, keyHint{"ctrl+d", "remove name"})
	}
	hints = append(hints, keyHint{"ctrl+s", "save"}, keyHint{"esc", "cancel"})

	return formScreenAt(m.width, m.height, "edit host",
		strings.Join(lines, "\n"), m.err, focusLine, hints...)
}

// editField describes one property input.
type editField struct {
	index    int
	label    string
	required bool
}

// editFields lists the editable properties in display order. Indices address
// editFormModel.inputs, whose order is historical and not the display order.
//
// SSH Options, RemoteCommand and RequestTTY are deliberately absent. Their
// inputs are still loaded from the host and read back on save, so a host that
// has them keeps them; they are simply not offered for editing here.
var editFields = []editField{
	{0, "Hostname", true},
	{1, "User", false},
	{2, "Port", false},
	{3, "Identity File", false},
	{4, "ProxyJump", false},
	{6, "Tags", false},
}

// editFieldPosition returns the display position of a property input.
func editFieldPosition(input int) int {
	for i, field := range editFields {
		if field.index == input {
			return i
		}
	}
	return 0
}

// Standalone wrapper for edit form
type standaloneEditForm struct {
	*editFormModel
}

func (m standaloneEditForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case editFormSubmitMsg:
		if msg.err != nil {
			m.editFormModel.err = msg.err.Error()
			return m, nil
		} else {
			// Success: quit the program
			return m, tea.Quit
		}
	case editFormCancelMsg:
		return m, tea.Quit
	}

	newForm, cmd := m.editFormModel.Update(msg)
	m.editFormModel = newForm.(*editFormModel)
	return m, cmd
}

// RunEditForm runs the edit form as a standalone program
func RunEditForm(hostName string, configFile string) error {
	styles := NewStyles(80) // Default width
	editForm, err := NewEditForm(hostName, styles, 80, 24, configFile)
	if err != nil {
		return err
	}

	m := standaloneEditForm{editForm}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

func (m *editFormModel) submitEditForm() tea.Cmd {
	return func() tea.Msg {
		// Collect host names
		var hostNames []string
		for _, input := range m.hostInputs {
			name := strings.TrimSpace(input.Value())
			if name != "" {
				hostNames = append(hostNames, name)
			}
		}

		if len(hostNames) == 0 {
			return editFormSubmitMsg{err: fmt.Errorf("at least one host name is required")}
		}

		// Get property values using direct indices
		hostname := strings.TrimSpace(m.inputs[0].Value())      // hostnameInput
		user := strings.TrimSpace(m.inputs[1].Value())          // userInput
		port := strings.TrimSpace(m.inputs[2].Value())          // portInput
		identity := strings.TrimSpace(m.inputs[3].Value())      // identityInput
		proxyJump := strings.TrimSpace(m.inputs[4].Value())     // proxyJumpInput
		options := strings.TrimSpace(m.inputs[5].Value())       // optionsInput
		remoteCommand := strings.TrimSpace(m.inputs[7].Value()) // remoteCommandInput
		requestTTY := strings.TrimSpace(m.inputs[8].Value())    // requestTTYInput

		// Set defaults
		if port == "" {
			port = "22"
		}

		// Validate hostname
		if hostname == "" {
			return editFormSubmitMsg{err: fmt.Errorf("hostname is required")}
		}

		// Validate all host names
		for _, hostName := range hostNames {
			if err := validation.ValidateHost(hostName, hostname, port, identity); err != nil {
				return editFormSubmitMsg{err: err}
			}
		}

		// Parse tags
		tagsStr := strings.TrimSpace(m.inputs[6].Value()) // tagsInput
		var tags []string
		if tagsStr != "" {
			for _, tag := range strings.Split(tagsStr, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					tags = append(tags, tag)
				}
			}
		}

		// Create the common host configuration
		commonHost := config.SSHHost{
			Hostname:      hostname,
			User:          user,
			Port:          port,
			Identity:      identity,
			ProxyJump:     proxyJump,
			Options:       config.ParseSSHOptionsFromCommand(options),
			RemoteCommand: remoteCommand,
			RequestTTY:    requestTTY,
			Tags:          tags,
		}

		var err error
		if len(hostNames) == 1 && len(m.originalHosts) == 1 {
			// Single host editing
			commonHost.Name = hostNames[0]
			if m.actualConfigFile != "" {
				err = config.UpdateSSHHostInFile(m.originalName, commonHost, m.actualConfigFile)
			} else {
				err = config.UpdateSSHHost(m.originalName, commonHost)
			}
		} else {
			// Multi-host editing or conversion from single to multi
			err = config.UpdateMultiHostBlock(m.originalHosts, hostNames, commonHost, m.actualConfigFile)
		}

		return editFormSubmitMsg{hostname: hostNames[0], err: err}
	}
}
