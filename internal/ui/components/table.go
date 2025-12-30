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
	RoundNumber    int  // Current round number (1-based)

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
	// Center the number in a 3-char wide cell
	numStyle := lipgloss.NewStyle().Width(3).Align(lipgloss.Center)
	return bc.Render("┌────────┬───┐") + "\n" +
		bc.Render("│") + " Tricks " + bc.Render("│") + numStyle.Render(fmt.Sprintf("%d", tricks)) + bc.Render("│") + "\n" +
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
	cardWidth := 7  // Card width
	cardHeight := 5 // Card height
	totalWidth := cardWidth*3 + 4 // 3 cards + spacing

	// During bidding, show the turned card in the center
	// Check TurnedCard is not zero value (which would be Nine of Clubs due to iota)
	hasTurnedCard := t.TurnedCard != (engine.Card{})
	if t.Trump == engine.NoSuit && hasTurnedCard && len(t.CurrentTrick) == 0 {
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
			Width(cardWidth).
			Height(cardHeight).
			Render("")

		// Top row (empty)
		topRow := lipgloss.NewStyle().Height(cardHeight).Render(
			lipgloss.PlaceHorizontal(totalWidth, lipgloss.Center, placeholder),
		)

		// Middle row with turned card in center
		middleRow := lipgloss.NewStyle().Height(cardHeight).Render(
			lipgloss.JoinHorizontal(lipgloss.Center,
				placeholder,
				"  ",
				turnedCard,
				"  ",
				placeholder,
			),
		)

		// Bottom row (empty)
		bottomRow := lipgloss.NewStyle().Height(cardHeight).Render(
			lipgloss.PlaceHorizontal(totalWidth, lipgloss.Center, placeholder),
		)

		content := lipgloss.JoinVertical(lipgloss.Center, topRow, middleRow, bottomRow)

		// Outer border
		style := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3498DB")).
			Padding(0, 1)

		return style.Render(content)
	}

	// During play: show played cards in diamond pattern (no center card)
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
							Width(cardWidth).
							Height(cardHeight).
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
			Width(cardWidth).
			Height(cardHeight).
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
	//          [You]

	// Top row (Partner's card centered)
	topRow := lipgloss.NewStyle().Height(cardHeight).Render(
		lipgloss.PlaceHorizontal(totalWidth, lipgloss.Center, topCard),
	)

	// Middle row (West and East cards on sides)
	middleRow := lipgloss.NewStyle().Height(cardHeight).Render(
		lipgloss.JoinHorizontal(lipgloss.Center,
			leftCard,
			lipgloss.NewStyle().Width(cardWidth+4).Render(""),
			rightCard,
		),
	)

	// Bottom row (Your card centered)
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

// renderTrumpIndicator shows the current trump, round number, and who called it
func (t *TableView) renderTrumpIndicator() string {
	var parts []string

	// Round number
	if t.RoundNumber > 0 {
		roundStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3498DB")).
			Bold(true)
		parts = append(parts, roundStyle.Render(fmt.Sprintf("Round %d", t.RoundNumber)))
	}

	if t.Trump == engine.NoSuit {
		if t.TurnedCard.Suit != engine.NoSuit {
			parts = append(parts, theme.Current.Muted.Render("Bidding..."))
		} else {
			parts = append(parts, theme.Current.Muted.Render("Dealing..."))
		}
		return strings.Join(parts, "  •  ")
	}

	// Trump suit
	trumpStyle := theme.Current.CardBlack
	if t.Trump == engine.Hearts || t.Trump == engine.Diamonds {
		trumpStyle = theme.Current.CardRed
	}
	parts = append(parts, fmt.Sprintf("Trump: %s", trumpStyle.Render(t.Trump.Symbol()+" "+t.Trump.String())))

	// Going alone indicator
	if t.MakerAlone {
		aloneStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E67E22")).
			Bold(true)
		parts = append(parts, aloneStyle.Render("ALONE"))
	}

	// Who called trump
	if t.Maker >= 0 && t.Maker < len(t.PlayerNames) {
		makerName := t.PlayerNames[t.Maker]
		parts = append(parts, theme.Current.Muted.Render(fmt.Sprintf("(%s)", makerName)))
	}

	return strings.Join(parts, "  •  ")
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
