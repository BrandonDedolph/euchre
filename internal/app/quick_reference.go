package app

import (
	"strings"

	"github.com/bran/euchre/internal/ui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// QuickReference shows the rules quick reference
type QuickReference struct {
	scroll int
	width  int
	height int
}

// NewQuickReference creates a new quick reference screen
func NewQuickReference() *QuickReference {
	return &QuickReference{}
}

// Init implements tea.Model
func (q *QuickReference) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (q *QuickReference) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		q.width = msg.Width
		q.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return q, Navigate(ScreenMainMenu)
		case "up", "k":
			if q.scroll > 0 {
				q.scroll--
			}
		case "down", "j":
			q.scroll++
		}
	}

	return q, nil
}

// View implements tea.Model
func (q *QuickReference) View() string {
	width := q.width
	height := q.height
	if width == 0 {
		width = 80
	}
	if height == 0 {
		height = 30
	}

	title := theme.Current.Title.Render("Euchre Quick Reference")

	// Wrap reference content in content box with dynamic size (min 60x20)
	refContent := q.getContent()
	boxWidth := max(60, width*2/3)
	boxHeight := max(20, height*2/3)
	contentBox := theme.Current.ContentBox.
		Width(boxWidth).
		Height(boxHeight).
		Render(refContent)

	help := theme.Current.Help.Render("↑/↓: Scroll • Esc: Back")

	innerContent := title + "\n\n" +
		contentBox + "\n\n" +
		help

	// Center content and wrap in screen border
	centeredContent := lipgloss.Place(width-4, height-4, lipgloss.Center, lipgloss.Center, innerContent)
	screenBox := theme.Current.ScreenBorder.
		Width(width - 2).
		Height(height - 2).
		Render(centeredContent)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, screenBox)
}

func (q *QuickReference) getContent() string {
	sections := []string{
		q.getTrumpHierarchy(),
		q.getBasicRules(),
		q.getScoring(),
		q.getBiddingRules(),
	}

	return strings.Join(sections, "\n\n")
}

func (q *QuickReference) getTrumpHierarchy() string {
	header := theme.Current.Primary.Render("Trump Card Hierarchy (Highest to Lowest)")

	content := `
  1. Right Bower - Jack of trump suit (highest)
  2. Left Bower  - Jack of same color as trump
  3. Ace of trump
  4. King of trump
  5. Queen of trump
  6. Ten of trump
  7. Nine of trump (lowest trump)

  Example: If Hearts is trump:
    ` + theme.Current.CardRed.Render("J♥") + ` Right Bower (highest)
    ` + theme.Current.CardRed.Render("J♦") + ` Left Bower (belongs to Hearts!)
    ` + theme.Current.CardRed.Render("A♥ K♥ Q♥ 10♥ 9♥") + ` (remaining hearts)`

	return header + content
}

func (q *QuickReference) getBasicRules() string {
	header := theme.Current.Primary.Render("Basic Rules")

	content := `
  • 4 players in 2 teams, partners sit across
  • 24-card deck: 9, 10, J, Q, K, A of each suit
  • 5 cards dealt to each player
  • One card turned up to suggest trump
  • Must follow suit if able
  • Highest card of led suit wins (unless trumped)
  • First team to 10 points wins`

	return header + content
}

func (q *QuickReference) getScoring() string {
	header := theme.Current.Primary.Render("Scoring")

	content := `
  Making Team (called trump):
  • 3 or 4 tricks: 1 point
  • All 5 tricks (March): 2 points
  • All 5 tricks alone: 4 points

  Defending Team:
  • Euchre (makers get < 3 tricks): 2 points`

	return header + content
}

func (q *QuickReference) getBiddingRules() string {
	header := theme.Current.Primary.Render("Bidding")

	content := `
  Round 1:
  • Each player can "order up" the turned card or pass
  • If ordered up, that suit becomes trump
  • Dealer picks up the card and discards one

  Round 2 (if all passed):
  • Each player can name any OTHER suit as trump
  • Or pass again

  Going Alone:
  • Player's partner sits out
  • Win all 5 = 4 points (instead of 2)`

	return header + content
}
