package components

import (
	"strings"

	"github.com/bran/euchre/internal/engine"
	"github.com/bran/euchre/internal/tutorial"
	"github.com/bran/euchre/internal/ui/theme"
	"github.com/charmbracelet/lipgloss"
)

// LessonVisualView renders visual elements for lessons
type LessonVisualView struct {
	Element *tutorial.VisualElement
	Width   int
	Height  int

	// Animation state
	AnimFrame     int  // Current frame in sequence
	IsAnimating   bool // Whether animation is playing
	SequenceIndex int  // Current step in sequence
}

// NewLessonVisualView creates a new visual renderer
func NewLessonVisualView(element *tutorial.VisualElement, width, height int) *LessonVisualView {
	return &LessonVisualView{
		Element: element,
		Width:   width,
		Height:  height,
	}
}

// Render returns the visual element as a string
func (v *LessonVisualView) Render() string {
	if v.Element == nil {
		return ""
	}

	var content string
	switch v.Element.Type {
	case tutorial.VisualSingleCard:
		content = v.renderSingleCard()
	case tutorial.VisualCardRow:
		content = v.renderCardRow()
	case tutorial.VisualCardGrid:
		content = v.renderCardGrid()
	case tutorial.VisualHand:
		content = v.renderHand()
	case tutorial.VisualCardComparison:
		content = v.renderComparison()
	case tutorial.VisualTrickDemo:
		content = v.renderTrickDemo()
	case tutorial.VisualTrumpHierarchy:
		content = v.renderHierarchy()
	case tutorial.VisualTableLayout:
		content = v.renderTableLayout()
	default:
		content = "[Unknown visual type]"
	}

	// Add caption if present
	if v.Element.Caption != "" {
		captionStyle := theme.Current.Muted.Italic(true).Align(lipgloss.Center)
		caption := captionStyle.Render(v.Element.Caption)
		content = lipgloss.JoinVertical(lipgloss.Center, content, "", caption)
	}

	return content
}

// renderSingleCard renders a single card with optional label
func (v *LessonVisualView) renderSingleCard() string {
	if len(v.Element.Cards) == 0 {
		return ""
	}

	card := v.Element.Cards[0]
	cv := NewCardView(card)
	cardStr := cv.Render()

	// Find annotation for this card
	var label string
	for _, ann := range v.Element.Annotations {
		if ann.CardIndex == 0 {
			label = ann.Label
			break
		}
	}

	if label != "" {
		labelStyle := theme.Current.Primary.Bold(true).Align(lipgloss.Center)
		labelStr := labelStyle.Render(label)
		return lipgloss.JoinVertical(lipgloss.Center, cardStr, "", labelStr)
	}

	return cardStr
}

// renderCardRow renders multiple cards in a horizontal row
func (v *LessonVisualView) renderCardRow() string {
	if len(v.Element.Cards) == 0 {
		return ""
	}

	cards := make([]string, len(v.Element.Cards))
	for i, card := range v.Element.Cards {
		cv := NewCardView(card)
		// Apply annotation style if present
		for _, ann := range v.Element.Annotations {
			if ann.CardIndex == i {
				cv.Style = annotationToCardStyle(ann.Style)
				break
			}
		}
		cards[i] = cv.Render()
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, cards...)
}

// renderCardGrid renders cards in a grid with multiple rows
func (v *LessonVisualView) renderCardGrid() string {
	if len(v.Element.Cards) == 0 {
		return ""
	}

	cardsPerRow := v.Element.CardsPerRow
	if cardsPerRow <= 0 {
		cardsPerRow = 6 // Default to 6 cards per row
	}

	var rows []string

	for i := 0; i < len(v.Element.Cards); i += cardsPerRow {
		end := i + cardsPerRow
		if end > len(v.Element.Cards) {
			end = len(v.Element.Cards)
		}

		rowCards := make([]string, end-i)
		for j, card := range v.Element.Cards[i:end] {
			cv := NewCardView(card)
			// Apply annotation style if present
			for _, ann := range v.Element.Annotations {
				if ann.CardIndex == i+j {
					cv.Style = annotationToCardStyle(ann.Style)
					break
				}
			}
			rowCards[j] = cv.Render()
		}

		row := lipgloss.JoinHorizontal(lipgloss.Top, rowCards...)
		rows = append(rows, row)
	}

	return lipgloss.JoinVertical(lipgloss.Center, rows...)
}

// renderHand renders a 5-card hand with highlights
func (v *LessonVisualView) renderHand() string {
	if len(v.Element.Cards) == 0 {
		return ""
	}

	// Build list of playable cards based on highlight indices
	var playableCards []engine.Card
	for _, idx := range v.Element.HighlightIndices {
		if idx >= 0 && idx < len(v.Element.Cards) {
			playableCards = append(playableCards, v.Element.Cards[idx])
		}
	}

	// Use -1 for selectedIdx since we don't want selection highlighting
	return RenderHand(v.Element.Cards, -1, playableCards)
}

// renderComparison renders cards side-by-side with labels
func (v *LessonVisualView) renderComparison() string {
	if len(v.Element.Cards) == 0 {
		return ""
	}

	// Assume first half is left, second half is right
	mid := len(v.Element.Cards) / 2
	if mid == 0 {
		mid = 1
	}

	leftCards := v.Element.Cards[:mid]
	rightCards := v.Element.Cards[mid:]

	// Render left side
	leftRendered := make([]string, len(leftCards))
	for i, card := range leftCards {
		cv := NewCardView(card)
		leftRendered[i] = cv.Render()
	}
	leftGroup := lipgloss.JoinHorizontal(lipgloss.Top, leftRendered...)

	// Render right side
	rightRendered := make([]string, len(rightCards))
	for i, card := range rightCards {
		cv := NewCardView(card)
		rightRendered[i] = cv.Render()
	}
	rightGroup := lipgloss.JoinHorizontal(lipgloss.Top, rightRendered...)

	// Add labels
	labelStyle := theme.Current.Primary.Bold(true).Align(lipgloss.Center)

	leftWithLabel := leftGroup
	if v.Element.LeftLabel != "" {
		leftWithLabel = lipgloss.JoinVertical(lipgloss.Center,
			labelStyle.Render(v.Element.LeftLabel),
			"",
			leftGroup,
		)
	}

	rightWithLabel := rightGroup
	if v.Element.RightLabel != "" {
		rightWithLabel = lipgloss.JoinVertical(lipgloss.Center,
			labelStyle.Render(v.Element.RightLabel),
			"",
			rightGroup,
		)
	}

	// Add "vs" in the middle
	vsStyle := theme.Current.Muted.Bold(true)
	vs := vsStyle.Render("  vs  ")

	return lipgloss.JoinHorizontal(lipgloss.Center, leftWithLabel, vs, rightWithLabel)
}

// renderTrickDemo renders a trick demonstration (4 cards in compass layout)
func (v *LessonVisualView) renderTrickDemo() string {
	// If we have a sequence, show the current step
	if len(v.Element.Sequence) > 0 {
		return v.renderSequenceStep()
	}

	// Static trick display - show all cards
	return v.renderTrickArea(v.Element.Cards)
}

// renderSequenceStep renders the current step of an animation sequence
func (v *LessonVisualView) renderSequenceStep() string {
	if v.SequenceIndex >= len(v.Element.Sequence) {
		// Show final state with all cards
		return v.renderTrickArea(v.Element.Cards)
	}

	// Collect cards played so far
	var cardsPlayed []engine.Card
	var positions []int

	for i := 0; i <= v.SequenceIndex; i++ {
		step := v.Element.Sequence[i]
		if step.Action == tutorial.ActionPlay {
			cardsPlayed = append(cardsPlayed, step.Card)
			positions = append(positions, step.PlayerIdx)
		}
	}

	// Build the trick area with only played cards
	trickCards := make([]engine.Card, 4)
	for i, card := range cardsPlayed {
		if i < len(positions) {
			trickCards[positions[i]] = card
		}
	}

	trick := v.renderTrickArea(trickCards)

	// Add current step message
	currentStep := v.Element.Sequence[v.SequenceIndex]
	if currentStep.Message != "" {
		msgStyle := theme.Current.Body.Italic(true).Align(lipgloss.Center)
		msg := msgStyle.Render(currentStep.Message)
		trick = lipgloss.JoinVertical(lipgloss.Center, trick, "", msg)
	}

	return trick
}

// renderTrickArea renders 4 cards in a compass layout
func (v *LessonVisualView) renderTrickArea(cards []engine.Card) string {
	// Layout:
	//     [Partner/Top]
	// [Left]         [Right]
	//     [You/Bottom]

	cardWidth := 7
	cardHeight := 5
	emptyCard := strings.Repeat(" ", cardWidth)
	emptyCardBlock := ""
	for i := 0; i < cardHeight; i++ {
		if i > 0 {
			emptyCardBlock += "\n"
		}
		emptyCardBlock += emptyCard
	}

	// Render each position's card
	renderPos := func(idx int) string {
		// Check if card exists and is not the zero value
		// (zero value would be Clubs+Nine, but we check if it's an intentional card)
		if idx < len(cards) {
			card := cards[idx]
			// A zero-value card has Suit=Clubs(0) and Rank=Nine(0)
			// Consider it empty only if it matches exactly
			emptyCard := engine.Card{}
			if card != emptyCard {
				cv := NewCardView(card)
				return cv.Render()
			}
		}
		return emptyCardBlock
	}

	// Position mapping: 0=you(bottom), 1=left, 2=partner(top), 3=right
	bottomCard := renderPos(0)
	leftCard := renderPos(1)
	topCard := renderPos(2)
	rightCard := renderPos(3)

	// Build layout
	topRow := lipgloss.NewStyle().Width(cardWidth*3 + 4).Align(lipgloss.Center).Render(topCard)

	middleRow := lipgloss.JoinHorizontal(lipgloss.Center,
		leftCard,
		strings.Repeat(" ", cardWidth+4),
		rightCard,
	)

	bottomRow := lipgloss.NewStyle().Width(cardWidth*3 + 4).Align(lipgloss.Center).Render(bottomCard)

	return lipgloss.JoinVertical(lipgloss.Center, topRow, middleRow, bottomRow)
}

// renderHierarchy renders trump cards in order with rank labels
func (v *LessonVisualView) renderHierarchy() string {
	if len(v.Element.Cards) == 0 {
		return ""
	}

	rankLabels := []string{"1st", "2nd", "3rd", "4th", "5th", "6th", "7th"}
	cardWidth := 7 // Width of a rendered card

	// Build labels row and cards row
	var labelParts []string
	var cardParts []string

	for i, card := range v.Element.Cards {
		cv := NewCardView(card)
		cardStr := cv.Render()
		cardParts = append(cardParts, cardStr)

		// Create centered label
		label := ""
		if i < len(rankLabels) {
			label = rankLabels[i]
		}

		// Find annotation for special labels
		for _, ann := range v.Element.Annotations {
			if ann.CardIndex == i && ann.Label != "" {
				label = ann.Label
				break
			}
		}

		// Style and center the label to match card width
		labelStyle := theme.Current.Muted
		if i < 2 {
			// Highlight bowers
			labelStyle = theme.Current.Primary.Bold(true)
		}
		centeredLabel := lipgloss.NewStyle().Width(cardWidth).Align(lipgloss.Center).Render(labelStyle.Render(label))
		labelParts = append(labelParts, centeredLabel)
	}

	labelsRow := lipgloss.JoinHorizontal(lipgloss.Top, labelParts...)
	cardsRow := lipgloss.JoinHorizontal(lipgloss.Top, cardParts...)

	return lipgloss.JoinVertical(lipgloss.Center, labelsRow, cardsRow)
}

// renderTableLayout renders 4 player positions
func (v *LessonVisualView) renderTableLayout() string {
	// Layout:
	//        [Partner]
	//   [Left]     [Right]
	//        [You]

	posStyle := func(pos tutorial.PlayerPosition) lipgloss.Style {
		style := theme.Current.Body
		switch pos.Style {
		case tutorial.AnnotationHighlight:
			style = theme.Current.Primary.Bold(true)
		case tutorial.AnnotationDim:
			style = theme.Current.Muted
		case tutorial.AnnotationWinner:
			style = theme.Current.Success.Bold(true)
		}
		return style
	}

	// Default labels if not specified
	players := v.Element.Players
	if players[0].Label == "" {
		players[0].Label = "You"
	}
	if players[1].Label == "" {
		players[1].Label = "Opponent"
	}
	if players[2].Label == "" {
		players[2].Label = "Partner"
	}
	if players[3].Label == "" {
		players[3].Label = "Opponent"
	}

	// Render each position
	renderPlayer := func(idx int) string {
		p := players[idx]
		style := posStyle(p)
		boxStyle := theme.Current.Border.Width(12).Align(lipgloss.Center)
		return boxStyle.Render(style.Render(p.Label))
	}

	topRow := lipgloss.NewStyle().Width(40).Align(lipgloss.Center).Render(renderPlayer(2))

	leftPlayer := renderPlayer(1)
	rightPlayer := renderPlayer(3)
	middleRow := lipgloss.JoinHorizontal(lipgloss.Center,
		leftPlayer,
		strings.Repeat(" ", 8),
		rightPlayer,
	)

	bottomRow := lipgloss.NewStyle().Width(40).Align(lipgloss.Center).Render(renderPlayer(0))

	return lipgloss.JoinVertical(lipgloss.Center, topRow, "", middleRow, "", bottomRow)
}

// AdvanceSequence moves to the next step in an animation sequence
func (v *LessonVisualView) AdvanceSequence() bool {
	if v.Element == nil || len(v.Element.Sequence) == 0 {
		return false
	}
	if v.SequenceIndex < len(v.Element.Sequence)-1 {
		v.SequenceIndex++
		return true
	}
	return false
}

// ResetSequence resets the animation sequence to the beginning
func (v *LessonVisualView) ResetSequence() {
	v.SequenceIndex = 0
	v.AnimFrame = 0
	v.IsAnimating = false
}

// IsSequenceComplete returns true if the sequence has finished
func (v *LessonVisualView) IsSequenceComplete() bool {
	if v.Element == nil || len(v.Element.Sequence) == 0 {
		return true
	}
	return v.SequenceIndex >= len(v.Element.Sequence)-1
}

// GetCurrentStepPause returns the pause duration for the current step (0 = wait for user)
func (v *LessonVisualView) GetCurrentStepPause() int {
	if v.Element == nil || len(v.Element.Sequence) == 0 {
		return 0
	}
	if v.SequenceIndex >= len(v.Element.Sequence) {
		return 0
	}
	return v.Element.Sequence[v.SequenceIndex].PauseMs
}

// annotationToCardStyle converts an annotation style to a CardStyle
func annotationToCardStyle(style tutorial.AnnotationStyle) CardStyle {
	switch style {
	case tutorial.AnnotationHighlight, tutorial.AnnotationRequired:
		return CardStylePlayable
	case tutorial.AnnotationDim, tutorial.AnnotationLoser:
		return CardStyleDisabled
	case tutorial.AnnotationWinner:
		return CardStylePlayable
	default:
		return CardStyleNormal
	}
}
