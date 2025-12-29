package components

import (
	"strings"

	"github.com/bran/euchre/internal/engine"
	"github.com/bran/euchre/internal/ui/theme"
	"github.com/charmbracelet/lipgloss"
)

// CardStyle defines the rendering style for a card
type CardStyle int

const (
	CardStyleNormal CardStyle = iota
	CardStyleSelected
	CardStylePlayable
	CardStyleSelectedPlayable // Selected AND playable - green border with selection indicator
	CardStyleDisabled
	CardStyleFaceDown
)

// CardView represents a visual card component
type CardView struct {
	Card     engine.Card
	Style    CardStyle
	FaceUp   bool
	Compact  bool
}

// NewCardView creates a new card view
func NewCardView(card engine.Card) *CardView {
	return &CardView{
		Card:   card,
		Style:  CardStyleNormal,
		FaceUp: true,
	}
}

// Render returns the visual representation of the card
func (c *CardView) Render() string {
	if !c.FaceUp {
		return c.renderFaceDown()
	}

	if c.Compact {
		return c.renderCompact()
	}

	return c.renderFull()
}

// renderFull renders a full-size card
func (c *CardView) renderFull() string {
	rank := c.Card.Rank.String()
	suit := c.Card.Suit.Symbol()

	// Pad rank for alignment
	rankPad := rank
	if len(rank) == 1 {
		rankPad = rank + " "
	}

	// Get styles
	_, borderStyle, _ := c.getStyles()

	// Get foreground color for content based on suit
	contentColor := lipgloss.Color("#2C3E50") // dark for clubs/spades
	if c.Card.Suit == engine.Hearts || c.Card.Suit == engine.Diamonds {
		contentColor = lipgloss.Color("#E74C3C") // red for hearts/diamonds
	}

	// Adjust colors based on card style
	whiteBg := lipgloss.Color("#FFFFFF")
	interiorBg := whiteBg

	switch c.Style {
	case CardStyleDisabled:
		contentColor = lipgloss.Color("#666666")
		interiorBg = lipgloss.Color("#CCCCCC")
	}

	// Create interior style with background
	interiorStyle := lipgloss.NewStyle().
		Background(interiorBg).
		Foreground(contentColor)

	// Build each interior line as a complete styled unit (5 chars wide)
	interior1 := interiorStyle.Render(rankPad + "   ")
	interior2 := interiorStyle.Render("  " + suit + "  ")
	interior3 := interiorStyle.Render("   " + rankPad)

	border := borderStyle.Render

	lines := []string{
		border("┌─────┐"),
		border("│") + interior1 + border("│"),
		border("│") + interior2 + border("│"),
		border("│") + interior3 + border("│"),
		border("└─────┘"),
	}

	cardStr := strings.Join(lines, "\n")

	return cardStr
}

// renderCompact renders a compact card representation
func (c *CardView) renderCompact() string {
	style := c.getStyle()
	return style.Render(c.Card.ShortString())
}

// renderFaceDown renders a face-down card
func (c *CardView) renderFaceDown() string {
	lines := []string{
		"┌─────┐",
		"│░░░░░│",
		"│░░░░░│",
		"│░░░░░│",
		"└─────┘",
	}

	style := theme.Current.Muted
	styled := make([]string, len(lines))
	for i, line := range lines {
		styled[i] = style.Render(line)
	}

	return strings.Join(styled, "\n")
}

// getStyle returns the appropriate lipgloss style (for compact rendering)
func (c *CardView) getStyle() lipgloss.Style {
	contentStyle, _, _ := c.getStyles()
	return contentStyle
}

// getStyles returns separate styles for content (rank/suit), border, and background
func (c *CardView) getStyles() (contentStyle, borderStyle, bgStyle lipgloss.Style) {
	// Default border is a neutral gray
	borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#7F8C8D"))

	// No background by default (use terminal default)
	bgStyle = lipgloss.NewStyle()

	// Content color based on suit
	if c.Card.Suit == engine.Hearts || c.Card.Suit == engine.Diamonds {
		contentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E74C3C"))
	} else {
		contentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#2C3E50"))
	}

	switch c.Style {
	case CardStyleSelected:
		// Selected: keep normal border (outer dashed border indicates selection)
		return contentStyle, borderStyle, bgStyle
	case CardStylePlayable, CardStyleSelectedPlayable:
		// Playable: green border, keep normal suit color for content
		// SelectedPlayable also gets green border (outer dashed border added in renderFull)
		greenBorder := lipgloss.NewStyle().Foreground(lipgloss.Color("#27AE60"))
		return contentStyle, greenBorder, bgStyle
	case CardStyleDisabled:
		// Disabled: dim gray text
		disabledStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
		return disabledStyle, disabledStyle, bgStyle
	default:
		return contentStyle, borderStyle, bgStyle
	}
}

// RenderHand renders a hand of cards horizontally
// Colors:
//   - Blue dashed border: Currently selected card (press Enter to play)
//   - Green border: Legal cards you can play
//   - Dimmed/gray: Cards you cannot play right now (must follow suit)
//   - Normal (red/black): When it's not your turn, all cards shown normally
// Set selectedIdx to -1 to disable selection highlighting
func RenderHand(cards []engine.Card, selectedIdx int, playableCards []engine.Card) string {
	if len(cards) == 0 {
		return ""
	}

	// Build playable set for quick lookup
	playable := make(map[string]bool)
	for _, c := range playableCards {
		playable[c.String()] = true
	}

	// If we have playable cards specified, mark non-playable as disabled
	hasPlayableInfo := len(playableCards) > 0

	// Render each card
	cardViews := make([]*CardView, len(cards))
	for i, card := range cards {
		cv := NewCardView(card)
		isSelected := selectedIdx >= 0 && i == selectedIdx
		isPlayable := hasPlayableInfo && playable[card.String()]

		if isSelected && isPlayable {
			cv.Style = CardStyleSelectedPlayable
		} else if isSelected {
			cv.Style = CardStyleSelected
		} else if isPlayable {
			cv.Style = CardStylePlayable
		} else if hasPlayableInfo && !playable[card.String()] {
			cv.Style = CardStyleDisabled
		}
		cardViews[i] = cv
	}

	// Render cards with raised effect for selected card
	renderedCards := make([]string, len(cardViews))
	cardWidth := 7 // width of a card "┌─────┐"
	emptyLine := strings.Repeat(" ", cardWidth)

	for i, cv := range cardViews {
		card := cv.Render()
		isSelected := selectedIdx >= 0 && i == selectedIdx

		if isSelected {
			// Selected card: no top padding (appears raised)
			// Add bottom padding to maintain alignment
			renderedCards[i] = card + "\n" + emptyLine
		} else {
			// Non-selected: add top padding (appears lower)
			renderedCards[i] = emptyLine + "\n" + card
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, renderedCards...)
}

// RenderCompactHand renders a hand in compact format
func RenderCompactHand(cards []engine.Card, selectedIdx int) string {
	parts := make([]string, len(cards))
	for i, card := range cards {
		cv := NewCardView(card)
		cv.Compact = true
		if i == selectedIdx {
			cv.Style = CardStyleSelected
		}
		parts[i] = cv.Render()
	}
	return strings.Join(parts, " ")
}

// RenderFaceDown renders multiple face-down cards horizontally with overlap
func RenderFaceDown(count int) string {
	if count == 0 {
		return ""
	}

	style := theme.Current.Muted

	// Build overlapping cards horizontally
	// Each card shows just left edge except the last one shows fully
	// Card structure:
	// ┌─────┐
	// │░░░░░│
	// │░░░░░│
	// │░░░░░│
	// └─────┘

	var lines [5]string

	for i := 0; i < count; i++ {
		if i < count-1 {
			// Overlapping card - just show left edge (2 chars)
			lines[0] += style.Render("┌─")
			lines[1] += style.Render("│░")
			lines[2] += style.Render("│░")
			lines[3] += style.Render("│░")
			lines[4] += style.Render("└─")
		} else {
			// Last card - show full
			lines[0] += style.Render("┌─────┐")
			lines[1] += style.Render("│░░░░░│")
			lines[2] += style.Render("│░░░░░│")
			lines[3] += style.Render("│░░░░░│")
			lines[4] += style.Render("└─────┘")
		}
	}

	return strings.Join(lines[:], "\n")
}

// RenderFaceDownVertical renders face-down cards stacked vertically (for side players)
// Cards are 9 wide x 4 tall, stacked with 1-line overlaps
// If reversed is true, cards stack upward (bottoms showing) instead of downward (tops showing)
func RenderFaceDownVertical(count int, reversed bool) string {
	style := theme.Current.Muted

	// Card (9 wide x 4 tall):
	// ┌───────┐
	// │░░░░░░░│
	// │░░░░░░░│
	// └───────┘
	const maxCards = 5
	const linesPerOverlap = 1
	const linesForLastCard = 4
	const totalLines = (maxCards-1)*linesPerOverlap + linesForLastCard // 4*1 + 4 = 8

	var sb strings.Builder
	cardWidth := 9 // "┌───────┐" is 9 chars wide
	emptyLine := strings.Repeat(" ", cardWidth)

	// Handle empty case
	if count == 0 {
		for i := 0; i < totalLines; i++ {
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(emptyLine)
		}
		return sb.String()
	}

	if reversed {
		// Calculate how many lines the cards will take
		cardLines := 4 + (count - 1) // full card + overlap lines
		paddingLines := totalLines - cardLines

		// Add padding at top first (cards grow upward)
		for i := 0; i < paddingLines; i++ {
			sb.WriteString(emptyLine)
			sb.WriteString("\n")
		}

		// Full card
		sb.WriteString(style.Render("┌───────┐"))
		sb.WriteString("\n")
		sb.WriteString(style.Render("│░░░░░░░│"))
		sb.WriteString("\n")
		sb.WriteString(style.Render("│░░░░░░░│"))
		sb.WriteString("\n")
		sb.WriteString(style.Render("└───────┘"))

		// Then bottom edges of cards behind
		for i := 1; i < count; i++ {
			sb.WriteString("\n")
			sb.WriteString(style.Render("└───────┘"))
		}
	} else {
		// Top edges of cards behind first
		for i := 0; i < count-1; i++ {
			sb.WriteString(style.Render("┌───────┐"))
			sb.WriteString("\n")
		}
		// Then full card at bottom
		sb.WriteString(style.Render("┌───────┐"))
		sb.WriteString("\n")
		sb.WriteString(style.Render("│░░░░░░░│"))
		sb.WriteString("\n")
		sb.WriteString(style.Render("│░░░░░░░│"))
		sb.WriteString("\n")
		sb.WriteString(style.Render("└───────┘"))

		// Pad with empty lines at bottom
		cardLines := 4 + (count - 1)
		for i := cardLines; i < totalLines; i++ {
			sb.WriteString("\n")
			sb.WriteString(emptyLine)
		}
	}

	return sb.String()
}
