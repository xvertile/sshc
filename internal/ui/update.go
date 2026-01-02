package ui

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/xvertile/sshc/internal/config"
	"github.com/xvertile/sshc/internal/connectivity"
	"github.com/xvertile/sshc/internal/transfer"
	"github.com/xvertile/sshc/internal/version"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Messages for SSH ping functionality and version checking
type (
	pingResultMsg   *connectivity.HostPingResult
	versionCheckMsg *version.UpdateInfo
	versionErrorMsg error
	errorMsg        string
)

// startPingAllCmd creates a command to ping all hosts concurrently
func (m Model) startPingAllCmd() tea.Cmd {
	if m.pingManager == nil {
		return nil
	}

	return tea.Batch(
		// Create individual ping commands for each host
		func() tea.Cmd {
			var cmds []tea.Cmd
			for _, host := range m.hosts {
				cmds = append(cmds, pingSingleHostCmd(m.pingManager, host))
			}
			return tea.Batch(cmds...)
		}(),
	)
}

// listenForPingResultsCmd is no longer needed since we use individual ping commands

// pingSingleHostCmd creates a command to ping a single host
func pingSingleHostCmd(pingManager *connectivity.PingManager, host config.SSHHost) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		result := pingManager.PingHost(ctx, host)
		return pingResultMsg(result)
	}
}

// checkVersionCmd creates a command to check for version updates
func checkVersionCmd(currentVersion string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		updateInfo, err := version.CheckForUpdates(ctx, currentVersion)
		if err != nil {
			return versionErrorMsg(err)
		}
		return versionCheckMsg(updateInfo)
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd

	// Basic initialization commands
	cmds = append(cmds, textinput.Blink)

	// Check for version updates if we have a current version
	if m.currentVersion != "" {
		cmds = append(cmds, checkVersionCmd(m.currentVersion))
	}

	return tea.Batch(cmds...)
}

// Update handles model updates
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle different message types
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update terminal size and recalculate styles
		m.width = msg.Width
		m.height = msg.Height
		m.styles = NewStyles(m.width)
		m.ready = true

		// Update table height and columns based on new window size
		m.updateTableHeight()
		m.updateTableColumns()

		// Update sub-forms if they exist
		if m.addForm != nil {
			m.addForm.width = m.width
			m.addForm.height = m.height
			m.addForm.styles = m.styles
		}
		if m.editForm != nil {
			m.editForm.width = m.width
			m.editForm.height = m.height
			m.editForm.styles = m.styles
		}
		if m.moveForm != nil {
			m.moveForm.width = m.width
			m.moveForm.height = m.height
			m.moveForm.styles = m.styles
		}
		if m.infoForm != nil {
			m.infoForm.width = m.width
			m.infoForm.height = m.height
			m.infoForm.styles = m.styles
		}
		if m.portForwardForm != nil {
			m.portForwardForm.width = m.width
			m.portForwardForm.height = m.height
			m.portForwardForm.styles = m.styles
		}
		if m.transferForm != nil {
			m.transferForm.width = m.width
			m.transferForm.height = m.height
			m.transferForm.styles = m.styles
		}
		if m.quickTransferForm != nil {
			m.quickTransferForm.width = m.width
			m.quickTransferForm.height = m.height
			m.quickTransferForm.styles = m.styles
		}
		if m.helpForm != nil {
			m.helpForm.width = m.width
			m.helpForm.height = m.height
			m.helpForm.styles = m.styles
		}
		if m.fileSelectorForm != nil {
			m.fileSelectorForm.width = m.width
			m.fileSelectorForm.height = m.height
			m.fileSelectorForm.styles = m.styles
		}
		return m, nil

	case pingResultMsg:
		// Handle ping result - update table display
		if msg != nil {
			// Update the table to reflect the new ping status
			m.updateTableRows()
		}
		return m, nil

	case versionCheckMsg:
		// Handle version check result
		if msg != nil {
			m.updateInfo = msg
		}
		return m, nil

	case versionErrorMsg:
		// Handle version check error (silently - not critical)
		// We don't want to show error messages for version checks
		// as it might disrupt the user experience
		return m, nil

	case errorMsg:
		// Handle general error messages
		if string(msg) == "clear" {
			m.showingError = false
			m.errorMessage = ""
		}
		return m, nil

	case addFormSubmitMsg:
		if msg.err != nil {
			// Show error in form
			if m.addForm != nil {
				m.addForm.err = msg.err.Error()
			}
			return m, nil
		} else {
			// Success: refresh hosts and return to list view
			var hosts []config.SSHHost
			var err error

			if m.configFile != "" {
				hosts, err = config.ParseSSHConfigFile(m.configFile)
			} else {
				hosts, err = config.ParseSSHConfig()
			}

			if err != nil {
				return m, tea.Quit
			}
			m.hosts = m.sortHosts(hosts)

			// Reapply search filter if there is one active
			if m.searchInput.Value() != "" {
				m.filteredHosts = m.filterHosts(m.searchInput.Value())
			} else {
				m.filteredHosts = m.hosts
			}

			m.updateTableRows()
			m.viewMode = ViewList
			m.addForm = nil
			m.table.Focus()
			return m, nil
		}

	case addFormCancelMsg:
		// Cancel: return to list view
		m.viewMode = ViewList
		m.addForm = nil
		m.table.Focus()
		return m, nil

	case editFormSubmitMsg:
		if msg.err != nil {
			// Show error in form
			if m.editForm != nil {
				m.editForm.err = msg.err.Error()
			}
			return m, nil
		} else {
			// Success: refresh hosts and return to list view
			var hosts []config.SSHHost
			var err error

			if m.configFile != "" {
				hosts, err = config.ParseSSHConfigFile(m.configFile)
			} else {
				hosts, err = config.ParseSSHConfig()
			}

			if err != nil {
				return m, tea.Quit
			}
			m.hosts = m.sortHosts(hosts)

			// Reapply search filter if there is one active
			if m.searchInput.Value() != "" {
				m.filteredHosts = m.filterHosts(m.searchInput.Value())
			} else {
				m.filteredHosts = m.hosts
			}

			m.updateTableRows()
			m.viewMode = ViewList
			m.editForm = nil
			m.table.Focus()
			return m, nil
		}

	case editFormCancelMsg:
		// Cancel: return to list view
		m.viewMode = ViewList
		m.editForm = nil
		m.table.Focus()
		return m, nil

	case moveFormSubmitMsg:
		if msg.err != nil {
			// En cas d'erreur, on pourrait afficher une notification ou retourner à la liste
			// Pour l'instant, on retourne simplement à la liste
			m.viewMode = ViewList
			m.moveForm = nil
			m.table.Focus()
			return m, nil
		} else {
			// Success: refresh hosts and return to list view
			var hosts []config.SSHHost
			var err error

			if m.configFile != "" {
				hosts, err = config.ParseSSHConfigFile(m.configFile)
			} else {
				hosts, err = config.ParseSSHConfig()
			}

			if err != nil {
				return m, tea.Quit
			}
			m.hosts = m.sortHosts(hosts)

			// Reapply search filter if there is one active
			if m.searchInput.Value() != "" {
				m.filteredHosts = m.filterHosts(m.searchInput.Value())
			} else {
				m.filteredHosts = m.hosts
			}

			m.updateTableRows()
			m.viewMode = ViewList
			m.moveForm = nil
			m.table.Focus()
			return m, nil
		}

	case moveFormCancelMsg:
		// Cancel: return to list view
		m.viewMode = ViewList
		m.moveForm = nil
		m.table.Focus()
		return m, nil

	case infoFormCancelMsg:
		// Cancel: return to list view
		m.viewMode = ViewList
		m.infoForm = nil
		m.table.Focus()
		return m, nil

	case fileSelectorMsg:
		if msg.cancelled {
			// Cancel: return to list view
			m.viewMode = ViewList
			m.fileSelectorForm = nil
			m.table.Focus()
			return m, nil
		} else {
			// File selected: proceed to add form with selected file
			m.addForm = NewAddForm("", m.styles, m.width, m.height, msg.selectedFile)
			m.viewMode = ViewAdd
			m.fileSelectorForm = nil
			return m, textinput.Blink
		}

	case infoFormEditMsg:
		// Switch from info to edit mode
		editForm, err := NewEditForm(msg.hostName, m.styles, m.width, m.height, m.configFile)
		if err != nil {
			// Handle error - could show in UI, for now just go back to list
			m.viewMode = ViewList
			m.infoForm = nil
			m.table.Focus()
			return m, nil
		}
		m.editForm = editForm
		m.infoForm = nil
		m.viewMode = ViewEdit
		return m, textinput.Blink

	case portForwardSubmitMsg:
		if msg.err != nil {
			// Show error in form
			if m.portForwardForm != nil {
				m.portForwardForm.err = msg.err.Error()
			}
			return m, nil
		} else {
			// Success: execute SSH command with port forwarding
			if len(msg.sshArgs) > 0 {
				sshCmd := exec.Command("ssh", msg.sshArgs...)

				// Record the connection in history
				if m.historyManager != nil && m.portForwardForm != nil {
					err := m.historyManager.RecordConnection(m.portForwardForm.hostName)
					if err != nil {
						fmt.Printf("Warning: Could not record connection history: %v\n", err)
					}
				}

				return m, tea.ExecProcess(sshCmd, func(err error) tea.Msg {
					return tea.Quit()
				})
			}

			// If no SSH args, just return to list view
			m.viewMode = ViewList
			m.portForwardForm = nil
			m.table.Focus()
			return m, nil
		}

	case portForwardCancelMsg:
		// Cancel: return to list view
		m.viewMode = ViewList
		m.portForwardForm = nil
		m.table.Focus()
		return m, nil

	case transferSubmitMsg:
		if msg.err != nil {
			// Show error in form
			if m.transferForm != nil {
				m.transferForm.err = msg.err.Error()
			}
			return m, nil
		} else {
			// Success: execute transfer command
			if msg.request != nil {
				// Record the transfer in history
				if m.historyManager != nil {
					direction := "upload"
					if msg.request.Direction == transfer.Download {
						direction = "download"
					}
					_ = m.historyManager.RecordTransfer(
						msg.request.Host,
						direction,
						msg.request.LocalPath,
						msg.request.RemotePath,
					)
				}

				// Build and execute scp command
				scpCmd := msg.request.BuildSCPCommand()
				return m, tea.ExecProcess(scpCmd, func(err error) tea.Msg {
					return tea.Quit()
				})
			}

			// If no request, just return to list view
			m.viewMode = ViewList
			m.transferForm = nil
			m.table.Focus()
			return m, nil
		}

	case transferCancelMsg:
		// Cancel: return to list view
		m.viewMode = ViewList
		m.transferForm = nil
		m.table.Focus()
		return m, nil

	case quickTransferCancelMsg:
		// Quick transfer cancelled or done: return to list view
		m.viewMode = ViewList
		m.quickTransferForm = nil
		m.table.Focus()
		return m, nil

	case quickLocalPickedMsg, quickRemotePickedMsg, quickTransferDoneMsg:
		// Route quick transfer async messages to the form
		if m.viewMode == ViewQuickTransfer && m.quickTransferForm != nil {
			var newForm *quickTransferModel
			newForm, cmd = m.quickTransferForm.Update(msg)
			m.quickTransferForm = newForm
			return m, cmd
		}
		return m, nil

	case openRemoteBrowserMsg:
		// Open the remote browser as a sub-view (not a nested program)
		m.remoteBrowserForm = NewRemoteBrowser(msg.host, msg.startPath, msg.configFile, msg.mode, m.styles, m.width, m.height)
		m.viewMode = ViewRemoteBrowser
		return m, m.remoteBrowserForm.Init()

	case remoteBrowserResultMsg:
		// Remote browser completed - route result back to quick transfer
		m.remoteBrowserForm = nil
		m.viewMode = ViewQuickTransfer
		if m.quickTransferForm != nil {
			// Convert to quickRemotePickedMsg
			pickedMsg := quickRemotePickedMsg{path: msg.path, selected: msg.selected}
			var newForm *quickTransferModel
			newForm, cmd = m.quickTransferForm.Update(pickedMsg)
			m.quickTransferForm = newForm
			return m, cmd
		}
		return m, nil

	case remoteBrowserLoadedMsg, remoteBrowserSearchMsg, searchDebounceMsg:
		// Route remote browser async messages to the form
		if m.viewMode == ViewRemoteBrowser && m.remoteBrowserForm != nil {
			var newForm *remoteBrowserModel
			newForm, cmd = m.remoteBrowserForm.Update(msg)
			m.remoteBrowserForm = newForm
			return m, cmd
		}
		return m, nil

	case helpCloseMsg:
		// Close help: return to list view
		m.viewMode = ViewList
		m.helpForm = nil
		m.table.Focus()
		return m, nil

	case k8sAddFormSubmitMsg:
		if msg.err != nil {
			// Show error in form
			if m.k8sAddForm != nil {
				m.k8sAddForm.err = msg.err.Error()
			}
			return m, nil
		} else {
			// Success: refresh k8s hosts and return to list view
			k8sHosts, err := config.ParseK8sConfig()
			if err != nil {
				return m, tea.Quit
			}
			m.k8sHosts = k8sHosts
			m.filteredK8sHosts = k8sHosts
			m.rebuildEntries()
			m.updateTableRows()
			m.viewMode = ViewList
			m.k8sAddForm = nil
			m.table.Focus()
			return m, nil
		}

	case k8sAddFormCancelMsg:
		// Cancel: return to list view
		m.viewMode = ViewList
		m.k8sAddForm = nil
		m.table.Focus()
		return m, nil

	case k8sEditFormSubmitMsg:
		if msg.err != nil {
			// Show error in form
			if m.k8sEditForm != nil {
				m.k8sEditForm.err = msg.err.Error()
			}
			return m, nil
		} else {
			// Success: refresh k8s hosts and return to list view
			k8sHosts, err := config.ParseK8sConfig()
			if err != nil {
				return m, tea.Quit
			}
			m.k8sHosts = k8sHosts
			m.filteredK8sHosts = k8sHosts
			m.rebuildEntries()
			m.updateTableRows()
			m.viewMode = ViewList
			m.k8sEditForm = nil
			m.table.Focus()
			return m, nil
		}

	case k8sEditFormCancelMsg:
		// Cancel: return to list view
		m.viewMode = ViewList
		m.k8sEditForm = nil
		m.table.Focus()
		return m, nil

	case themePickerSubmitMsg:
		// Theme selected: save to config and update styles
		if m.appConfig != nil {
			m.appConfig.Theme = msg.themeName
			_ = config.SaveAppConfig(m.appConfig)
		}
		SetThemeByName(msg.themeName)
		m.styles = NewStyles(m.width)
		m.updateTableStyles()
		m.viewMode = ViewList
		m.themePicker = nil
		m.table.Focus()
		return m, nil

	case themePickerCancelMsg:
		// Cancel: restore original theme and return to list view
		if m.appConfig != nil {
			SetThemeByName(m.appConfig.Theme)
		}
		m.styles = NewStyles(m.width)
		m.updateTableStyles()
		m.viewMode = ViewList
		m.themePicker = nil
		m.table.Focus()
		return m, nil

	case tea.KeyMsg:
		// Handle view-specific key presses
		switch m.viewMode {
		case ViewAdd:
			if m.addForm != nil {
				var newForm *addFormModel
				newForm, cmd = m.addForm.Update(msg)
				m.addForm = newForm
				return m, cmd
			}
		case ViewEdit:
			if m.editForm != nil {
				var updatedModel tea.Model
				updatedModel, cmd = m.editForm.Update(msg)
				m.editForm = updatedModel.(*editFormModel)
				return m, cmd
			}
		case ViewMove:
			if m.moveForm != nil {
				var newForm *moveFormModel
				newForm, cmd = m.moveForm.Update(msg)
				m.moveForm = newForm
				return m, cmd
			}
		case ViewInfo:
			if m.infoForm != nil {
				var newForm *infoFormModel
				newForm, cmd = m.infoForm.Update(msg)
				m.infoForm = newForm
				return m, cmd
			}
		case ViewPortForward:
			if m.portForwardForm != nil {
				var newForm *portForwardModel
				newForm, cmd = m.portForwardForm.Update(msg)
				m.portForwardForm = newForm
				return m, cmd
			}
		case ViewTransfer:
			if m.transferForm != nil {
				var newForm *transferFormModel
				newForm, cmd = m.transferForm.Update(msg)
				m.transferForm = newForm
				return m, cmd
			}
		case ViewQuickTransfer:
			if m.quickTransferForm != nil {
				var newForm *quickTransferModel
				newForm, cmd = m.quickTransferForm.Update(msg)
				m.quickTransferForm = newForm
				return m, cmd
			}
		case ViewRemoteBrowser:
			if m.remoteBrowserForm != nil {
				var newForm *remoteBrowserModel
				newForm, cmd = m.remoteBrowserForm.Update(msg)
				m.remoteBrowserForm = newForm
				return m, cmd
			}
		case ViewHelp:
			if m.helpForm != nil {
				var newForm *helpModel
				newForm, cmd = m.helpForm.Update(msg)
				m.helpForm = newForm
				return m, cmd
			}
		case ViewFileSelector:
			if m.fileSelectorForm != nil {
				var newForm *fileSelectorModel
				newForm, cmd = m.fileSelectorForm.Update(msg)
				m.fileSelectorForm = newForm
				return m, cmd
			}
		case ViewK8sAdd:
			if m.k8sAddForm != nil {
				var newForm *k8sAddFormModel
				newForm, cmd = m.k8sAddForm.Update(msg)
				m.k8sAddForm = newForm
				return m, cmd
			}
		case ViewK8sEdit:
			if m.k8sEditForm != nil {
				var newForm *k8sEditFormModel
				newForm, cmd = m.k8sEditForm.Update(msg)
				m.k8sEditForm = newForm
				return m, cmd
			}
		case ViewTheme:
			if m.themePicker != nil {
				var newPicker *themePickerModel
				newPicker, cmd = m.themePicker.Update(msg)
				m.themePicker = newPicker
				// Update styles after theme picker changes
				m.styles = NewStyles(m.width)
				return m, cmd
			}
		case ViewList:
			// Handle list view keys
			return m.handleListViewKeys(msg)
		}
	}

	return m, cmd
}

func (m Model) handleListViewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	key := msg.String()

	switch key {
	case "esc", "ctrl+c":
		if m.deleteMode {
			// Exit delete mode
			m.deleteMode = false
			m.deleteHost = ""
			m.table.Focus()
			return m, nil
		}
		if m.searchMode {
			// Exit search mode
			m.searchMode = false
			m.updateTableStyles()
			m.searchInput.Blur()
			m.table.Focus()
			return m, nil
		}
		// Use configurable key bindings for quit
		if m.appConfig != nil && m.appConfig.KeyBindings.ShouldQuitOnKey(key) {
			return m, tea.Quit
		}
	case "q":
		if !m.searchMode && !m.deleteMode {
			// Use configurable key bindings for quit
			if m.appConfig != nil && m.appConfig.KeyBindings.ShouldQuitOnKey(key) {
				return m, tea.Quit
			}
		}
	case "/", "ctrl+f":
		if !m.searchMode && !m.deleteMode {
			// Enter search mode
			m.searchMode = true
			m.updateTableStyles()
			m.table.Blur()
			m.searchInput.Focus()
			// Don't trigger filtering when entering search mode - wait for user input
			return m, textinput.Blink
		}
	case "tab":
		if !m.deleteMode {
			// Switch focus between search input and table
			if m.searchMode {
				// Switch from search to table
				m.searchMode = false
				m.updateTableStyles()
				m.searchInput.Blur()
				m.table.Focus()
			} else {
				// Switch from table to search
				m.searchMode = true
				m.updateTableStyles()
				m.table.Blur()
				m.searchInput.Focus()
				// Don't trigger filtering when switching to search mode
				return m, textinput.Blink
			}
			return m, nil
		}
	case "enter":
		if m.searchMode {
			// Validate search and return to table mode to allow commands
			m.searchMode = false
			m.updateTableStyles()
			m.searchInput.Blur()
			m.table.Focus()
			return m, nil
		} else if m.deleteMode {
			// Confirm deletion - handle both SSH and K8s hosts
			var err error
			if m.deleteHostIsK8s {
				// Delete K8s host
				err = config.DeleteK8sHost(m.deleteHost)
				if err == nil {
					// Refresh k8s hosts
					k8sHosts, parseErr := config.ParseK8sConfig()
					if parseErr == nil {
						m.k8sHosts = k8sHosts
						m.filteredK8sHosts = k8sHosts
					}
				}
			} else {
				// Delete SSH host
				if m.configFile != "" {
					err = config.DeleteSSHHostFromFile(m.deleteHost, m.configFile)
				} else {
					err = config.DeleteSSHHost(m.deleteHost)
				}
				if err == nil {
					// Refresh SSH hosts
					var hosts []config.SSHHost
					if m.configFile != "" {
						hosts, _ = config.ParseSSHConfigFile(m.configFile)
					} else {
						hosts, _ = config.ParseSSHConfig()
					}
					m.hosts = m.sortHosts(hosts)
					if m.searchInput.Value() != "" {
						m.filteredHosts = m.filterHosts(m.searchInput.Value())
					} else {
						m.filteredHosts = m.hosts
					}
				}
			}
			if err != nil {
				// Could display an error message here
				m.deleteMode = false
				m.deleteHost = ""
				m.deleteHostIsK8s = false
				m.table.Focus()
				return m, nil
			}

			// Rebuild unified entries and update table
			m.rebuildEntries()
			m.updateTableRows()
			m.deleteMode = false
			m.deleteHost = ""
			m.deleteHostIsK8s = false
			m.table.Focus()
			return m, nil
		} else {
			// Connect to the selected host
			selected := m.table.SelectedRow()
			if len(selected) > 0 {
				hostName := extractHostNameFromTableRow(selected[0])
				isK8s := isK8sHostFromTableRow(selected[0])

				// Record the connection in history
				if m.historyManager != nil {
					err := m.historyManager.RecordConnection(hostName)
					if err != nil {
						fmt.Printf("Warning: Could not record connection history: %v\n", err)
					}
				}

				if isK8s {
					// Get k8s host and build kubectl exec command
					k8sHost, err := config.GetK8sHost(hostName)
					if err != nil {
						fmt.Printf("Error: Could not find k8s host: %v\n", err)
						return m, nil
					}
					kubectlCmd := k8sHost.BuildKubectlCommand()
					return m, tea.ExecProcess(kubectlCmd, func(err error) tea.Msg {
						return tea.Quit()
					})
				} else {
					// Build the SSH command with the appropriate config file
					var sshCmd *exec.Cmd
					if m.configFile != "" {
						sshCmd = exec.Command("ssh", "-F", m.configFile, hostName)
					} else {
						sshCmd = exec.Command("ssh", hostName)
					}
					return m, tea.ExecProcess(sshCmd, func(err error) tea.Msg {
						return tea.Quit()
					})
				}
			}
		}
	case "e":
		if !m.searchMode && !m.deleteMode {
			// Edit the selected host
			selected := m.table.SelectedRow()
			if len(selected) > 0 {
				hostName := extractHostNameFromTableRow(selected[0])
				isK8s := isK8sHostFromTableRow(selected[0])

				if isK8s {
					// Edit k8s host
					k8sEditForm, err := NewK8sEditForm(hostName, m.styles, m.width, m.height)
					if err != nil {
						return m, nil
					}
					m.k8sEditForm = k8sEditForm
					m.viewMode = ViewK8sEdit
				} else {
					// Edit SSH host
					editForm, err := NewEditForm(hostName, m.styles, m.width, m.height, m.configFile)
					if err != nil {
						return m, nil
					}
					m.editForm = editForm
					m.viewMode = ViewEdit
				}
				return m, textinput.Blink
			}
		}
	case "m":
		if !m.searchMode && !m.deleteMode {
			// Move the selected host to another config file
			selected := m.table.SelectedRow()
			if len(selected) > 0 {
				// Check if it's a k8s host
				if isK8sHostFromTableRow(selected[0]) {
					m.errorMessage = "Move is not supported for Kubernetes hosts"
					m.showingError = true
					return m, func() tea.Msg {
						time.Sleep(2 * time.Second)
						return errorMsg("clear")
					}
				}
				hostName := extractHostNameFromTableRow(selected[0])
				moveForm, err := NewMoveForm(hostName, m.styles, m.width, m.height, m.configFile)
				if err != nil {
					// Show error message to user
					m.errorMessage = err.Error()
					m.showingError = true
					return m, func() tea.Msg {
						time.Sleep(3 * time.Second) // Show error for 3 seconds
						return errorMsg("clear")
					}
				}
				m.moveForm = moveForm
				m.viewMode = ViewMove
				return m, textinput.Blink
			}
		}
	case "i":
		if !m.searchMode && !m.deleteMode {
			// Show info for the selected host
			selected := m.table.SelectedRow()
			if len(selected) > 0 {
				// Check if it's a k8s host - show basic info in error message for now
				if isK8sHostFromTableRow(selected[0]) {
					hostName := extractHostNameFromTableRow(selected[0])
					k8sHost, err := config.GetK8sHost(hostName)
					if err != nil {
						return m, nil
					}
					info := fmt.Sprintf("K8s: %s | NS: %s | Pod: %s | Context: %s",
						k8sHost.Name, k8sHost.Namespace, k8sHost.Pod, k8sHost.Context)
					if k8sHost.Context == "" {
						info = fmt.Sprintf("K8s: %s | NS: %s | Pod: %s",
							k8sHost.Name, k8sHost.Namespace, k8sHost.Pod)
					}
					m.errorMessage = info
					m.showingError = true
					return m, func() tea.Msg {
						time.Sleep(4 * time.Second)
						return errorMsg("clear")
					}
				}
				hostName := extractHostNameFromTableRow(selected[0])
				infoForm, err := NewInfoForm(hostName, m.styles, m.width, m.height, m.configFile)
				if err != nil {
					// Handle error - could show in UI
					return m, nil
				}
				m.infoForm = infoForm
				m.viewMode = ViewInfo
				return m, nil
			}
		}
	case "a":
		if !m.searchMode && !m.deleteMode {
			// Check if there are multiple config files starting from the current base config
			var configFiles []string
			var err error

			if m.configFile != "" {
				// Use the specified config file as base
				configFiles, err = config.GetAllConfigFilesFromBase(m.configFile)
			} else {
				// Use the default config file as base
				configFiles, err = config.GetAllConfigFiles()
			}

			if err != nil || len(configFiles) <= 1 {
				// Only one config file (or error), go directly to add form
				var configFile string
				if len(configFiles) == 1 {
					configFile = configFiles[0]
				} else {
					configFile = m.configFile
				}
				m.addForm = NewAddForm("", m.styles, m.width, m.height, configFile)
				m.viewMode = ViewAdd
			} else {
				// Multiple config files, show file selector
				fileSelectorForm, err := NewFileSelectorFromBase("Select config file to add host to:", m.styles, m.width, m.height, m.configFile)
				if err != nil {
					// Fallback to default behavior if file selector fails
					m.addForm = NewAddForm("", m.styles, m.width, m.height, m.configFile)
					m.viewMode = ViewAdd
				} else {
					m.fileSelectorForm = fileSelectorForm
					m.viewMode = ViewFileSelector
				}
			}
			return m, textinput.Blink
		}
	case "d":
		if !m.searchMode && !m.deleteMode {
			// Delete the selected host
			selected := m.table.SelectedRow()
			if len(selected) > 0 {
				hostName := extractHostNameFromTableRow(selected[0])
				isK8s := isK8sHostFromTableRow(selected[0])
				m.deleteMode = true
				m.deleteHost = hostName
				m.deleteHostIsK8s = isK8s
				m.table.Blur()
				return m, nil
			}
		}
	case "K":
		if !m.searchMode && !m.deleteMode {
			// Add new k8s host
			m.k8sAddForm = NewK8sAddForm(m.styles, m.width, m.height)
			m.viewMode = ViewK8sAdd
			return m, textinput.Blink
		}
	case "p":
		if !m.searchMode && !m.deleteMode {
			// Ping all hosts
			return m, m.startPingAllCmd()
		}
	case "f":
		if !m.searchMode && !m.deleteMode {
			// Port forwarding for the selected host
			selected := m.table.SelectedRow()
			if len(selected) > 0 {
				// Check if it's a k8s host
				if isK8sHostFromTableRow(selected[0]) {
					m.errorMessage = "Port forwarding is not supported for Kubernetes hosts"
					m.showingError = true
					return m, func() tea.Msg {
						time.Sleep(2 * time.Second)
						return errorMsg("clear")
					}
				}
				hostName := extractHostNameFromTableRow(selected[0])
				m.portForwardForm = NewPortForwardForm(hostName, m.styles, m.width, m.height, m.configFile, m.historyManager)
				m.viewMode = ViewPortForward
				return m, textinput.Blink
			}
		}
	case "t":
		if !m.searchMode && !m.deleteMode {
			// Quick file transfer for the selected host
			selected := m.table.SelectedRow()
			if len(selected) > 0 {
				// Check if it's a k8s host
				if isK8sHostFromTableRow(selected[0]) {
					m.errorMessage = "File transfer is not supported for Kubernetes hosts"
					m.showingError = true
					return m, func() tea.Msg {
						time.Sleep(2 * time.Second)
						return errorMsg("clear")
					}
				}
				hostName := extractHostNameFromTableRow(selected[0])
				m.quickTransferForm = NewQuickTransfer(hostName, m.styles, m.width, m.height, m.configFile)
				m.viewMode = ViewQuickTransfer
				return m, nil
			}
		}
	case "h":
		if !m.searchMode && !m.deleteMode {
			// Show help
			m.helpForm = NewHelpForm(m.styles, m.width, m.height)
			m.viewMode = ViewHelp
			return m, nil
		}
	case "c":
		if !m.searchMode && !m.deleteMode {
			// Open theme picker (c for colors)
			m.themePicker = NewThemePicker(m.styles, m.width, m.height, m.appConfig)
			m.viewMode = ViewTheme
			return m, nil
		}
	case "s":
		if !m.searchMode && !m.deleteMode {
			// Cycle through sort modes (only 2 modes now)
			m.sortMode = (m.sortMode + 1) % 2
			// Re-apply the current filter/sort with the new sort mode
			if m.searchInput.Value() != "" {
				m.filteredEntries = m.sortEntries(m.filterEntries(m.searchInput.Value()))
			} else {
				m.filteredEntries = m.sortEntries(m.allEntries)
			}
			m.updateTableRows()
			return m, nil
		}
	case "r":
		if !m.searchMode && !m.deleteMode {
			// Switch to sort by recent (last used)
			m.sortMode = SortByLastUsed
			// Re-apply the current filter/sort with the new sort mode
			if m.searchInput.Value() != "" {
				m.filteredEntries = m.sortEntries(m.filterEntries(m.searchInput.Value()))
			} else {
				m.filteredEntries = m.sortEntries(m.allEntries)
			}
			m.updateTableRows()
			return m, nil
		}
	case "n":
		if !m.searchMode && !m.deleteMode {
			// Switch to sort by name
			m.sortMode = SortByName
			// Re-apply the current filter/sort with the new sort mode
			if m.searchInput.Value() != "" {
				m.filteredEntries = m.sortEntries(m.filterEntries(m.searchInput.Value()))
			} else {
				m.filteredEntries = m.sortEntries(m.allEntries)
			}
			m.updateTableRows()
			return m, nil
		}
	}

	// Update the appropriate component based on mode
	if m.searchMode {
		oldValue := m.searchInput.Value()
		m.searchInput, cmd = m.searchInput.Update(msg)
		// Update filtered entries only if the search value has changed
		if m.searchInput.Value() != oldValue {
			currentCursor := m.table.Cursor()
			if m.searchInput.Value() != "" {
				m.filteredEntries = m.filterEntries(m.searchInput.Value())
			} else {
				m.filteredEntries = m.allEntries
			}
			m.updateTableRows()
			// If the current cursor position is beyond the filtered results, reset to 0
			if currentCursor >= len(m.filteredEntries) && len(m.filteredEntries) > 0 {
				m.table.SetCursor(0)
			}
		}
	} else {
		m.table, cmd = m.table.Update(msg)
	}

	return m, cmd
}
