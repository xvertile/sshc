package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xvertile/sshc/internal/history"
	"github.com/xvertile/sshc/internal/transfer"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// QuickTransferState represents the current state of the quick transfer flow
type QuickTransferState int

const (
	QTStateChooseDirection    QuickTransferState = iota
	QTStateChooseUploadType                      // File or Folder selection (only for uploads)
	QTStateChooseDownloadType                    // File or Folder selection (for downloads)
	QTStateSelectingLocal
	QTStateSelectingRemote
	QTStateTransferring
	QTStateError // New state for error with retry option
	QTStateDone
)

// quickTransferModel is a streamlined transfer UI
type quickTransferModel struct {
	state           QuickTransferState
	direction       transfer.Direction
	uploadType      UploadType // File or Folder (reuse from transfer_form.go)
	downloadType    UploadType // File or Folder for downloads (reuses UploadType enum)
	selectedIdx     int        // 0 = upload/file, 1 = download/folder (for arrow key nav)
	hostName        string
	configFile      string
	localPath       string
	remotePath      string
	styles          Styles
	width           int
	height          int
	err             string
	historyManager  *history.HistoryManager
	runningTransfer *transfer.RunningTransfer // For cancellation
	retryCount      int                       // Number of retry attempts
}

// quickTransferDoneMsg signals transfer complete
type quickTransferDoneMsg struct {
	success bool
	err     error
}

// quickTransferCancelMsg signals cancellation
type quickTransferCancelMsg struct{}


// quickLocalPickedMsg is sent when local file is picked
type quickLocalPickedMsg struct {
	path     string
	selected bool
}

// quickRemotePickedMsg is sent when remote file is picked
type quickRemotePickedMsg struct {
	path     string
	selected bool
}

// openRemoteBrowserMsg requests the main app to open the remote browser
type openRemoteBrowserMsg struct {
	host       string
	startPath  string
	configFile string
	mode       BrowserMode
}

// NewQuickTransfer creates a new quick transfer model
func NewQuickTransfer(hostName string, styles Styles, width, height int, configFile string) *quickTransferModel {
	historyManager, _ := history.NewHistoryManager()
	return &quickTransferModel{
		state:          QTStateChooseDirection,
		hostName:       hostName,
		configFile:     configFile,
		styles:         styles,
		width:          width,
		height:         height,
		historyManager: historyManager,
	}
}

func (m *quickTransferModel) Init() tea.Cmd {
	return nil
}

func (m *quickTransferModel) Update(msg tea.Msg) (*quickTransferModel, tea.Cmd) {
	switch msg := msg.(type) {
	case quickLocalPickedMsg:
		if !msg.selected {
			// Cancelled - go back or exit
			return m, func() tea.Msg { return quickTransferCancelMsg{} }
		}
		m.localPath = msg.path

		if m.direction == transfer.Download {
			// For downloads: both paths set (remote first, then local), execute transfer
			m.state = QTStateTransferring
			return m, m.executeTransfer()
		}
		// For uploads: local picked, now ask for remote destination
		m.state = QTStateSelectingRemote
		return m, m.openRemotePicker()

	case quickRemotePickedMsg:
		if !msg.selected {
			// Cancelled - go back or exit
			return m, func() tea.Msg { return quickTransferCancelMsg{} }
		}
		m.remotePath = msg.path

		if m.direction == transfer.Download {
			// For downloads: remote picked, now ask for local destination
			m.state = QTStateSelectingLocal
			return m, m.openLocalPicker()
		}
		// For uploads: both paths set, execute transfer
		m.state = QTStateTransferring
		return m, m.executeTransfer()

	case quickTransferDoneMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.state = QTStateError // Go to error state for retry option
			return m, nil
		}
		m.state = QTStateDone
		return m, func() tea.Msg { return quickTransferCancelMsg{} }

	case tea.KeyMsg:
		// Global cancel with ctrl+c from any state
		if msg.Type == tea.KeyCtrlC {
			// Cancel running transfer if any
			if m.runningTransfer != nil {
				m.runningTransfer.Cancel()
			}
			return m, func() tea.Msg { return quickTransferCancelMsg{} }
		}

		switch m.state {
		case QTStateChooseDirection:
			// Handle escape to exit
			if msg.Type == tea.KeyEsc {
				return m, func() tea.Msg { return quickTransferCancelMsg{} }
			}
			switch msg.String() {
			case "u", "U", "1":
				m.direction = transfer.Upload
				m.selectedIdx = 0 // Reset for upload type selection
				m.state = QTStateChooseUploadType
				return m, nil
			case "d", "D", "2":
				m.direction = transfer.Download
				m.selectedIdx = 0 // Reset for download type selection
				m.state = QTStateChooseDownloadType
				return m, nil
			case "left", "h", "up", "k":
				m.selectedIdx = 0 // Upload
				return m, nil
			case "right", "l", "down", "j":
				m.selectedIdx = 1 // Download
				return m, nil
			case "tab":
				m.selectedIdx = (m.selectedIdx + 1) % 2
				return m, nil
			case "enter", " ":
				if m.selectedIdx == 0 {
					m.direction = transfer.Upload
					m.selectedIdx = 0 // Reset for upload type selection
					m.state = QTStateChooseUploadType
					return m, nil
				} else {
					m.direction = transfer.Download
					m.selectedIdx = 0 // Reset for download type selection
					m.state = QTStateChooseDownloadType
					return m, nil
				}
			case "q":
				return m, func() tea.Msg { return quickTransferCancelMsg{} }
			}

		case QTStateChooseUploadType:
			// Handle escape to go back
			if msg.Type == tea.KeyEsc {
				m.state = QTStateChooseDirection
				m.selectedIdx = 0
				return m, nil
			}
			switch msg.String() {
			case "f", "F", "1":
				m.uploadType = UploadFile
				m.state = QTStateSelectingLocal
				return m, m.openLocalPicker()
			case "d", "D", "2":
				m.uploadType = UploadFolder
				m.state = QTStateSelectingLocal
				return m, m.openLocalPicker()
			case "left", "h", "up", "k":
				m.selectedIdx = 0 // File
				return m, nil
			case "right", "l", "down", "j":
				m.selectedIdx = 1 // Folder
				return m, nil
			case "tab":
				m.selectedIdx = (m.selectedIdx + 1) % 2
				return m, nil
			case "enter", " ":
				if m.selectedIdx == 0 {
					m.uploadType = UploadFile
				} else {
					m.uploadType = UploadFolder
				}
				m.state = QTStateSelectingLocal
				return m, m.openLocalPicker()
			case "q":
				// Go back to direction selection
				m.state = QTStateChooseDirection
				m.selectedIdx = 0
				return m, nil
			}

		case QTStateChooseDownloadType:
			// Handle escape to go back
			if msg.Type == tea.KeyEsc {
				m.state = QTStateChooseDirection
				m.selectedIdx = 1 // Keep download selected
				return m, nil
			}
			switch msg.String() {
			case "f", "F", "1":
				m.downloadType = UploadFile
				m.state = QTStateSelectingRemote
				return m, m.openRemotePicker()
			case "d", "D", "2":
				m.downloadType = UploadFolder
				m.state = QTStateSelectingRemote
				return m, m.openRemotePicker()
			case "left", "h", "up", "k":
				m.selectedIdx = 0 // File
				return m, nil
			case "right", "l", "down", "j":
				m.selectedIdx = 1 // Folder
				return m, nil
			case "tab":
				m.selectedIdx = (m.selectedIdx + 1) % 2
				return m, nil
			case "enter", " ":
				if m.selectedIdx == 0 {
					m.downloadType = UploadFile
				} else {
					m.downloadType = UploadFolder
				}
				m.state = QTStateSelectingRemote
				return m, m.openRemotePicker()
			case "q":
				// Go back to direction selection
				m.state = QTStateChooseDirection
				m.selectedIdx = 1 // Keep download selected
				return m, nil
			}

		case QTStateSelectingLocal, QTStateSelectingRemote:
			// While file picker is open, allow cancel
			if msg.Type == tea.KeyEsc {
				return m, func() tea.Msg { return quickTransferCancelMsg{} }
			}
			if msg.String() == "q" {
				return m, func() tea.Msg { return quickTransferCancelMsg{} }
			}

		case QTStateTransferring:
			// Transfer in progress - handled at top with ctrl+c
			break

		case QTStateError:
			// Error state - allow retry or exit
			switch msg.String() {
			case "r", "R":
				// Retry the transfer
				m.err = ""
				m.retryCount++
				m.state = QTStateTransferring
				return m, m.executeTransfer()
			case "q", "esc":
				return m, func() tea.Msg { return quickTransferCancelMsg{} }
			}

		case QTStateDone:
			// Any key exits
			return m, func() tea.Msg { return quickTransferCancelMsg{} }
		}
	}

	return m, nil
}

func (m *quickTransferModel) openLocalPicker() tea.Cmd {
	return func() tea.Msg {
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

		startDir, _ := os.Getwd()
		result, err := transfer.OpenFilePicker(mode, title, startDir)
		if err != nil || result == nil || !result.Selected {
			return quickLocalPickedMsg{selected: false}
		}

		return quickLocalPickedMsg{path: result.Path, selected: true}
	}
}

func (m *quickTransferModel) openRemotePicker() tea.Cmd {
	// Send a message to the main app to open the remote browser
	// This avoids nested tea.Program issues
	var mode BrowserMode
	if m.direction == transfer.Upload {
		mode = BrowseDirectories
	} else {
		// Download: use directories mode for folder downloads, files mode for file downloads
		if m.downloadType == UploadFolder {
			mode = BrowseDirectories
		} else {
			mode = BrowseFiles
		}
	}

	return func() tea.Msg {
		return openRemoteBrowserMsg{
			host:       m.hostName,
			startPath:  "~",
			configFile: m.configFile,
			mode:       mode,
		}
	}
}

func (m *quickTransferModel) executeTransfer() tea.Cmd {
	localPath := m.localPath
	recursive := false

	if m.direction == transfer.Upload {
		// Check if uploading a directory
		info, err := os.Stat(localPath)
		if err == nil && info.IsDir() {
			recursive = true
		}
	} else {
		// Download: if local path is a directory, append the remote filename/foldername
		info, err := os.Stat(localPath)
		if err == nil && info.IsDir() {
			remoteFilename := filepath.Base(m.remotePath)
			localPath = filepath.Join(localPath, remoteFilename)
		}
		// Set recursive for folder downloads
		if m.downloadType == UploadFolder {
			recursive = true
		}
	}

	req := &transfer.TransferRequest{
		Host:       m.hostName,
		Direction:  m.direction,
		LocalPath:  localPath,
		RemotePath: m.remotePath,
		Recursive:  recursive,
		ConfigFile: m.configFile,
	}

	// Start the transfer (non-blocking)
	m.runningTransfer = req.StartTransfer()

	// Return a command that waits for the transfer to complete
	return func() tea.Msg {
		result := <-m.runningTransfer.Done()
		if !result.Success {
			return quickTransferDoneMsg{success: false, err: result.Error}
		}

		// Record in history
		if m.historyManager != nil {
			direction := "upload"
			if m.direction == transfer.Download {
				direction = "download"
			}
			_ = m.historyManager.RecordTransfer(m.hostName, direction, m.localPath, m.remotePath)
		}

		return quickTransferDoneMsg{success: true}
	}
}

func (m *quickTransferModel) View() string {
	theme := GetCurrentTheme()
	var sections []string

	// Title
	title := m.styles.Header.Render("Quick Transfer")
	sections = append(sections, title)
	sections = append(sections, m.styles.HelpText.Render(fmt.Sprintf("Host: %s", m.hostName)))
	sections = append(sections, "")

	switch m.state {
	case QTStateChooseDirection:
		sections = append(sections, m.styles.Label.Render("What would you like to do?"))
		sections = append(sections, "")

		var uploadBtn, downloadBtn string
		if m.selectedIdx == 0 {
			uploadBtn = m.styles.ActiveTab.Render("  ↑ Upload  ")
			downloadBtn = m.styles.InactiveTab.Render("  ↓ Download  ")
		} else {
			uploadBtn = m.styles.InactiveTab.Render("  ↑ Upload  ")
			downloadBtn = m.styles.ActiveTab.Render("  ↓ Download  ")
		}
		buttons := lipgloss.JoinHorizontal(lipgloss.Center, uploadBtn, "    ", downloadBtn)
		sections = append(sections, buttons)
		sections = append(sections, "")
		sections = append(sections, m.styles.HelpText.Render("←/→ or Tab: switch • Enter: confirm • Esc: cancel"))

	case QTStateChooseUploadType:
		sections = append(sections, m.styles.Label.Render("What do you want to upload?"))
		sections = append(sections, "")

		var fileBtn, folderBtn string
		if m.selectedIdx == 0 {
			fileBtn = m.styles.ActiveTab.Render("  File  ")
			folderBtn = m.styles.InactiveTab.Render("  Folder  ")
		} else {
			fileBtn = m.styles.InactiveTab.Render("  File  ")
			folderBtn = m.styles.ActiveTab.Render("  Folder  ")
		}
		buttons := lipgloss.JoinHorizontal(lipgloss.Center, fileBtn, "    ", folderBtn)
		sections = append(sections, buttons)
		sections = append(sections, "")
		sections = append(sections, m.styles.HelpText.Render("←/→ or Tab: switch • Enter: confirm • Esc: back"))

	case QTStateChooseDownloadType:
		sections = append(sections, m.styles.Label.Render("What do you want to download?"))
		sections = append(sections, "")

		var fileBtn, folderBtn string
		if m.selectedIdx == 0 {
			fileBtn = m.styles.ActiveTab.Render("  File  ")
			folderBtn = m.styles.InactiveTab.Render("  Folder  ")
		} else {
			fileBtn = m.styles.InactiveTab.Render("  File  ")
			folderBtn = m.styles.ActiveTab.Render("  Folder  ")
		}
		buttons := lipgloss.JoinHorizontal(lipgloss.Center, fileBtn, "    ", folderBtn)
		sections = append(sections, buttons)
		sections = append(sections, "")
		sections = append(sections, m.styles.HelpText.Render("←/→ or Tab: switch • Enter: confirm • Esc: back"))

	case QTStateSelectingLocal:
		if m.direction == transfer.Upload {
			if m.uploadType == UploadFolder {
				sections = append(sections, m.styles.Label.Render("Select folder to upload..."))
			} else {
				sections = append(sections, m.styles.Label.Render("Select file to upload..."))
			}
		} else {
			sections = append(sections, m.styles.Label.Render("Select download destination..."))
		}
		sections = append(sections, "")
		loadingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
		sections = append(sections, loadingStyle.Render("Opening file picker..."))

	case QTStateSelectingRemote:
		if m.direction == transfer.Upload {
			sections = append(sections, m.styles.Label.Render("Select remote destination..."))
		} else {
			if m.downloadType == UploadFolder {
				sections = append(sections, m.styles.Label.Render("Select remote folder to download..."))
			} else {
				sections = append(sections, m.styles.Label.Render("Select remote file to download..."))
			}
		}
		sections = append(sections, "")
		if m.localPath != "" {
			sections = append(sections, m.styles.HelpText.Render("Local: "+m.localPath))
			sections = append(sections, "")
		}
		loadingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
		sections = append(sections, loadingStyle.Render("Opening remote browser..."))

	case QTStateTransferring:
		direction := "Uploading"
		icon := "↑"
		if m.direction == transfer.Download {
			icon = "↓"
			if m.downloadType == UploadFolder {
				direction = "Downloading folder"
			} else {
				direction = "Downloading"
			}
		}
		transferStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.Primary))
		sections = append(sections, transferStyle.Render(fmt.Sprintf("%s %s...", icon, direction)))
		sections = append(sections, "")
		sections = append(sections, m.styles.HelpText.Render("From: "+m.localPath))
		sections = append(sections, m.styles.HelpText.Render("  To: "+m.remotePath))

	case QTStateError:
		// Error state with retry option
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		sections = append(sections, errorStyle.Render("✗ Transfer Failed"))
		sections = append(sections, "")
		sections = append(sections, m.styles.HelpText.Render("From: "+m.localPath))
		sections = append(sections, m.styles.HelpText.Render("  To: "+m.remotePath))
		sections = append(sections, "")
		sections = append(sections, errorStyle.Render(m.err))
		if m.retryCount > 0 {
			sections = append(sections, "")
			sections = append(sections, m.styles.HelpText.Render(fmt.Sprintf("Retry attempts: %d", m.retryCount)))
		}
		sections = append(sections, "")
		retryStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.Primary))
		sections = append(sections, retryStyle.Render("Press 'r' to retry • Esc to cancel"))

	case QTStateDone:
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
		sections = append(sections, successStyle.Render("✓ Transfer Complete!"))
		sections = append(sections, "")
		sections = append(sections, m.styles.HelpText.Render("From: "+m.localPath))
		sections = append(sections, m.styles.HelpText.Render("  To: "+m.remotePath))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Container with primary color border
	container := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Primary)).
		Padding(1, 2).
		Render(content)

	// Logo outside the container
	logo := m.styles.Header.Render(asciiTitle)

	// Stack logo and container
	fullContent := lipgloss.JoinVertical(lipgloss.Center, logo, "", container)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		fullContent,
	)
}

// Standalone wrapper
type standaloneQuickTransfer struct {
	*quickTransferModel
}

func (m standaloneQuickTransfer) Init() tea.Cmd {
	return m.quickTransferModel.Init()
}

func (m standaloneQuickTransfer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.quickTransferModel.width = msg.Width
		m.quickTransferModel.height = msg.Height
		m.quickTransferModel.styles = NewStyles(msg.Width)
		return m, nil

	case quickTransferCancelMsg:
		return m, tea.Quit

	case openRemoteBrowserMsg:
		// Standalone mode: launch remote browser as external program
		return m, func() tea.Msg {
			path, selected, err := RunRemoteBrowser(msg.host, msg.startPath, msg.configFile, msg.mode)
			if err != nil || !selected {
				return quickRemotePickedMsg{selected: false}
			}
			return quickRemotePickedMsg{path: path, selected: true}
		}
	}

	newModel, cmd := m.quickTransferModel.Update(msg)
	m.quickTransferModel = newModel
	return m, cmd
}

func (m standaloneQuickTransfer) View() string {
	return m.quickTransferModel.View()
}

// RunQuickTransfer runs the quick transfer UI
func RunQuickTransfer(hostName, configFile string) error {
	styles := NewStyles(80)
	qt := NewQuickTransfer(hostName, styles, 80, 24, configFile)
	m := standaloneQuickTransfer{qt}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
