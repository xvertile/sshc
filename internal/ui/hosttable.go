package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// hostTable renders the host list.
//
// It replaces bubbles/table, which passes every cell through
// runewidth.Truncate — an ANSI-unaware call that slices through escape
// sequences and leaves the codes on screen as literal text. That makes
// per-cell colour impossible there. Owning the render is what allows the
// status glyphs, tag colours, row striping and a selection bar that reaches
// the full width of the terminal.
//
// Cells arrive pre-styled and already padded to their column width by
// padCell, so this type only stacks them.
type hostTable struct {
	columns []tableColumn
	rows    []tableRow
	cursor  int
	offset  int // index of the first visible row
	height  int // total lines drawn, header included
	width   int
	focused bool

	headerStyle lipgloss.Style
	emptyStyle  lipgloss.Style
}

// tableColumn is one column heading and its width.
type tableColumn struct {
	Title string
	Width int
}

// tableRow holds one row's cells, pre-styled and padded to column width.
type tableRow []string

// SetColumns replaces the column set.
func (t *hostTable) SetColumns(columns []tableColumn) {
	t.columns = columns
}

// Columns returns the current columns.
func (t *hostTable) Columns() []tableColumn {
	return t.columns
}

// SetRows replaces the rows, keeping the cursor inside the new set.
func (t *hostTable) SetRows(rows []tableRow) {
	t.rows = rows
	t.clampCursor()
}

// Rows returns the current rows.
func (t *hostTable) Rows() []tableRow {
	return t.rows
}

// SetHeight sets the total number of lines the table draws, header included.
func (t *hostTable) SetHeight(height int) {
	t.height = height
	t.clampCursor()
}

// SetWidth records the terminal width available to the table.
func (t *hostTable) SetWidth(width int) {
	t.width = width
}

// Cursor returns the selected row index.
func (t *hostTable) Cursor() int {
	return t.cursor
}

// SetCursor selects a row by index.
func (t *hostTable) SetCursor(cursor int) {
	t.cursor = cursor
	t.clampCursor()
}

// Focus marks the table as accepting key input.
func (t *hostTable) Focus() {
	t.focused = true
}

// Blur marks the table as not accepting key input.
func (t *hostTable) Blur() {
	t.focused = false
}

// Focused reports whether the table has focus.
func (t *hostTable) Focused() bool {
	return t.focused
}

// visibleRows is the number of data rows that fit below the header.
func (t *hostTable) visibleRows() int {
	rows := t.height - 1 // the header claims one line
	if rows < 0 {
		return 0
	}
	return rows
}

// clampCursor keeps the cursor within the rows and scrolls the window so the
// cursor stays visible.
func (t *hostTable) clampCursor() {
	if t.cursor >= len(t.rows) {
		t.cursor = len(t.rows) - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}

	visible := t.visibleRows()
	if visible <= 0 {
		t.offset = 0
		return
	}

	if t.cursor < t.offset {
		t.offset = t.cursor
	}
	if t.cursor >= t.offset+visible {
		t.offset = t.cursor - visible + 1
	}

	// Never leave a gap at the bottom while rows remain above.
	if maxOffset := len(t.rows) - visible; t.offset > maxOffset {
		t.offset = maxOffset
	}
	if t.offset < 0 {
		t.offset = 0
	}
}

// MoveUp moves the selection towards the top of the list.
func (t *hostTable) MoveUp(n int) {
	t.cursor -= n
	t.clampCursor()
}

// MoveDown moves the selection towards the bottom of the list.
func (t *hostTable) MoveDown(n int) {
	t.cursor += n
	t.clampCursor()
}

// Update handles navigation keys. It mirrors the bindings bubbles/table
// offered, so muscle memory carries over.
//
// Focus is not checked here: the caller decides whether keys belong to the
// table or the search input, and checking again would silently swallow
// navigation whenever the two disagreed.
func (t hostTable) Update(msg tea.Msg) (hostTable, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return t, nil
	}

	switch key.String() {
	case "up", "k", "ctrl+p":
		t.MoveUp(1)
	case "down", "j", "ctrl+n":
		t.MoveDown(1)
	case "pgup":
		t.MoveUp(t.visibleRows())
	case "pgdown":
		t.MoveDown(t.visibleRows())
	case "home", "g":
		t.MoveUp(len(t.rows))
	case "end", "G":
		t.MoveDown(len(t.rows))
	}

	return t, nil
}

// View renders the header and the visible rows, padded to exactly the height
// set by SetHeight so the surrounding layout can rely on it.
func (t *hostTable) View() string {
	lines := make([]string, 0, t.height)
	lines = append(lines, t.headerView())

	visible := t.visibleRows()

	switch {
	case len(t.rows) == 0 && visible > 0:
		lines = append(lines, t.emptyStyle.Render("  no hosts match"))

	default:
		end := t.offset + visible
		if end > len(t.rows) {
			end = len(t.rows)
		}
		for _, row := range t.rows[t.offset:end] {
			lines = append(lines, strings.Join(row, ""))
		}
	}

	for len(lines) < t.height {
		lines = append(lines, "")
	}
	if len(lines) > t.height {
		lines = lines[:t.height]
	}

	return strings.Join(lines, "\n")
}

// headerView renders the column headings.
func (t *hostTable) headerView() string {
	var b strings.Builder

	// The same gutters the cells use, so headings sit over their columns.
	for i, column := range t.columns {
		b.WriteString(t.headerStyle.
			Width(column.Width).
			MaxWidth(column.Width).
			Render(strings.Repeat(" ", gutterFor(i)) + column.Title))
	}

	return b.String()
}
