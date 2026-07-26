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
			if m.selectedIndex > 0 {
				m.selectedIndex--
			}
			// Apply theme live for preview
			SetTheme(m.selectedIndex)
			m.styles = NewStyles(m.width)
			return m, nil

		case "down", "j":
			if m.selectedIndex < len(Themes)-1 {
				m.selectedIndex++
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

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.SelectionFg)).
		Background(lipgloss.Color(theme.SelectionBg)).
		Bold(true)

	// Each row previews its own theme's colours, so the whole palette is
	// comparable at a glance instead of one theme at a time.
	lines := make([]string, 0, len(Themes))
	for i, t := range Themes {
		swatches := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary)).Render("███") +
			lipgloss.NewStyle().Foreground(lipgloss.Color(t.Accent)).Render("███") +
			lipgloss.NewStyle().Foreground(lipgloss.Color(t.Success)).Render("███") +
			lipgloss.NewStyle().Foreground(lipgloss.Color(t.Error)).Render("███")

		name := lipgloss.NewStyle().Width(20).Render(t.Name)

		if i == m.selectedIndex {
			lines = append(lines, selectedStyle.Render(" ▸ "+name)+" "+swatches)
		} else {
			lines = append(lines, "   "+muted(name)+" "+swatches)
		}
	}

	return formScreen(m.width, m.height,
		fmt.Sprintf("theme %d/%d", m.selectedIndex+1, len(Themes)),
		strings.Join(lines, "\n"), "",
		keyHint{"↑↓", "preview"},
		keyHint{"↵", "confirm"},
		keyHint{"esc", "cancel"},
	)
}
