package ui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Gu1llaum-3/sshm/internal/transfer"
	tea "github.com/charmbracelet/bubbletea"
)

// BrowserMode defines whether we're selecting files or directories
type BrowserMode int

const (
	BrowseFiles BrowserMode = iota
	BrowseDirectories
)

// searchDebounceTime is how long to wait after typing before searching
const searchDebounceTime = 400 * time.Millisecond

// remoteBrowserModel is the TUI file browser for remote files
type remoteBrowserModel struct {
	host        string
	configFile  string
	currentDir  string
	files       []transfer.RemoteFile // All files from directory
	visibleFiles []transfer.RemoteFile // Filtered files (respects showHidden)
	cursor      int
	selected    string
	err         string
	loading     bool
	mode        BrowserMode
	styles      Styles
	width       int
	height      int
	session     *transfer.SFTPSession
	searchMode  bool
	searchQuery string
	searchFiles []transfer.RemoteFile // Search results
	hasLocate   bool                  // Whether locate is available on remote
	showHidden  bool                  // Whether to show dotfiles

	// Debounce state
	pendingSearch   string // Query waiting to be searched
	searchTriggered bool   // Whether a search has been triggered for current query
}

// remoteBrowserResultMsg is sent when browsing is complete
type remoteBrowserResultMsg struct {
	path     string
	selected bool
	err      error
}

// remoteBrowserLoadedMsg is sent when directory listing completes
type remoteBrowserLoadedMsg struct {
	files []transfer.RemoteFile
	dir   string
	err   error
}

// remoteBrowserSearchMsg is sent when search completes
type remoteBrowserSearchMsg struct {
	files []transfer.RemoteFile
	query string
	err   error
}

// searchDebounceMsg is sent after debounce delay to trigger actual search
type searchDebounceMsg struct {
	query string
}

// NewRemoteBrowser creates a new remote file browser
func NewRemoteBrowser(host, startPath, configFile string, mode BrowserMode, styles Styles, width, height int) *remoteBrowserModel {
	if startPath == "" {
		startPath = "~"
	}

	return &remoteBrowserModel{
		host:       host,
		configFile: configFile,
		currentDir: startPath,
		mode:       mode,
		styles:     styles,
		width:      width,
		height:     height,
		loading:    true,
		cursor:     0,
	}
}

func (m *remoteBrowserModel) Init() tea.Cmd {
	return m.loadDirectory(m.currentDir)
}

func (m *remoteBrowserModel) loadDirectory(path string) tea.Cmd {
	return func() tea.Msg {
		// Create SFTP session if needed
		if m.session == nil {
			session, err := transfer.NewSFTPSession(m.host, m.configFile)
			if err != nil {
				return remoteBrowserLoadedMsg{err: err}
			}
			m.session = session
		}

		// Expand ~ on first load
		if path == "~" {
			home, err := m.session.GetHomeDirectory()
			if err == nil {
				path = home
			}
		}

		files, err := m.session.ListDirectory(path)
		if err != nil {
			return remoteBrowserLoadedMsg{err: err}
		}

		return remoteBrowserLoadedMsg{files: files, dir: path}
	}
}

func (m *remoteBrowserModel) runSearch() tea.Cmd {
	query := m.searchQuery
	return func() tea.Msg {
		if m.session == nil {
			return remoteBrowserSearchMsg{err: fmt.Errorf("no session"), query: query}
		}

		files, err := m.session.QuickSearch(query, m.currentDir, 30)
		if err != nil {
			return remoteBrowserSearchMsg{err: err, query: query}
		}

		return remoteBrowserSearchMsg{files: files, query: query}
	}
}

func (m *remoteBrowserModel) scheduleSearch(query string) tea.Cmd {
	return tea.Tick(searchDebounceTime, func(t time.Time) tea.Msg {
		return searchDebounceMsg{query: query}
	})
}

// filterFiles updates visibleFiles based on showHidden setting
func (m *remoteBrowserModel) filterFiles() {
	if m.showHidden {
		m.visibleFiles = m.files
		return
	}

	m.visibleFiles = nil
	for _, f := range m.files {
		// Always show ".." for navigation
		if f.Name == ".." || !strings.HasPrefix(f.Name, ".") {
			m.visibleFiles = append(m.visibleFiles, f)
		}
	}
}

// filterSearchResults filters existing search results by current query (for backspace)
func (m *remoteBrowserModel) filterSearchResults() {
	if len(m.searchQuery) < 3 {
		return
	}
	query := strings.ToLower(m.searchQuery)
	var filtered []transfer.RemoteFile
	for _, f := range m.searchFiles {
		if strings.Contains(strings.ToLower(f.Name), query) ||
			strings.Contains(strings.ToLower(f.Path), query) {
			filtered = append(filtered, f)
		}
	}
	m.searchFiles = filtered
	if m.cursor >= len(m.searchFiles) {
		m.cursor = len(m.searchFiles) - 1
		if m.cursor < 0 {
			m.cursor = 0
		}
	}
}

// sortSearchResults sorts results: exact filename matches first, then by path length
func (m *remoteBrowserModel) sortSearchResults() {
	if len(m.searchFiles) == 0 || len(m.searchQuery) < 3 {
		return
	}
	query := strings.ToLower(m.searchQuery)

	sort.SliceStable(m.searchFiles, func(i, j int) bool {
		fi, fj := m.searchFiles[i], m.searchFiles[j]
		nameI, nameJ := strings.ToLower(fi.Name), strings.ToLower(fj.Name)

		// Exact filename match gets highest priority
		exactI := nameI == query
		exactJ := nameJ == query
		if exactI != exactJ {
			return exactI
		}

		// Filename starts with query
		startsI := strings.HasPrefix(nameI, query)
		startsJ := strings.HasPrefix(nameJ, query)
		if startsI != startsJ {
			return startsI
		}

		// Filename contains query (vs only path contains)
		containsI := strings.Contains(nameI, query)
		containsJ := strings.Contains(nameJ, query)
		if containsI != containsJ {
			return containsI
		}

		// Shorter paths first (less nested = more relevant)
		return len(fi.Path) < len(fj.Path)
	})
}

func (m *remoteBrowserModel) Update(msg tea.Msg) (*remoteBrowserModel, tea.Cmd) {
	switch msg := msg.(type) {
	case remoteBrowserLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.files = msg.files
		m.currentDir = msg.dir
		m.cursor = 0
		m.err = ""
		m.searchMode = false
		m.searchQuery = ""
		m.searchFiles = nil
		m.filterFiles()
		return m, nil

	case remoteBrowserSearchMsg:
		// Only process if this is for the current query (ignore stale results)
		if msg.query != m.searchQuery {
			return m, nil
		}
		m.loading = false
		m.searchTriggered = true
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.searchFiles = msg.files
		m.sortSearchResults()
		m.cursor = 0
		m.err = ""
		return m, nil

	case searchDebounceMsg:
		// Only search if query hasn't changed since debounce was scheduled
		if msg.query == m.searchQuery && len(m.searchQuery) >= 3 && !m.searchTriggered {
			m.loading = true
			m.pendingSearch = ""
			return m, m.runSearch()
		}
		return m, nil

	case tea.KeyMsg:
		// Allow navigation even while loading
		if m.loading && m.searchMode {
			switch msg.String() {
			case "esc":
				m.searchMode = false
				m.searchQuery = ""
				m.searchFiles = nil
				m.pendingSearch = ""
				m.searchTriggered = false
				m.loading = false
				m.cursor = 0
				return m, nil
			case "up", "ctrl+p":
				if m.cursor > 0 {
					m.cursor--
				}
				return m, nil
			case "down", "ctrl+n":
				if m.cursor < len(m.searchFiles)-1 {
					m.cursor++
				}
				return m, nil
			}
			return m, nil
		}

		if m.loading {
			return m, nil
		}

		// Handle search mode input
		if m.searchMode {
			switch msg.String() {
			case "esc", "ctrl+c":
				// Exit search mode (ctrl+c exits search, not the app)
				m.searchMode = false
				m.searchQuery = ""
				m.searchFiles = nil
				m.pendingSearch = ""
				m.searchTriggered = false
				m.cursor = 0
				return m, nil

			case "enter":
				if len(m.searchFiles) > 0 && m.cursor < len(m.searchFiles) {
					// Select from search results
					file := m.searchFiles[m.cursor]
					if file.IsDir {
						// Navigate to directory
						m.searchMode = false
						m.searchQuery = ""
						m.searchFiles = nil
						m.pendingSearch = ""
						m.searchTriggered = false
						m.loading = true
						return m, m.loadDirectory(file.Path)
					} else if m.mode == BrowseFiles {
						// Select file
						if m.session != nil {
							m.session.Close()
						}
						return m, func() tea.Msg {
							return remoteBrowserResultMsg{path: file.Path, selected: true}
						}
					}
				}
				return m, nil

			case "up", "ctrl+p":
				if m.cursor > 0 {
					m.cursor--
				}
				return m, nil

			case "down", "ctrl+n":
				if m.cursor < len(m.searchFiles)-1 {
					m.cursor++
				}
				return m, nil

			case "backspace":
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
					m.searchTriggered = false
					if len(m.searchQuery) < 3 {
						m.searchFiles = nil
						m.pendingSearch = ""
					} else {
						// Filter existing results locally instead of new search
						// This makes backspace instant
						if len(m.searchFiles) > 0 {
							m.filterSearchResults()
						}
						m.pendingSearch = ""
					}
				}
				return m, nil

			default:
				// Add character to search query
				char := msg.String()
				if len(char) == 1 && char[0] >= 32 && char[0] < 127 {
					m.searchQuery += char
					m.searchTriggered = false
					if len(m.searchQuery) >= 3 {
						// Schedule debounced search
						m.pendingSearch = m.searchQuery
						return m, m.scheduleSearch(m.searchQuery)
					}
				}
				return m, nil
			}
		}

		// Normal mode
		switch msg.String() {
		case "q", "ctrl+c":
			// Cancel
			if m.session != nil {
				m.session.Close()
			}
			return m, func() tea.Msg {
				return remoteBrowserResultMsg{selected: false}
			}

		case "esc":
			// Cancel or exit search
			if m.searchMode {
				m.searchMode = false
				m.searchQuery = ""
				m.searchFiles = nil
				return m, nil
			}
			if m.session != nil {
				m.session.Close()
			}
			return m, func() tea.Msg {
				return remoteBrowserResultMsg{selected: false}
			}

		case "/":
			// Enter search mode
			m.searchMode = true
			m.searchQuery = ""
			m.searchFiles = nil
			m.cursor = 0
			return m, nil

		case ".":
			// Toggle hidden files
			m.showHidden = !m.showHidden
			m.filterFiles()
			// Adjust cursor if it's now out of bounds
			if m.cursor >= len(m.visibleFiles) {
				m.cursor = len(m.visibleFiles) - 1
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
			return m, nil

		case "r", "R":
			// Retry connection / reload current directory
			m.err = ""
			m.loading = true
			// Close existing session to force reconnect
			if m.session != nil {
				m.session.Close()
				m.session = nil
			}
			return m, m.loadDirectory(m.currentDir)

		case "enter":
			if len(m.visibleFiles) == 0 {
				return m, nil
			}

			file := m.visibleFiles[m.cursor]

			if file.IsDir {
				// Enter directory
				m.loading = true
				return m, m.loadDirectory(file.Path)
			}
			// File selected
			if m.mode == BrowseFiles {
				if m.session != nil {
					m.session.Close()
				}
				return m, func() tea.Msg {
					return remoteBrowserResultMsg{path: file.Path, selected: true}
				}
			}
			return m, nil

		case "s", " ":
			// Select current directory (for BrowseDirectories mode)
			if m.mode == BrowseDirectories {
				path := m.currentDir
				// If in search mode and on a directory, select that
				if m.searchMode && len(m.searchFiles) > 0 && m.searchFiles[m.cursor].IsDir {
					path = m.searchFiles[m.cursor].Path
				}
				if m.session != nil {
					m.session.Close()
				}
				return m, func() tea.Msg {
					return remoteBrowserResultMsg{path: path, selected: true}
				}
			}
			return m, nil

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case "down", "j":
			if m.cursor < len(m.visibleFiles)-1 {
				m.cursor++
			}
			return m, nil

		case "home", "g":
			m.cursor = 0
			return m, nil

		case "end", "G":
			if len(m.visibleFiles) > 0 {
				m.cursor = len(m.visibleFiles) - 1
			}
			return m, nil

		case "backspace", "h", "left":
			// Go to parent directory
			parent := filepath.Dir(m.currentDir)
			if parent != m.currentDir {
				m.loading = true
				return m, m.loadDirectory(parent)
			}
			return m, nil

		case "~":
			// Go to home directory
			m.loading = true
			return m, m.loadDirectory("~")

		case "right", "l":
			// Enter directory if on one
			if len(m.visibleFiles) > 0 && m.visibleFiles[m.cursor].IsDir {
				m.loading = true
				return m, m.loadDirectory(m.visibleFiles[m.cursor].Path)
			}
			return m, nil
		}
	}

	return m, nil
}

func (m *remoteBrowserModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(m.styles.Header.Render(fmt.Sprintf("📂 Remote Browser: %s", m.host)))
	b.WriteString("\n")

	// Current path or search mode indicator
	if m.searchMode {
		cursor := "_"
		if m.loading {
			cursor = ""
		}
		if len(m.searchQuery) < 3 {
			b.WriteString(fmt.Sprintf("  🔍 Search: %s%s (type %d more)\n", m.searchQuery, cursor, 3-len(m.searchQuery)))
		} else {
			b.WriteString(fmt.Sprintf("  🔍 Search: %s%s\n", m.searchQuery, cursor))
		}
		b.WriteString("  in: " + m.currentDir + "\n")
	} else {
		b.WriteString(m.styles.DirStyle.Render("  "+m.currentDir) + "\n")
	}
	b.WriteString("\n")

	// Error message
	if m.err != "" {
		b.WriteString(m.styles.Error.Render("Error: "+m.err) + "\n\n")
	}

	// Loading indicator or file list
	if m.loading {
		if m.searchMode {
			b.WriteString("  Searching...\n")
		} else {
			b.WriteString("  Loading...\n")
		}
	} else {
		// Choose which file list to display
		displayFiles := m.visibleFiles
		if m.searchMode && len(m.searchFiles) > 0 {
			displayFiles = m.searchFiles
		} else if m.searchMode && len(m.searchQuery) >= 3 && m.searchTriggered && len(m.searchFiles) == 0 {
			b.WriteString("  No files found\n")
			displayFiles = nil
		} else if m.searchMode {
			displayFiles = nil
		}

		if displayFiles != nil {
			visibleHeight := m.height - 10
			if visibleHeight < 5 {
				visibleHeight = 5
			}

			start := 0
			if m.cursor >= visibleHeight {
				start = m.cursor - visibleHeight + 1
			}
			end := start + visibleHeight
			if end > len(displayFiles) {
				end = len(displayFiles)
			}

			for i := start; i < end; i++ {
				file := displayFiles[i]
				if m.searchMode {
					b.WriteString(m.renderSearchResultLine(file, i == m.cursor) + "\n")
				} else {
					b.WriteString(m.renderFileLine(file, i == m.cursor) + "\n")
				}
			}

			if len(displayFiles) > visibleHeight {
				b.WriteString(fmt.Sprintf("  [%d/%d]\n", m.cursor+1, len(displayFiles)))
			}
		}
	}

	b.WriteString("\n")

	// Hidden files indicator and help
	if !m.searchMode {
		if m.showHidden {
			b.WriteString("  [hidden: on]\n")
		} else {
			b.WriteString("  [hidden: off]\n")
		}
	}

	if m.searchMode {
		b.WriteString(" ↑/↓: navigate | Enter: select | Esc: back\n")
	} else if m.mode == BrowseDirectories {
		b.WriteString(" ↑/↓: navigate | Enter: open | s: select | r: retry | Esc: cancel\n")
	} else {
		b.WriteString(" ↑/↓: navigate | Enter: select | /: search | r: retry | Esc: cancel\n")
	}

	return b.String()
}

// ANSI escape codes for fast rendering (avoid lipgloss.Render in hot loop)
const (
	ansiReset    = "\x1b[0m"
	ansiSelected = "\x1b[38;5;229;48;2;0;173;216m" // white on cyan (matches Selected style)
	ansiDir      = "\x1b[38;5;39m"                 // blue (matches DirStyle)
)

func (m *remoteBrowserModel) renderFileLine(file transfer.RemoteFile, selected bool) string {
	var icon, name string

	if file.Name == ".." {
		icon = "⬆"
		name = ".."
	} else if file.IsDir {
		icon = "📁"
		name = file.Name + "/"
	} else {
		icon = "  "
		name = file.Name
	}

	// Simple truncation
	if len(name) > 40 {
		name = name[:37] + "..."
	}

	if selected {
		return ansiSelected + "  " + icon + " " + name + ansiReset
	}
	if file.IsDir {
		return ansiDir + "  " + icon + " " + name + ansiReset
	}
	return "  " + icon + " " + name
}

// renderSearchResultLine renders a search result showing the full path
func (m *remoteBrowserModel) renderSearchResultLine(file transfer.RemoteFile, selected bool) string {
	icon := "📁"
	if !file.IsDir {
		icon = "  "
	}

	path := file.Path
	if len(path) > 50 {
		path = "..." + path[len(path)-47:]
	}

	if selected {
		return ansiSelected + "  " + icon + " " + path + ansiReset
	}
	if file.IsDir {
		return ansiDir + "  " + icon + " " + path + ansiReset
	}
	return "  " + icon + " " + path
}

func formatSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.1fG", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.1fM", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.1fK", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%dB", size)
	}
}

// Standalone browser for CLI use

type standaloneRemoteBrowser struct {
	*remoteBrowserModel
}

func (m standaloneRemoteBrowser) Init() tea.Cmd {
	return m.remoteBrowserModel.Init()
}

func (m standaloneRemoteBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Only update if dimensions actually changed
		if m.remoteBrowserModel.width != msg.Width || m.remoteBrowserModel.height != msg.Height {
			m.remoteBrowserModel.width = msg.Width
			m.remoteBrowserModel.height = msg.Height
			m.remoteBrowserModel.styles = NewStyles(msg.Width)
		}
		return m, nil

	case remoteBrowserResultMsg:
		// Store result for retrieval
		if msg.selected {
			m.remoteBrowserModel.selected = msg.path
		}
		return m, tea.Quit
	}

	newModel, cmd := m.remoteBrowserModel.Update(msg)
	m.remoteBrowserModel = newModel
	return m, cmd
}

func (m standaloneRemoteBrowser) View() string {
	return m.remoteBrowserModel.View()
}

// RunRemoteBrowser runs the remote browser as a standalone TUI and returns the selected path
func RunRemoteBrowser(host, startPath, configFile string, mode BrowserMode) (string, bool, error) {
	styles := NewStyles(80)
	browser := NewRemoteBrowser(host, startPath, configFile, mode, styles, 80, 24)
	m := standaloneRemoteBrowser{browser}

	p := tea.NewProgram(m,
		tea.WithAltScreen(),
	)
	finalModel, err := p.Run()
	if err != nil {
		return "", false, err
	}

	if result, ok := finalModel.(standaloneRemoteBrowser); ok {
		if result.remoteBrowserModel.selected != "" {
			return result.remoteBrowserModel.selected, true, nil
		}
	}

	return "", false, nil
}
