package theme

import "github.com/charmbracelet/lipgloss"

// Palette holds the adaptive base colors. Each automatically resolves to the
// Light or Dark value based on the terminal background, so text stays legible
// whether the user runs a light or dark terminal. Card interiors deliberately
// use fixed colors (a physical card is white with red/black ink regardless of
// terminal), so those are not part of this palette.
var (
	ColBlue  = lipgloss.AdaptiveColor{Light: "#2178C4", Dark: "#3498DB"} // accents, borders
	ColGreen = lipgloss.AdaptiveColor{Light: "#1E8449", Dark: "#2ECC71"} // your team, success
	ColRed   = lipgloss.AdaptiveColor{Light: "#C0392B", Dark: "#E74C3C"} // opponents, error
	ColGold  = lipgloss.AdaptiveColor{Light: "#B9770E", Dark: "#F1C40F"} // dealer badge, celebration
	ColMuted = lipgloss.AdaptiveColor{Light: "#7F8C8D", Dark: "#95A5A6"} // secondary text
	ColText  = lipgloss.AdaptiveColor{Light: "#2C3E50", Dark: "#ECF0F1"} // primary body text
	ColPip   = lipgloss.AdaptiveColor{Light: "#2563EB", Dark: "#60A5FA"} // card-back pattern
)

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

	// Game table
	TeamYou     lipgloss.Style // your team's name/score (green)
	TeamOpp     lipgloss.Style // opponents' name/score (red)
	DealerBadge lipgloss.Style // gold "DEALER" chip
	CardPattern lipgloss.Style // face-down card-back pattern
	PanelBorder lipgloss.Style // side-panel divider color

	// Visual lesson elements
	AnnotationLabel  lipgloss.Style
	WinnerHighlight  lipgloss.Style
	LoserDim         lipgloss.Style
	VisualCaption    lipgloss.Style
	LessonText       lipgloss.Style
}

// Default returns the default theme
func Default() *Theme {
	return &Theme{
		// Card colors
		CardRed: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E74C3C")),
		CardBlack: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")),
		CardBackground: lipgloss.NewStyle().
			Background(lipgloss.Color("#FFFFFF")).
			Foreground(lipgloss.Color("#000000")),
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
			Foreground(ColBlue),
		Secondary: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9B59B6")),
		Accent: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E67E22")),
		Muted: lipgloss.NewStyle().
			Foreground(ColMuted),

		// Status
		Success: lipgloss.NewStyle().
			Foreground(ColGreen),
		Warning: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F39C12")),
		Error: lipgloss.NewStyle().
			Foreground(ColRed),

		// Layout
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColBlue).
			Padding(1, 2),
		ScreenBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColBlue),
		ContentBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColMuted).
			Padding(1, 2),
		Title: lipgloss.NewStyle().
			Foreground(ColBlue).
			Bold(true).
			MarginBottom(1),
		Subtitle: lipgloss.NewStyle().
			Foreground(ColMuted).
			Italic(true),
		Body: lipgloss.NewStyle().
			Foreground(ColText),
		Help: lipgloss.NewStyle().
			Foreground(ColMuted).
			Italic(true),

		// Menu
		MenuItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF8E7")).
			PaddingLeft(2),
		MenuItemSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColBlue).
			Bold(true).
			PaddingLeft(2),
		MenuItemDisabled: lipgloss.NewStyle().
			Foreground(ColMuted).
			PaddingLeft(2),

		// Game table
		TeamYou: lipgloss.NewStyle().
			Foreground(ColGreen).
			Bold(true),
		TeamOpp: lipgloss.NewStyle().
			Foreground(ColRed).
			Bold(true),
		DealerBadge: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(ColGold).
			Bold(true).
			Padding(0, 1),
		CardPattern: lipgloss.NewStyle().
			Foreground(ColPip),
		PanelBorder: lipgloss.NewStyle().
			Foreground(ColBlue),

		// Visual lesson elements
		AnnotationLabel: lipgloss.NewStyle().
			Foreground(ColBlue).
			Bold(true),
		WinnerHighlight: lipgloss.NewStyle().
			Foreground(ColGreen).
			Bold(true),
		LoserDim: lipgloss.NewStyle().
			Foreground(ColMuted),
		VisualCaption: lipgloss.NewStyle().
			Foreground(ColMuted).
			Italic(true),
		LessonText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF8E7")),
	}
}

// Current holds the active theme
var Current = Default()
