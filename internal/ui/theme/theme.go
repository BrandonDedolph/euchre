package theme

import "github.com/charmbracelet/lipgloss"

// Theme defines the visual styling for the application
type Theme struct {
	// Card colors
	CardRed        lipgloss.Style
	CardBlack      lipgloss.Style
	CardBackground lipgloss.Style
	CardSelected   lipgloss.Style
	CardPlayable   lipgloss.Style
	CardDisabled   lipgloss.Style

	// UI elements
	Primary   lipgloss.Style
	Secondary lipgloss.Style
	Accent    lipgloss.Style
	Muted     lipgloss.Style

	// Status colors
	Success lipgloss.Style
	Warning lipgloss.Style
	Error   lipgloss.Style

	// Layout
	Border       lipgloss.Style
	ScreenBorder lipgloss.Style
	ContentBox   lipgloss.Style
	Title        lipgloss.Style
	Subtitle     lipgloss.Style
	Body         lipgloss.Style
	Help         lipgloss.Style

	// Menu
	MenuItem         lipgloss.Style
	MenuItemSelected lipgloss.Style
	MenuItemDisabled lipgloss.Style

	// Visual lesson elements
	AnnotationLabel  lipgloss.Style
	WinnerHighlight  lipgloss.Style
	LoserDim         lipgloss.Style
	VisualCaption    lipgloss.Style
}

// Default returns the default theme
func Default() *Theme {
	return &Theme{
		// Card colors
		CardRed: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E74C3C")),
		CardBlack: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2C3E50")),
		CardBackground: lipgloss.NewStyle().
			Background(lipgloss.Color("#FFFFFF")).
			Foreground(lipgloss.Color("#2C3E50")),
		CardSelected: lipgloss.NewStyle().
			Background(lipgloss.Color("#3498DB")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true),
		CardPlayable: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#27AE60")),
		CardDisabled: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#95A5A6")),

		// UI elements
		Primary: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3498DB")),
		Secondary: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9B59B6")),
		Accent: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E67E22")),
		Muted: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#95A5A6")),

		// Status
		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#27AE60")),
		Warning: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F39C12")),
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E74C3C")),

		// Layout
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3498DB")).
			Padding(1, 2),
		ScreenBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3498DB")),
		ContentBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7F8C8D")).
			Padding(1, 2),
		Title: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3498DB")).
			Bold(true).
			MarginBottom(1),
		Subtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7F8C8D")).
			Italic(true),
		Body: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2C3E50")),
		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#95A5A6")).
			Italic(true),

		// Menu
		MenuItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2C3E50")).
			PaddingLeft(2),
		MenuItemSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#3498DB")).
			Bold(true).
			PaddingLeft(2),
		MenuItemDisabled: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#95A5A6")).
			PaddingLeft(2),

		// Visual lesson elements
		AnnotationLabel: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3498DB")).
			Bold(true),
		WinnerHighlight: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#27AE60")).
			Bold(true),
		LoserDim: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#95A5A6")),
		VisualCaption: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7F8C8D")).
			Italic(true),
	}
}

// Current holds the active theme
var Current = Default()
