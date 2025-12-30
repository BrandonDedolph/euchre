package app

import (
	"github.com/bran/euchre/internal/ui/components"
	"github.com/bran/euchre/internal/ui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// GameSetup is the game setup screen
type GameSetup struct {
	menu           *components.Menu
	variant        string
	stickTheDealer bool
	width          int
	height         int
}

// NewGameSetup creates a new game setup screen
func NewGameSetup() *GameSetup {
	items := []components.MenuItem{
		{
			Label:       "Start Game",
			Description: "Begin a new game with current settings",
		},
		{
			Label:       "Variant: Standard",
			Description: "4-player Euchre with 24-card deck",
		},
		{
			Label:       "Stick the Dealer: Off",
			Description: "Dealer must call if everyone passes",
		},
		{
			Label:       "Back to Menu",
			Description: "Return to the main menu",
		},
	}

	return &GameSetup{
		menu:    components.NewMenu("", items),
		variant: "Standard",
	}
}

// Init implements tea.Model
func (g *GameSetup) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (g *GameSetup) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		g.width = msg.Width
		g.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			g.menu.MoveUp()
		case "down", "j":
			g.menu.MoveDown()
		case "enter", " ":
			return g.handleSelect()
		case "q", "esc":
			return g, Navigate(ScreenMainMenu)
		}
	}

	return g, nil
}

// handleSelect handles menu selection
func (g *GameSetup) handleSelect() (tea.Model, tea.Cmd) {
	switch g.menu.Selected {
	case 0: // Start Game
		return g, Navigate(ScreenGamePlay)
	case 1: // Variant toggle
		// TODO: Cycle through variants
	case 2: // Stick the Dealer toggle
		g.stickTheDealer = !g.stickTheDealer
		if g.stickTheDealer {
			g.menu.Items[2].Label = "Stick the Dealer: On"
		} else {
			g.menu.Items[2].Label = "Stick the Dealer: Off"
		}
	case 3: // Back
		return g, Navigate(ScreenMainMenu)
	}

	return g, nil
}

// View implements tea.Model
func (g *GameSetup) View() string {
	width := g.width
	height := g.height
	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 24
	}

	title := theme.Current.Title.Render("Game Setup")

	// Wrap menu in content box
	menuBox := theme.Current.ContentBox.
		Width(48).
		Render(g.menu.Render())

	help := theme.Current.Help.Render("↑/↓: Navigate • Enter: Select/Toggle • Esc: Back")

	innerContent := title + "\n\n" +
		menuBox + "\n\n" +
		help

	// Center content and wrap in screen border
	centeredContent := lipgloss.Place(width-4, height-4, lipgloss.Center, lipgloss.Center, innerContent)
	screenBox := theme.Current.ScreenBorder.
		Width(width - 2).
		Height(height - 2).
		Render(centeredContent)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, screenBox)
}
