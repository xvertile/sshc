package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xvertile/sshc/internal/history"
	"github.com/xvertile/sshc/internal/transfer"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Input field indices for transfer form
const (
	tfDirectionInput = iota
	tfUploadTypeInput // File or Folder toggle (only shown for uploads)
	tfLocalPathInput
	tfRemotePathInput
)

// UploadType determines whether to upload a file or folder
type UploadType int

const (
	UploadFile UploadType = iota
	UploadFolder
)

type transferFormModel struct {
	inputs         []textinput.Model
	focused        int
	direction      transfer.Direction
	uploadType     UploadType // File or Folder
	hostName       string
	err            string
	styles         Styles
	width          int
	height         int
	configFile     string
	historyManager *history.HistoryManager
	historyItems   []history.TransferHistoryEntry
	historyIndex   int // -1 means no history item selected
	showHistory    bool
}

// transferSubmitMsg is sent when the transfer form is submitted
type transferSubmitMsg struct {
	err     error
	request *transfer.TransferRequest
}

// transferCancelMsg is sent when the transfer form is cancelled
type transferCancelMsg struct{}

// filePickerResultMsg is sent when a file picker dialog completes
type filePickerResultMsg struct {
	path     string
	selected bool
	isLocal  bool // true for local path, false for remote
}

// NewTransferForm creates a new transfer form model
func NewTransferForm(hostName string, styles Styles, width, height int, configFile string, direction transfer.Direction) *transferFormModel {
	// Initialize history manager
	historyManager, _ := history.NewHistoryManager()

	inputs := make([]textinput.Model, 4)

	// Direction input (display only, controlled by arrow keys)
	inputs[tfDirectionInput] = textinput.New()
	inputs[tfDirectionInput].Placeholder = "Use ←/→ to change direction"
	inputs[tfDirectionInput].Focus()
	inputs[tfDirectionInput].Width = 40

	// Upload type input (display only, controlled by arrow keys) - only used for uploads
	inputs[tfUploadTypeInput] = textinput.New()
	inputs[tfUploadTypeInput].Placeholder = "Use ←/→ to change type"
	inputs[tfUploadTypeInput].Width = 40

	// Get current working directory for default local path
	cwd, _ := os.Getwd()

	// Local path input
	inputs[tfLocalPathInput] = textinput.New()
	inputs[tfLocalPathInput].Placeholder = cwd
	inputs[tfLocalPathInput].CharLimit = 500
	inputs[tfLocalPathInput].Width = 60

	// Remote path input
	inputs[tfRemotePathInput] = textinput.New()
	inputs[tfRemotePathInput].Placeholder = "~/"
	inputs[tfRemotePathInput].CharLimit = 500
	inputs[tfRemotePathInput].Width = 60

	m := &transferFormModel{
		inputs:         inputs,
		focused:        0,
		direction:      direction,
		uploadType:     UploadFile, // Default to file
		hostName:       hostName,
		styles:         styles,
		width:          width,
		height:         height,
		configFile:     configFile,
		historyManager: historyManager,
		historyIndex:   -1,
		showHistory:    true,
	}

	// Set initial direction display
	if direction == transfer.Upload {
		inputs[tfDirectionInput].SetValue("↑ Upload")
	} else {
		inputs[tfDirectionInput].SetValue("↓ Download")
	}

	// Load transfer history
	m.loadHistory()

	// Update placeholders based on direction
	m.updatePlaceholders()

	return m
}

func (m *transferFormModel) loadHistory() {
	if m.historyManager != nil {
		m.historyItems = m.historyManager.GetTransferHistory(m.hostName)
	}
}

func (m *transferFormModel) updatePlaceholders() {
	if m.direction == transfer.Upload {
		m.inputs[tfLocalPathInput].Placeholder = "Local file or directory to upload"
		m.inputs[tfRemotePathInput].Placeholder = "Remote destination (default: ~/)"
	} else {
		m.inputs[tfLocalPathInput].Placeholder = "Local destination (default: ./)"
		m.inputs[tfRemotePathInput].Placeholder = "Remote file or directory to download"
	}
}

func (m *transferFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *transferFormModel) openLocalFilePicker() tea.Cmd {
	return func() tea.Msg {
		// Determine picker mode based on direction and upload type
		var mode transfer.PickerMode
		var title string

		if m.direction == transfer.Upload {
			if m.uploadType == UploadFolder {
				mode = transfer.PickDirectory
				title = "Select folder to upload"
			} else {
				mode = transfer.PickFile
				title = "Select file to upload"
			}
		} else {
			mode = transfer.PickDirectory
			title = "Select download destination"
		}

		// Get starting directory
		startDir, _ := os.Getwd()
		if currentPath := m.inputs[tfLocalPathInput].Value(); currentPath != "" {
			if expanded, err := transfer.ExpandPath(currentPath); err == nil {
				startDir = expanded
			}
		}

		result, err := transfer.OpenFilePicker(mode, title, startDir)
		if err != nil || result == nil || !result.Selected {
			return filePickerResultMsg{selected: false, isLocal: true}
		}

		return filePickerResultMsg{
			path:     result.Path,
			selected: true,
			isLocal:  true,
		}
	}
}

func (m *transferFormModel) openRemoteFilePicker() tea.Cmd {
	return func() tea.Msg {
		// Determine browser mode based on direction
		var mode BrowserMode
		if m.direction == transfer.Upload {
			mode = BrowseDirectories
		} else {
			mode = BrowseFiles
		}

		// Get starting path
		startPath := m.inputs[tfRemotePathInput].Value()
		if startPath == "" {
			startPath = "~"
		}

		// Run the TUI browser
		path, selected, err := RunRemoteBrowser(m.hostName, startPath, m.configFile, mode)
		if err != nil {
			return filePickerResultMsg{selected: false, isLocal: false}
		}
		if !selected {
			return filePickerResultMsg{selected: false, isLocal: false}
		}

		return filePickerResultMsg{
			path:     path,
			selected: true,
			isLocal:  false,
		}
	}
}

// getNextFocusField returns the next focusable field index
func (m *transferFormModel) getNextFocusField(current int) int {
	next := current + 1
	// Skip upload type field if in download mode
	if next == tfUploadTypeInput && m.direction == transfer.Download {
		next++
	}
	if next > tfRemotePathInput {
		next = tfRemotePathInput
	}
	return next
}

// getPrevFocusField returns the previous focusable field index
func (m *transferFormModel) getPrevFocusField(current int) int {
	prev := current - 1
	// Skip upload type field if in download mode
	if prev == tfUploadTypeInput && m.direction == transfer.Download {
		prev--
	}
	if prev < tfDirectionInput {
		prev = tfDirectionInput
	}
	return prev
}

func (m *transferFormModel) Update(msg tea.Msg) (*transferFormModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case filePickerResultMsg:
		if msg.selected {
			if msg.isLocal {
				m.inputs[tfLocalPathInput].SetValue(msg.path)
			} else {
				m.inputs[tfRemotePathInput].SetValue(msg.path)
			}
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			return m, func() tea.Msg { return transferCancelMsg{} }

		case "enter":
			// If we're on direction, move to next field
			if m.focused == tfDirectionInput {
				m.inputs[m.focused].Blur()
				m.focused = m.getNextFocusField(m.focused)
				m.inputs[m.focused].Focus()
				return m, textinput.Blink
			}
			// If on upload type, move to local path
			if m.focused == tfUploadTypeInput {
				m.inputs[m.focused].Blur()
				m.focused = tfLocalPathInput
				m.inputs[m.focused].Focus()
				return m, textinput.Blink
			}
			// If on local path, move to remote path
			if m.focused == tfLocalPathInput {
				m.inputs[m.focused].Blur()
				m.focused = tfRemotePathInput
				m.inputs[m.focused].Focus()
				return m, textinput.Blink
			}
			// If on remote path, submit
			return m, m.submitForm()

		case "shift+tab", "up":
			prev := m.getPrevFocusField(m.focused)
			if prev != m.focused {
				m.inputs[m.focused].Blur()
				m.focused = prev
				m.inputs[m.focused].Focus()
				return m, textinput.Blink
			}

		case "tab", "down":
			next := m.getNextFocusField(m.focused)
			if next != m.focused {
				m.inputs[m.focused].Blur()
				m.focused = next
				m.inputs[m.focused].Focus()
				return m, textinput.Blink
			}

		case "left", "right":
			if m.focused == tfDirectionInput {
				// Toggle direction
				if m.direction == transfer.Upload {
					m.direction = transfer.Download
					m.inputs[tfDirectionInput].SetValue("↓ Download")
				} else {
					m.direction = transfer.Upload
					m.inputs[tfDirectionInput].SetValue("↑ Upload")
				}
				m.updatePlaceholders()
				return m, nil
			}
			if m.focused == tfUploadTypeInput {
				// Toggle upload type (File/Folder)
				if m.uploadType == UploadFile {
					m.uploadType = UploadFolder
				} else {
					m.uploadType = UploadFile
				}
				return m, nil
			}

		case "ctrl+h":
			// Toggle history display
			m.showHistory = !m.showHistory
			return m, nil

		case "ctrl+p", "ctrl+n":
			// Navigate history
			if len(m.historyItems) > 0 {
				if msg.String() == "ctrl+p" {
					// Previous (older) history item
					if m.historyIndex < len(m.historyItems)-1 {
						m.historyIndex++
						m.applyHistoryItem(m.historyIndex)
					}
				} else {
					// Next (newer) history item
					if m.historyIndex > 0 {
						m.historyIndex--
						m.applyHistoryItem(m.historyIndex)
					} else if m.historyIndex == 0 {
						m.historyIndex = -1
						// Clear inputs
						m.inputs[tfLocalPathInput].SetValue("")
						m.inputs[tfRemotePathInput].SetValue("")
					}
				}
				return m, nil
			}

		case "1", "2", "3", "4", "5":
			// Quick select history item (1-5)
			if m.focused == tfDirectionInput && len(m.historyItems) > 0 {
				idx := int(msg.String()[0] - '1')
				if idx < len(m.historyItems) {
					m.historyIndex = idx
					m.applyHistoryItem(idx)
					return m, nil
				}
			}

		case "o", "O":
			// Open native file picker
			if m.focused == tfLocalPathInput {
				return m, m.openLocalFilePicker()
			}
			if m.focused == tfRemotePathInput {
				return m, m.openRemoteFilePicker()
			}

		case "b", "B":
			// Browse - same as 'o' for convenience
			if m.focused == tfLocalPathInput {
				return m, m.openLocalFilePicker()
			}
			if m.focused == tfRemotePathInput {
				return m, m.openRemoteFilePicker()
			}
		}
	}

	// Update the focused input
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

func (m *transferFormModel) applyHistoryItem(idx int) {
	if idx >= 0 && idx < len(m.historyItems) {
		item := m.historyItems[idx]
		m.inputs[tfLocalPathInput].SetValue(item.LocalPath)
		m.inputs[tfRemotePathInput].SetValue(item.RemotePath)
		if item.Direction == "upload" {
			m.direction = transfer.Upload
			m.inputs[tfDirectionInput].SetValue("↑ Upload")
		} else {
			m.direction = transfer.Download
			m.inputs[tfDirectionInput].SetValue("↓ Download")
		}
		m.updatePlaceholders()
	}
}

func (m *transferFormModel) View() string {
	var sections []string

	// Title
	title := m.styles.Header.Render("📁 File Transfer")
	sections = append(sections, title)

	// Host info
	hostInfo := fmt.Sprintf("Host: %s", m.hostName)
	sections = append(sections, m.styles.HelpText.Render(hostInfo))
	sections = append(sections, "")

	// Error message
	if m.err != "" {
		sections = append(sections, m.styles.Error.Render("Error: "+m.err))
		sections = append(sections, "")
	}

	// Direction selector
	dirLabel := "Direction:"
	if m.focused == tfDirectionInput {
		dirLabel = m.styles.FocusedLabel.Render(dirLabel)
	} else {
		dirLabel = m.styles.Label.Render(dirLabel)
	}
	sections = append(sections, dirLabel)

	// Direction buttons
	uploadBtn := " ↑ Upload "
	downloadBtn := " ↓ Download "
	if m.direction == transfer.Upload {
		uploadBtn = m.styles.ActiveTab.Render(uploadBtn)
		downloadBtn = m.styles.InactiveTab.Render(downloadBtn)
	} else {
		uploadBtn = m.styles.InactiveTab.Render(uploadBtn)
		downloadBtn = m.styles.ActiveTab.Render(downloadBtn)
	}
	dirButtons := lipgloss.JoinHorizontal(lipgloss.Center, uploadBtn, "  ", downloadBtn)
	sections = append(sections, dirButtons)
	if m.focused == tfDirectionInput {
		sections = append(sections, m.styles.HelpText.Render("Use ←/→ to change direction"))
	}
	sections = append(sections, "")

	// Upload type selector (only shown for uploads)
	if m.direction == transfer.Upload {
		typeLabel := "Upload Type:"
		if m.focused == tfUploadTypeInput {
			typeLabel = m.styles.FocusedLabel.Render(typeLabel)
		} else {
			typeLabel = m.styles.Label.Render(typeLabel)
		}
		sections = append(sections, typeLabel)

		// File/Folder toggle buttons
		fileBtn := " 📄 File "
		folderBtn := " 📁 Folder "
		if m.uploadType == UploadFile {
			fileBtn = m.styles.ActiveTab.Render(fileBtn)
			folderBtn = m.styles.InactiveTab.Render(folderBtn)
		} else {
			fileBtn = m.styles.InactiveTab.Render(fileBtn)
			folderBtn = m.styles.ActiveTab.Render(folderBtn)
		}
		typeButtons := lipgloss.JoinHorizontal(lipgloss.Center, fileBtn, "  ", folderBtn)
		sections = append(sections, typeButtons)
		if m.focused == tfUploadTypeInput {
			sections = append(sections, m.styles.HelpText.Render("Use ←/→ to change type"))
		}
		sections = append(sections, "")
	}

	// Local path
	localLabel := "Local Path:"
	if m.focused == tfLocalPathInput {
		localLabel = m.styles.FocusedLabel.Render(localLabel)
	} else {
		localLabel = m.styles.Label.Render(localLabel)
	}
	sections = append(sections, localLabel)
	sections = append(sections, m.inputs[tfLocalPathInput].View())

	// Show file picker hint when focused on local path
	if m.focused == tfLocalPathInput && transfer.IsPickerAvailable() {
		sections = append(sections, m.styles.HelpText.Render("Press 'o' to browse"))
	}
	sections = append(sections, "")

	// Remote path
	remoteLabel := "Remote Path:"
	if m.focused == tfRemotePathInput {
		remoteLabel = m.styles.FocusedLabel.Render(remoteLabel)
	} else {
		remoteLabel = m.styles.Label.Render(remoteLabel)
	}
	sections = append(sections, remoteLabel)
	sections = append(sections, m.inputs[tfRemotePathInput].View())

	// Show file picker hint when focused on remote path
	if m.focused == tfRemotePathInput {
		sections = append(sections, m.styles.HelpText.Render("Press 'o' to browse remote files"))
	}
	sections = append(sections, "")

	// Transfer history
	if m.showHistory && len(m.historyItems) > 0 {
		sections = append(sections, m.styles.Label.Render("Recent Transfers (press 1-5 to select):"))

		maxItems := 5
		if len(m.historyItems) < maxItems {
			maxItems = len(m.historyItems)
		}

		for i := 0; i < maxItems; i++ {
			item := m.historyItems[i]
			arrow := "↑"
			if item.Direction == "download" {
				arrow = "↓"
			}
			timeAgo := formatTimeAgo(item.Timestamp)

			historyLine := fmt.Sprintf(" %d. %s %s → %s (%s)",
				i+1, arrow,
				truncatePath(item.LocalPath, 25),
				truncatePath(item.RemotePath, 25),
				timeAgo)

			if i == m.historyIndex {
				historyLine = m.styles.Selected.Render(historyLine)
			} else {
				historyLine = m.styles.HelpText.Render(historyLine)
			}
			sections = append(sections, historyLine)
		}
		sections = append(sections, "")
	}

	// Help text
	helpText := " Tab/↓: next • Shift+Tab/↑: prev • Enter: transfer • Ctrl+H: toggle history • Esc: cancel"
	sections = append(sections, m.styles.HelpText.Render(helpText))

	// Join all sections
	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Center the form
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		m.styles.FormContainer.Render(content),
	)
}

func (m *transferFormModel) submitForm() tea.Cmd {
	return func() tea.Msg {
		localPath := strings.TrimSpace(m.inputs[tfLocalPathInput].Value())
		remotePath := strings.TrimSpace(m.inputs[tfRemotePathInput].Value())

		// Validate inputs based on direction
		if m.direction == transfer.Upload {
			if localPath == "" {
				return transferSubmitMsg{err: fmt.Errorf("local path is required for upload")}
			}
			// Expand and validate local path
			expandedPath, err := transfer.ExpandPath(localPath)
			if err != nil {
				return transferSubmitMsg{err: fmt.Errorf("invalid local path: %w", err)}
			}
			if err := transfer.ValidateLocalPath(expandedPath, transfer.Upload); err != nil {
				return transferSubmitMsg{err: err}
			}
			localPath = expandedPath

			if remotePath == "" {
				remotePath = "~/"
			}
		} else {
			if remotePath == "" {
				return transferSubmitMsg{err: fmt.Errorf("remote path is required for download")}
			}
			if localPath == "" {
				localPath = "./"
			}
			// Expand local path
			expandedPath, err := transfer.ExpandPath(localPath)
			if err != nil {
				return transferSubmitMsg{err: fmt.Errorf("invalid local path: %w", err)}
			}
			localPath = expandedPath
		}

		// Check if local path is a directory for uploads
		recursive := false
		if m.direction == transfer.Upload {
			info, err := os.Stat(localPath)
			if err == nil && info.IsDir() {
				recursive = true
			}
		}

		req := &transfer.TransferRequest{
			Host:       m.hostName,
			Direction:  m.direction,
			LocalPath:  localPath,
			RemotePath: remotePath,
			Recursive:  recursive,
			ConfigFile: m.configFile,
		}

		return transferSubmitMsg{err: nil, request: req}
	}
}

// truncatePath truncates a path to fit in maxLen characters
func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	// Keep the last part of the path
	return "..." + path[len(path)-maxLen+3:]
}

// formatTimeAgo formats a time as "X ago" (already exists in tui.go, but we need it here too)
func formatTransferTimeAgo(t time.Time) string {
	duration := time.Since(t)
	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2")
	}
}

// Standalone wrapper for transfer form
type standaloneTransferForm struct {
	*transferFormModel
}

func (m standaloneTransferForm) Init() tea.Cmd {
	return m.transferFormModel.Init()
}

func (m standaloneTransferForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.transferFormModel.width = msg.Width
		m.transferFormModel.height = msg.Height
		m.transferFormModel.styles = NewStyles(msg.Width)
		return m, nil

	case transferSubmitMsg:
		if msg.err != nil {
			m.transferFormModel.err = msg.err.Error()
			return m, nil
		}
		// Execute the transfer
		if msg.request != nil {
			fmt.Printf("\nTransferring %s...\n", msg.request.LocalPath)
			result := msg.request.ExecuteWithProgress()
			if !result.Success {
				m.transferFormModel.err = result.Error.Error()
				return m, nil
			}

			// Record in history
			if m.transferFormModel.historyManager != nil {
				direction := "upload"
				if msg.request.Direction == transfer.Download {
					direction = "download"
				}
				_ = m.transferFormModel.historyManager.RecordTransfer(
					m.transferFormModel.hostName,
					direction,
					msg.request.LocalPath,
					msg.request.RemotePath,
				)
			}

			fmt.Println("Transfer complete!")
		}
		return m, tea.Quit

	case transferCancelMsg:
		return m, tea.Quit
	}

	newForm, cmd := m.transferFormModel.Update(msg)
	m.transferFormModel = newForm
	return m, cmd
}

func (m standaloneTransferForm) View() string {
	return m.transferFormModel.View()
}

// RunTransferForm runs the transfer form as a standalone TUI
func RunTransferForm(hostName, configFile string) error {
	styles := NewStyles(80)
	form := NewTransferForm(hostName, styles, 80, 24, configFile, transfer.Upload)
	m := standaloneTransferForm{form}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// RunTransferFormWithDirection runs the transfer form with a preset direction
func RunTransferFormWithDirection(hostName, configFile string, direction transfer.Direction) error {
	styles := NewStyles(80)
	form := NewTransferForm(hostName, styles, 80, 24, configFile, direction)
	m := standaloneTransferForm{form}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// GetLocalFilesForBrowser returns files in a directory for the file browser
func GetLocalFilesForBrowser(path string) ([]transfer.FileInfo, error) {
	expandedPath, err := transfer.ExpandPath(path)
	if err != nil {
		return nil, err
	}

	// If path is a file, get its directory
	info, err := os.Stat(expandedPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		expandedPath = filepath.Dir(expandedPath)
	}

	return transfer.GetLocalFiles(expandedPath)
}
