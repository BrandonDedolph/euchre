package app

import (
	"github.com/BrandonDedolph/euchre/internal/ai"
	"github.com/BrandonDedolph/euchre/internal/ui/components"
	"github.com/BrandonDedolph/euchre/internal/ui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// GameSettings is the payload passed from the setup screen to game play,
// describing the rule toggles chosen by the player.
type GameSettings struct {
	Variant        string
	StickTheDealer bool
	DefendAlone    bool
	Difficulty     ai.Difficulty // opponent AI skill level (defaults to Medium)
	Tutorial       bool          // enable the interactive coach (random hand + per-move tips)
}

// GameSetup is the game setup screen
type GameSetup struct {
	menu           *components.Menu
	variant        string
	stickTheDealer bool
	defendAlone    bool
	difficulty     ai.Difficulty
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
			Label:       "Defend Alone: Off",
			Description: "Allow defenders to go alone for 4 points on euchre",
		},
		{
			Label:       "AI Difficulty: Medium",
			Description: "Skill level of the computer opponents",
		},
		{
			Label:       "Back to Menu",
			Description: "Return to the main menu",
		},
	}

	return &GameSetup{
		menu:       components.NewMenu("", items),
		variant:    "Standard",
		difficulty: ai.DifficultyMedium,
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
		return g, NavigateWithData(ScreenGamePlay, GameSettings{
			Variant:        g.variant,
			StickTheDealer: g.stickTheDealer,
			DefendAlone:    g.defendAlone,
			Difficulty:     g.difficulty,
		})
	case 1: // Variant toggle
		// TODO: Cycle through variants (only Standard exists for now)
	case 2: // Stick the Dealer toggle
		g.stickTheDealer = !g.stickTheDealer
		if g.stickTheDealer {
			g.menu.Items[2].Label = "Stick the Dealer: On"
		} else {
			g.menu.Items[2].Label = "Stick the Dealer: Off"
		}
	case 3: // Defend Alone toggle
		g.defendAlone = !g.defendAlone
		if g.defendAlone {
			g.menu.Items[3].Label = "Defend Alone: On"
		} else {
			g.menu.Items[3].Label = "Defend Alone: Off"
		}
	case 4: // AI Difficulty cycle (Easy -> Medium -> Hard -> Easy)
		switch g.difficulty {
		case ai.DifficultyEasy:
			g.difficulty = ai.DifficultyMedium
		case ai.DifficultyMedium:
			g.difficulty = ai.DifficultyHard
		default: // Hard (or any unexpected value) wraps back to Easy
			g.difficulty = ai.DifficultyEasy
		}
		g.menu.Items[4].Label = "AI Difficulty: " + g.difficulty.String()
	case 5: // Back
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
