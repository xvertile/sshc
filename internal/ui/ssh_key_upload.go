package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xvertile/sshc/internal/config"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SSHKeyOption represents a key selection option
type SSHKeyOption int

const (
	KeyOptionSelect SSHKeyOption = iota
	KeyOptionPaste
)

// sshKeyUploadModel is the model for SSH key upload
type sshKeyUploadModel struct {
	hostName   string
	configFile string
	styles     Styles
	width      int
	height     int

	// Available SSH keys
	availableKeys []string
	selectedKey   int

	// Mode: selecting key or pasting
	mode SSHKeyOption

	// For paste mode
	pasteInput textinput.Model

	// State
	uploading          bool
	err                string
	success            string
	askingConfigUpdate bool   // Whether we're asking about config update
	uploadedKeyPath    string // Path to the key that was uploaded (without .pub)
	configUpdateDone   bool   // Whether config update was completed
}

// sshKeyUploadSubmitMsg is sent when key upload completes
type sshKeyUploadSubmitMsg struct {
	err     error
	keyPath string // Path to the key that was uploaded
}

// sshKeyUploadCancelMsg is sent when user cancels
type sshKeyUploadCancelMsg struct{}

// NewSSHKeyUploadForm creates a new SSH key upload form
func NewSSHKeyUploadForm(hostName string, styles Styles, width, height int, configFile string) *sshKeyUploadModel {
	// Find available SSH keys
	keys := findSSHKeys()

	// Create paste input
	ti := textinput.New()
	ti.Placeholder = "Paste public key here (ssh-rsa AAAA... or ssh-ed25519 AAAA...)"
	ti.CharLimit = 2048
	ti.Width = 60

	return &sshKeyUploadModel{
		hostName:      hostName,
		configFile:    configFile,
		styles:        styles,
		width:         width,
		height:        height,
		availableKeys: keys,
		selectedKey:   0,
		mode:          KeyOptionSelect,
		pasteInput:    ti,
	}
}

// findSSHKeys finds all public keys in ~/.ssh/
func findSSHKeys() []string {
	var keys []string

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return keys
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return keys
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Look for .pub files (public keys)
		if strings.HasSuffix(name, ".pub") {
			keys = append(keys, filepath.Join(sshDir, name))
		}
	}

	return keys
}

func (m *sshKeyUploadModel) Init() tea.Cmd {
	return nil
}

func (m *sshKeyUploadModel) Update(msg tea.Msg) (*sshKeyUploadModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case sshKeyUploadSubmitMsg:
		m.uploading = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.success = "Key uploaded successfully!"
		// If we have a key path, ask about config update
		if msg.keyPath != "" {
			m.askingConfigUpdate = true
			m.uploadedKeyPath = msg.keyPath
		}
		return m, nil

	case tea.KeyMsg:
		if m.uploading {
			return m, nil
		}

		// Handle config update prompt
		if m.askingConfigUpdate {
			switch msg.String() {
			case "y", "Y":
				// Update SSH config with identity file
				if err := m.updateSSHConfig(); err != nil {
					m.err = err.Error()
				} else {
					m.configUpdateDone = true
					m.success = "Key uploaded and config updated!"
				}
				m.askingConfigUpdate = false
				return m, nil
			case "n", "N", "esc", "q":
				// Don't update config, just continue
				m.askingConfigUpdate = false
				return m, nil
			}
			return m, nil
		}

		// Clear success message on any key (after config decision)
		if m.success != "" && !m.askingConfigUpdate {
			return m, func() tea.Msg {
				return sshKeyUploadCancelMsg{}
			}
		}

		switch msg.String() {
		case "esc", "q":
			if m.mode == KeyOptionPaste && m.pasteInput.Focused() {
				// Exit paste mode
				m.mode = KeyOptionSelect
				m.pasteInput.Blur()
				m.err = ""
				return m, nil
			}
			return m, func() tea.Msg {
				return sshKeyUploadCancelMsg{}
			}

		case "ctrl+c":
			return m, func() tea.Msg {
				return sshKeyUploadCancelMsg{}
			}

		case "tab":
			// Toggle between select and paste mode
			if m.mode == KeyOptionSelect {
				m.mode = KeyOptionPaste
				m.pasteInput.Focus()
				return m, textinput.Blink
			} else {
				m.mode = KeyOptionSelect
				m.pasteInput.Blur()
				return m, nil
			}

		case "up", "k":
			if m.mode == KeyOptionSelect && len(m.availableKeys) > 0 {
				if m.selectedKey > 0 {
					m.selectedKey--
				}
			}
			return m, nil

		case "down", "j":
			if m.mode == KeyOptionSelect && len(m.availableKeys) > 0 {
				if m.selectedKey < len(m.availableKeys)-1 {
					m.selectedKey++
				}
			}
			return m, nil

		case "enter":
			if m.mode == KeyOptionSelect {
				// Upload selected key
				if len(m.availableKeys) > 0 {
					m.uploading = true
					m.err = ""
					return m, m.uploadKey(m.availableKeys[m.selectedKey])
				}
			} else {
				// Upload pasted key
				key := strings.TrimSpace(m.pasteInput.Value())
				if key != "" {
					if !isValidSSHPublicKey(key) {
						m.err = "Invalid SSH public key format"
						return m, nil
					}
					m.uploading = true
					m.err = ""
					return m, m.uploadPastedKey(key)
				}
			}
			return m, nil
		}

		// Handle text input for paste mode
		if m.mode == KeyOptionPaste {
			m.pasteInput, cmd = m.pasteInput.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// isValidSSHPublicKey checks if the key looks like a valid SSH public key
func isValidSSHPublicKey(key string) bool {
	key = strings.TrimSpace(key)
	// Check for common SSH key prefixes
	validPrefixes := []string{
		"ssh-rsa",
		"ssh-ed25519",
		"ssh-dss",
		"ecdsa-sha2-nistp256",
		"ecdsa-sha2-nistp384",
		"ecdsa-sha2-nistp521",
	}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(key, prefix+" ") {
			return true
		}
	}
	return false
}

// updateSSHConfig updates the SSH config to add IdentityFile for this host
func (m *sshKeyUploadModel) updateSSHConfig() error {
	// Get the private key path (remove .pub suffix)
	privateKeyPath := strings.TrimSuffix(m.uploadedKeyPath, ".pub")

	// Get the current host configuration
	var host *config.SSHHost
	var err error

	if m.configFile != "" {
		hosts, parseErr := config.ParseSSHConfigFile(m.configFile)
		if parseErr != nil {
			return fmt.Errorf("failed to parse config: %v", parseErr)
		}
		for i := range hosts {
			if hosts[i].Name == m.hostName {
				host = &hosts[i]
				break
			}
		}
	} else {
		hosts, parseErr := config.ParseSSHConfig()
		if parseErr != nil {
			return fmt.Errorf("failed to parse config: %v", parseErr)
		}
		for i := range hosts {
			if hosts[i].Name == m.hostName {
				host = &hosts[i]
				break
			}
		}
	}

	if host == nil {
		return fmt.Errorf("host not found in config")
	}

	// Update the identity field
	host.Identity = privateKeyPath

	// Save the updated config
	if m.configFile != "" {
		err = config.UpdateSSHHostInFile(m.hostName, *host, m.configFile)
	} else {
		err = config.UpdateSSHHost(m.hostName, *host)
	}

	if err != nil {
		return fmt.Errorf("failed to update config: %v", err)
	}

	return nil
}

// uploadKey uploads a key file using ssh-copy-id (interactive)
func (m *sshKeyUploadModel) uploadKey(keyPath string) tea.Cmd {
	var args []string

	// Add config file if specified
	if m.configFile != "" {
		args = append(args, "-F", m.configFile)
	}

	args = append(args, "-i", keyPath, m.hostName)

	cmd := exec.Command("ssh-copy-id", args...)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return sshKeyUploadSubmitMsg{err: err, keyPath: keyPath}
	})
}

// uploadPastedKey uploads a pasted key by writing to temp file and using ssh-copy-id (interactive)
func (m *sshKeyUploadModel) uploadPastedKey(key string) tea.Cmd {
	// Create temp file for the key (sync, before running command)
	tmpFile, err := os.CreateTemp("", "sshc-key-*.pub")
	if err != nil {
		return func() tea.Msg {
			return sshKeyUploadSubmitMsg{err: fmt.Errorf("failed to create temp file: %v", err)}
		}
	}

	if _, err := tmpFile.WriteString(key + "\n"); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return func() tea.Msg {
			return sshKeyUploadSubmitMsg{err: fmt.Errorf("failed to write key: %v", err)}
		}
	}
	tmpFile.Close()

	var args []string

	// Add config file if specified
	if m.configFile != "" {
		args = append(args, "-F", m.configFile)
	}

	args = append(args, "-i", tmpFile.Name(), m.hostName)
	tmpPath := tmpFile.Name()

	cmd := exec.Command("ssh-copy-id", args...)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		// Clean up temp file after command completes
		os.Remove(tmpPath)
		// For pasted keys, we don't offer config update since it was a temp file
		return sshKeyUploadSubmitMsg{err: err, keyPath: ""}
	})
}

func (m *sshKeyUploadModel) View() string {
	theme := GetCurrentTheme()

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.Primary))
	b.WriteString(titleStyle.Render("UPLOAD SSH KEY"))
	b.WriteString("\n\n")

	// Host info
	hostStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Accent))
	b.WriteString(hostStyle.Render(fmt.Sprintf("Host: %s", m.hostName)))
	b.WriteString("\n\n")

	// Success message and config update prompt
	if m.success != "" {
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
		b.WriteString(successStyle.Render(m.success))
		b.WriteString("\n\n")

		if m.askingConfigUpdate {
			// Show config update prompt
			questionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Foreground))
			b.WriteString(questionStyle.Render("Update SSH config to use this key automatically?"))
			b.WriteString("\n")
			keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Muted))
			privateKey := strings.TrimSuffix(m.uploadedKeyPath, ".pub")
			b.WriteString(keyStyle.Render(fmt.Sprintf("  IdentityFile %s", privateKey)))
			b.WriteString("\n\n")

			helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Muted))
			b.WriteString(helpStyle.Render("y: yes, update config • n: no, skip"))
		} else {
			helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Muted))
			b.WriteString(helpStyle.Render("Press any key to continue..."))
		}
	} else if m.uploading {
		b.WriteString("Uploading key...")
	} else {
		// Mode tabs
		selectTab := "[ Select Key ]"
		pasteTab := "[ Paste Key ]"

		if m.mode == KeyOptionSelect {
			selectTab = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.Primary)).Render(selectTab)
			pasteTab = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Muted)).Render(pasteTab)
		} else {
			selectTab = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Muted)).Render(selectTab)
			pasteTab = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.Primary)).Render(pasteTab)
		}
		b.WriteString(selectTab + "  " + pasteTab + "  (Tab to switch)")
		b.WriteString("\n\n")

		if m.mode == KeyOptionSelect {
			// Show available keys
			if len(m.availableKeys) == 0 {
				b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Muted)).Render("No SSH keys found in ~/.ssh/"))
				b.WriteString("\n")
			} else {
				for i, key := range m.availableKeys {
					keyName := filepath.Base(key)
					if i == m.selectedKey {
						b.WriteString(lipgloss.NewStyle().
							Bold(true).
							Foreground(lipgloss.Color(theme.SelectionFg)).
							Background(lipgloss.Color(theme.SelectionBg)).
							Render(fmt.Sprintf(" > %s ", keyName)))
					} else {
						b.WriteString(fmt.Sprintf("   %s", keyName))
					}
					b.WriteString("\n")
				}
			}
		} else {
			// Show paste input
			b.WriteString("Paste your public key:\n")
			b.WriteString(m.pasteInput.View())
			b.WriteString("\n")
		}

		// Error message
		if m.err != "" {
			b.WriteString("\n")
			errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
			b.WriteString(errorStyle.Render("Error: " + m.err))
		}

		// Help
		b.WriteString("\n\n")
		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Muted))
		if m.mode == KeyOptionSelect {
			b.WriteString(helpStyle.Render("↑/↓: navigate • Enter: upload • Tab: paste mode • Esc: cancel"))
		} else {
			b.WriteString(helpStyle.Render("Enter: upload • Tab: select mode • Esc: cancel"))
		}
	}

	content := b.String()

	// Container
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Primary)).
		Padding(1, 2)

	// Logo
	logo := m.styles.Header.Render(asciiTitle)

	// Stack logo and container
	fullContent := lipgloss.JoinVertical(lipgloss.Center, logo, "", box.Render(content))

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		fullContent,
	)
}
