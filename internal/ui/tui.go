package ui

import (
	"fmt"
	"time"

	"github.com/xvertile/sshc/internal/config"
	"github.com/xvertile/sshc/internal/connectivity"
	"github.com/xvertile/sshc/internal/history"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// NewModel creates a new TUI model with the given SSH hosts
func NewModel(hosts []config.SSHHost, configFile, currentVersion string) Model {
	// Load application configuration
	appConfig, err := config.LoadAppConfig()
	if err != nil {
		// Log the error but continue with default configuration
		fmt.Printf("Warning: Could not load application config: %v, using defaults\n", err)
		defaultConfig := config.GetDefaultAppConfig()
		appConfig = &defaultConfig
	}

	// Apply saved theme
	if appConfig.Theme != "" {
		SetThemeByName(appConfig.Theme)
	}

	// Initialize the history manager
	historyManager, err := history.NewHistoryManager()
	if err != nil {
		// Log the error but continue without the history functionality
		fmt.Printf("Warning: Could not initialize history manager: %v\n", err)
		historyManager = nil
	}

	// Load k8s hosts if config exists (feature is off by default)
	var k8sHosts []config.K8sHost
	if config.K8sConfigExists() {
		k8sHosts, err = config.ParseK8sConfig()
		if err != nil {
			// Log the error but continue without k8s hosts
			fmt.Printf("Warning: Could not load k8s config: %v\n", err)
			k8sHosts = []config.K8sHost{}
		}
	}

	// Determine sort mode from config
	sortMode := SortByName
	if appConfig != nil && appConfig.SortMode == "recent" {
		sortMode = SortByLastUsed
	}

	// The filter line renders its own label, so the input carries no prompt.
	searchInput := textinput.New()
	searchInput.Placeholder = "name, hostname or #tag"
	searchInput.Prompt = ""
	searchInput.CharLimit = 50
	searchInput.Width = 40

	m := Model{
		hosts:          hosts,
		k8sHosts:       k8sHosts,
		historyManager: historyManager,
		pingManager:    connectivity.NewPingManager(5 * time.Second),
		sortMode:       sortMode,
		configFile:     configFile,
		currentVersion: currentVersion,
		appConfig:      appConfig,
		searchInput:    searchInput,
		styles:         NewStyles(80),
		width:          80,
		height:         24,
		ready:          false,
		viewMode:       ViewList,
	}

	// Build the unified SSH + K8s entry list, sorted by the active sort mode.
	m.rebuildEntries()

	// Columns are sized on the first WindowSizeMsg, once the real terminal
	// width is known; the titles are correct from the start.
	m.table.Focus()
	m.table.SetHeight(10)
	m.updateTableStyles()

	m.updateTableRows()

	// Start in search mode if configured
	if appConfig != nil && appConfig.StartInSearchMode {
		m.searchMode = true
		m.searchInput.Focus()
		m.table.Blur()
	}

	m.updateTableStyles()

	return m
}

// RunInteractiveMode starts the interactive TUI interface
func RunInteractiveMode(hosts []config.SSHHost, configFile, currentVersion string) error {
	m := NewModel(hosts, configFile, currentVersion)

	// Start the application in alt screen mode for clean output
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}
