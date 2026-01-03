package ui

import (
	"github.com/xvertile/sshc/internal/config"
	"github.com/xvertile/sshc/internal/connectivity"
	"github.com/xvertile/sshc/internal/history"
	"github.com/xvertile/sshc/internal/version"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// SortMode defines the available sorting modes
type SortMode int

const (
	SortByName SortMode = iota
	SortByLastUsed
)

func (s SortMode) String() string {
	switch s {
	case SortByName:
		return "Name (A-Z)"
	case SortByLastUsed:
		return "Last Login"
	default:
		return "Name (A-Z)"
	}
}

// ViewMode defines the current view state
type ViewMode int

const (
	ViewList ViewMode = iota
	ViewAdd
	ViewEdit
	ViewMove
	ViewInfo
	ViewPortForward
	ViewTransfer
	ViewQuickTransfer
	ViewRemoteBrowser
	ViewHelp
	ViewFileSelector
	ViewK8sAdd
	ViewK8sEdit
	ViewTheme
	ViewConnectionError
	ViewSSHKeyUpload
)

// PortForwardType defines the type of port forwarding
type PortForwardType int

const (
	LocalForward PortForwardType = iota
	RemoteForward
	DynamicForward
)

func (p PortForwardType) String() string {
	switch p {
	case LocalForward:
		return "Local (-L)"
	case RemoteForward:
		return "Remote (-R)"
	case DynamicForward:
		return "Dynamic (-D)"
	default:
		return "Local (-L)"
	}
}

// HostEntry represents a unified host entry that can be either SSH or K8s
type HostEntry struct {
	Name     string
	IsK8s    bool
	SSHHost  *config.SSHHost
	K8sHost  *config.K8sHost
	Tags     []string
	Hostname string // For display: SSH hostname or K8s namespace/pod
}

// Model represents the state of the user interface
type Model struct {
	table          table.Model
	searchInput    textinput.Model
	hosts          []config.SSHHost
	filteredHosts  []config.SSHHost
	searchMode     bool
	deleteMode     bool
	deleteHost     string
	deleteHostIsK8s bool // Track if delete target is a k8s host
	historyManager *history.HistoryManager
	pingManager    *connectivity.PingManager
	sortMode       SortMode
	configFile     string // Path to the SSH config file

	// Kubernetes hosts
	k8sHosts         []config.K8sHost
	filteredK8sHosts []config.K8sHost

	// Unified host entries for display
	allEntries      []HostEntry
	filteredEntries []HostEntry

	// Application configuration
	appConfig      *config.AppConfig

	// Version update information
	updateInfo     *version.UpdateInfo
	currentVersion string

	// View management
	viewMode          ViewMode
	addForm           *addFormModel
	editForm          *editFormModel
	moveForm          *moveFormModel
	infoForm          *infoFormModel
	portForwardForm   *portForwardModel
	transferForm      *transferFormModel
	quickTransferForm *quickTransferModel
	remoteBrowserForm *remoteBrowserModel
	helpForm          *helpModel
	fileSelectorForm  *fileSelectorModel
	k8sAddForm        *k8sAddFormModel
	k8sEditForm       *k8sEditFormModel
	themePicker       *themePickerModel
	sshKeyUploadForm  *sshKeyUploadModel

	// Terminal size and styles
	width  int
	height int
	styles Styles
	ready  bool

	// Error handling
	errorMessage string
	showingError bool

	// Connection retry state
	connectionHost  string // Host being connected to
	connectionIsK8s bool   // Whether it's a k8s host
	connectionError string // Last connection error
}

// updateTableStyles updates the table header border color based on focus state
func (m *Model) updateTableStyles() {
	s := table.DefaultStyles()
	s.Selected = m.styles.Selected

	if m.searchMode {
		// When in search mode, use secondary color for table header
		s.Header = s.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(SecondaryColor)).
			BorderBottom(true).
			Bold(false)
	} else {
		// When table is focused, use primary color for table header
		s.Header = s.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(PrimaryColor)).
			BorderBottom(true).
			Bold(false)
	}

	m.table.SetStyles(s)
}
