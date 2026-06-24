package components

import (
	"fmt"
	"strings"

	"github.com/BrandonDedolph/euchre/internal/engine"
	"github.com/BrandonDedolph/euchre/internal/ui/theme"
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
	Maker          int       // Player who called trump (-1 if none)
	MakerAlone     bool      // Whether maker is going alone
	TurnPulseFrame int       // Animation frame for turn indicator pulse
	RoundNumber    int       // Current round number (1-based)
	PlayerActions  [4]string // Latest per-seat action label (e.g. "passes"); "" = none
	TrickWinner    int       // Seat of the just-won trick's card to crown; -1 = none

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
		TrickWinner: -1,
	}
}

// teamAccent returns the accent color for a seat's team: green for your team
// (seats 0,2) and red for the opponents (seats 1,3).
func teamAccent(seat int) lipgloss.TerminalColor {
	if engine.Team(seat) == engine.Team(0) {
		return theme.ColGreen
	}
	return theme.ColRed
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

	return sb.String()
}

// RenderTricksTable renders a small 1x2 table for tricks
func RenderTricksTable(tricks int) string {
	bc := theme.Current.Muted
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
		dealerBadge = " " + theme.Current.DealerBadge.Render("DEALER")
	}

	// Compact header with inline tricks
	tricksStyle := theme.Current.Muted
	tricksStr := tricksStyle.Render(fmt.Sprintf("(%d)", tricks))
	header := fmt.Sprintf("%s%s %s%s", name, indicator, tricksStr, dealerBadge)
	header = lipgloss.PlaceHorizontal(t.Width, lipgloss.Center, header)

	// Reserved action line directly under the name (always present, blank when
	// no action, so the seat's fixed height never changes).
	actionLine := lipgloss.PlaceHorizontal(t.Width, lipgloss.Center, renderActionLabel(t.PlayerActions[2], t.Width))

	// Show face-down cards (always show space for 5 cards even if fewer)
	cardDisplay := RenderFaceDown(min(cards, 5))
	cardDisplay = lipgloss.PlaceHorizontal(t.Width, lipgloss.Center, cardDisplay)

	content := header + "\n" + actionLine + "\n" + cardDisplay

	// Fixed height to prevent layout shift (header + action + 5-card block).
	return lipgloss.NewStyle().Height(8).Render(content)
}

// renderMiddle renders the middle section with left player, trick, right player
func (t *TableView) renderMiddle() string {
	leftPlayer := t.renderSidePlayer(1, true) // West
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

	// Compact header with inline tricks
	tricksStyle := theme.Current.Muted
	header := fmt.Sprintf("%s%s", name, indicator)
	tricksStr := tricksStyle.Render(fmt.Sprintf("(%d)", tricks))

	// Render vertical face-down cards (West is reversed)
	cardDisplay := RenderFaceDownVertical(min(cards, 5), isLeft)

	// Build the side player display
	var sb strings.Builder

	// Show DEALER on separate line above name if this player is dealer
	if t.Dealer == playerIdx {
		sb.WriteString(theme.Current.DealerBadge.Render("DEALER"))
		sb.WriteString("\n")
	}

	sb.WriteString(header)
	sb.WriteString(" ")
	sb.WriteString(tricksStr)
	sb.WriteString("\n")
	// Reserved action line directly under the name (always present, blank when
	// no action, so the seat's fixed height never changes). Width 14 matches the
	// seat box; truncate so a long label can't widen the seat.
	sb.WriteString(renderActionLabel(t.PlayerActions[playerIdx], 14))
	sb.WriteString("\n")
	sb.WriteString(cardDisplay)

	// Fixed width and height to prevent layout shift
	style := lipgloss.NewStyle().Width(14).Height(13)
	if isLeft {
		style = style.Align(lipgloss.Right)
	} else {
		style = style.Align(lipgloss.Left)
	}

	return style.Render(sb.String())
}

// renderTrickArea renders the center area with played cards
func (t *TableView) renderTrickArea() string {
	cardWidth := 7                // Card width
	cardHeight := 5               // Card height
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
			BorderForeground(theme.ColBlue).
			Padding(0, 1)

		return style.Render(content)
	}

	// emptySlot is a blank card-sized placeholder.
	emptySlot := func() string {
		return lipgloss.NewStyle().Width(cardWidth).Height(cardHeight).Render("")
	}

	// During play: show played cards in diamond pattern (no center card).
	//
	// Convergence padding: during the collect sweep the three LOSING cards step
	// horizontally toward the winner/center by a bounded offset so the motion
	// stays strictly inside the fixed trick box. We render the card into a
	// cardWidth-wide slot and shift it with left/right padding that never
	// exceeds the slack available within the slot/box.
	collecting := t.TrickCollectAnim != nil
	var collectProgress float64
	if collecting && t.TrickCollectAnim.TotalFrames > 0 {
		collectProgress = float64(t.TrickCollectAnim.Frame) / float64(t.TrickCollectAnim.TotalFrames)
	}

	renderCard := func(playerIdx int) string {
		// Check if this card is being animated in
		if t.CardPlayAnim != nil && t.CardPlayAnim.FromPlayer == playerIdx {
			cv := NewCardView(t.CardPlayAnim.Card)
			return cv.Render()
		}

		// Trick collection sweep: the winner's card stays bright and crowned,
		// the losers converge toward it and fade out.
		if collecting {
			for _, pc := range t.TrickCollectAnim.Cards {
				if pc.Player != playerIdx {
					continue
				}
				if playerIdx == t.TrickCollectAnim.Winner {
					// Winner stays put, bright, crowned in team color.
					cv := NewCardView(pc.Card)
					cv.Trump = t.Trump
					cv.Style = CardStyleTrickWinner
					cv.AccentColor = teamAccent(playerIdx)
					return cv.Render()
				}
				// Loser: fade then blank.
				if collectProgress >= 0.75 {
					return emptySlot()
				}
				cv := NewCardView(pc.Card)
				if collectProgress >= 0.4 {
					cv.Style = CardStyleDisabled
				}
				// Lean the loser horizontally toward the box center as it fades.
				// The card is placed inside a slot of exactly cardWidth (its own
				// width), so the alignment shift can never push it outside the
				// fixed trick box — convergence is conveyed by the lean + fade.
				card := cv.Render()
				slot := lipgloss.NewStyle().Width(cardWidth).MaxWidth(cardWidth)
				// You(0)/Partner(2) are centered already; West(1) leans right
				// toward center, East(3) leans left toward center.
				switch playerIdx {
				case 1: // West: lean right (toward center)
					return slot.Align(lipgloss.Right).Render(card)
				case 3: // East: lean left (toward center)
					return slot.Align(lipgloss.Left).Render(card)
				default:
					return slot.Align(lipgloss.Center).Render(card)
				}
			}
		}

		// Static crown: while reviewing a finished trick (no collect anim), the
		// winning card wears a bold team-colored double border and a ★.
		if t.TrickWinner >= 0 && playerIdx == t.TrickWinner && !collecting {
			for _, pc := range t.CurrentTrick {
				if pc.Player == playerIdx {
					cv := NewCardView(pc.Card)
					cv.Trump = t.Trump
					cv.Style = CardStyleTrickWinner
					cv.AccentColor = teamAccent(playerIdx)
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
		return emptySlot()
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
		BorderForeground(theme.ColBlue).
		Padding(0, 1)

	return style.Render(content)
}

// renderActionLabel renders a seat's latest action on a reserved line beneath
// its name. The line is ALWAYS present (a single space when action is empty) so
// the seat's fixed height never changes. The text is styled muted/dim and
// truncated to maxWidth so a long label can't widen the seat.
func renderActionLabel(action string, maxWidth int) string {
	if action == "" {
		return " "
	}
	return theme.Current.Muted.MaxWidth(maxWidth).Render(action)
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
	borderStyle := theme.Current.Muted
	patternStyle := theme.Current.CardPattern

	border := borderStyle.Render
	pattern := patternStyle.Render

	if progress < 0.25 {
		// Stage 1: Face-down card
		lines := []string{
			border("┌─────┐"),
			border("│") + pattern("░░░░░") + border("│"),
			border("│") + pattern("░░░░░") + border("│"),
			border("│") + pattern("░░░░░") + border("│"),
			border("└─────┘"),
		}
		return strings.Join(lines, "\n")
	} else if progress < 0.5 {
		// Stage 2: Card flipping (narrow)
		lines := []string{
			"  " + border("┌─┐") + "  ",
			"  " + border("│") + pattern("░") + border("│") + "  ",
			"  " + border("│") + pattern("░") + border("│") + "  ",
			"  " + border("│") + pattern("░") + border("│") + "  ",
			"  " + border("└─┘") + "  ",
		}
		return strings.Join(lines, "\n")
	} else if progress < 0.75 {
		// Stage 3: Card flipping back (narrow, showing face)
		cv := NewCardView(t.TurnedCard)
		return cv.Render()
	}

	// Stage 4: Full face-up card
	cv := NewCardView(t.TurnedCard)
	return cv.Render()
}
