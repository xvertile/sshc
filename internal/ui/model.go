package ui

import (
	"github.com/xvertile/sshc/internal/config"
	"github.com/xvertile/sshc/internal/connectivity"
	"github.com/xvertile/sshc/internal/history"
	"github.com/xvertile/sshc/internal/version"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// devVersion is the version an unreleased build reports: cmd.AppVersion
// defaults to it, and a real version is injected at link time.
const devVersion = "dev"

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

// short returns the one-word form used in the status line.
func (s SortMode) short() string {
	if s == SortByLastUsed {
		return "recent"
	}
	return "name"
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
	ViewPortForwardSession
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
	table           hostTable
	searchInput     textinput.Model
	hosts           []config.SSHHost
	searchMode      bool
	deleteMode      bool
	deleteHost      string
	deleteHostIsK8s bool // Track if delete target is a k8s host
	historyManager  *history.HistoryManager
	pingManager     *connectivity.PingManager
	sortMode        SortMode
	configFile      string // Path to the SSH config file

	// Kubernetes hosts
	k8sHosts []config.K8sHost

	// Unified host entries for display
	allEntries      []HostEntry
	filteredEntries []HostEntry

	// tagPositions gives each distinct tag its slot on the colour wheel.
	tagPositions map[string]int

	// Application configuration
	appConfig *config.AppConfig

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
	portForwardActive *portForwardSessionModel
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

// applyTheme re-renders everything that caches colours from the active theme.
//
// Rows are pre-styled strings, so a theme change is invisible until they are
// rebuilt. Refreshing the styles alone left the previous theme's colours on
// screen until some unrelated action — moving the cursor — happened to
// rebuild the rows.
func (m *Model) applyTheme() {
	m.styles = NewStyles(m.width)
	m.updateTableStyles()
	m.updateTableRows()
}

// updateTableStyles applies the styling hostTable needs. Row colours are
// applied per cell in updateTableRows, so only the header and the empty-state
// message are set here.
func (m *Model) updateTableStyles() {
	theme := GetCurrentTheme()

	m.table.headerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Secondary)).
		Bold(true)

	m.table.emptyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Muted)).
		Italic(true)
}
