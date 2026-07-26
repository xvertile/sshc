package ui

import (
	"fmt"
	"math"
	"slices"

	"github.com/charmbracelet/lipgloss"
)

// Theme represents a color theme for the application
type Theme struct {
	Name        string
	Primary     string
	Secondary   string
	Accent      string
	Error       string
	Success     string
	Background  string
	Foreground  string
	SelectionBg string
	SelectionFg string
	Muted       string
}

// Available themes
var Themes = []Theme{
	{
		Name:        "Default",
		Primary:     "#3B82F6", // Bright Blue
		Secondary:   "#64748B", // Slate
		Accent:      "#06B6D4", // Cyan
		Error:       "#EF4444", // Red
		Success:     "#22C55E", // Green
		Background:  "#0F172A", // Dark Slate
		Foreground:  "#F1F5F9", // Slate White
		SelectionBg: "#1E293B", // Lighter Slate
		SelectionFg: "#60A5FA", // Light Blue
		Muted:       "#94A3B8",
	},
	{
		Name:        "Dracula",
		Primary:     "#BD93F9", // Purple
		Secondary:   "#6272A4", // Comment Blue
		Accent:      "#FF79C6", // Pink
		Error:       "#FF5555", // Red
		Success:     "#50FA7B", // Green
		Background:  "#282A36",
		Foreground:  "#F8F8F2",
		SelectionBg: "#44475A",
		SelectionFg: "#FF79C6",
		Muted:       "#8894B8",
	},
	{
		Name:        "Nord",
		Primary:     "#88C0D0", // Frost Cyan
		Secondary:   "#4C566A", // Grey
		Accent:      "#81A1C1", // Blue
		Error:       "#BF616A", // Red
		Success:     "#A3BE8C", // Green
		Background:  "#2E3440", // Dark Grey
		Foreground:  "#D8DEE9", // White-ish
		SelectionBg: "#3B4252", // Lighter Grey
		SelectionFg: "#88C0D0",
		Muted:       "#7B88A1",
	},
	{
		Name:        "Monokai",
		Primary:     "#F92672", // Pink
		Secondary:   "#75715E", // Grey
		Accent:      "#66D9EF", // Light Blue
		Error:       "#F92672",
		Success:     "#A6E22E", // Green
		Background:  "#272822",
		Foreground:  "#F8F8F2",
		SelectionBg: "#3E3D32",
		SelectionFg: "#A6E22E",
		Muted:       "#A09B85",
	},
	{
		Name:        "Solarized Dark",
		Primary:     "#268BD2", // Blue
		Secondary:   "#657B83",
		Accent:      "#2AA198", // Cyan
		Error:       "#DC322F", // Red
		Success:     "#859900", // Green
		Background:  "#002B36",
		Foreground:  "#839496",
		SelectionBg: "#073642",
		SelectionFg: "#93A1A1",
		Muted:       "#93A1A1",
	},
	{
		Name:        "Gruvbox",
		Primary:     "#FE8019", // Orange
		Secondary:   "#928374", // Gray
		Accent:      "#FABD2F", // Yellow
		Error:       "#FB4934", // Red
		Success:     "#B8BB26", // Green
		Background:  "#282828",
		Foreground:  "#EBDBB2",
		SelectionBg: "#3C3836",
		SelectionFg: "#FE8019",
		Muted:       "#A89984",
	},
	{
		Name:        "Tokyo Night",
		Primary:     "#7AA2F7", // Blue
		Secondary:   "#565F89",
		Accent:      "#BB9AF7", // Purple
		Error:       "#F7768E",
		Success:     "#9ECE6A",
		Background:  "#1A1B26",
		Foreground:  "#C0CAF5",
		SelectionBg: "#292E42",
		SelectionFg: "#7AA2F7",
		Muted:       "#9AA5CE",
	},
	{
		Name:        "Catppuccin Mocha",
		Primary:     "#CBA6F7", // Mauve
		Secondary:   "#9399B2", // Overlay
		Accent:      "#F5C2E7", // Pink
		Error:       "#F38BA8", // Red
		Success:     "#A6E3A1", // Green
		Background:  "#1E1E2E", // Base
		Foreground:  "#CDD6F4", // Text
		SelectionBg: "#313244", // Surface
		SelectionFg: "#CBA6F7",
		Muted:       "#9399B2",
	},
	{
		Name:        "One Dark",
		Primary:     "#61AFEF", // Blue
		Secondary:   "#5C6370", // Grey
		Accent:      "#C678DD", // Purple
		Error:       "#E06C75", // Red
		Success:     "#98C379", // Green
		Background:  "#282C34",
		Foreground:  "#ABB2BF",
		SelectionBg: "#3E4451",
		SelectionFg: "#61AFEF",
		Muted:       "#7F848E",
	},
	{
		Name:        "Cyberpunk",
		Primary:     "#00E5FF", // Neon Cyan
		Secondary:   "#FF00E5", // Neon Magenta
		Accent:      "#F9F871", // Neon Yellow
		Error:       "#FF2A6D", // Red
		Success:     "#00FF9C", // Green
		Background:  "#050505", // Almost Black
		Foreground:  "#F0F0F0",
		SelectionBg: "#212121",
		SelectionFg: "#00E5FF",
		Muted:       "#A0A0A0",
	},
	{
		Name:        "Teal Ocean",
		Primary:     "#2DD4BF", // Teal 400
		Secondary:   "#5EEAD4", // Teal 300
		Accent:      "#0D9488", // Teal 600
		Error:       "#FDA4AF", // Rose
		Success:     "#6EE7B7", // Emerald
		Background:  "#132F35", // Deep Teal Dark
		Foreground:  "#F0FDFA", // Azure White
		SelectionBg: "#115E59", // Deep Teal
		SelectionFg: "#CCFBF1", // Light Teal
		Muted:       "#6B9CA6",
	},
}

// CurrentThemeIndex tracks the active theme
var CurrentThemeIndex = 0

// SetTheme applies a theme by index
func SetTheme(index int) {
	if index < 0 || index >= len(Themes) {
		return
	}
	CurrentThemeIndex = index
}

// SetThemeByName applies a theme by name
func SetThemeByName(name string) {
	for i, theme := range Themes {
		if theme.Name == name {
			SetTheme(i)
			return
		}
	}
}

// GetCurrentTheme returns the current theme
func GetCurrentTheme() Theme {
	return Themes[CurrentThemeIndex]
}

// anchors returns the theme's saturated colours, in the order they are blended
// through. Greys are left out, because a grey tag reads as disabled, and Error
// is left out because the offline glyph owns that colour.
//
// A colour is rejected by value rather than by which field it came from: some
// themes define a nominally coloured field as grey, as Solarized Dark does
// with SelectionFg.
func (t Theme) anchors() []string {
	grey := []string{t.Muted, t.Secondary}

	palette := make([]string, 0, 4)
	for _, color := range []string{t.Primary, t.Accent, t.Success, t.SelectionFg} {
		if !slices.Contains(grey, color) && !slices.Contains(palette, color) {
			palette = append(palette, color)
		}
	}

	if len(palette) == 0 {
		palette = append(palette, t.Foreground)
	}

	return palette
}

// blend returns a colour a given fraction of the way around the theme's own
// anchor colours, mixing between neighbouring pairs.
//
// Every result therefore lies on a line between two colours the theme already
// uses, so tags stay inside the theme's own range. Rotating the hue wheel
// instead would give Gruvbox — an orange and yellow theme — blue and cyan
// tags, which is a different palette wearing the theme's name.
func (t Theme) blend(fraction float64) string {
	anchors := t.anchors()
	if len(anchors) == 1 {
		return anchors[0]
	}

	// The anchors form a loop, so the last blends back into the first.
	position := math.Mod(fraction, 1) * float64(len(anchors))
	index := int(position)
	weight := position - float64(index)

	return mixHex(anchors[index%len(anchors)], anchors[(index+1)%len(anchors)], weight)
}

// mixHex blends two "#RRGGBB" colours, weight running from 0 (all of from) to
// 1 (all of to).
func mixHex(from, to string, weight float64) string {
	var a, b [3]int
	if _, err := fmt.Sscanf(from, "#%02x%02x%02x", &a[0], &a[1], &a[2]); err != nil {
		return from
	}
	if _, err := fmt.Sscanf(to, "#%02x%02x%02x", &b[0], &b[1], &b[2]); err != nil {
		return from
	}

	mix := func(i int) int {
		return int(math.Round(float64(a[i]) + (float64(b[i])-float64(a[i]))*weight))
	}

	return fmt.Sprintf("#%02X%02X%02X", mix(0), mix(1), mix(2))
}

// rgbToHSL converts a "#RRGGBB" colour to hue in degrees, plus saturation and
// lightness in [0,1]. An unparseable colour yields mid grey.
func rgbToHSL(hex string) (h, s, l float64) {
	var rgb [3]int
	if _, err := fmt.Sscanf(hex, "#%02x%02x%02x", &rgb[0], &rgb[1], &rgb[2]); err != nil {
		return 0, 0, 0.5
	}

	r, g, b := float64(rgb[0])/255, float64(rgb[1])/255, float64(rgb[2])/255
	high := math.Max(r, math.Max(g, b))
	low := math.Min(r, math.Min(g, b))
	span := high - low

	l = (high + low) / 2
	if span == 0 {
		return 0, 0, l // grey: hue is undefined
	}

	if l > 0.5 {
		s = span / (2 - high - low)
	} else {
		s = span / (high + low)
	}

	switch high {
	case r:
		h = (g - b) / span
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/span + 2
	default:
		h = (r-g)/span + 4
	}

	return h * 60, s, l
}

// hslToHex converts hue in degrees plus saturation and lightness in [0,1] to
// a "#RRGGBB" colour.
func hslToHex(h, s, l float64) string {
	if s == 0 {
		channel := int(math.Round(l * 255))
		return fmt.Sprintf("#%02X%02X%02X", channel, channel, channel)
	}

	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q

	h /= 360
	channel := func(t float64) int {
		switch {
		case t < 0:
			t++
		case t > 1:
			t--
		}

		switch {
		case t < 1.0/6:
			return int(math.Round((p + (q-p)*6*t) * 255))
		case t < 1.0/2:
			return int(math.Round(q * 255))
		case t < 2.0/3:
			return int(math.Round((p + (q-p)*(2.0/3-t)*6) * 255))
		default:
			return int(math.Round(p * 255))
		}
	}

	return fmt.Sprintf("#%02X%02X%02X",
		channel(h+1.0/3), channel(h), channel(h-1.0/3))
}

// Styles holds the styles that are threaded through the form constructors.
//
// It once carried two dozen fields. The layout rewrite moved colour into
// layout.go and hosttable.go, which read the active theme directly, and the
// rest became a second, silently diverging definition of the same colours.
type Styles struct {
	// Selected styles the highlighted row of a simple list.
	Selected lipgloss.Style
}

// NewStyles builds the styles for the active theme.
func NewStyles(width int) Styles {
	theme := GetCurrentTheme()

	return Styles{
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.SelectionFg)).
			Background(lipgloss.Color(theme.SelectionBg)).
			Bold(true),
	}
}
