package ui

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/xvertile/sshc/internal/config"
)

// terminalSizes spans from absurdly small to large. No screen may ever refuse
// to render or overflow, however little room it is given, so the small sizes
// are the point of this list rather than an edge case bolted on the end.
// ansiPattern matches a complete SGR escape sequence.
var ansiPattern = regexp.MustCompile("\x1b\\[[0-9;]*m")

var terminalSizes = [][2]int{
	{20, 5}, {30, 6}, {40, 8}, {60, 12}, {80, 24}, {120, 40}, {200, 60},
}

// newLayoutModel builds a model with enough hosts to overflow a short
// terminal, so clipping is exercised as well as padding.
func newLayoutModel(width, height int) Model {
	var hosts []config.SSHHost
	for _, name := range []string{
		"web-01", "db-primary", "cache-01", "bastion",
		"staging-01", "build-runner", "metrics", "logs-01",
	} {
		hosts = append(hosts, config.SSHHost{
			Name:     name,
			Hostname: name + ".internal",
			Tags:     []string{"prod"},
		})
	}

	searchInput := textinput.New()
	searchInput.Prompt = "" // as NewModel configures it

	m := Model{
		hosts:          hosts,
		searchInput:    searchInput,
		table:          hostTable{},
		ready:          true,
		width:          width,
		height:         height,
		styles:         NewStyles(width),
		currentVersion: "1.6.0",
	}

	m.rebuildEntries()
	m.updateTableStyles()
	m.updateTableColumns()
	m.updateTableRows()

	return m
}

// assertFillsViewport checks a screen occupies exactly the terminal height and
// never overflows its width. Both matter: a screen one line too tall scrolls
// the terminal and smears the alt-screen buffer.
func assertFillsViewport(t *testing.T, name string, view string, width, height int) {
	t.Helper()

	lines := strings.Split(view, "\n")
	if len(lines) != height {
		t.Errorf("%s at %dx%d: rendered %d lines, want %d",
			name, width, height, len(lines), height)
	}

	for i, line := range lines {
		if w := lipgloss.Width(line); w > width {
			t.Errorf("%s at %dx%d: line %d is %d columns wide, want <= %d",
				name, width, height, i+1, w, width)
		}
	}
}

func TestListViewFillsViewport(t *testing.T) {
	for _, size := range terminalSizes {
		width, height := size[0], size[1]
		m := newLayoutModel(width, height)

		assertFillsViewport(t, "list", m.View(), width, height)

		// Search mode swaps the filter line and footer contents; both must
		// stay single-line.
		m.searchMode = true
		m.searchInput.Focus()
		assertFillsViewport(t, "list/search", m.View(), width, height)

		// The delete prompt is inline, so it must not add a line either.
		m.searchMode = false
		m.deleteMode = true
		m.deleteHost = "cache-01"
		assertFillsViewport(t, "list/delete", m.View(), width, height)
	}
}

func TestConnectionErrorViewFillsViewport(t *testing.T) {
	for _, size := range terminalSizes {
		width, height := size[0], size[1]

		m := newLayoutModel(width, height)
		m.viewMode = ViewConnectionError
		m.connectionHost = "cache-01"
		m.connectionError = "ssh: connect to host 10.0.0.9 port 22: Operation timed out"

		assertFillsViewport(t, "connection-error", m.View(), width, height)
	}
}

func TestFormScreensFillViewport(t *testing.T) {
	for _, size := range terminalSizes {
		width, height := size[0], size[1]
		styles := NewStyles(width)

		screens := map[string]string{
			"add":   NewAddForm("", styles, width, height, "").View(),
			"help":  NewHelpForm(styles, width, height).View(),
			"theme": NewThemePicker(styles, width, height, nil).View(),
			"k8s":   NewK8sAddForm(styles, width, height).View(),
		}

		for name, view := range screens {
			assertFillsViewport(t, name, view, width, height)
		}
	}
}

// TestListChromeMatchesTableHeight guards the invariant the old hardcoded
// reservedHeight got wrong: the lines the chrome draws plus the lines the
// table draws must equal the terminal height exactly, at every size.
func TestListChromeMatchesTableHeight(t *testing.T) {
	for _, size := range terminalSizes {
		width, height := size[0], size[1]
		m := newLayoutModel(width, height)

		tableLines := lipgloss.Height(m.table.View())
		chromeLines := len(strings.Split(m.View(), "\n")) - tableLines

		// The table renders its own column header, which chromeFor counts,
		// hence the -1.
		if want := chromeFor(height).lines() - 1; chromeLines != want {
			t.Errorf("at %dx%d: chrome drew %d lines, want %d",
				width, height, chromeLines, want)
		}
	}
}

// TestListShowsHostsAtEverySize checks the list never degrades into pure
// chrome: whatever the terminal size, at least one host row is visible.
func TestListShowsHostsAtEverySize(t *testing.T) {
	for _, size := range terminalSizes {
		width, height := size[0], size[1]
		m := newLayoutModel(width, height)

		// One line of the table is its column header; anything beyond that
		// is an actual host.
		if rows := lipgloss.Height(m.table.View()) - 1; rows < 1 {
			t.Errorf("at %dx%d: no host rows visible", width, height)
		}
	}
}

// TestNarrowTerminalDropsColumns checks a cramped terminal gives up whole
// columns rather than clipping every one of them into illegibility.
func TestNarrowTerminalDropsColumns(t *testing.T) {
	wide := newLayoutModel(120, 24)
	if got := len(wide.table.Columns()); got != tableColumnCount {
		t.Errorf("at 120 columns: showed %d columns, want %d", got, tableColumnCount)
	}

	narrow := newLayoutModel(30, 24)
	columns := narrow.table.Columns()

	if len(columns) >= tableColumnCount {
		t.Errorf("at 30 columns: showed %d columns, expected some to be dropped", len(columns))
	}
	if len(columns) < 1 {
		t.Fatal("at 30 columns: every column was dropped, want at least the name")
	}
	if !strings.HasPrefix(columns[0].Title, "NAME") {
		t.Errorf("at 30 columns: first column is %q, want the name column", columns[0].Title)
	}
}

// TestEditFormRendersOnShortTerminal covers the regression this replaced: the
// edit form used to refuse to draw, showing a "terminal height is too small"
// warning instead, on terminals it now handles by scrolling.
func TestEditFormRendersOnShortTerminal(t *testing.T) {
	for _, size := range terminalSizes {
		width, height := size[0], size[1]

		body := strings.Join([]string{
			"config a", "", "host names",
			"Name 1", "", "tabs", "",
			"Hostname", "User", "Port", "Identity", "ProxyJump", "Tags",
		}, "\n")

		// Focus the final field: it is the one that falls off the bottom.
		view := formScreenAt(width, height, "edit host", body, "", 12,
			keyHint{"ctrl+s", "save"})

		assertFillsViewport(t, "edit", view, width, height)

		if strings.Contains(view, "too small") {
			t.Errorf("at %dx%d: form refused to render", width, height)
		}
		if height >= formChromeLines+1 && !strings.Contains(view, "Tags") {
			t.Errorf("at %dx%d: focused field scrolled out of view", width, height)
		}
	}
}

// TestRowsCarryColourWithoutLeaking is a regression test for the reason
// bubbles/table was replaced: it truncated cells with runewidth.Truncate,
// which is ANSI-unaware and cut through escape sequences, leaving the codes
// on screen as literal text.
func TestRowsCarryColourWithoutLeaking(t *testing.T) {
	restore := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(restore)

	const width, height = 80, 20
	m := newLayoutModel(width, height)

	for i, row := range m.table.Rows() {
		line := strings.Join(row, "")

		// A half-written escape sequence shows up as a stray control byte
		// once the complete ones are removed.
		if stripped := ansiPattern.ReplaceAllString(line, ""); strings.ContainsRune(stripped, '\x1b') {
			t.Errorf("row %d: escape sequence was cut mid-way: %q", i, stripped)
		}

		// Rows span the panel's interior, not the whole terminal.
		if got, want := lipgloss.Width(line), m.contentWidth(); got != want {
			t.Errorf("row %d: rendered %d columns, want %d", i, got, want)
		}
	}

	// The colours have to actually be there, or the check above passes on a
	// plain-text table.
	if !strings.Contains(strings.Join(m.table.Rows()[0], ""), "\x1b[") {
		t.Error("rows carry no colour at all")
	}
}

// TestHostTableNavigation covers the cursor and scrolling behaviour that moved
// in-tree with hostTable.
func TestHostTableNavigation(t *testing.T) {
	m := newLayoutModel(80, 10)
	rows := len(m.filteredEntries)

	m.table.SetCursor(0)
	m.table.MoveUp(1)
	if got := m.table.Cursor(); got != 0 {
		t.Errorf("cursor moved above the first row to %d", got)
	}

	m.table.MoveDown(rows * 2)
	if got, want := m.table.Cursor(), rows-1; got != want {
		t.Errorf("cursor ran past the last row to %d, want %d", got, want)
	}

	// The window must have scrolled to keep the cursor on screen.
	if visible := m.table.visibleRows(); m.table.Cursor() >= m.table.offset+visible {
		t.Errorf("cursor %d is outside the visible window [%d,%d)",
			m.table.Cursor(), m.table.offset, m.table.offset+visible)
	}
}

// TestThemeChangeRepaintsRows covers a bug where a new theme only took effect
// on the rows once the cursor moved: rows are pre-styled strings, and the
// theme handlers refreshed the styles without rebuilding them.
func TestThemeChangeRepaintsRows(t *testing.T) {
	restore := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(restore)

	original := CurrentThemeIndex
	defer SetTheme(original)

	SetTheme(0)
	m := newLayoutModel(80, 14)
	// appConfig is nil here, so the handler does not touch the user's config.
	before := strings.Join(m.table.Rows()[0], "")

	updated, _ := m.Update(themePickerSubmitMsg{themeName: "Gruvbox"})
	m = updated.(Model)

	if after := strings.Join(m.table.Rows()[0], ""); after == before {
		t.Error("rows kept the previous theme's colours after the theme changed")
	}

	// The whole screen must be repainted, not just the rows.
	if !strings.Contains(m.View(), "\x1b[") {
		t.Error("view lost its colour entirely")
	}
}

// TestResizeRebuildsRows covers a first-start bug: rows are pre-padded to the
// column widths, and the resize handler updated the columns without them, so
// the table rendered skewed until an unrelated action rebuilt the rows.
func TestResizeRebuildsRows(t *testing.T) {
	restore := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(restore)

	// Built at the default 80x24 that NewModel starts from.
	m := newLayoutModel(80, 24)

	// The first WindowSizeMsg carries the real terminal size.
	const width, height = 120, 30
	updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	m = updated.(Model)

	want := m.contentWidth()
	for i, row := range m.table.Rows() {
		if got := lipgloss.Width(strings.Join(row, "")); got != want {
			t.Fatalf("row %d is %d columns after resize, want %d", i, got, want)
		}
	}

	assertFillsViewport(t, "list/resized", m.View(), width, height)
}

// TestEditPreservesHiddenFields checks that editing a host keeps the settings
// the form no longer shows. SSH Options, RemoteCommand and RequestTTY are not
// editable in the TUI, so a save must carry them through untouched rather than
// dropping them from the config file.
func TestEditPreservesHiddenFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")

	original := config.SSHHost{
		Name:          "web-01",
		Hostname:      "10.0.0.4",
		User:          "deploy",
		Port:          "22",
		Options:       "Compression yes",
		RemoteCommand: "tmux attach",
		RequestTTY:    "force",
	}
	if err := config.AddSSHHostToFile(original, path); err != nil {
		t.Fatal(err)
	}

	form, err := NewEditForm("web-01", NewStyles(80), 80, 24, path)
	if err != nil {
		t.Fatal(err)
	}

	// Change only a visible field, as a user editing the hostname would.
	form.inputs[0].SetValue("10.0.0.5")

	if msg, ok := form.submitEditForm()().(editFormSubmitMsg); !ok || msg.err != nil {
		t.Fatalf("submit failed: %+v", msg)
	}

	saved, err := config.GetSSHHostFromFile("web-01", path)
	if err != nil {
		t.Fatal(err)
	}

	if saved.Hostname != "10.0.0.5" {
		t.Errorf("hostname is %q, want the edited value", saved.Hostname)
	}
	for _, field := range []struct{ name, got, want string }{
		{"Options", saved.Options, original.Options},
		{"RemoteCommand", saved.RemoteCommand, original.RemoteCommand},
		{"RequestTTY", saved.RequestTTY, original.RequestTTY},
	} {
		if field.got != field.want {
			t.Errorf("%s is %q after edit, want %q", field.name, field.got, field.want)
		}
	}
}

// TestTagColoursAreUniqueAndThemed checks that every distinct tag gets its own
// colour. The previous scheme hashed tags into four fixed theme colours, so a
// fifth tag necessarily shared with an earlier one.
func TestTagColoursAreUniqueAndThemed(t *testing.T) {
	original := CurrentThemeIndex
	defer SetTheme(original)

	tags := []string{"prod", "web", "db", "infra", "ci", "stg", "k8s", "eu", "us", "edge", "batch", "cache"}

	for themeIndex, theme := range Themes {
		SetTheme(themeIndex)

		var hosts []config.SSHHost
		for i, tag := range tags {
			hosts = append(hosts, config.SSHHost{
				Name: fmt.Sprintf("host-%d", i),
				Tags: []string{tag},
			})
		}

		m := Model{hosts: hosts, searchInput: textinput.New()}
		m.rebuildEntries()

		seen := make(map[string]string, len(tags))
		for _, tag := range tags {
			colour := m.tagColor(tag)

			if owner, taken := seen[colour]; taken {
				t.Errorf("%s: tags %q and %q share colour %s", theme.Name, owner, tag, colour)
			}
			seen[colour] = tag

			// Every colour must lie within the box bounded by the theme's
			// own anchors: a blend of theme colours, not an invented one.
			if !withinAnchors(colour, theme.anchors()) {
				t.Errorf("%s: tag %q colour %s falls outside the theme's colours %v",
					theme.Name, tag, colour, theme.anchors())
			}
		}
	}
}

// withinAnchors reports whether a colour's channels all fall inside the range
// spanned by the given colours.
func withinAnchors(colour string, anchors []string) bool {
	channels := func(hex string) [3]int {
		var c [3]int
		fmt.Sscanf(hex, "#%02x%02x%02x", &c[0], &c[1], &c[2])
		return c
	}

	got := channels(colour)

	var low, high [3]int
	for i := range low {
		low[i], high[i] = 255, 0
	}
	for _, anchor := range anchors {
		c := channels(anchor)
		for i := range c {
			low[i] = min(low[i], c[i])
			high[i] = max(high[i], c[i])
		}
	}

	for i := range got {
		if got[i] < low[i] || got[i] > high[i] {
			return false
		}
	}
	return true
}

// TestItalicFollowsTerminalCapability checks the wordmark only asks for italic
// where the terminal declares it. screen-256color and the Linux console do not
// support it, and some terminals show reverse video instead of slanted text.
func TestItalicFollowsTerminalCapability(t *testing.T) {
	restore := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(restore)

	// supportsItalic caches its answer, so each case starts from a clean once.
	defer func() { italicOnce.Once = sync.Once{} }()

	for _, probe := range []struct {
		term  string
		want  bool
		notes string
	}{
		{"xterm-256color", true, "declares sitm"},
		{"tmux-256color", true, "declares sitm"},
		{"screen-256color", false, "no sitm: tmux under the wrong TERM"},
		{"linux", false, "no sitm: the console"},
		{"no-such-terminal-xyz", true, "unknown: assumed capable"},
	} {
		t.Setenv("TERM", probe.term)
		italicOnce.Once = sync.Once{}

		if got := supportsItalic(); got != probe.want {
			t.Errorf("TERM=%s: supportsItalic()=%v, want %v (%s)",
				probe.term, got, probe.want, probe.notes)
		}

		// SGR 3 is the italic attribute; it must be absent when unsupported.
		hasItalic := strings.Contains(wordmark(), ";3;")
		if hasItalic != probe.want {
			t.Errorf("TERM=%s: italic in output=%v, want %v", probe.term, hasItalic, probe.want)
		}
	}
}

// TestFormNavigationStopsAtEnds checks that holding a direction runs out at
// the first and last field instead of wrapping around to the other end.
func TestFormNavigationStopsAtEnds(t *testing.T) {
	styles := NewStyles(80)

	t.Run("add", func(t *testing.T) {
		form := NewAddForm("", styles, 80, 24, "")

		press := func(key tea.KeyType, times int) {
			for i := 0; i < times; i++ {
				form, _ = form.Update(tea.KeyMsg{Type: key})
			}
		}

		press(tea.KeyDown, len(form.inputs)+5)
		if want := len(form.inputs) - 1; form.focused != want {
			t.Errorf("after running past the end, focus is %d, want %d", form.focused, want)
		}

		press(tea.KeyUp, len(form.inputs)+5)
		if form.focused != 0 {
			t.Errorf("after running past the start, focus is %d, want 0", form.focused)
		}
	})

	t.Run("k8s", func(t *testing.T) {
		form := NewK8sAddForm(styles, 80, 24)

		for i := 0; i < len(form.inputs)+5; i++ {
			form.handleNavigation("down")
		}
		if want := len(form.inputs) - 1; form.focused != want {
			t.Errorf("after running past the end, focus is %d, want %d", form.focused, want)
		}

		for i := 0; i < len(form.inputs)+5; i++ {
			form.handleNavigation("up")
		}
		if form.focused != 0 {
			t.Errorf("after running past the start, focus is %d, want 0", form.focused)
		}
	})

	t.Run("theme picker", func(t *testing.T) {
		original := CurrentThemeIndex
		defer SetTheme(original)

		picker := NewThemePicker(styles, 80, 24, nil)
		for i := 0; i < len(Themes)+5; i++ {
			picker, _ = picker.Update(tea.KeyMsg{Type: tea.KeyDown})
		}
		if want := len(Themes) - 1; picker.selectedIndex != want {
			t.Errorf("after running past the end, selection is %d, want %d", picker.selectedIndex, want)
		}

		for i := 0; i < len(Themes)+5; i++ {
			picker, _ = picker.Update(tea.KeyMsg{Type: tea.KeyUp})
		}
		if picker.selectedIndex != 0 {
			t.Errorf("after running past the start, selection is %d, want 0", picker.selectedIndex)
		}
	})
}
