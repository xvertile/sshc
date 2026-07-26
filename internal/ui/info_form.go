package ui

import (
	"github.com/xvertile/sshc/internal/config"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type infoFormModel struct {
	host       *config.SSHHost
	styles     Styles
	width      int
	height     int
	configFile string
	hostName   string
}

// Messages for communication with parent model
type infoFormEditMsg struct {
	hostName string
}

type infoFormCancelMsg struct{}

// NewInfoForm creates a new info form model for displaying host details in read-only mode
func NewInfoForm(hostName string, styles Styles, width, height int, configFile string) (*infoFormModel, error) {
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

	return &infoFormModel{
		host:       host,
		hostName:   hostName,
		configFile: configFile,
		styles:     styles,
		width:      width,
		height:     height,
	}, nil
}

func (m *infoFormModel) Init() tea.Cmd {
	return nil
}

func (m *infoFormModel) Update(msg tea.Msg) (*infoFormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.styles = NewStyles(m.width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			return m, func() tea.Msg { return infoFormCancelMsg{} }

		case "e", "enter":
			// Switch to edit mode
			return m, func() tea.Msg { return infoFormEditMsg{hostName: m.hostName} }
		}
	}

	return m, nil
}

func (m *infoFormModel) View() string {
	theme := GetCurrentTheme()

	// Create info sections with consistent formatting
	sections := []struct {
		label string
		value string
	}{
		{"Host Name", m.host.Name},
		{"Config File", formatConfigFile(m.host.SourceFile)},
		{"Hostname/IP", m.host.Hostname},
		{"User", formatOptionalValue(m.host.User)},
		{"Port", formatOptionalValue(m.host.Port)},
		{"Identity File", formatOptionalValue(m.host.Identity)},
		{"ProxyJump", formatOptionalValue(m.host.ProxyJump)},
		{"SSH Options", formatSSHOptions(m.host.Options)},
		{"Tags", formatTags(m.host.Tags)},
	}

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Muted)).
		Width(formLabelWidth)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Foreground))

	unsetStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Muted))

	lines := make([]string, 0, len(sections))
	for _, section := range sections {
		value := valueStyle.Render(section.value)
		if section.value == "Not set" {
			value = unsetStyle.Render(section.value)
		}
		lines = append(lines, "  "+labelStyle.Render(section.label)+value)
	}

	return formScreen(m.width, m.height,
		"host "+m.host.Name, strings.Join(lines, "\n"), "",
		keyHint{"e", "edit"},
		keyHint{"esc", "back"},
	)
}

// Helper functions for formatting values

func formatOptionalValue(value string) string {
	if value == "" {
		return "Not set"
	}
	return value
}

func formatSSHOptions(options string) string {
	if options == "" {
		return "Not set"
	}
	return options
}

func formatTags(tags []string) string {
	if len(tags) == 0 {
		return "Not set"
	}
	return strings.Join(tags, ", ")
}

// Standalone wrapper for info form (for testing or standalone use)
type standaloneInfoForm struct {
	*infoFormModel
}

func (m standaloneInfoForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case infoFormCancelMsg:
		return m, tea.Quit
	case infoFormEditMsg:
		// For standalone mode, just quit - parent should handle edit transition
		return m, tea.Quit
	}

	newForm, cmd := m.infoFormModel.Update(msg)
	m.infoFormModel = newForm
	return m, cmd
}

// RunInfoForm provides a standalone info form for testing
func RunInfoForm(hostName string, configFile string) error {
	styles := NewStyles(80)
	infoForm, err := NewInfoForm(hostName, styles, 80, 24, configFile)
	if err != nil {
		return err
	}
	m := standaloneInfoForm{infoForm}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
