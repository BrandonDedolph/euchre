package ai

import "github.com/BrandonDedolph/euchre/internal/engine"

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
