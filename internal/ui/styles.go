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
		Primary:     "#00ADD8", // Go blue
		Secondary:   "240",
		Accent:      "#00ADD8",
		Error:       "1",
		Success:     "36",
		Background:  "",
		Foreground:  "",
		SelectionBg: "#00ADD8",
		SelectionFg: "229",
		Muted:       "#626262",
	},
	{
		Name:        "Dracula",
		Primary:     "#BD93F9", // Purple
		Secondary:   "#6272A4",
		Accent:      "#FF79C6", // Pink
		Error:       "#FF5555",
		Success:     "#50FA7B",
		Background:  "#282A36",
		Foreground:  "#F8F8F2",
		SelectionBg: "#44475A",
		SelectionFg: "#F8F8F2",
		Muted:       "#6272A4",
	},
	{
		Name:        "Nord",
		Primary:     "#88C0D0", // Frost
		Secondary:   "#4C566A",
		Accent:      "#81A1C1",
		Error:       "#BF616A",
		Success:     "#A3BE8C",
		Background:  "#2E3440",
		Foreground:  "#ECEFF4",
		SelectionBg: "#434C5E",
		SelectionFg: "#ECEFF4",
		Muted:       "#4C566A",
	},
	{
		Name:        "Monokai",
		Primary:     "#F92672", // Pink
		Secondary:   "#75715E",
		Accent:      "#A6E22E", // Green
		Error:       "#F92672",
		Success:     "#A6E22E",
		Background:  "#272822",
		Foreground:  "#F8F8F2",
		SelectionBg: "#49483E",
		SelectionFg: "#F8F8F2",
		Muted:       "#75715E",
	},
	{
		Name:        "Solarized",
		Primary:     "#268BD2", // Blue
		Secondary:   "#586E75",
		Accent:      "#2AA198", // Cyan
		Error:       "#DC322F",
		Success:     "#859900",
		Background:  "#002B36",
		Foreground:  "#839496",
		SelectionBg: "#073642",
		SelectionFg: "#93A1A1",
		Muted:       "#586E75",
	},
	{
		Name:        "Gruvbox",
		Primary:     "#FE8019", // Orange
		Secondary:   "#928374",
		Accent:      "#FABD2F", // Yellow
		Error:       "#FB4934",
		Success:     "#B8BB26",
		Background:  "#282828",
		Foreground:  "#EBDBB2",
		SelectionBg: "#3C3836",
		SelectionFg: "#EBDBB2",
		Muted:       "#928374",
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
		SelectionBg: "#283457",
		SelectionFg: "#C0CAF5",
		Muted:       "#565F89",
	},
	{
		Name:        "Catppuccin",
		Primary:     "#CBA6F7", // Mauve
		Secondary:   "#6C7086",
		Accent:      "#F5C2E7", // Pink
		Error:       "#F38BA8",
		Success:     "#A6E3A1",
		Background:  "#1E1E2E",
		Foreground:  "#CDD6F4",
		SelectionBg: "#313244",
		SelectionFg: "#CDD6F4",
		Muted:       "#6C7086",
	},
	{
		Name:        "One Dark",
		Primary:     "#61AFEF", // Blue
		Secondary:   "#5C6370",
		Accent:      "#C678DD", // Purple
		Error:       "#E06C75",
		Success:     "#98C379",
		Background:  "#282C34",
		Foreground:  "#ABB2BF",
		SelectionBg: "#3E4451",
		SelectionFg: "#ABB2BF",
		Muted:       "#5C6370",
	},
	{
		Name:        "Cyberpunk",
		Primary:     "#00FFFF", // Cyan
		Secondary:   "#FF00FF",
		Accent:      "#FFFF00", // Yellow
		Error:       "#FF0000",
		Success:     "#00FF00",
		Background:  "#0D0D0D",
		Foreground:  "#FFFFFF",
		SelectionBg: "#FF00FF",
		SelectionFg: "#000000",
		Muted:       "#808080",
	},
}

// Current theme colors (set by SetTheme)
var (
	PrimaryColor   = "#00ADD8"
	SecondaryColor = "240"
	ErrorColor     = "1"
	SuccessColor   = "36"
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
			Padding(1),

		// Header style
		Header: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Primary)).
			Bold(true).
			Align(lipgloss.Center),

		// Search styles
		SearchFocused: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Primary)).
			Padding(0, 1),

		SearchUnfocused: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Secondary)).
			Padding(0, 1),

		// Table styles
		TableFocused: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(theme.Primary)),

		TableUnfocused: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(theme.Secondary)),

		// Style for selected items
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.SelectionFg)).
			Background(lipgloss.Color(theme.SelectionBg)).
			Bold(false),

		// Info styles
		SortInfo: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Secondary)),

		HelpText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Secondary)),

		// Error style
		Error: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Error)).
			Padding(1, 2),

		// Error text style (no border, just red text)
		ErrorText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Error)).
			Bold(true),

		// Form styles
		FormTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color(theme.Primary)).
			Padding(0, 1),

		FormField: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Primary)),

		FormHelp: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Muted)),

		FormContainer: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Primary)).
			Padding(1, 2),

		Label: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Secondary)),

		FocusedLabel: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Primary)),

		HelpSection: lipgloss.NewStyle().
			Padding(0, 2),

		// Tab styles
		ActiveTab: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color(theme.Primary)).
			Padding(0, 2).
			Bold(true),

		InactiveTab: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Secondary)).
			Background(lipgloss.Color("#333333")).
			Padding(0, 2),

		DirStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Accent)),

		// Theme picker styles
		ThemeItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.Foreground)).
			Padding(0, 2),

		ThemeItemSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.SelectionFg)).
			Background(lipgloss.Color(theme.SelectionBg)).
			Padding(0, 2).
			Bold(true),

		ThemePreview: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(theme.Primary)).
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
