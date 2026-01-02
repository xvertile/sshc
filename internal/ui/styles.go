package ui

import "github.com/charmbracelet/lipgloss"

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
		Muted:       "#475569",
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
		Muted:       "#6272A4",
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
		Muted:       "#434C5E",
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
		Muted:       "#75715E",
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
		Muted:       "#586E75",
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
		Muted:       "#7C6F64",
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
		Muted:       "#565F89",
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
		Muted:       "#6C7086",
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
		Muted:       "#4B5263",
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
		Muted:       "#696969",
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
		Muted:       "#365961",
	},
}

// Current theme colors (set by SetTheme)
var (
	PrimaryColor   = "#3B82F6"
	SecondaryColor = "#64748B"
	ErrorColor     = "#EF4444"
	SuccessColor   = "#22C55E"
)

// CurrentThemeIndex tracks the active theme
var CurrentThemeIndex = 0

// SetTheme applies a theme by index
func SetTheme(index int) {
	if index < 0 || index >= len(Themes) {
		return
	}
	CurrentThemeIndex = index
	theme := Themes[index]
	PrimaryColor = theme.Primary
	SecondaryColor = theme.Secondary
	ErrorColor = theme.Error
	SuccessColor = theme.Success
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

// Styles struct centralizes all lipgloss styles
type Styles struct {
	// Layout
	App    lipgloss.Style
	Header lipgloss.Style

	// Search styles
	SearchFocused   lipgloss.Style
	SearchUnfocused lipgloss.Style

	// Table styles
	TableFocused   lipgloss.Style
	TableUnfocused lipgloss.Style
	Selected       lipgloss.Style

	// Info and help styles
	SortInfo lipgloss.Style
	HelpText lipgloss.Style

	// Error and confirmation styles
	Error     lipgloss.Style
	ErrorText lipgloss.Style

	// Form styles (for add/edit forms)
	FormTitle     lipgloss.Style
	FormField     lipgloss.Style
	FormHelp      lipgloss.Style
	FormContainer lipgloss.Style
	Label         lipgloss.Style
	FocusedLabel  lipgloss.Style
	HelpSection   lipgloss.Style

	// Tab styles (for toggle buttons)
	ActiveTab   lipgloss.Style
	InactiveTab lipgloss.Style

	// File browser styles
	DirStyle lipgloss.Style

	// Theme picker styles
	ThemeItem         lipgloss.Style
	ThemeItemSelected lipgloss.Style
	ThemePreview      lipgloss.Style
}

// NewStyles creates a new Styles struct with the given terminal width
func NewStyles(width int) Styles {
	theme := GetCurrentTheme()

	return Styles{
		// Main app container
		App: lipgloss.NewStyle().
			Padding(1, 2), // More horizontal breathing room

		// Header style
		Header: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Primary)).
			Bold(true).
			Align(lipgloss.Center).
			PaddingBottom(1),

		// Search styles
		SearchFocused: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Primary)).
			Foreground(lipgloss.Color(theme.Foreground)).
			Padding(0, 1),

		SearchUnfocused: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Muted)).
			Foreground(lipgloss.Color(theme.Secondary)).
			Padding(0, 1),

		// Table styles
		TableFocused: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Primary)),

		TableUnfocused: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Muted)),

		// Style for selected items - Clean modern highlight
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.SelectionFg)).
			Background(lipgloss.Color(theme.SelectionBg)).
			Bold(true),

		// Info styles
		SortInfo: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Secondary)).
			Italic(true),

		HelpText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Muted)).
			PaddingTop(1),

		// Error style
		Error: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Error)).
			Padding(0, 1).
			MarginTop(1),

		// Error text style
		ErrorText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Error)).
			Bold(true),

		// Form styles
		FormTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Background)).
			Background(lipgloss.Color(theme.Primary)).
			Bold(true).
			Padding(0, 1),

		FormField: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Foreground)),

		FormHelp: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Muted)).
			Italic(true),

		FormContainer: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Muted)).
			Padding(1, 2),

		Label: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Secondary)),

		FocusedLabel: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Primary)).
			Bold(true),

		HelpSection: lipgloss.NewStyle().
			Padding(1, 2),

		// Tab styles
		ActiveTab: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Background)).
			Background(lipgloss.Color(theme.Primary)).
			Padding(0, 2).
			Bold(true),

		InactiveTab: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Secondary)).
			Background(lipgloss.Color(theme.SelectionBg)). // Slight background for depth
			Padding(0, 2),

		DirStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Accent)).
			Bold(true),

		// Theme picker styles
		ThemeItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Foreground)).
			Padding(0, 2),

		ThemeItemSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Primary)).
			Border(lipgloss.NormalBorder(), false, false, false, true). // Left border only
			BorderForeground(lipgloss.Color(theme.Primary)).
			Padding(0, 1),

		ThemePreview: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Secondary)).
			Padding(1, 2),
	}
}

// Application ASCII title
const asciiTitle = `
              __        
   __________/ /_  _____
  / ___/ ___/ __ \/ ___/
 (__  |__  ) / / / /__  
/____/____/_/ /_/\___/  
                        
`
