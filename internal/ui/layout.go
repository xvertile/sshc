package ui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/xo/terminfo"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// This file defines the chrome shared by every screen: a status line at the
// top, a body that absorbs all remaining height, and key hints pinned to the
// bottom. Screens differ only in what they put in the body, so vertical
// budgeting lives here rather than being re-guessed per view.

// listChrome records which parts of the list view's chrome a terminal is tall
// enough to afford. On a short terminal the decoration is given up so that
// host rows stay visible, rather than the screen refusing to render.
type listChrome struct {
	box   bool // full border around the screen
	hints bool // the key hints, in their own section
}

// chromeFor picks the chrome for a given terminal height.
//
// Decoration is surrendered before information. The border goes first, since
// it costs two lines and carries nothing; then the hints; the status and
// filter lines are the last to go, because they are the only way to tell what
// is shown and to narrow it.
func chromeFor(height int) listChrome {
	switch {
	case height >= 10:
		return listChrome{box: true, hints: true}
	case height >= 8:
		return listChrome{box: true}
	case height >= 5:
		return listChrome{hints: true}
	default:
		return listChrome{}
	}
}

// sections counts the bordered sections the list is divided into: the header,
// the rows, and the key hints.
func (c listChrome) sections() int {
	total := 2 // header and rows

	if c.hints {
		total++
	}

	return total
}

// lines counts the rows this chrome occupies, including the column header the
// table draws itself. renderListView and updateTableHeight both derive their
// budget from this, so they cannot disagree.
func (c listChrome) lines() int {
	total := 2 // header line, plus the table's own column header

	if c.hints {
		total++
	}

	if c.box {
		// Two border edges, and a divider between each pair of sections.
		total += 2 + c.sections() - 1
	}

	return total
}

// formChromeLines is the non-field overhead of a form panel: two border edges,
// a title, two dividers, and the key hints.
const formChromeLines = 6

// sep separates segments within a status line or hint row.
const sep = " · "

// formLabelWidth is the column that form inputs align on, shared across every
// form so switching between them does not shift the fields.
const formLabelWidth = 16

// clearInputPrompt removes a text input's built-in prompt. formField already
// marks the focused row with a caret, so the default "> " is a second marker
// saying the same thing.
func clearInputPrompt(input textinput.Model) textinput.Model {
	input.Prompt = ""
	return input
}

// fitInputs sizes text inputs to the space remaining after the caret and the
// label column, so a form adapts to the terminal instead of being clipped at
// the right edge.
func fitInputs(inputs []textinput.Model, width int) {
	available := width - formLabelWidth - 4

	const minInputWidth = 10
	if available < minInputWidth {
		available = minInputWidth
	}

	for i := range inputs {
		inputs[i].Width = available
	}
}

// panelPadding is the space between the border and the content on each side,
// so panelChrome columns are unavailable to content.
const (
	panelPadding = 1
	panelChrome  = 2 * (1 + panelPadding) // border plus padding, both sides
)

// contentWidth returns the columns available inside a panel of a given width.
func contentWidth(width int) int {
	inner := width - panelChrome
	if inner < 1 {
		return 1
	}
	return inner
}

// panel frames sections in a full border, with a divider between each one.
//
//	╭──────────────────────────────╮
//	│ sshc          8 hosts · name │
//	├──────────────────────────────┤
//	│ NAME        HOSTNAME         │
//	│ ● web-01    10.0.0.4         │
//	├──────────────────────────────┤
//	│ ↵ connect          q quit    │
//	╰──────────────────────────────╯
//
// The section at flexIndex absorbs whatever height is left over, so the
// bottom edge always lands on the last line of the terminal.
func panel(width, height int, sections [][]string, flexIndex int) string {
	frame := lipgloss.NewStyle().Foreground(lipgloss.Color(GetCurrentTheme().Secondary))

	span := width - 2
	if span < 0 {
		span = 0
	}

	edge := func(open, close string) string {
		return frame.Render(open + strings.Repeat("─", span) + close)
	}

	inner := contentWidth(width)
	pad := strings.Repeat(" ", panelPadding)
	side := frame.Render("│")

	row := func(content string) string {
		return side + pad + fitLine(content, inner) + pad + side
	}

	// Everything except the flexible section is fixed, so the leftover is
	// what that section gets.
	fixed := 2 + (len(sections) - 1) // edges and dividers
	for i, section := range sections {
		if i != flexIndex {
			fixed += len(section)
		}
	}

	flex := height - fixed
	if flex < 0 {
		flex = 0
	}

	lines := []string{edge("╭", "╮")}

	for i, section := range sections {
		if i > 0 {
			lines = append(lines, edge("├", "┤"))
		}

		if i == flexIndex {
			section = resize(section, flex)
		}
		for _, content := range section {
			lines = append(lines, row(content))
		}
	}

	lines = append(lines, edge("╰", "╯"))

	return strings.Join(resize(lines, height), "\n")
}

// resize pads a slice with blanks or clips it to exactly n entries.
func resize(lines []string, n int) []string {
	if n < 0 {
		n = 0
	}
	if len(lines) > n {
		return lines[:n]
	}
	for len(lines) < n {
		lines = append(lines, "")
	}
	return lines
}

// fitLine pads or clips a line to exactly width columns, so a panel's right
// border stays flush no matter what the content does.
func fitLine(content string, width int) string {
	used := lipgloss.Width(content)

	switch {
	case used > width:
		return lipgloss.NewStyle().MaxWidth(width).Render(content)
	case used < width:
		return content + strings.Repeat(" ", width-used)
	default:
		return content
	}
}

// rule renders a full-width horizontal line, used where a screen is not
// framed.
func rule(width int) string {
	if width <= 0 {
		return ""
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(GetCurrentTheme().Muted)).
		Render(strings.Repeat("─", width))
}

// max returns the larger of two ints.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// statusLine places left and right segments on a single line of exactly the
// given width, truncating the left segment if the two would collide.
func statusLine(width int, left, right string) string {
	if width <= 0 {
		return ""
	}

	rightWidth := lipgloss.Width(right)
	if rightWidth >= width {
		return lipgloss.NewStyle().MaxWidth(width).Render(right)
	}

	available := width - rightWidth
	if right != "" {
		available-- // keep at least one space between the segments
	}
	left = lipgloss.NewStyle().MaxWidth(available).Render(left)

	gap := width - lipgloss.Width(left) - rightWidth
	if gap < 0 {
		gap = 0
	}
	return left + strings.Repeat(" ", gap) + right
}

// italicOnce guards the one-time terminfo lookup behind supportsItalic.
var italicOnce struct {
	sync.Once
	supported bool
}

// supportsItalic reports whether the terminal declares the italics capability
// (terminfo's sitm).
//
// Not every terminal has it: screen-256color and the Linux console do not, and
// a terminal without it may show reverse video instead of slanted text. When
// the terminfo database cannot be read at all the answer is assumed to be yes,
// since that means an unusual environment rather than a known-incapable one.
func supportsItalic() bool {
	italicOnce.Do(func() {
		info, err := terminfo.LoadFromEnv()
		if err != nil {
			italicOnce.supported = true
			return
		}

		italicOnce.supported = len(info.Strings[terminfo.EnterItalicsMode]) > 0
	})

	return italicOnce.supported
}

// wordmark renders the application name, set in bold and in italic where the
// terminal supports it.
func wordmark() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(GetCurrentTheme().Primary)).
		Bold(true).
		Italic(supportsItalic()).
		Render("sshc")
}

// fitSegments joins segments with the separator, dropping whole trailing
// segments that would not fit rather than truncating one mid-word.
func fitSegments(width int, segments ...string) string {
	var b strings.Builder
	used := 0

	for _, segment := range segments {
		cost := lipgloss.Width(segment)

		prefix := ""
		if used > 0 {
			prefix = muted(sep)
			cost += len([]rune(sep))
		}

		if used+cost > width {
			break
		}

		b.WriteString(prefix)
		b.WriteString(segment)
		used += cost
	}

	return b.String()
}

// keyHint is a single "key action" pair shown in the footer.
type keyHint struct {
	key    string
	action string
}

// keyHints renders footer hints on one line, dropping hints that do not fit
// rather than wrapping onto a second line and breaking the height budget.
func keyHints(width int, hints ...keyHint) string {
	theme := GetCurrentTheme()
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Accent)).Bold(true)
	actionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Muted))

	var b strings.Builder
	used := 0

	for _, h := range hints {
		segment := keyStyle.Render(h.key) + " " + actionStyle.Render(h.action)
		cost := lipgloss.Width(segment)

		prefix := ""
		if used > 0 {
			prefix = actionStyle.Render(sep)
			cost += len([]rune(sep))
		}

		if used+cost > width {
			break
		}

		b.WriteString(prefix)
		b.WriteString(segment)
		used += cost
	}

	return b.String()
}

// screen composes header, body and footer into exactly height lines, padding
// or clipping the body so the footer always lands on the final line.
func screen(width, height int, header, body, footer string) string {
	headerHeight := 0
	if header != "" {
		headerHeight = lipgloss.Height(header)
	}

	footerHeight := 0
	if footer != "" {
		footerHeight = lipgloss.Height(footer)
	}

	bodyHeight := height - headerHeight - footerHeight
	if bodyHeight < 0 {
		bodyHeight = 0
	}

	var lines []string
	if body != "" {
		lines = strings.Split(body, "\n")
	}
	if len(lines) > bodyHeight {
		lines = lines[:bodyHeight]
	}
	for len(lines) < bodyHeight {
		lines = append(lines, "")
	}

	// Clip overlong lines rather than letting the terminal wrap them: a
	// wrapped line pushes everything below it down and breaks the height
	// budget the rest of this function just balanced.
	clip := lipgloss.NewStyle().MaxWidth(width)
	for i, line := range lines {
		if lipgloss.Width(line) > width {
			lines[i] = clip.Render(line)
		}
	}

	parts := make([]string, 0, 3)
	if headerHeight > 0 {
		parts = append(parts, header)
	}
	if bodyHeight > 0 {
		parts = append(parts, strings.Join(lines, "\n"))
	}
	if footerHeight > 0 {
		parts = append(parts, footer)
	}

	return strings.Join(parts, "\n")
}

// formField renders one labelled input row, marking the focused field with a
// caret so focus is readable without relying on colour alone.
func formField(label string, required, focused bool, input string, labelWidth int) string {
	theme := GetCurrentTheme()

	if required {
		label += lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Error)).Render("*")
	}

	marker := "  "
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Muted)).Width(labelWidth)

	if focused {
		marker = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Primary)).Bold(true).Render("▸ ")
		labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Primary)).Bold(true).Width(labelWidth)
	}

	return marker + labelStyle.Render(label) + input
}

// toggleRow renders a set of mutually exclusive options on one line, marking
// the selected one. Used where a form would otherwise spend three lines on a
// label, a button row and a hint.
func toggleRow(options []string, selected int) string {
	theme := GetCurrentTheme()

	activeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Background)).
		Background(lipgloss.Color(theme.Primary)).
		Bold(true).
		Padding(0, 1)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Muted)).
		Padding(0, 1)

	rendered := make([]string, 0, len(options))
	for i, option := range options {
		if i == selected {
			rendered = append(rendered, activeStyle.Render(option))
		} else {
			rendered = append(rendered, inactiveStyle.Render(option))
		}
	}

	return strings.Join(rendered, " ")
}

// formTitle renders the screen title shown in a form's status line.
func formTitle(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(GetCurrentTheme().Primary)).
		Bold(true).
		Render(text)
}

// formError renders an inline error message for a form body.
func formError(message string) string {
	if message == "" {
		return ""
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(GetCurrentTheme().Error)).
		Render("! " + message)
}

// noFocusLine tells formScreenAt that a body has no focused row to keep in
// view, so it may simply be clipped.
const noFocusLine = -1

// formScreen wraps a form body in the standard chrome: title, rules, hints.
func formScreen(width, height int, title, body, errMessage string, hints ...keyHint) string {
	return formScreenAt(width, height, title, body, errMessage, noFocusLine, hints...)
}

// formScreenAt is formScreen for bodies that can outgrow the terminal. The
// body is scrolled so the line at focusLine stays visible, rather than the
// form refusing to render on a short terminal.
func formScreenAt(width, height int, title, body, errMessage string, focusLine int, hints ...keyHint) string {
	if errMessage != "" {
		body += "\n\n" + formError(errMessage)
	}

	inner := contentWidth(width)

	lines := strings.Split(body, "\n")
	available := height - formChromeLines

	scrolled, above, below := windowLines(lines, focusLine, available)

	// Off-screen rows are reported in the title's right slot, so a scrolled
	// form never looks like a truncated one.
	position := ""
	switch {
	case above > 0 && below > 0:
		position = fmt.Sprintf("↑%d ↓%d", above, below)
	case above > 0:
		position = fmt.Sprintf("↑%d", above)
	case below > 0:
		position = fmt.Sprintf("↓%d", below)
	}

	// A form is the same panel as the list: title, body, hints.
	sections := [][]string{
		{statusLine(inner, formTitle(title), muted(position))},
		scrolled,
	}

	if len(hints) > 0 {
		sections = append(sections, []string{keyHints(inner, hints...)})
	}

	return panel(width, height, sections, 1)
}

// windowLines returns the slice of lines that fits in available rows while
// keeping focusLine visible, plus how many lines fell off each end.
func windowLines(lines []string, focusLine, available int) (window []string, above, below int) {
	if available < 1 {
		available = 1
	}
	if len(lines) <= available {
		return lines, 0, 0
	}

	start := 0
	if focusLine >= 0 {
		// Keep the focused line centred once it would leave the window.
		if focusLine >= available {
			start = focusLine - available + 1
		}
		if max := len(lines) - available; start > max {
			start = max
		}
	}

	return lines[start : start+available], start, len(lines) - start - available
}

// muted renders secondary text in the theme's muted colour.
func muted(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(GetCurrentTheme().Muted)).
		Render(text)
}

// accent renders text in the theme's accent colour.
func accent(text string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(GetCurrentTheme().Accent)).
		Render(text)
}
