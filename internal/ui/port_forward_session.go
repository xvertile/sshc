package ui

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// portForwardState is what the tunnel is currently doing.
type portForwardState int

const (
	forwardStarting portForwardState = iota
	forwardActive
	forwardFailed
	forwardStopped
)

// portForwardSessionModel supervises a running ssh port forward.
//
// The forward runs as a background child with -N, so no remote shell is
// started and the terminal stays with this UI. Previously the command was
// handed the terminal outright, which logged you in and left the tunnel alive
// only as long as you sat in that shell.
type portForwardSessionModel struct {
	host    string
	summary string   // what is being forwarded, in words
	args    []string // the ssh arguments, for display and restart

	command *exec.Cmd
	stderr  *bytes.Buffer

	state   portForwardState
	started time.Time
	now     time.Time
	err     string

	styles Styles
	width  int
	height int
}

// portForwardExitedMsg is sent when the ssh child exits, for any reason.
type portForwardExitedMsg struct {
	err    error
	output string
}

// portForwardTickMsg drives the elapsed-time display.
type portForwardTickMsg time.Time

// portForwardClosedMsg asks the parent to return to the list.
type portForwardClosedMsg struct{}

// NewPortForwardSession creates a session for a forward that is about to start.
func NewPortForwardSession(host, summary string, args []string, styles Styles, width, height int) *portForwardSessionModel {
	return &portForwardSessionModel{
		host:    host,
		summary: summary,
		args:    args,
		state:   forwardStarting,
		styles:  styles,
		width:   width,
		height:  height,
	}
}

func (m *portForwardSessionModel) Init() tea.Cmd {
	return tea.Batch(m.start(), portForwardTick())
}

// portForwardTick schedules the next elapsed-time update.
func portForwardTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return portForwardTickMsg(t)
	})
}

// start launches ssh and returns a command that waits on it.
func (m *portForwardSessionModel) start() tea.Cmd {
	return m.launch("ssh", m.args...)
}

// stop ends the forward if it is still running.
func (m *portForwardSessionModel) stop() {
	if m.state == forwardActive && m.command != nil && m.command.Process != nil {
		_ = m.command.Process.Kill()
	}
	m.state = forwardStopped
}

func (m *portForwardSessionModel) Update(msg tea.Msg) (*portForwardSessionModel, tea.Cmd) {
	switch msg := msg.(type) {
	case portForwardTickMsg:
		m.now = time.Time(msg)
		if m.state == forwardActive {
			return m, portForwardTick()
		}
		return m, nil

	case portForwardExitedMsg:
		// A kill is expected; anything else means the tunnel dropped.
		if m.state != forwardStopped {
			m.state = forwardFailed
			m.err = strings.TrimSpace(msg.output)
			if m.err == "" && msg.err != nil {
				m.err = msg.err.Error()
			}
			if m.err == "" {
				m.err = "the connection closed"
			}
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "ctrl+c":
			m.stop()
			return m, func() tea.Msg { return portForwardClosedMsg{} }

		case "r":
			// Retry after a failure, reusing the same arguments.
			if m.state == forwardFailed {
				m.err = ""
				m.state = forwardStarting
				return m, tea.Batch(m.start(), portForwardTick())
			}
		}
	}

	return m, nil
}

// elapsed is how long the forward has been up.
func (m *portForwardSessionModel) elapsed() string {
	if m.state != forwardActive {
		return ""
	}

	seconds := int(m.now.Sub(m.started).Seconds())
	if seconds < 0 {
		seconds = 0
	}

	return fmt.Sprintf("%02d:%02d:%02d", seconds/3600, (seconds%3600)/60, seconds%60)
}

func (m *portForwardSessionModel) View() string {
	theme := GetCurrentTheme()

	var status, hint string
	var hints []keyHint

	switch m.state {
	case forwardStarting:
		status = accent("◌ starting…")
		hints = []keyHint{{"esc", "cancel"}}

	case forwardActive:
		status = lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Success)).
			Bold(true).
			Render("● forwarding")
		hint = muted("leave this open to keep the tunnel up")
		hints = []keyHint{{"esc", "stop"}}

	case forwardFailed:
		status = lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Error)).
			Bold(true).
			Render("✗ forwarding stopped")
		hints = []keyHint{{"r", "retry"}, {"esc", "back"}}

	case forwardStopped:
		status = muted("stopped")
		hints = []keyHint{{"esc", "back"}}
	}

	lines := []string{
		status,
		"",
		muted("host      ") + m.host,
		muted("forward   ") + m.summary,
	}

	if elapsed := m.elapsed(); elapsed != "" {
		lines = append(lines, muted("uptime    ")+elapsed)
	}

	lines = append(lines, "", muted("ssh "+strings.Join(m.args, " ")))

	if hint != "" {
		lines = append(lines, "", hint)
	}

	return formScreen(m.width, m.height,
		"port forward", strings.Join(lines, "\n"), m.err, hints...)
}

// launch starts the child process and returns a command that waits on it.
//
// The program is a parameter so tests can stand in for ssh without needing a
// server to connect to.
func (m *portForwardSessionModel) launch(program string, args ...string) tea.Cmd {
	m.stderr = &bytes.Buffer{}
	m.command = exec.Command(program, args...)
	m.command.Stderr = m.stderr

	if err := m.command.Start(); err != nil {
		m.state = forwardFailed
		m.err = err.Error()
		return nil
	}

	m.state = forwardActive
	m.started = time.Now()
	m.now = m.started

	command, stderr := m.command, m.stderr

	return func() tea.Msg {
		err := command.Wait()
		return portForwardExitedMsg{err: err, output: stderr.String()}
	}
}
