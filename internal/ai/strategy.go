package ai

import "github.com/bran/euchre/internal/engine"

// Strategy defines the interface for AI decision-making strategies
type Strategy interface {
	// EvaluateHandForBid scores a hand for bidding purposes (0-100)
	EvaluateHandForBid(hand []engine.Card, turnedCard engine.Card, round int, position int) int

	// ShouldGoAlone decides if the player should go alone
	ShouldGoAlone(hand []engine.Card, trump engine.Suit) bool

	// ChooseTrump selects the best trump suit for round 2 bidding
	ChooseTrump(hand []engine.Card, excludeSuit engine.Suit) (engine.Suit, bool)

	// SelectPlay chooses the best card to play
	SelectPlay(hand []engine.Card, trick *engine.Trick, state *engine.GameState) engine.Card

	// SelectDiscard chooses which card to discard
	SelectDiscard(hand []engine.Card, trump engine.Suit) engine.Card

	// SelectLead chooses the best card to lead
	SelectLead(hand []engine.Card, trump engine.Suit, state *engine.GameState) engine.Card
}

// handAnalysis contains analyzed information about a hand
type handAnalysis struct {
	trumpCount    int
	hasRightBower bool
	hasLeftBower  bool
	offAces       int
	voidSuits     int
	strength      int // Overall strength rating 0-100
}

// analyzeHand analyzes a hand for strategic value
func analyzeHand(hand []engine.Card, trump engine.Suit) handAnalysis {
	analysis := handAnalysis{}

	suitCounts := make(map[engine.Suit]int)

	for _, card := range hand {
		effectiveSuit := card.EffectiveSuit(trump)
		suitCounts[effectiveSuit]++

		if card.IsTrump(trump) {
			analysis.trumpCount++
			if card.IsRightBower(trump) {
				analysis.hasRightBower = true
			}
			if card.IsLeftBower(trump) {
				analysis.hasLeftBower = true
			}
		} else if card.Rank == engine.Ace {
			analysis.offAces++
		}
	}

	// Count void suits
	for _, suit := range []engine.Suit{engine.Clubs, engine.Diamonds, engine.Hearts, engine.Spades} {
		if suit != trump && suitCounts[suit] == 0 {
			analysis.voidSuits++
		}
	}

	// Calculate overall strength
	strength := 0

	// Trump count is very important
	strength += analysis.trumpCount * 15

	// Bowers are crucial
	if analysis.hasRightBower {
		strength += 20
	}
	if analysis.hasLeftBower {
		strength += 15
	}

	// Off-suit aces are valuable
	strength += analysis.offAces * 10

	// Voids help for trumping
	strength += analysis.voidSuits * 5

	if strength > 100 {
		strength = 100
	}
	analysis.strength = strength

	return analysis
}

// countTrumpsIfOrdered counts potential trumps if a suit is made trump
func countTrumpsIfOrdered(hand []engine.Card, potentialTrump engine.Suit) int {
	count := 0
	for _, card := range hand {
		if card.Suit == potentialTrump || card.IsLeftBower(potentialTrump) {
			count++
		}
	}
	return count
}

// findSameColorJack finds the Jack of the same color as a suit
func findSameColorJack(suit engine.Suit) engine.Card {
	switch suit {
	case engine.Hearts:
		return engine.NewCard(engine.Diamonds, engine.Jack)
	case engine.Diamonds:
		return engine.NewCard(engine.Hearts, engine.Jack)
	case engine.Spades:
		return engine.NewCard(engine.Clubs, engine.Jack)
	case engine.Clubs:
		return engine.NewCard(engine.Spades, engine.Jack)
	default:
		return engine.Card{}
	}
}

// hasCard checks if hand contains a specific card
func hasCard(hand []engine.Card, target engine.Card) bool {
	for _, c := range hand {
		if c.Suit == target.Suit && c.Rank == target.Rank {
			return true
		}
	}
	return false
}
