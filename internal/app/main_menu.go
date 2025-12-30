package app

import (
	"github.com/bran/euchre/internal/ui/components"
	"github.com/bran/euchre/internal/ui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MainMenu is the main menu screen
type MainMenu struct {
	menu   *components.Menu
	width  int
	height int
}

// NewMainMenu creates a new main menu
func NewMainMenu() *MainMenu {
	items := []components.MenuItem{
		{
			Label:       "Play Game",
			Description: "Start a new game against AI opponents",
		},
		{
			Label:       "Learn to Play",
			Description: "Interactive tutorials to learn Euchre",
		},
		{
			Label:       "Quick Reference",
			Description: "View rules and card rankings",
		},
		{
			Label:       "Quit",
			Description: "Exit the application",
		},
	}

	return &MainMenu{
		menu: components.NewMenu("", items),
	}
}

// Init implements tea.Model
func (m *MainMenu) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m *MainMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.menu.MoveUp()
		case "down", "j":
			m.menu.MoveDown()
		case "enter", " ":
			return m.handleSelect()
		case "q", "esc":
			return m, Quit()
		}
	}

	return m, nil
}

// handleSelect handles menu selection
func (m *MainMenu) handleSelect() (tea.Model, tea.Cmd) {
	item := m.menu.CurrentItem()
	if item == nil || item.Disabled {
		return m, nil
	}

	switch m.menu.Selected {
	case 0: // Play Game
		return m, Navigate(ScreenGameSetup)
	case 1: // Learn to Play
		return m, Navigate(ScreenLearningJourney)
	case 2: // Quick Reference
		return m, Navigate(ScreenQuickReference)
	case 3: // Quit
		return m, Quit()
	}

	return m, nil
}

// View implements tea.Model
func (m *MainMenu) View() string {
	width := m.width
	height := m.height
	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 24
	}

	// ASCII art title
	title := `
    ███████╗██╗   ██╗ ██████╗██╗  ██╗██████╗ ███████╗
    ██╔════╝██║   ██║██╔════╝██║  ██║██╔══██╗██╔════╝
    █████╗  ██║   ██║██║     ███████║██████╔╝█████╗
    ██╔══╝  ██║   ██║██║     ██╔══██║██╔══██╗██╔══╝
    ███████╗╚██████╔╝╚██████╗██║  ██║██║  ██║███████╗
    ╚══════╝ ╚═════╝  ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝
`

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3498DB")).
		Bold(true)

	subtitle := theme.Current.Subtitle.Render("Learn and play the classic trick-taking card game")

	// Wrap menu in content box with fixed size
	menuBox := theme.Current.ContentBox.
		Width(40).
		Render(m.menu.Render())

	help := theme.Current.Help.Render("↑/↓ or j/k: Navigate • Enter: Select • Esc: Quit")

	innerContent := titleStyle.Render(title) + "\n" +
		subtitle + "\n\n" +
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
