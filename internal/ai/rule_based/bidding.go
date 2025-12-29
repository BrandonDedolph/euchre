package rule_based

import "github.com/bran/euchre/internal/engine"

// BiddingEvaluator handles bidding decisions
type BiddingEvaluator struct {
	threshold int // Minimum strength to bid (0-100)
}

// NewBiddingEvaluator creates a new bidding evaluator
func NewBiddingEvaluator(threshold int) *BiddingEvaluator {
	return &BiddingEvaluator{threshold: threshold}
}

// EvaluateRound1 evaluates whether to order up in round 1
func (e *BiddingEvaluator) EvaluateRound1(hand []engine.Card, turnedCard engine.Card, position int, isDealer bool) (bool, bool) {
	trump := turnedCard.Suit
	strength := e.evaluateHandStrength(hand, trump)

	// Adjust for position
	// Dealer gets a boost since they pick up the card
	if isDealer {
		strength += 10
	}

	// Player to dealer's left needs stronger hand
	if position == 1 {
		strength -= 5
	}

	// Partner of dealer can order up with slightly weaker hand
	if position == 2 {
		strength += 5
	}

	shouldBid := strength >= e.threshold
	shouldGoAlone := strength >= 85 // Very strong hand for alone

	return shouldBid, shouldGoAlone
}

// EvaluateRound2 evaluates whether to call trump in round 2
func (e *BiddingEvaluator) EvaluateRound2(hand []engine.Card, excludeSuit engine.Suit, isDealer bool, stickTheDealer bool) (bool, engine.Suit, bool) {
	bestSuit := engine.NoSuit
	bestStrength := 0

	// Evaluate each possible trump suit
	for _, suit := range []engine.Suit{engine.Clubs, engine.Diamonds, engine.Hearts, engine.Spades} {
		if suit == excludeSuit {
			continue
		}

		strength := e.evaluateHandStrength(hand, suit)
		if strength > bestStrength {
			bestStrength = strength
			bestSuit = suit
		}
	}

	// Dealer must call if stick the dealer is on
	if isDealer && stickTheDealer && bestSuit != engine.NoSuit {
		shouldGoAlone := bestStrength >= 85
		return true, bestSuit, shouldGoAlone
	}

	// Otherwise, need threshold strength
	if bestStrength >= e.threshold {
		shouldGoAlone := bestStrength >= 85
		return true, bestSuit, shouldGoAlone
	}

	return false, engine.NoSuit, false
}

// evaluateHandStrength calculates the bidding strength of a hand (0-100)
func (e *BiddingEvaluator) evaluateHandStrength(hand []engine.Card, trump engine.Suit) int {
	strength := 0

	trumpCount := 0
	hasRightBower := false
	hasLeftBower := false
	offAces := 0

	for _, card := range hand {
		if card.IsTrump(trump) {
			trumpCount++
			if card.IsRightBower(trump) {
				hasRightBower = true
			}
			if card.IsLeftBower(trump) {
				hasLeftBower = true
			}
		} else if card.Rank == engine.Ace {
			offAces++
		}
	}

	// Base strength from trump count
	// 3+ trumps is generally good
	switch trumpCount {
	case 0:
		strength += 0
	case 1:
		strength += 15
	case 2:
		strength += 35
	case 3:
		strength += 55
	case 4:
		strength += 75
	case 5:
		strength += 95
	}

	// Bower bonuses
	if hasRightBower {
		strength += 15
	}
	if hasLeftBower {
		strength += 10
	}
	if hasRightBower && hasLeftBower {
		strength += 5 // Extra bonus for both
	}

	// Off-suit aces are valuable
	strength += offAces * 8

	// Cap at 100
	if strength > 100 {
		strength = 100
	}

	return strength
}

// SelectDiscard chooses the best card to discard when dealer picks up
func (e *BiddingEvaluator) SelectDiscard(hand []engine.Card, turnedCard engine.Card) engine.Card {
	if len(hand) == 0 {
		return engine.Card{}
	}

	trump := turnedCard.Suit

	// Find the weakest card to discard
	worstCard := hand[0] // Initialize with first card
	worstValue := 1000

	for _, card := range hand {
		// Never discard the turned card itself
		if card.Suit == turnedCard.Suit && card.Rank == turnedCard.Rank {
			continue
		}

		value := e.cardDiscardValue(card, trump)
		if value < worstValue {
			worstValue = value
			worstCard = card
		}
	}

	return worstCard
}

// cardDiscardValue rates how valuable a card is to keep (higher = keep)
func (e *BiddingEvaluator) cardDiscardValue(card engine.Card, trump engine.Suit) int {
	// Trump cards are valuable
	if card.IsTrump(trump) {
		return 100 + card.TrumpValue(trump)
	}

	// Aces are valuable
	if card.Rank == engine.Ace {
		return 80
	}

	// Other cards by rank
	return card.OffSuitValue()
}
