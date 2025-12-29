package components

import (
	"fmt"
	"strings"

	"github.com/bran/euchre/internal/engine"
	"github.com/bran/euchre/internal/ui/theme"
	"github.com/charmbracelet/lipgloss"
)

// CardPlayAnim represents a card being played animation
type CardPlayAnim struct {
	Card        engine.Card
	FromPlayer  int
	Frame       int
	TotalFrames int
}

// TrickCollectAnim represents cards being collected after a trick
type TrickCollectAnim struct {
	Winner      int
	Cards       []engine.PlayedCard
	Frame       int
	TotalFrames int
}

// TableView represents the game table visualization
type TableView struct {
	Width          int
	Height         int
	Trump          engine.Suit
	TurnedCard     engine.Card
	CurrentTrick   []engine.PlayedCard
	PlayerHands    []int // Card counts for each player
	Dealer         int
	CurrentPlayer  int
	PlayerNames    []string
	TricksWon      []int
	Maker          int  // Player who called trump (-1 if none)
	MakerAlone     bool // Whether maker is going alone
	TurnPulseFrame int  // Animation frame for turn indicator pulse

	// Animation states
	CardPlayAnim     *CardPlayAnim     // Card being played animation
	TrickCollectAnim *TrickCollectAnim // Trick collection animation
	CardFlipFrames   int               // Frames remaining for card flip reveal
	CardFlipTotal    int               // Total frames for card flip animation
}

// NewTableView creates a new table view
func NewTableView() *TableView {
	return &TableView{
		Width:       60,
		Height:      20,
		PlayerNames: []string{"You", "West", "Partner", "East"},
		PlayerHands: []int{5, 5, 5, 5},
		TricksWon:   []int{0, 0, 0, 0},
		Maker:       -1,
	}
}

// Render returns the visual representation of the table
func (t *TableView) Render() string {
	var sb strings.Builder

	// Top player (partner, position 2)
	sb.WriteString(t.renderTopPlayer())
	sb.WriteString("\n")

	// Middle section with left player, trick area, right player
	sb.WriteString(t.renderMiddle())
	sb.WriteString("\n")

	// Trump indicator
	sb.WriteString(t.renderTrumpIndicator())
	sb.WriteString("\n")

	return sb.String()
}

// RenderTricksTable renders a small 1x2 table for tricks
func RenderTricksTable(tricks int) string {
	bc := lipgloss.NewStyle().Foreground(lipgloss.Color("#7F8C8D"))
	return bc.Render("┌────────┬───┐") + "\n" +
		bc.Render("│") + " Tricks " + bc.Render("│") + fmt.Sprintf(" %d ", tricks) + bc.Render("│") + "\n" +
		bc.Render("└────────┴───┘")
}

// renderTopPlayer renders the top player area
func (t *TableView) renderTopPlayer() string {
	name := t.PlayerNames[2]
	cards := t.PlayerHands[2]
	tricks := t.TricksWon[2]

	indicator := ""
	if t.CurrentPlayer == 2 {
		indicator = t.renderTurnIndicator()
	}

	dealerBadge := ""
	if t.Dealer == 2 {
		dealerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000")).
			Background(lipgloss.Color("#f1c40f")).
			Bold(true).
			Padding(0, 1)
		dealerBadge = " " + dealerStyle.Render("DEALER")
	}

	header := fmt.Sprintf("%s%s%s", name, indicator, dealerBadge)
	header = lipgloss.PlaceHorizontal(t.Width, lipgloss.Center, header)

	tricksTable := RenderTricksTable(tricks)
	tricksTable = lipgloss.PlaceHorizontal(t.Width, lipgloss.Center, tricksTable)

	// Show face-down cards (always show space for 5 cards even if fewer)
	cardDisplay := RenderFaceDown(min(cards, 5))
	cardDisplay = lipgloss.PlaceHorizontal(t.Width, lipgloss.Center, cardDisplay)

	content := header + "\n" + tricksTable + "\n" + cardDisplay

	// Fixed height to prevent layout shift
	return lipgloss.NewStyle().Height(10).Render(content)
}

// renderMiddle renders the middle section with left player, trick, right player
func (t *TableView) renderMiddle() string {
	leftPlayer := t.renderSidePlayer(1, true)   // West
	trickArea := t.renderTrickArea()
	rightPlayer := t.renderSidePlayer(3, false) // East

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		leftPlayer,
		"  ",
		trickArea,
		"  ",
		rightPlayer,
	)
}

// renderSidePlayer renders a side player (East or West)
func (t *TableView) renderSidePlayer(playerIdx int, isLeft bool) string {
	name := t.PlayerNames[playerIdx]
	cards := t.PlayerHands[playerIdx]
	tricks := t.TricksWon[playerIdx]

	indicator := ""
	if t.CurrentPlayer == playerIdx {
		indicator = t.renderTurnIndicator()
	}

	dealerBadge := ""
	if t.Dealer == playerIdx {
		dealerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000")).
			Background(lipgloss.Color("#f1c40f")).
			Bold(true)
		dealerBadge = "\n" + dealerStyle.Render("DEALER")
	}

	header := fmt.Sprintf("%s%s%s", name, indicator, dealerBadge)
	tricksTable := RenderTricksTable(tricks)

	// Render vertical face-down cards (West is reversed)
	cardDisplay := RenderFaceDownVertical(min(cards, 5), isLeft)

	// Build the side player display
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n")
	sb.WriteString(tricksTable)
	sb.WriteString("\n")
	sb.WriteString(cardDisplay)

	// Fixed width and height to prevent layout shift
	style := lipgloss.NewStyle().Width(14).Height(16)
	if isLeft {
		style = style.Align(lipgloss.Right)
	} else {
		style = style.Align(lipgloss.Left)
	}

	return style.Render(sb.String())
}

// renderTrickArea renders the center area with played cards
func (t *TableView) renderTrickArea() string {
	// Player border colors for card attribution
	playerColors := []lipgloss.Color{
		lipgloss.Color("#2ecc71"), // You - Green
		lipgloss.Color("#3498db"), // West - Blue
		lipgloss.Color("#1abc9c"), // Partner - Cyan
		lipgloss.Color("#9b59b6"), // East - Magenta
	}

	_ = playerColors // Keep for potential future use

	cardWidth := 11  // Card width with border
	totalWidth := cardWidth*3 + 4

	// During bidding, show the turned card in the center
	if t.Trump == engine.NoSuit && t.TurnedCard.Suit != engine.NoSuit && len(t.CurrentTrick) == 0 {
		var turnedCard string
		if t.CardFlipFrames > 0 && t.CardFlipTotal > 0 {
			// Show card flip animation
			progress := float64(t.CardFlipTotal-t.CardFlipFrames) / float64(t.CardFlipTotal)
			turnedCard = t.renderFlipAnimation(progress)
		} else {
			cv := NewCardView(t.TurnedCard)
			turnedCard = cv.Render()
		}

		// Empty placeholder for surrounding positions
		placeholder := lipgloss.NewStyle().
			Width(7).
			Height(5).
			Render("")

		// Top row (empty)
		topRow := lipgloss.PlaceHorizontal(totalWidth, lipgloss.Center, placeholder)

		// Wrap turned card in fixed height to prevent layout shift
		turnedCard = lipgloss.NewStyle().Height(5).Render(turnedCard)

		// Middle row with turned card in center
		middleRow := lipgloss.JoinHorizontal(lipgloss.Center,
			placeholder,
			lipgloss.NewStyle().Width(4).Render(""), // spacing
			turnedCard,
			lipgloss.NewStyle().Width(4).Render(""), // spacing
			placeholder,
		)

		// Bottom row (empty)
		bottomRow := lipgloss.PlaceHorizontal(totalWidth, lipgloss.Center, placeholder)

		content := lipgloss.JoinVertical(lipgloss.Center, topRow, middleRow, bottomRow)

		// Outer border
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3498DB")).
			Padding(0, 1)

		return style.Render(content)
	}

	// Render each player's card (or empty placeholder)
	renderCard := func(playerIdx int) string {
		// Check if this card is being animated in
		if t.CardPlayAnim != nil && t.CardPlayAnim.FromPlayer == playerIdx {
			cv := NewCardView(t.CardPlayAnim.Card)
			return cv.Render()
		}

		// Check if trick is being collected (cards fade out)
		if t.TrickCollectAnim != nil {
			for _, pc := range t.TrickCollectAnim.Cards {
				if pc.Player == playerIdx {
					progress := float64(t.TrickCollectAnim.Frame) / float64(t.TrickCollectAnim.TotalFrames)
					if progress > 0.75 {
						// Almost done - show placeholder
						return lipgloss.NewStyle().
							Width(7).
							Height(5).
							Render("")
					} else if progress > 0.5 {
						// Fading - show dimmed card
						cv := NewCardView(pc.Card)
						cv.Style = CardStyleDisabled
						return cv.Render()
					}
					// Still showing normally early in animation
					cv := NewCardView(pc.Card)
					return cv.Render()
				}
			}
		}

		for _, pc := range t.CurrentTrick {
			if pc.Player == playerIdx {
				cv := NewCardView(pc.Card)
				return cv.Render()
			}
		}
		// Empty placeholder (same size as card)
		placeholder := lipgloss.NewStyle().
			Width(7).
			Height(5).
			Align(lipgloss.Center, lipgloss.Center).
			Render("")
		return placeholder
	}

	topCard := renderCard(2)    // Partner
	leftCard := renderCard(1)   // West
	rightCard := renderCard(3)  // East
	bottomCard := renderCard(0) // You

	// Build layout:
	//        [Partner]
	// [West]           [East]
	//        [You]

	cardHeight := 5 // Fixed card height

	// Top row (Partner's card centered) - fixed height
	topRow := lipgloss.NewStyle().Height(cardHeight).Render(
		lipgloss.PlaceHorizontal(totalWidth, lipgloss.Center, topCard),
	)

	// Middle row (West and East cards on sides) - fixed height
	middleRow := lipgloss.NewStyle().Height(cardHeight).Render(
		lipgloss.JoinHorizontal(lipgloss.Center,
			leftCard,
			lipgloss.NewStyle().Width(cardWidth+4).Render(""),
			rightCard,
		),
	)

	// Bottom row (Your card centered) - fixed height
	bottomRow := lipgloss.NewStyle().Height(cardHeight).Render(
		lipgloss.PlaceHorizontal(totalWidth, lipgloss.Center, bottomCard),
	)

	content := lipgloss.JoinVertical(lipgloss.Center, topRow, middleRow, bottomRow)

	// Outer border
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3498DB")).
		Padding(0, 1)

	return style.Render(content)
}

// renderTrumpIndicator shows the current trump and who called it
func (t *TableView) renderTrumpIndicator() string {
	if t.Trump == engine.NoSuit {
		if t.TurnedCard.Suit != engine.NoSuit {
			// Card is shown visually in trick area, just show text label here
			return "Bidding in progress..."
		}
		return "Trump not selected"
	}

	trumpStyle := theme.Current.CardBlack
	if t.Trump == engine.Hearts || t.Trump == engine.Diamonds {
		trumpStyle = theme.Current.CardRed
	}

	trumpStr := fmt.Sprintf("Trump: %s", trumpStyle.Render(t.Trump.Symbol()+" "+t.Trump.String()))

	// Show who called trump
	if t.Maker >= 0 && t.Maker < len(t.PlayerNames) {
		makerName := t.PlayerNames[t.Maker]
		makerInfo := fmt.Sprintf(" (called by %s", makerName)
		if t.MakerAlone {
			makerInfo += ", going alone"
		}
		makerInfo += ")"
		trumpStr += theme.Current.Muted.Render(makerInfo)
	}

	return trumpStr
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// renderTurnIndicator returns an animated turn indicator
func (t *TableView) renderTurnIndicator() string {
	// Pulse through different styles based on frame
	indicators := []string{"◀", "◁", "◀", "◂"}
	colors := []string{"#E74C3C", "#FF6B6B", "#E74C3C", "#C0392B"}

	frame := t.TurnPulseFrame % len(indicators)
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(colors[frame])).
		Bold(frame%2 == 0)

	return " " + style.Render(indicators[frame])
}

// renderFlipAnimation renders the card flip animation at given progress (0.0 to 1.0)
func (t *TableView) renderFlipAnimation(progress float64) string {
	style := theme.Current.Muted

	if progress < 0.25 {
		// Stage 1: Face-down card
		lines := []string{
			"┌─────┐",
			"│░░░░░│",
			"│░░░░░│",
			"│░░░░░│",
			"└─────┘",
		}
		styled := make([]string, len(lines))
		for i, line := range lines {
			styled[i] = style.Render(line)
		}
		return strings.Join(styled, "\n")
	} else if progress < 0.5 {
		// Stage 2: Card flipping (narrow)
		lines := []string{
			"  ┌─┐  ",
			"  │░│  ",
			"  │░│  ",
			"  │░│  ",
			"  └─┘  ",
		}
		styled := make([]string, len(lines))
		for i, line := range lines {
			styled[i] = style.Render(line)
		}
		return strings.Join(styled, "\n")
	} else if progress < 0.75 {
		// Stage 3: Card flipping back (narrow, showing face)
		cv := NewCardView(t.TurnedCard)
		return cv.Render()
	}

	// Stage 4: Full face-up card
	cv := NewCardView(t.TurnedCard)
	return cv.Render()
}
