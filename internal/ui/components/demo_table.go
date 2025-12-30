package components

import (
	"strings"

	"github.com/bran/euchre/internal/engine"
	"github.com/bran/euchre/internal/ui/theme"
	"github.com/charmbracelet/lipgloss"
)

// DemoTableView renders a simplified 4-player table for demonstrations
// without game state dependencies
type DemoTableView struct {
	Width  int
	Height int

	// Player labels for each position (0=bottom/you, 1=left, 2=top/partner, 3=right)
	PlayerLabels [4]string

	// Cards in the trick area (one per position, empty Card for no card)
	TrickCards [4]engine.Card

	// Whether each position shows cards (vs empty placeholder)
	ShowCards [4]bool

	// Hand cards for each position (for showing hands)
	HandCards [4][]engine.Card

	// Winner position (-1 for no winner)
	WinnerPos int

	// Annotations for each position
	Annotations [4]string

	// Whether each player is highlighted
	Highlighted [4]bool

	// Whether each player is dimmed
	Dimmed [4]bool

	// Trump suit for display
	TrumpSuit engine.Suit
}

// NewDemoTableView creates a new demo table view
func NewDemoTableView(width, height int) *DemoTableView {
	return &DemoTableView{
		Width:  width,
		Height: height,
		PlayerLabels: [4]string{
			"You",
			"Opponent",
			"Partner",
			"Opponent",
		},
		WinnerPos: -1,
	}
}

// Render returns the demo table as a string
func (v *DemoTableView) Render() string {
	// Layout:
	//           [Partner]
	//              cards
	//
	//   [Left]    TRICK    [Right]
	//   cards     AREA     cards
	//
	//           [You]
	//           cards

	cardWidth := 7
	cardHeight := 5

	// Render player boxes
	renderPlayerBox := func(idx int) string {
		label := v.PlayerLabels[idx]

		style := theme.Current.Body
		if v.Highlighted[idx] {
			style = theme.Current.Primary.Bold(true)
		}
		if v.Dimmed[idx] {
			style = theme.Current.Muted
		}
		if v.WinnerPos == idx {
			style = theme.Current.Success.Bold(true)
		}

		boxStyle := theme.Current.Border.
			Width(14).
			Align(lipgloss.Center).
			BorderForeground(lipgloss.Color("#7F8C8D"))

		if v.Highlighted[idx] || v.WinnerPos == idx {
			boxStyle = boxStyle.BorderForeground(lipgloss.Color("#27AE60"))
		}
		if v.Dimmed[idx] {
			boxStyle = boxStyle.BorderForeground(lipgloss.Color("#95A5A6"))
		}

		content := style.Render(label)
		if v.Annotations[idx] != "" {
			annStyle := theme.Current.Muted.Italic(true)
			content += "\n" + annStyle.Render(v.Annotations[idx])
		}

		return boxStyle.Render(content)
	}

	// Render hand for a position
	renderHand := func(idx int) string {
		if len(v.HandCards[idx]) == 0 {
			return ""
		}
		if v.Dimmed[idx] {
			// Show face-down cards for dimmed players
			return RenderFaceDown(len(v.HandCards[idx]))
		}
		return RenderHand(v.HandCards[idx], -1, nil)
	}

	// Render trick area (center)
	renderTrickArea := func() string {
		emptyCard := strings.Repeat(" ", cardWidth)
		emptyCardBlock := ""
		for i := 0; i < cardHeight; i++ {
			if i > 0 {
				emptyCardBlock += "\n"
			}
			emptyCardBlock += emptyCard
		}

		renderCard := func(idx int) string {
			if v.ShowCards[idx] && v.TrickCards[idx].Rank != 0 {
				cv := NewCardView(v.TrickCards[idx])
				if v.WinnerPos == idx {
					cv.Style = CardStylePlayable
				}
				return cv.Render()
			}
			return emptyCardBlock
		}

		// Layout cards in compass formation
		topCard := renderCard(2)
		leftCard := renderCard(1)
		rightCard := renderCard(3)
		bottomCard := renderCard(0)

		topRow := lipgloss.NewStyle().Width(cardWidth*3 + 4).Align(lipgloss.Center).Render(topCard)
		middleRow := lipgloss.JoinHorizontal(lipgloss.Center,
			leftCard,
			strings.Repeat(" ", cardWidth),
			rightCard,
		)
		bottomRow := lipgloss.NewStyle().Width(cardWidth*3 + 4).Align(lipgloss.Center).Render(bottomCard)

		return lipgloss.JoinVertical(lipgloss.Center, topRow, middleRow, bottomRow)
	}

	// Build the layout
	partnerBox := renderPlayerBox(2)
	partnerHand := renderHand(2)
	partnerSection := lipgloss.JoinVertical(lipgloss.Center, partnerBox, partnerHand)

	leftBox := renderPlayerBox(1)
	leftHand := renderHand(1)
	leftSection := lipgloss.JoinVertical(lipgloss.Center, leftBox, leftHand)

	rightBox := renderPlayerBox(3)
	rightHand := renderHand(3)
	rightSection := lipgloss.JoinVertical(lipgloss.Center, rightBox, rightHand)

	youBox := renderPlayerBox(0)
	youHand := renderHand(0)
	youSection := lipgloss.JoinVertical(lipgloss.Center, youHand, youBox)

	trickArea := renderTrickArea()

	// Assemble the table
	topRow := lipgloss.NewStyle().Width(v.Width).Align(lipgloss.Center).Render(partnerSection)

	middleRow := lipgloss.JoinHorizontal(lipgloss.Center,
		leftSection,
		strings.Repeat(" ", 4),
		trickArea,
		strings.Repeat(" ", 4),
		rightSection,
	)
	middleRow = lipgloss.NewStyle().Width(v.Width).Align(lipgloss.Center).Render(middleRow)

	bottomRow := lipgloss.NewStyle().Width(v.Width).Align(lipgloss.Center).Render(youSection)

	return lipgloss.JoinVertical(lipgloss.Center, topRow, "", middleRow, "", bottomRow)
}

// SetPlayerLabel sets the label for a player position
func (v *DemoTableView) SetPlayerLabel(pos int, label string) {
	if pos >= 0 && pos < 4 {
		v.PlayerLabels[pos] = label
	}
}

// SetTrickCard sets a card in the trick area
func (v *DemoTableView) SetTrickCard(pos int, card engine.Card) {
	if pos >= 0 && pos < 4 {
		v.TrickCards[pos] = card
		v.ShowCards[pos] = true
	}
}

// ClearTrickCards removes all cards from the trick area
func (v *DemoTableView) ClearTrickCards() {
	for i := range v.TrickCards {
		v.TrickCards[i] = engine.Card{}
		v.ShowCards[i] = false
	}
}

// SetHand sets the hand for a player position
func (v *DemoTableView) SetHand(pos int, cards []engine.Card) {
	if pos >= 0 && pos < 4 {
		v.HandCards[pos] = cards
	}
}

// SetHighlighted highlights or unhighlights a player
func (v *DemoTableView) SetHighlighted(pos int, highlighted bool) {
	if pos >= 0 && pos < 4 {
		v.Highlighted[pos] = highlighted
	}
}

// SetDimmed dims or undims a player
func (v *DemoTableView) SetDimmed(pos int, dimmed bool) {
	if pos >= 0 && pos < 4 {
		v.Dimmed[pos] = dimmed
	}
}

// SetWinner marks a position as the winner
func (v *DemoTableView) SetWinner(pos int) {
	v.WinnerPos = pos
}

// SetAnnotation sets an annotation for a player
func (v *DemoTableView) SetAnnotation(pos int, annotation string) {
	if pos >= 0 && pos < 4 {
		v.Annotations[pos] = annotation
	}
}
