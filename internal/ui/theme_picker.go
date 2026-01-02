package ui

import (
	"fmt"
	"strings"

	"github.com/xvertile/sshc/internal/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type themePickerModel struct {
	selectedIndex int
	styles        Styles
	width         int
	height        int
	appConfig     *config.AppConfig
}

// Messages for communication with parent model
type themePickerSubmitMsg struct {
	themeName string
}

type themePickerCancelMsg struct{}

func NewThemePicker(styles Styles, width, height int, appConfig *config.AppConfig) *themePickerModel {
	// Find current theme index
	selectedIndex := 0
	if appConfig != nil {
		for i, theme := range Themes {
			if theme.Name == appConfig.Theme {
				selectedIndex = i
				break
			}
		}
	}

	return &themePickerModel{
		selectedIndex: selectedIndex,
		styles:        styles,
		width:         width,
		height:        height,
		appConfig:     appConfig,
	}
}

func (m *themePickerModel) Init() tea.Cmd {
	return nil
}

func (m *themePickerModel) Update(msg tea.Msg) (*themePickerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			// Cancel and revert to original theme
			if m.appConfig != nil {
				SetThemeByName(m.appConfig.Theme)
			}
			return m, func() tea.Msg { return themePickerCancelMsg{} }

		case "enter":
			// Confirm selection
			themeName := Themes[m.selectedIndex].Name
			return m, func() tea.Msg { return themePickerSubmitMsg{themeName: themeName} }

		case "up", "k":
			m.selectedIndex--
			if m.selectedIndex < 0 {
				m.selectedIndex = len(Themes) - 1
			}
			// Apply theme live for preview
			SetTheme(m.selectedIndex)
			m.styles = NewStyles(m.width)
			return m, nil

		case "down", "j":
			m.selectedIndex++
			if m.selectedIndex >= len(Themes) {
				m.selectedIndex = 0
			}
			// Apply theme live for preview
			SetTheme(m.selectedIndex)
			m.styles = NewStyles(m.width)
			return m, nil
		}
	}

	return m, nil
}

func (m *themePickerModel) View() string {
	theme := GetCurrentTheme()

	// Title style
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Primary)).
		Bold(true)

	// Container style
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Primary)).
		Padding(1, 3).
		Align(lipgloss.Center)

	// Help style
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Muted))

	// Build theme list items
	var themeItems []string
	for i, t := range Themes {
		var line string
		if i == m.selectedIndex {
			// Selected item
			selectedStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.SelectionFg)).
				Background(lipgloss.Color(theme.SelectionBg)).
				Bold(true).
				Padding(0, 2)
			line = selectedStyle.Render(t.Name)
		} else {
			// Normal item
			normalStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.Foreground)).
				Padding(0, 2)
			line = normalStyle.Render(t.Name)
		}
		themeItems = append(themeItems, line)
	}

	// Color preview for selected theme
	selectedTheme := Themes[m.selectedIndex]
	previewStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(selectedTheme.Primary)).
		Padding(0, 1).
		Align(lipgloss.Center)

	var previewContent strings.Builder
	previewContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(selectedTheme.Primary)).Render("Primary"))
	previewContent.WriteString("  ")
	previewContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(selectedTheme.Accent)).Render("Accent"))
	previewContent.WriteString("  ")
	previewContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(selectedTheme.Success)).Render("Success"))
	previewContent.WriteString("  ")
	previewContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(selectedTheme.Error)).Render("Error"))

	// Stack everything centered
	content := lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render("Select Theme"),
		"",
		lipgloss.JoinVertical(lipgloss.Center, themeItems...),
		"",
		previewStyle.Render(previewContent.String()),
		"",
		helpStyle.Render(fmt.Sprintf("Theme %d of %d", m.selectedIndex+1, len(Themes))),
		helpStyle.Render("Up/Down: navigate • Enter: confirm • Esc: cancel"),
	)

	// Center the dialog
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		containerStyle.Render(content),
	)
}
