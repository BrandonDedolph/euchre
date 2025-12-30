package app

import (
	"fmt"

	"github.com/bran/euchre/internal/engine"
	"github.com/bran/euchre/internal/ui/components"
	"github.com/bran/euchre/internal/ui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Tab represents a section in the quick reference
type Tab int

const (
	TabTrumpHierarchy Tab = iota
	TabBasicRules
	TabScoring
	TabBidding
	TabCount // sentinel for counting tabs
)

func (t Tab) String() string {
	switch t {
	case TabTrumpHierarchy:
		return "Trump Hierarchy"
	case TabBasicRules:
		return "Basic Rules"
	case TabScoring:
		return "Scoring"
	case TabBidding:
		return "Bidding"
	default:
		return ""
	}
}

// QuickReference shows the rules quick reference
type QuickReference struct {
	activeTab Tab
	width     int
	height    int
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
		case "left", "h":
			if q.activeTab > 0 {
				q.activeTab--
			} else {
				q.activeTab = TabCount - 1
			}
		case "right", "l":
			if q.activeTab < TabCount-1 {
				q.activeTab++
			} else {
				q.activeTab = 0
			}
		case "1":
			q.activeTab = TabTrumpHierarchy
		case "2":
			q.activeTab = TabBasicRules
		case "3":
			q.activeTab = TabScoring
		case "4":
			q.activeTab = TabBidding
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

	// Header: title + tabs (fixed at top)
	title := theme.Current.Title.Render("Euchre Quick Reference")
	tabBar := q.renderTabBar()
	header := lipgloss.PlaceHorizontal(width, lipgloss.Center, title) + "\n" +
		lipgloss.PlaceHorizontal(width, lipgloss.Center, tabBar)
	headerHeight := lipgloss.Height(header)

	// Footer: help text (fixed at bottom)
	help := theme.Current.Help.Render("←/→: Switch tabs • 1-4: Jump to tab • Esc: Back")
	footer := lipgloss.PlaceHorizontal(width, lipgloss.Center, help)
	footerHeight := lipgloss.Height(footer)

	// Content area: fills remaining space
	contentHeight := height - headerHeight - footerHeight - 2 // 2 for spacing

	// Render active panel content
	var panelContent string
	switch q.activeTab {
	case TabTrumpHierarchy:
		panelContent = q.renderTrumpPanel()
	case TabBasicRules:
		panelContent = q.renderBasicRulesPanel()
	case TabScoring:
		panelContent = q.renderScoringPanel()
	case TabBidding:
		panelContent = q.renderBiddingPanel()
	}

	// Wrap content in panel box
	contentBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3498DB")).
		Padding(0, 1).
		Render(panelContent)

	// Center content horizontally and place in middle area
	centeredContent := lipgloss.Place(width, contentHeight, lipgloss.Center, lipgloss.Center, contentBox)

	// Assemble: header at top, content in middle, footer at bottom
	return header + "\n" + centeredContent + "\n" + footer
}

// renderTabBar renders the tab navigation bar
func (q *QuickReference) renderTabBar() string {
	var tabs []string

	activeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#3498DB")).
		Bold(true).
		Padding(0, 2)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7F8C8D")).
		Padding(0, 2)

	for i := Tab(0); i < TabCount; i++ {
		label := fmt.Sprintf("%d. %s", i+1, i.String())
		if i == q.activeTab {
			tabs = append(tabs, activeStyle.Render(label))
		} else {
			tabs = append(tabs, inactiveStyle.Render(label))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, tabs...)
}

// renderTrumpPanel renders the trump hierarchy with actual cards
func (q *QuickReference) renderTrumpPanel() string {
	header := theme.Current.Primary.Bold(true).Render("Trump Card Hierarchy")
	subtitle := theme.Current.Muted.Render("When Hearts (♥) is trump:")

	// Create the cards for the hierarchy
	rightBower := engine.Card{Rank: engine.Jack, Suit: engine.Hearts}
	leftBower := engine.Card{Rank: engine.Jack, Suit: engine.Diamonds}
	aceHearts := engine.Card{Rank: engine.Ace, Suit: engine.Hearts}
	kingHearts := engine.Card{Rank: engine.King, Suit: engine.Hearts}
	queenHearts := engine.Card{Rank: engine.Queen, Suit: engine.Hearts}

	// Render cards
	cards := []engine.Card{rightBower, leftBower, aceHearts, kingHearts, queenHearts}
	renderedCards := make([]string, len(cards))
	for i, card := range cards {
		cv := components.NewCardView(card)
		renderedCards[i] = cv.Render()
	}

	// Labels for each card
	labelStyle := theme.Current.Primary
	labels := []string{
		labelStyle.Render("Right"),
		labelStyle.Render("Left"),
		labelStyle.Render("Ace"),
		labelStyle.Render("King"),
		labelStyle.Render("Queen"),
	}

	// Join cards horizontally with labels below
	cardRow := lipgloss.JoinHorizontal(lipgloss.Top, renderedCards...)

	// Create label row (centered under each card)
	labelParts := make([]string, len(labels))
	for i, label := range labels {
		labelParts[i] = lipgloss.NewStyle().Width(7).Align(lipgloss.Center).Render(label)
	}
	labelRow := lipgloss.JoinHorizontal(lipgloss.Top, labelParts...)

	// Arrow showing order
	arrowStyle := theme.Current.Accent
	orderArrow := arrowStyle.Render("HIGHEST ──────────────────────────────▶ LOWER")

	// Key concepts box
	conceptsBox := q.renderConceptsBox()

	return lipgloss.JoinVertical(lipgloss.Center,
		header,
		"",
		subtitle,
		"",
		cardRow,
		labelRow,
		"",
		orderArrow,
		"",
		conceptsBox,
	)
}

// renderConceptsBox renders key bower concepts
func (q *QuickReference) renderConceptsBox() string {
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#9B59B6")).
		Padding(0, 1)

	content := theme.Current.Secondary.Bold(true).Render("Key Concepts") + "\n\n" +
		theme.Current.CardRed.Render("Right Bower") + " = Jack of trump suit (highest card)\n" +
		theme.Current.CardRed.Render("Left Bower") + "  = Jack of SAME COLOR as trump\n" +
		theme.Current.Muted.Render("              (counts as trump, not its printed suit!)")

	return borderStyle.Render(content)
}

// renderBasicRulesPanel renders basic rules with position diagram
func (q *QuickReference) renderBasicRulesPanel() string {
	header := theme.Current.Primary.Bold(true).Render("Basic Rules")

	// Player position diagram
	diagram := q.renderPositionDiagram()

	// Rules in two columns
	rulesLeft := `• 4 players in 2 teams
• Partners sit across
• 24-card deck (9-A)
• 5 cards dealt each`

	rulesRight := `• Must follow suit if able
• Trump beats other suits
• Highest card wins trick
• First to 10 points wins`

	leftCol := lipgloss.NewStyle().Width(24).Render(rulesLeft)
	rightCol := lipgloss.NewStyle().Width(24).Render(rulesRight)
	rulesRow := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, "    ", rightCol)

	// Card deck info
	deckHeader := theme.Current.Secondary.Bold(true).Render("The Deck")
	deckInfo := "9  10  J  Q  K  A   of each suit (24 cards total)"

	return lipgloss.JoinVertical(lipgloss.Center,
		header,
		"",
		diagram,
		"",
		rulesRow,
		"",
		deckHeader,
		deckInfo,
	)
}

// renderPositionDiagram renders the player seating diagram
func (q *QuickReference) renderPositionDiagram() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#27AE60")).
		Padding(0, 1)

	teamAStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3498DB")).
		Bold(true)

	teamBStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E67E22")).
		Bold(true)

	// Build the diagram
	diagram := `
         ` + teamAStyle.Render("Partner") + `
        (Team A)

  ` + teamBStyle.Render("West") + `              ` + teamBStyle.Render("East") + `
(Team B)          (Team B)

          ` + teamAStyle.Render("You") + `
        (Team A)
`

	return boxStyle.Render(diagram)
}

// renderScoringPanel renders scoring as a table
func (q *QuickReference) renderScoringPanel() string {
	header := theme.Current.Primary.Bold(true).Render("Scoring")

	// Table styles
	bc := lipgloss.NewStyle().Foreground(lipgloss.Color("#7F8C8D"))
	headerStyle := theme.Current.Secondary.Bold(true)
	cellStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF8E7"))
	pointsStyle := theme.Current.Success.Bold(true)

	// Making team table
	makingHeader := theme.Current.Primary.Render("Making Team (called trump)")
	makingTable := bc.Render("┌──────────────────────────┬────────┐") + "\n" +
		bc.Render("│") + headerStyle.Render(" Result                   ") + bc.Render("│") + headerStyle.Render(" Points ") + bc.Render("│") + "\n" +
		bc.Render("├──────────────────────────┼────────┤") + "\n" +
		bc.Render("│") + cellStyle.Render(" 3 or 4 tricks            ") + bc.Render("│") + pointsStyle.Render("   1    ") + bc.Render("│") + "\n" +
		bc.Render("│") + cellStyle.Render(" All 5 tricks (March)     ") + bc.Render("│") + pointsStyle.Render("   2    ") + bc.Render("│") + "\n" +
		bc.Render("│") + cellStyle.Render(" All 5 tricks (Alone)     ") + bc.Render("│") + pointsStyle.Render("   4    ") + bc.Render("│") + "\n" +
		bc.Render("└──────────────────────────┴────────┘")

	// Defending team table
	defendingHeader := theme.Current.Primary.Render("Defending Team")
	defendingTable := bc.Render("┌──────────────────────────┬────────┐") + "\n" +
		bc.Render("│") + headerStyle.Render(" Result                   ") + bc.Render("│") + headerStyle.Render(" Points ") + bc.Render("│") + "\n" +
		bc.Render("├──────────────────────────┼────────┤") + "\n" +
		bc.Render("│") + cellStyle.Render(" Euchre (makers < 3)      ") + bc.Render("│") + pointsStyle.Render("   2    ") + bc.Render("│") + "\n" +
		bc.Render("└──────────────────────────┴────────┘")

	// Side-by-side layout
	leftTable := lipgloss.JoinVertical(lipgloss.Left, makingHeader, makingTable)
	rightTable := lipgloss.JoinVertical(lipgloss.Left, defendingHeader, defendingTable)

	tables := lipgloss.JoinHorizontal(lipgloss.Top, leftTable, "   ", rightTable)

	// Win condition
	winBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#F1C40F")).
		Padding(0, 2).
		Render(theme.Current.Warning.Bold(true).Render("★ First team to 10 points wins! ★"))

	return lipgloss.JoinVertical(lipgloss.Center,
		header,
		"",
		tables,
		"",
		winBox,
	)
}

// renderBiddingPanel renders bidding rules with visual flow
func (q *QuickReference) renderBiddingPanel() string {
	header := theme.Current.Primary.Bold(true).Render("Bidding")

	// Round 1 section
	round1Header := theme.Current.Secondary.Bold(true).Render("Round 1: Order Up")

	// Show a sample turned up card
	turnedCard := engine.Card{Rank: engine.King, Suit: engine.Spades}
	cv := components.NewCardView(turnedCard)
	cardView := cv.Render()

	turnedLabel := theme.Current.Muted.Render("Turned up card")

	round1Text := `Each player can:
  • ` + theme.Current.Success.Render("Order Up") + ` - Make this suit trump
  • ` + theme.Current.Muted.Render("Pass") + ` - Decline

If ordered up:
  → Dealer picks up card
  → Dealer discards one`

	round1Content := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.JoinVertical(lipgloss.Center, cardView, turnedLabel),
		"   ",
		round1Text,
	)

	// Round 2 section
	round2Header := theme.Current.Secondary.Bold(true).Render("Round 2: Call Suit")
	round2Text := `If all pass in Round 1:
  • Name ANY other suit as trump
  • Or pass again`

	// Going alone section
	aloneHeader := theme.Current.Accent.Bold(true).Render("Going Alone")
	aloneText := `• Your partner sits out
• Win all 5 = ` + theme.Current.Success.Render("4 points") + `
  (instead of 2)`

	// Dealer indicator
	dealerBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#F1C40F")).
		Padding(0, 1)

	dealerContent := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#000")).
		Background(lipgloss.Color("#F1C40F")).
		Bold(true).
		Padding(0, 1).
		Render("DEALER") +
		theme.Current.Muted.Render(" always acts last in each round")

	return lipgloss.JoinVertical(lipgloss.Center,
		header,
		"",
		round1Header,
		round1Content,
		"",
		round2Header,
		round2Text,
		"",
		aloneHeader,
		aloneText,
		"",
		dealerBox.Render(dealerContent),
	)
}
