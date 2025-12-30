package tutorial

import "github.com/bran/euchre/internal/engine"

// VisualElementType defines the type of visual content in a lesson
type VisualElementType int

const (
	VisualSingleCard     VisualElementType = iota // Display one card with optional label
	VisualCardRow                                 // Multiple cards in a row
	VisualCardGrid                                // Cards in a grid (multiple rows)
	VisualHand                                    // Full 5-card hand display with highlights
	VisualCardComparison                          // Side-by-side card comparison
	VisualTrickDemo                               // 4 cards in trick area layout
	VisualTrumpHierarchy                          // Vertical ranked list of trump cards
	VisualTableLayout                             // 4-player table positions
)

// AnnotationStyle defines how a card annotation should be rendered
type AnnotationStyle string

const (
	AnnotationHighlight AnnotationStyle = "highlight" // Emphasize this card
	AnnotationDim       AnnotationStyle = "dim"       // De-emphasize this card
	AnnotationWinner    AnnotationStyle = "winner"    // Mark as winning card
	AnnotationLoser     AnnotationStyle = "loser"     // Mark as losing card
	AnnotationPlayable  AnnotationStyle = "playable"  // Mark as playable
	AnnotationRequired  AnnotationStyle = "required"  // Must play this
)

// CardAnnotation adds a label or visual style to a specific card
type CardAnnotation struct {
	CardIndex int             // Which card in the Cards slice (0-indexed)
	Label     string          // Text label (e.g., "Right Bower", "Highest")
	Style     AnnotationStyle // Visual style to apply
}

// SequenceAction defines what happens in a sequence step
type SequenceAction string

const (
	ActionShow      SequenceAction = "show"      // Show a card or element
	ActionHighlight SequenceAction = "highlight" // Highlight a card
	ActionPlay      SequenceAction = "play"      // Play a card to the trick
	ActionCollect   SequenceAction = "collect"   // Collect the trick
	ActionMessage   SequenceAction = "message"   // Just show a message
)

// SequenceStep represents one frame in an animated demonstration
type SequenceStep struct {
	Action    SequenceAction // What action to take
	PlayerIdx int            // Which player position (0=you, 1=left, 2=partner, 3=right)
	Card      engine.Card    // Card involved in this step
	Message   string         // Text to display during this step
	PauseMs   int            // Milliseconds to pause (0 = wait for user to advance)
}

// PlayerPosition represents a position at the table
type PlayerPosition struct {
	Label    string          // Display label (e.g., "You", "Partner")
	Style    AnnotationStyle // Visual style (e.g., "highlight", "dim")
	CardBack bool            // Show card back instead of face
}

// VisualElement represents a visual demonstration in a lesson
type VisualElement struct {
	Type VisualElementType

	// Cards to display (interpretation depends on Type)
	Cards []engine.Card

	// Trump suit context (for bower highlighting and trump identification)
	Trump engine.Suit

	// Annotations for specific cards
	Annotations []CardAnnotation

	// For animated sequences (VisualTrickDemo)
	Sequence []SequenceStep

	// For table layouts
	Players [4]PlayerPosition

	// Highlight specific card indices (for VisualHand)
	HighlightIndices []int

	// Caption text displayed below the visual
	Caption string

	// For comparisons - labels for left and right sides
	LeftLabel  string
	RightLabel string

	// For grid layout - number of cards per row
	CardsPerRow int
}

// VisualSection represents a lesson section with visual content
type VisualSection struct {
	Title      string         // Section heading
	TextBefore string         // Explanatory text before the visual
	Visual     *VisualElement // The visual demonstration (nil for text-only)
	TextAfter  string         // Explanatory text after the visual
}

// Helper functions for creating common visual elements

// NewSingleCardVisual creates a visual showing one card with a label
func NewSingleCardVisual(card engine.Card, label string, trump engine.Suit) *VisualElement {
	annotations := []CardAnnotation{}
	if label != "" {
		annotations = append(annotations, CardAnnotation{
			CardIndex: 0,
			Label:     label,
			Style:     AnnotationHighlight,
		})
	}
	return &VisualElement{
		Type:        VisualSingleCard,
		Cards:       []engine.Card{card},
		Trump:       trump,
		Annotations: annotations,
	}
}

// NewCardRowVisual creates a visual showing multiple cards in a row
func NewCardRowVisual(cards []engine.Card, caption string) *VisualElement {
	return &VisualElement{
		Type:    VisualCardRow,
		Cards:   cards,
		Caption: caption,
	}
}

// NewCardGridVisual creates a visual showing cards in a grid layout
func NewCardGridVisual(cards []engine.Card, cardsPerRow int, caption string) *VisualElement {
	return &VisualElement{
		Type:        VisualCardGrid,
		Cards:       cards,
		CardsPerRow: cardsPerRow,
		Caption:     caption,
	}
}

// NewCardComparisonVisual creates a side-by-side comparison of cards
func NewCardComparisonVisual(leftCards, rightCards []engine.Card, leftLabel, rightLabel string, trump engine.Suit) *VisualElement {
	// Combine cards: left cards first, then right cards
	allCards := make([]engine.Card, 0, len(leftCards)+len(rightCards))
	allCards = append(allCards, leftCards...)
	allCards = append(allCards, rightCards...)

	return &VisualElement{
		Type:       VisualCardComparison,
		Cards:      allCards,
		Trump:      trump,
		LeftLabel:  leftLabel,
		RightLabel: rightLabel,
	}
}

// NewHandVisual creates a visual showing a 5-card hand with optional highlights
func NewHandVisual(cards []engine.Card, highlightIndices []int, trump engine.Suit, caption string) *VisualElement {
	return &VisualElement{
		Type:             VisualHand,
		Cards:            cards,
		Trump:            trump,
		HighlightIndices: highlightIndices,
		Caption:          caption,
	}
}

// NewTrickDemoVisual creates an animated trick demonstration
func NewTrickDemoVisual(sequence []SequenceStep, trump engine.Suit) *VisualElement {
	return &VisualElement{
		Type:     VisualTrickDemo,
		Trump:    trump,
		Sequence: sequence,
	}
}

// NewTrumpHierarchyVisual creates a vertical display of trump card ranking
func NewTrumpHierarchyVisual(trump engine.Suit) *VisualElement {
	// Build the trump hierarchy: Right Bower, Left Bower, A, K, Q, 10, 9
	var leftBowerSuit engine.Suit
	switch trump {
	case engine.Hearts:
		leftBowerSuit = engine.Diamonds
	case engine.Diamonds:
		leftBowerSuit = engine.Hearts
	case engine.Spades:
		leftBowerSuit = engine.Clubs
	case engine.Clubs:
		leftBowerSuit = engine.Spades
	}

	cards := []engine.Card{
		{Suit: trump, Rank: engine.Jack},          // Right Bower
		{Suit: leftBowerSuit, Rank: engine.Jack},  // Left Bower
		{Suit: trump, Rank: engine.Ace},           // Ace
		{Suit: trump, Rank: engine.King},          // King
		{Suit: trump, Rank: engine.Queen},         // Queen
		{Suit: trump, Rank: engine.Ten},           // 10
		{Suit: trump, Rank: engine.Nine},          // 9
	}

	annotations := []CardAnnotation{
		{CardIndex: 0, Label: "Right", Style: AnnotationHighlight},
		{CardIndex: 1, Label: "Left", Style: AnnotationHighlight},
	}

	return &VisualElement{
		Type:        VisualTrumpHierarchy,
		Cards:       cards,
		Trump:       trump,
		Annotations: annotations,
		Caption:     trump.String() + " is trump",
	}
}

// NewTableLayoutVisual creates a 4-player table position display
func NewTableLayoutVisual(players [4]PlayerPosition, caption string) *VisualElement {
	return &VisualElement{
		Type:    VisualTableLayout,
		Players: players,
		Caption: caption,
	}
}
