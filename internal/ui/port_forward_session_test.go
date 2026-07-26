package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// newTestSession builds a session without launching anything.
func newTestSession() *portForwardSessionModel {
	return NewPortForwardSession(
		"web-01",
		"localhost:8080 here → localhost:80 on the remote",
		[]string{"-L", "8080:localhost:80", "-N", "web-01"},
		NewStyles(72), 72, 16,
	)
}

// TestPortForwardRunsWithoutATerminal checks the forward runs as a background
// child rather than taking over the terminal, which is what logged the user
// into the server instead of just forwarding.
func TestPortForwardRunsWithoutATerminal(t *testing.T) {
	m := newTestSession()

	wait := m.launch("sleep", "30")
	if wait == nil {
		t.Fatal("launch returned no wait command")
	}
	if m.state != forwardActive {
		t.Fatalf("state is %v after launch, want active (%s)", m.state, m.err)
	}

	// Uptime is reported from the clock, not guessed.
	m.now = m.started.Add(75 * time.Second)
	if got := m.elapsed(); got != "00:01:15" {
		t.Errorf("elapsed is %q, want 00:01:15", got)
	}

	view := m.View()
	if !strings.Contains(view, "forwarding") {
		t.Error("the active view does not say it is forwarding")
	}

	// Stopping must actually kill the child, not just change the label.
	process := m.command.Process
	m.stop()
	if m.state != forwardStopped {
		t.Errorf("state is %v after stop, want stopped", m.state)
	}
	if _, err := process.Wait(); err != nil {
		t.Errorf("waiting on the killed child failed: %v", err)
	}

	// The expected exit after a kill must not be reported as a failure.
	m, _ = m.Update(portForwardExitedMsg{err: fmt.Errorf("signal: killed")})
	if m.state != forwardStopped {
		t.Errorf("a kill was reported as %v, want it to stay stopped", m.state)
	}
}

// TestPortForwardReportsBindFailure checks that ssh's reason for exiting is
// shown, rather than the failure passing silently.
func TestPortForwardReportsBindFailure(t *testing.T) {
	m := newTestSession()
	m.launch("sh", "-c", "exit 1")

	m, _ = m.Update(portForwardExitedMsg{
		err:    fmt.Errorf("exit status 1"),
		output: "bind [127.0.0.1]:80: Address already in use\n",
	})

	if m.state != forwardFailed {
		t.Fatalf("state is %v after a failed forward, want failed", m.state)
	}
	if !strings.Contains(m.View(), "Address already in use") {
		t.Error("the failure view does not show why ssh exited")
	}
}

// TestPortForwardEscapeCloses checks esc stops the forward and hands control
// back to the list.
func TestPortForwardEscapeCloses(t *testing.T) {
	m := newTestSession()
	m.launch("sleep", "30")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc produced no command")
	}
	if _, ok := cmd().(portForwardClosedMsg); !ok {
		t.Errorf("esc sent %T, want portForwardClosedMsg", cmd())
	}
	if updated.state != forwardStopped {
		t.Errorf("state is %v after esc, want stopped", updated.state)
	}
}
