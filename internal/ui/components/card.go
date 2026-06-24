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
	CardStyleCoachPick   // tutorial: the coach's recommended card - gold double border
	CardStyleTrickWinner // the card that won the just-completed trick - bold team-colored double border + ★
)

// isCoachPick reports whether the style is the coach's recommended card.
func (s CardStyle) isCoachPick() bool { return s == CardStyleCoachPick }

// isTrickWinner reports whether the style is the trick-winning card.
func (s CardStyle) isTrickWinner() bool { return s == CardStyleTrickWinner }

// hasDoubleBorder reports whether the style uses the heavier double-line border.
func (s CardStyle) hasDoubleBorder() bool { return s.isCoachPick() || s.isTrickWinner() }

// CardView represents a visual card component
type CardView struct {
	Card    engine.Card
	Style   CardStyle
	FaceUp  bool
	Compact bool
	// Trump, when set (not NoSuit), lets the card flag itself as the left bower
	// — the off-suit jack that actually plays as trump — with a small trump pip.
	Trump engine.Suit
	// AccentColor tints the border for CardStyleTrickWinner (the winning team's
	// color). Ignored by other styles.
	AccentColor lipgloss.TerminalColor
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
	contentColor := lipgloss.Color("#000000") // black for clubs/spades
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

	// Build each interior line as a complete styled unit (5 chars wide).
	// For the left bower (the off-suit jack that plays as trump), tuck a small
	// trump pip into the top-right corner so a learner can see it counts as
	// trump despite its printed suit.
	topLine := rankPad + "   "
	if c.Trump != engine.NoSuit && c.Card.IsLeftBower(c.Trump) {
		topLine = rankPad + "  " + c.Trump.Symbol()
	}
	// The trick-winner crown takes precedence over the left-bower pip: stamp a
	// ★ into the same top-right interior cell.
	if c.Style.isTrickWinner() {
		topLine = rankPad + "  ★"
	}
	interior1 := interiorStyle.Render(topLine)
	interior2 := interiorStyle.Render("  " + suit + "  ")
	interior3 := interiorStyle.Render("   " + rankPad)

	border := borderStyle.Render

	// The coach's recommended card and the trick winner get a double border to
	// stand out from the single-border normal/playable cards.
	tl, tr, bl, br, h, v := "┌", "┐", "└", "┘", "─", "│"
	if c.Style.hasDoubleBorder() {
		tl, tr, bl, br, h, v = "╔", "╗", "╚", "╝", "═", "║"
	}
	rule := strings.Repeat(h, 5)

	lines := []string{
		border(tl + rule + tr),
		border(v) + interior1 + border(v),
		border(v) + interior2 + border(v),
		border(v) + interior3 + border(v),
		border(bl + rule + br),
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
	borderStyle := theme.Current.Muted
	patternStyle := theme.Current.CardPattern

	border := borderStyle.Render
	pattern := patternStyle.Render

	lines := []string{
		border("┌─────┐"),
		border("│") + pattern("░░░░░") + border("│"),
		border("│") + pattern("░░░░░") + border("│"),
		border("│") + pattern("░░░░░") + border("│"),
		border("└─────┘"),
	}

	return strings.Join(lines, "\n")
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
		contentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#000000"))
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
	case CardStyleCoachPick:
		// Coach's recommended card: bold gold (double) border.
		goldBorder := lipgloss.NewStyle().Foreground(theme.ColGold).Bold(true)
		return contentStyle, goldBorder, bgStyle
	case CardStyleTrickWinner:
		// Trick-winning card: bold double border in the winning team's accent
		// color. Falls back to gold if no accent was supplied.
		accent := c.AccentColor
		if accent == nil {
			accent = theme.ColGold
		}
		winnerBorder := lipgloss.NewStyle().Foreground(accent).Bold(true)
		return contentStyle, winnerBorder, bgStyle
	default:
		return contentStyle, borderStyle, bgStyle
	}
}

// RenderHand renders a hand of cards horizontally
// Colors / cues:
//   - Blue dashed border: Currently selected card (press Enter to play)
//   - Green border + a "▾" marker above: Legal cards you can play (the marker
//     is a colorblind-safe secondary cue so green isn't the only signal)
//   - Dimmed/gray: Cards you cannot play right now (must follow suit)
//   - Normal (red/black): When it's not your turn, all cards shown normally
//   - A small trump pip on the left bower when trump is set
//   - Gold double border + a gold "▼" arrow above: the coach's recommended card
//     (tutorial mode); overrides the green legal-move marker on that card
//
// Set selectedIdx to -1 to disable selection highlighting; pass trump as
// engine.NoSuit to skip the left-bower flag, and coachPick as -1 for no pick.
func RenderHand(cards []engine.Card, selectedIdx int, playableCards []engine.Card, trump engine.Suit, coachPick int) string {
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
	playableFlags := make([]bool, len(cards))
	for i, card := range cards {
		cv := NewCardView(card)
		cv.Trump = trump
		isSelected := selectedIdx >= 0 && i == selectedIdx
		isPlayable := hasPlayableInfo && playable[card.String()]
		playableFlags[i] = isPlayable

		switch {
		case i == coachPick:
			cv.Style = CardStyleCoachPick // gold spotlight wins over other styles
		case isSelected && isPlayable:
			cv.Style = CardStyleSelectedPlayable
		case isSelected:
			cv.Style = CardStyleSelected
		case isPlayable:
			cv.Style = CardStylePlayable
		case hasPlayableInfo && !playable[card.String()]:
			cv.Style = CardStyleDisabled
		}
		cardViews[i] = cv
	}

	// Render cards with raised effect for selected card, plus a marker row on
	// top: a gold "▼" over the coach's pick, else a green "▾" over legal cards.
	renderedCards := make([]string, len(cardViews))
	cardWidth := 7 // width of a card "┌─────┐"
	emptyLine := strings.Repeat(" ", cardWidth)
	legalMark := theme.Current.Success.Bold(true)
	coachMark := lipgloss.NewStyle().Foreground(theme.ColGold).Bold(true)

	for i, cv := range cardViews {
		card := cv.Render()
		isSelected := selectedIdx >= 0 && i == selectedIdx

		marker := emptyLine
		switch {
		case i == coachPick:
			marker = coachMark.Render(lipgloss.PlaceHorizontal(cardWidth, lipgloss.Center, "▼"))
		case playableFlags[i]:
			marker = legalMark.Render(lipgloss.PlaceHorizontal(cardWidth, lipgloss.Center, "▾"))
		}

		// The marker hugs the top of its own card so it tracks the selection's
		// raise/lower instead of floating at a fixed top row: directly above the
		// raised (selected) card, and dropped a row to stay directly above the
		// lowered (non-selected) cards. Each column stays 7 rows tall either way.
		if isSelected {
			// Selected card: raised (marker, then card, then bottom padding).
			renderedCards[i] = marker + "\n" + card + "\n" + emptyLine
		} else {
			// Non-selected: lowered (top padding, then marker, then card).
			renderedCards[i] = emptyLine + "\n" + marker + "\n" + card
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

	borderStyle := theme.Current.Muted
	patternStyle := theme.Current.CardPattern

	border := borderStyle.Render
	pattern := patternStyle.Render

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
			lines[0] += border("┌─")
			lines[1] += border("│") + pattern("░")
			lines[2] += border("│") + pattern("░")
			lines[3] += border("│") + pattern("░")
			lines[4] += border("└─")
		} else {
			// Last card - show full
			lines[0] += border("┌─────┐")
			lines[1] += border("│") + pattern("░░░░░") + border("│")
			lines[2] += border("│") + pattern("░░░░░") + border("│")
			lines[3] += border("│") + pattern("░░░░░") + border("│")
			lines[4] += border("└─────┘")
		}
	}

	return strings.Join(lines[:], "\n")
}

// RenderFaceDownVertical renders face-down cards stacked vertically (for side players)
// Cards are 9 wide x 4 tall, stacked with 1-line overlaps
// If reversed is true, cards stack upward (bottoms showing) instead of downward (tops showing)
func RenderFaceDownVertical(count int, reversed bool) string {
	borderStyle := theme.Current.Muted
	patternStyle := theme.Current.CardPattern

	border := borderStyle.Render
	pattern := patternStyle.Render

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

	// Helper for interior line
	interiorLine := border("│") + pattern("░░░░░░░") + border("│")

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
		sb.WriteString(border("┌───────┐"))
		sb.WriteString("\n")
		sb.WriteString(interiorLine)
		sb.WriteString("\n")
		sb.WriteString(interiorLine)
		sb.WriteString("\n")
		sb.WriteString(border("└───────┘"))

		// Then bottom edges of cards behind
		for i := 1; i < count; i++ {
			sb.WriteString("\n")
			sb.WriteString(border("└───────┘"))
		}
	} else {
		// Top edges of cards behind first
		for i := 0; i < count-1; i++ {
			sb.WriteString(border("┌───────┐"))
			sb.WriteString("\n")
		}
		// Then full card at bottom
		sb.WriteString(border("┌───────┐"))
		sb.WriteString("\n")
		sb.WriteString(interiorLine)
		sb.WriteString("\n")
		sb.WriteString(interiorLine)
		sb.WriteString("\n")
		sb.WriteString(border("└───────┘"))

		// Pad with empty lines at bottom
		cardLines := 4 + (count - 1)
		for i := cardLines; i < totalLines; i++ {
			sb.WriteString("\n")
			sb.WriteString(emptyLine)
		}
	}

	return sb.String()
}
