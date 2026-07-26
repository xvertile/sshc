package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type helpModel struct {
	styles Styles
	width  int
	height int
}

// helpCloseMsg is sent when the help window is closed
type helpCloseMsg struct{}

// NewHelpForm creates a new help form model
func NewHelpForm(styles Styles, width, height int) *helpModel {
	return &helpModel{
		styles: styles,
		width:  width,
		height: height,
	}
}

func (m *helpModel) Init() tea.Cmd {
	return nil
}

func (m *helpModel) Update(msg tea.Msg) (*helpModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "h", "enter", "ctrl+c":
			return m, func() tea.Msg { return helpCloseMsg{} }
		}
	}
	return m, nil
}

// helpEntry is one key binding shown in the help screen.
type helpEntry struct {
	key    string
	action string
}

// helpSection groups related bindings under a heading.
type helpSection struct {
	title   string
	entries []helpEntry
}

// helpSections is the full key map, kept here as data so the help screen and
// the list footer cannot drift apart silently.
var helpSections = []helpSection{
	{"navigation", []helpEntry{
		{"↵", "connect to selected host"},
		{"↑ ↓", "move selection"},
		{"/", "search hosts"},
		{"tab", "switch focus"},
		{"i", "show host information"},
	}},
	{"hosts", []helpEntry{
		{"a", "add new host"},
		{"e", "edit selected host"},
		{"m", "move host to another config"},
		{"d", "delete selected host"},
		{"K", "add kubernetes host"},
	}},
	{"actions", []helpEntry{
		{"p", "ping all hosts"},
		{"f", "set up port forwarding"},
		{"t", "quick file transfer"},
		{"k", "upload SSH key to host"},
	}},
	{"view", []helpEntry{
		{"s", "cycle sort modes"},
		{"n", "sort by name"},
		{"r", "sort by recent connection"},
		{"c", "change theme"},
		{"ctrl+s", "toggle start-in-search"},
	}},
}

// helpColumnWidth is the width one column of bindings needs. Below two of
// these the help falls back to a single column rather than being clipped.
const helpColumnWidth = 38

func (m *helpModel) View() string {
	var body string

	inner := contentWidth(m.width)

	if inner < helpColumnWidth*2 {
		body = renderHelpSections(helpSections)
	} else {
		// Two columns, split so both are roughly the same height.
		half := (len(helpSections) + 1) / 2

		body = lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(helpColumnWidth).Render(renderHelpSections(helpSections[:half])),
			renderHelpSections(helpSections[half:]),
		)
	}

	return formScreen(m.width, m.height, "commands", body, "",
		keyHint{"esc", "close"},
	)
}

// renderHelpSections renders a group of sections as a single column.
func renderHelpSections(sections []helpSection) string {
	var lines []string

	for i, section := range sections {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, accent(section.title))

		for _, entry := range sections[i].entries {
			key := lipgloss.NewStyle().
				Foreground(lipgloss.Color(GetCurrentTheme().Primary)).
				Bold(true).
				Width(8).
				Render(entry.key)
			lines = append(lines, "  "+key+muted(entry.action))
		}
	}

	return strings.Join(lines, "\n")
}
