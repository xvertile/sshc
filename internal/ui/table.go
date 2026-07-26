package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Minimum widths below which a column stops being readable.
const (
	minNameWidth     = 12
	minHostnameWidth = 14
	minTagsWidth     = 8
	minLastWidth     = 4
)

// tableColumnCount is the number of columns in the host table.
//
// tableGutter is the blank space separating one column from the next. The
// first column has none, so rows start hard against the left edge in line with
// the status and hint lines. It is drawn as part of the cell so that it carries
// the row background, leaving no gap in the selection bar.
const (
	tableColumnCount = 4
	tableGutter      = 2
)

// gutterFor returns the blank space preceding a column. The first column sits
// flush with the left edge; the rest are separated by tableGutter.
func gutterFor(column int) int {
	if column == 0 {
		return 0
	}
	return tableGutter
}

// tableColumnHeaders are the column titles, also the floor for column widths.
var tableColumnHeaders = [tableColumnCount]string{"NAME", "HOSTNAME", "TAGS", "LAST"}

// columnDropOrder lists columns in the order they are surrendered when the
// terminal is too narrow to show them all: tags first, then last-login, then
// hostname. Name is never dropped, because it is what you select by.
var columnDropOrder = []int{2, 3, 1}

// sortIndicator is appended to whichever column the list is sorted by. Its
// width is reserved on every sortable column so widths do not shift when the
// sort mode changes.
const sortIndicator = " ↓"

// columnTitles returns the column titles with the sort indicator applied.
func (m *Model) columnTitles() [tableColumnCount]string {
	titles := tableColumnHeaders
	switch m.sortMode {
	case SortByName:
		titles[0] += sortIndicator
	case SortByLastUsed:
		titles[3] += sortIndicator
	}
	return titles
}

// formatEntryTags renders an entry's tags as they appear in the table.
func formatEntryTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}

	formatted := make([]string, 0, len(tags))
	for _, tag := range tags {
		formatted = append(formatted, "#"+tag)
	}
	return strings.Join(formatted, " ")
}

// entryRowCells returns the plain, unstyled cell values for an entry. Widths
// are measured from these, so measurement never has to reason about escape
// sequences.
func (m *Model) entryRowCells(entry *HostEntry) [tableColumnCount]string {
	indicator := m.getPingStatusIndicator(entry.Name)
	if entry.IsK8s {
		indicator = k8sIndicator
	}

	last := "—"
	if m.historyManager != nil {
		if lastConnect, exists := m.historyManager.GetLastConnectionTime(entry.Name); exists {
			last = formatTimeCompact(lastConnect)
		}
	}

	return [tableColumnCount]string{
		indicator + " " + entry.Name,
		entry.Hostname,
		formatEntryTags(entry.Tags),
		last,
	}
}

// k8sIndicator marks a Kubernetes entry in the status column.
const k8sIndicator = "⬢"

// styledEntryCells returns the cell values with colour applied: the status
// glyph by connectivity, tags by name, and last-login faded by age.
//
// A row background is applied to every coloured span rather than wrapped
// around the finished row. Wrapping looks correct until a span ends: its reset
// clears the background too, punching holes in the band. Carrying the
// background on each span keeps it continuous.
func (m *Model) styledEntryCells(entry *HostEntry, background string, bold bool) [tableColumnCount]string {
	cells := m.entryRowCells(entry)
	theme := GetCurrentTheme()

	color := func(hex, text string) string {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Bold(bold)
		if background != "" {
			style = style.Background(lipgloss.Color(background))
		}
		return style.Render(text)
	}

	// Status glyph and name: the glyph carries connectivity, the name stays
	// bright so it remains the thing the eye lands on.
	glyph := m.statusColor(entry.Name)
	if entry.IsK8s {
		glyph = theme.Accent
	}

	indicator, name, _ := strings.Cut(cells[0], " ")
	cells[0] = color(glyph, indicator) + color(theme.Foreground, " "+name)

	cells[1] = color(theme.Muted, cells[1])

	if len(entry.Tags) > 0 {
		// Separators take the colour of the tag that follows them, so no part
		// of the tag column is drawn in grey.
		var tags strings.Builder
		for i, tag := range entry.Tags {
			hex := m.tagColor(tag)
			if i > 0 {
				tags.WriteString(color(hex, " "))
			}
			tags.WriteString(color(hex, "#"+tag))
		}
		cells[2] = tags.String()
	}

	lastConnect, exists := time.Time{}, false
	if m.historyManager != nil {
		lastConnect, exists = m.historyManager.GetLastConnectionTime(entry.Name)
	}
	cells[3] = color(recencyColor(lastConnect, exists), cells[3])

	return cells
}

// padCell lays a cell out to exactly width columns, including the gutter that
// separates it from the previous one, filling any slack with the row
// background so a shaded band runs unbroken across the row.
func padCell(content string, width, gutter int, background string) string {
	style := lipgloss.NewStyle()
	if background != "" {
		style = style.Background(lipgloss.Color(background))
	}

	available := width - gutter
	if available < 0 {
		available = 0
	}

	used := lipgloss.Width(content)
	if used > available {
		content = lipgloss.NewStyle().MaxWidth(available).Render(content)
		used = available
	}

	return style.Render(strings.Repeat(" ", gutter)) +
		content +
		style.Render(strings.Repeat(" ", available-used))
}

// computeColumnWidths sizes the columns to exactly fill the terminal width.
//
// Widths are measured with lipgloss.Width rather than len, so multi-byte host
// names and tags no longer skew the layout by their byte count. Columns are
// measured across all entries, not the filtered subset, so they stay put while
// the user types a search.
func (m *Model) computeColumnWidths() (columns []int, widths []int) {
	// Sortable columns reserve room for the indicator whether or not they are
	// currently the sort column, so widths stay stable across sort changes.
	var natural [tableColumnCount]int
	for i, header := range tableColumnHeaders {
		natural[i] = lipgloss.Width(header)
		if i == 0 || i == 3 {
			natural[i] += lipgloss.Width(sortIndicator)
		}
	}

	for i := range m.allEntries {
		cells := m.entryRowCells(&m.allEntries[i])
		for col, cell := range cells {
			if w := lipgloss.Width(cell); w > natural[col] {
				natural[col] = w
			}
		}
	}

	minimums := [tableColumnCount]int{minNameWidth, minHostnameWidth, minTagsWidth, minLastWidth}

	// Start with every column, then give them up in priority order until the
	// survivors fit at their minimum widths. Showing three readable columns
	// beats showing four clipped ones.
	columns = []int{0, 1, 2, 3}

	fits := func(cols []int) bool {
		total := (len(cols) - 1) * tableGutter
		for _, c := range cols {
			total += minimums[c]
		}
		return total <= m.contentWidth()
	}

	for _, drop := range columnDropOrder {
		if fits(columns) {
			break
		}
		kept := columns[:0:0]
		for _, c := range columns {
			if c != drop {
				kept = append(kept, c)
			}
		}
		columns = kept
	}

	available := m.contentWidth() - (len(columns)-1)*tableGutter

	widths = make([]int, len(columns))
	total := 0
	for i, c := range columns {
		widths[i] = natural[c]
		total += widths[i]
	}

	// Everything fits: hand the surplus to the last column. The text inside it
	// stays left-aligned, so this reads as trailing space exactly as before,
	// but the row now spans the full width and the selection bar and row
	// stripes reach the right-hand edge instead of stopping mid-screen.
	if total <= available {
		widths[len(widths)-1] += available - total
		return columns, widths
	}

	// Over budget: shed the excess from the widest column first, but never
	// below its minimum.
	for excess := total - available; excess > 0; excess-- {
		widest, widestIndex := 0, -1
		for i, c := range columns {
			if widths[i] > minimums[c] && widths[i] > widest {
				widest, widestIndex = widths[i], i
			}
		}
		if widestIndex == -1 {
			break
		}
		widths[widestIndex]--
	}

	// Nothing left to give: clamp so bubbles is never handed a negative width.
	for i := range widths {
		if widths[i] < 1 {
			widths[i] = 1
		}
	}

	return columns, widths
}

// updateTableRows rebuilds the table rows from the filtered entries.
func (m *Model) updateTableRows() {
	visible, widths := m.computeColumnWidths()
	cursor := m.table.Cursor()
	theme := GetCurrentTheme()

	rows := make([]tableRow, 0, len(m.filteredEntries))
	for i := range m.filteredEntries {
		// Only the selected row carries a background; the rest are separated
		// by their own colour alone.
		background, bold := "", false
		if i == cursor {
			background, bold = theme.SelectionBg, true
		}

		cells := m.styledEntryCells(&m.filteredEntries[i], background, bold)

		row := make(tableRow, 0, len(visible))
		for column, index := range visible {
			row = append(row, padCell(cells[index], widths[column]+gutterFor(column), gutterFor(column), background))
		}
		rows = append(rows, row)
	}

	m.table.SetRows(rows)

	// Keep the cursor inside the new row set, otherwise selection would point
	// at a stale index after a filter narrows the list.
	if cursor := m.table.Cursor(); cursor >= len(rows) {
		m.table.SetCursor(0)
	}

	m.updateTableHeight()
	m.updateTableColumns()
}

// updateTableHeight sizes the table to fill everything the chrome does not use.
//
// The budget comes from chromeFor, the same source renderListView draws from,
// so the two can never disagree about how many lines are spoken for.
func (m *Model) updateTableHeight() {
	if !m.ready {
		return
	}

	// The chrome count includes the column header the table draws itself, so
	// add it back to get the table's own height.
	height := m.height - chromeFor(m.height).lines() + 1

	const minTableHeight = 1 // the column header alone
	if height < minTableHeight {
		height = minTableHeight
	}

	m.table.SetHeight(height)
}

// updateTableColumns applies freshly computed widths and the sort indicator.
func (m *Model) updateTableColumns() {
	if !m.ready {
		return
	}

	visible, widths := m.computeColumnWidths()
	titles := m.columnTitles()

	columns := make([]tableColumn, 0, len(visible))
	for i, column := range visible {
		columns = append(columns, tableColumn{Title: titles[column], Width: widths[i] + gutterFor(i)})
	}

	m.table.SetColumns(columns)
	m.table.SetWidth(m.contentWidth())
}
