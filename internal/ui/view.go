package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the complete user interface
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Handle different view modes
	switch m.viewMode {
	case ViewAdd:
		if m.addForm != nil {
			return m.addForm.View()
		}
	case ViewEdit:
		if m.editForm != nil {
			return m.editForm.View()
		}
	case ViewMove:
		if m.moveForm != nil {
			return m.moveForm.View()
		}
	case ViewInfo:
		if m.infoForm != nil {
			return m.infoForm.View()
		}
	case ViewPortForward:
		if m.portForwardForm != nil {
			return m.portForwardForm.View()
		}
	case ViewPortForwardSession:
		if m.portForwardActive != nil {
			return m.portForwardActive.View()
		}
	case ViewTransfer:
		if m.transferForm != nil {
			return m.transferForm.View()
		}
	case ViewQuickTransfer:
		if m.quickTransferForm != nil {
			return m.quickTransferForm.View()
		}
	case ViewRemoteBrowser:
		if m.remoteBrowserForm != nil {
			return m.remoteBrowserForm.View()
		}
	case ViewHelp:
		if m.helpForm != nil {
			return m.helpForm.View()
		}
	case ViewFileSelector:
		if m.fileSelectorForm != nil {
			return m.fileSelectorForm.View()
		}
	case ViewK8sAdd:
		if m.k8sAddForm != nil {
			return m.k8sAddForm.View()
		}
	case ViewK8sEdit:
		if m.k8sEditForm != nil {
			return m.k8sEditForm.View()
		}
	case ViewTheme:
		if m.themePicker != nil {
			return m.themePicker.View()
		}
	case ViewConnectionError:
		return m.renderConnectionErrorView()
	case ViewSSHKeyUpload:
		if m.sshKeyUploadForm != nil {
			return m.sshKeyUploadForm.View()
		}
	case ViewList:
		return m.renderListView()
	}

	return m.renderListView()
}

// renderListView renders the main list interface.
//
// The layout is fixed-height and top-anchored: a status line, a filter line, a
// rule, the table, a rule, and key hints. Every line not claimed by that
// chrome belongs to the table, so the host list grows with the terminal
// instead of floating in the middle of it.
func (m Model) renderListView() string {
	chrome := chromeFor(m.height)
	width := m.contentWidth()

	sections := [][]string{{m.renderHeaderLine(width)}}

	rows := len(sections) // the table section goes next
	sections = append(sections, strings.Split(m.table.View(), "\n"))

	if chrome.hints {
		sections = append(sections, []string{m.renderFooterLine(width)})
	}

	if chrome.box {
		return panel(m.width, m.height, sections, rows)
	}

	// Too short to spend two lines on a border: stack the sections plain.
	var header, footer []string
	for i, section := range sections {
		switch {
		case i < rows:
			header = append(header, section...)
		case i > rows:
			footer = append(footer, section...)
		}
	}

	return screen(m.width, m.height,
		strings.Join(header, "\n"),
		m.table.View(),
		strings.Join(footer, "\n"),
	)
}

// contentWidth returns the columns available to content, which the border and
// its padding reduce when the screen is framed.
func (m Model) contentWidth() int {
	if chromeFor(m.height).box {
		return contentWidth(m.width)
	}
	return m.width
}

// renderHeaderLine renders the single top row: the wordmark and search on the
// left, the host count on the right.
func (m Model) renderHeaderLine(width int) string {
	theme := GetCurrentTheme()

	left := wordmark()

	// "dev" is the placeholder a build carries when no version was injected at
	// link time, so it says nothing worth a slot in the header.
	if m.currentVersion != "" && m.currentVersion != devVersion {
		left += " " + muted(m.currentVersion)
	}

	// Show "shown/total" only while a filter is actually narrowing the list.
	total := len(m.allEntries)
	shown := len(m.filteredEntries)

	right := fmt.Sprintf("%d hosts", total)
	if shown != total {
		right = fmt.Sprintf("%d/%d hosts", shown, total)
	}
	right = muted(right)

	// A transient error takes the right slot outright, then an available
	// update; either displaces the count, which matters less than both.
	switch {
	case m.showingError && m.errorMessage != "":
		right = lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Error)).
			Bold(true).
			Render("! " + m.errorMessage)

	case m.updateInfo != nil && m.updateInfo.Available:
		right = lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Success)).
			Render("⇧ "+m.updateInfo.LatestVer) + "  " + right
	}

	return statusLine(width, left+"  "+m.renderSearch(), right)
}

// renderSearch renders the search, which sits beside the wordmark rather than
// on a line of its own.
func (m Model) renderSearch() string {
	if m.searchMode {
		label := lipgloss.NewStyle().
			Foreground(lipgloss.Color(GetCurrentTheme().Primary)).
			Bold(true).
			Render("search ▸ ")
		return label + m.searchInput.View()
	}

	if query := m.searchInput.Value(); query != "" {
		return muted("search ") + accent(query)
	}

	return muted("press / to search")
}

// renderFooterLine renders the frame's bottom edge, carrying either the key
// hints or the delete confirmation prompt. The prompt is inline rather than a
// modal so the host list stays visible while confirming.
func (m Model) renderFooterLine(width int) string {

	if m.deleteMode {
		kind := "host"
		if m.deleteHostIsK8s {
			kind = "k8s host"
		}

		prompt := lipgloss.NewStyle().
			Foreground(lipgloss.Color(GetCurrentTheme().Error)).
			Bold(true).
			Render(fmt.Sprintf("delete %s %q?", kind, m.deleteHost))

		hints := keyHints(width,
			keyHint{"enter", "confirm"},
			keyHint{"esc", "cancel"},
		)

		return statusLine(width, prompt+"  "+hints, "")
	}

	if m.searchMode {
		return keyHints(width,
			keyHint{"enter", "apply"},
			keyHint{"tab", "list"},
			keyHint{"esc", "exit"},
		)
	}

	// help and quit are pinned right: on a narrow terminal the action hints
	// are dropped from the middle, but the way out stays visible.
	pinned := keyHints(width, keyHint{"h", "help"}, keyHint{"q", "quit"})

	actions := keyHints(width-lipgloss.Width(pinned)-2,
		keyHint{"↵", "connect"},
		keyHint{"a", "add"},
		keyHint{"e", "edit"},
		keyHint{"d", "delete"},
		keyHint{"i", "info"},
		keyHint{"f", "forward"},
		keyHint{"t", "transfer"},
		keyHint{"k", "key"},
		keyHint{"p", "ping"},
		keyHint{"s", "sort"},
		keyHint{"c", "theme"},
	)

	return statusLine(width, actions, pinned)
}

// renderConnectionErrorView renders the connection failure screen with retry.
func (m Model) renderConnectionErrorView() string {
	theme := GetCurrentTheme()

	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Error)).
		Bold(true).
		Render("connection failed")

	inner := contentWidth(m.width)

	// Wrap on the panel's width rather than the previous fixed 60 columns.
	body := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Error)).
		Width(inner).
		Render(m.connectionError)

	sections := [][]string{
		{statusLine(inner, title, muted(m.connectionHost))},
		strings.Split(body, "\n"),
		{keyHints(inner, keyHint{"r", "retry"}, keyHint{"esc", "back"})},
	}

	return panel(m.width, m.height, sections, 1)
}
