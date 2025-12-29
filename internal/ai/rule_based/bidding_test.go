package rule_based

import (
	"testing"

	"github.com/bran/euchre/internal/engine"
)

func TestBiddingEvaluator_EvaluateRound1_StrongHand(t *testing.T) {
	evaluator := NewBiddingEvaluator(55)

	// Strong hand with right bower, left bower, and ace of trump
	hand := []engine.Card{
		{Suit: engine.Hearts, Rank: engine.Jack},    // Right bower
		{Suit: engine.Diamonds, Rank: engine.Jack},  // Left bower
		{Suit: engine.Hearts, Rank: engine.Ace},
		{Suit: engine.Spades, Rank: engine.Ace},
		{Suit: engine.Clubs, Rank: engine.King},
	}
	turnedCard := engine.Card{Suit: engine.Hearts, Rank: engine.Nine}

	shouldBid, goAlone := evaluator.EvaluateRound1(hand, turnedCard, 1, false)

	if !shouldBid {
		t.Error("Should bid with strong trump hand")
	}
	if !goAlone {
		t.Error("Should go alone with both bowers and ace of trump")
	}
}

func TestBiddingEvaluator_EvaluateRound1_WeakHand(t *testing.T) {
	evaluator := NewBiddingEvaluator(55)

	// Weak hand with no trumps
	hand := []engine.Card{
		{Suit: engine.Spades, Rank: engine.Nine},
		{Suit: engine.Spades, Rank: engine.Ten},
		{Suit: engine.Clubs, Rank: engine.Nine},
		{Suit: engine.Clubs, Rank: engine.Ten},
		{Suit: engine.Diamonds, Rank: engine.Nine},
	}
	turnedCard := engine.Card{Suit: engine.Hearts, Rank: engine.Ace}

	shouldBid, goAlone := evaluator.EvaluateRound1(hand, turnedCard, 1, false)

	if shouldBid {
		t.Error("Should not bid with no trumps")
	}
	if goAlone {
		t.Error("Should not go alone with weak hand")
	}
}

func TestBiddingEvaluator_EvaluateRound1_DealerBonus(t *testing.T) {
	evaluator := NewBiddingEvaluator(55)

	// Medium hand - might bid as dealer but not otherwise
	hand := []engine.Card{
		{Suit: engine.Hearts, Rank: engine.King},
		{Suit: engine.Hearts, Rank: engine.Queen},
		{Suit: engine.Spades, Rank: engine.Ace},
		{Suit: engine.Clubs, Rank: engine.Ace},
		{Suit: engine.Diamonds, Rank: engine.King},
	}
	turnedCard := engine.Card{Suit: engine.Hearts, Rank: engine.Ten}

	// As non-dealer
	shouldBidNonDealer, _ := evaluator.EvaluateRound1(hand, turnedCard, 1, false)

	// As dealer (gets bonus)
	shouldBidDealer, _ := evaluator.EvaluateRound1(hand, turnedCard, 0, true)

	// Dealer should be more likely to bid
	if shouldBidNonDealer && !shouldBidDealer {
		t.Error("Dealer should be at least as likely to bid as non-dealer")
	}
}

func TestBiddingEvaluator_EvaluateRound2(t *testing.T) {
	evaluator := NewBiddingEvaluator(55)

	// Hand with good spades
	hand := []engine.Card{
		{Suit: engine.Spades, Rank: engine.Jack},  // Would be right bower
		{Suit: engine.Clubs, Rank: engine.Jack},   // Would be left bower
		{Suit: engine.Spades, Rank: engine.Ace},
		{Suit: engine.Hearts, Rank: engine.Ace},
		{Suit: engine.Diamonds, Rank: engine.King},
	}

	// Hearts was turned down
	shouldBid, suit, goAlone := evaluator.EvaluateRound2(hand, engine.Hearts, false, false)

	if !shouldBid {
		t.Error("Should bid with strong spades")
	}
	if suit != engine.Spades {
		t.Errorf("Should call spades, got %s", suit)
	}
	if !goAlone {
		t.Error("Should go alone with both bowers")
	}
}

func TestBiddingEvaluator_EvaluateRound2_StickTheDealer(t *testing.T) {
	evaluator := NewBiddingEvaluator(55)

	// Weak hand
	hand := []engine.Card{
		{Suit: engine.Spades, Rank: engine.Nine},
		{Suit: engine.Clubs, Rank: engine.Nine},
		{Suit: engine.Hearts, Rank: engine.Nine},
		{Suit: engine.Diamonds, Rank: engine.Nine},
		{Suit: engine.Spades, Rank: engine.Ten},
	}

	// Without stick the dealer - should pass
	shouldBid1, _, _ := evaluator.EvaluateRound2(hand, engine.Hearts, true, false)
	if shouldBid1 {
		t.Error("Should pass with weak hand when not stuck")
	}

	// With stick the dealer - must bid
	shouldBid2, suit, _ := evaluator.EvaluateRound2(hand, engine.Hearts, true, true)
	if !shouldBid2 {
		t.Error("Dealer must bid when stuck")
	}
	if suit == engine.Hearts {
		t.Error("Should not call the turned-down suit")
	}
}

func TestBiddingEvaluator_SelectDiscard(t *testing.T) {
	evaluator := NewBiddingEvaluator(55)

	hand := []engine.Card{
		{Suit: engine.Hearts, Rank: engine.Jack},   // Right bower - keep!
		{Suit: engine.Hearts, Rank: engine.Ace},    // Strong trump - keep!
		{Suit: engine.Spades, Rank: engine.Ace},    // Off-ace - keep!
		{Suit: engine.Clubs, Rank: engine.Nine},    // Weak - discard this
		{Suit: engine.Hearts, Rank: engine.Nine},   // Picked up card
		{Suit: engine.Diamonds, Rank: engine.Ten},  // Weak
	}
	turnedCard := engine.Card{Suit: engine.Hearts, Rank: engine.Nine}

	discard := evaluator.SelectDiscard(hand, turnedCard)

	// Should discard a weak off-suit card, not a trump or ace
	if discard.IsTrump(engine.Hearts) {
		t.Errorf("Should not discard trump, got %s", discard)
	}
	if discard.Rank == engine.Ace {
		t.Errorf("Should not discard ace, got %s", discard)
	}
}

func TestBiddingEvaluator_SelectDiscard_NotTurnedCard(t *testing.T) {
	evaluator := NewBiddingEvaluator(55)

	turnedCard := engine.Card{Suit: engine.Hearts, Rank: engine.Nine}
	hand := []engine.Card{
		turnedCard, // The turned card itself
		{Suit: engine.Hearts, Rank: engine.Ace},
		{Suit: engine.Spades, Rank: engine.Ace},
		{Suit: engine.Clubs, Rank: engine.Nine},
		{Suit: engine.Diamonds, Rank: engine.Ten},
		{Suit: engine.Diamonds, Rank: engine.Nine},
	}

	discard := evaluator.SelectDiscard(hand, turnedCard)

	// Should not discard the turned card (it was just picked up)
	if discard.Suit == turnedCard.Suit && discard.Rank == turnedCard.Rank {
		t.Error("Should not discard the turned card that was just picked up")
	}
}
