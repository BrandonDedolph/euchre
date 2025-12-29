package rule_based

import (
	"testing"

	"github.com/bran/euchre/internal/engine"
)

func TestPlayStrategy_SelectPlay_Leading(t *testing.T) {
	strategy := NewPlayStrategy()
	trump := engine.Hearts

	// Hand with multiple trumps - should lead trump
	hand := []engine.Card{
		{Suit: engine.Hearts, Rank: engine.Jack},  // Right bower
		{Suit: engine.Hearts, Rank: engine.Ace},
		{Suit: engine.Spades, Rank: engine.Ace},
		{Suit: engine.Clubs, Rank: engine.King},
	}

	trick := engine.NewTrick(trump)
	play := strategy.SelectPlay(hand, trick, 0, trump)

	// With multiple trumps, should lead highest trump
	if !play.IsRightBower(trump) {
		t.Errorf("Should lead right bower, got %s", play)
	}
}

func TestPlayStrategy_SelectPlay_LeadingAce(t *testing.T) {
	strategy := NewPlayStrategy()
	trump := engine.Hearts

	// Hand with only one trump but off-suit ace
	hand := []engine.Card{
		{Suit: engine.Hearts, Rank: engine.Nine},  // Only trump
		{Suit: engine.Spades, Rank: engine.Ace},   // Off-suit ace
		{Suit: engine.Clubs, Rank: engine.King},
		{Suit: engine.Diamonds, Rank: engine.Queen},
	}

	trick := engine.NewTrick(trump)
	play := strategy.SelectPlay(hand, trick, 0, trump)

	// With only one trump, should lead off-suit ace
	if play.Suit != engine.Spades || play.Rank != engine.Ace {
		t.Errorf("Should lead off-suit ace, got %s", play)
	}
}

func TestPlayStrategy_SelectPlay_Following_PartnerWinning(t *testing.T) {
	strategy := NewPlayStrategy()
	trump := engine.Hearts

	// Partner (player 2) is winning
	trick := engine.NewTrick(trump)
	trick.Play(2, engine.Card{Suit: engine.Spades, Rank: engine.Ace})
	trick.Play(3, engine.Card{Suit: engine.Spades, Rank: engine.King})

	// Player 0's hand - has spades
	hand := []engine.Card{
		{Suit: engine.Spades, Rank: engine.Queen},
		{Suit: engine.Spades, Rank: engine.Nine},
		{Suit: engine.Hearts, Rank: engine.Ace},
	}

	play := strategy.SelectPlay(hand, trick, 0, trump)

	// Partner is winning, should play lowest spade
	if play.Suit != engine.Spades || play.Rank != engine.Nine {
		t.Errorf("Should play lowest spade when partner winning, got %s", play)
	}
}

func TestPlayStrategy_SelectPlay_Following_OpponentWinning(t *testing.T) {
	strategy := NewPlayStrategy()
	trump := engine.Hearts

	// Opponent (player 1) is winning with King
	trick := engine.NewTrick(trump)
	trick.Play(1, engine.Card{Suit: engine.Spades, Rank: engine.King})

	// Player 0's hand - can beat it
	hand := []engine.Card{
		{Suit: engine.Spades, Rank: engine.Ace},   // Can beat
		{Suit: engine.Spades, Rank: engine.Queen}, // Can't beat
		{Suit: engine.Hearts, Rank: engine.Nine},
	}

	play := strategy.SelectPlay(hand, trick, 0, trump)

	// Should beat opponent with ace
	if play.Suit != engine.Spades || play.Rank != engine.Ace {
		t.Errorf("Should beat opponent with ace, got %s", play)
	}
}

func TestPlayStrategy_SelectPlay_Trumping(t *testing.T) {
	strategy := NewPlayStrategy()
	trump := engine.Hearts

	// Opponent leads spades, we're void
	trick := engine.NewTrick(trump)
	trick.Play(1, engine.Card{Suit: engine.Spades, Rank: engine.Ace})

	// Player 0's hand - void in spades, has trumps
	hand := []engine.Card{
		{Suit: engine.Hearts, Rank: engine.Nine},  // Low trump
		{Suit: engine.Hearts, Rank: engine.Ace},   // High trump
		{Suit: engine.Clubs, Rank: engine.King},
	}

	play := strategy.SelectPlay(hand, trick, 0, trump)

	// Should trump with lowest trump that wins
	if play.Suit != engine.Hearts || play.Rank != engine.Nine {
		t.Errorf("Should trump with lowest trump, got %s", play)
	}
}

func TestPlayStrategy_SelectPlay_DiscardWhenPartnerWinning(t *testing.T) {
	strategy := NewPlayStrategy()
	trump := engine.Hearts

	// Partner leads spades, we're void but have off-suit to discard
	trick := engine.NewTrick(trump)
	trick.Play(2, engine.Card{Suit: engine.Spades, Rank: engine.Ace}) // Partner

	// Player 0's hand - void in spades, has off-suit cards (not just trumps)
	hand := []engine.Card{
		{Suit: engine.Clubs, Rank: engine.King},
		{Suit: engine.Diamonds, Rank: engine.Queen},
		{Suit: engine.Diamonds, Rank: engine.Nine},
	}

	play := strategy.SelectPlay(hand, trick, 0, trump)

	// Partner is winning - should discard lowest off-suit
	if play.IsTrump(trump) {
		t.Errorf("Should discard, not trump when partner is winning, got %s", play)
	}
	if play.Rank != engine.Nine {
		t.Errorf("Should discard lowest card, got %s", play)
	}
}

func TestPlayStrategy_SelectPlay_OnlyOneChoice(t *testing.T) {
	strategy := NewPlayStrategy()
	trump := engine.Hearts

	trick := engine.NewTrick(trump)
	trick.Play(1, engine.Card{Suit: engine.Spades, Rank: engine.Ace})

	// Only one card in hand
	hand := []engine.Card{
		{Suit: engine.Spades, Rank: engine.Nine},
	}

	play := strategy.SelectPlay(hand, trick, 0, trump)

	if play.Suit != engine.Spades || play.Rank != engine.Nine {
		t.Errorf("Should play only card in hand, got %s", play)
	}
}

func TestPlayStrategy_SelectPlay_MustFollowSuit(t *testing.T) {
	strategy := NewPlayStrategy()
	trump := engine.Hearts

	// Spades led
	trick := engine.NewTrick(trump)
	trick.Play(1, engine.Card{Suit: engine.Spades, Rank: engine.King})

	// Hand with one spade and better off-suit cards
	hand := []engine.Card{
		{Suit: engine.Spades, Rank: engine.Nine},  // Must play this
		{Suit: engine.Hearts, Rank: engine.Jack},  // Right bower - can't play
		{Suit: engine.Clubs, Rank: engine.Ace},    // Can't play
	}

	play := strategy.SelectPlay(hand, trick, 0, trump)

	// Must follow suit even though we have better cards
	if play.Suit != engine.Spades {
		t.Errorf("Must follow suit with spade, got %s", play)
	}
}

func TestPlayStrategy_Discard(t *testing.T) {
	strategy := NewPlayStrategy()
	trump := engine.Hearts

	// Void in lead suit, partner winning, no option to trump effectively
	trick := engine.NewTrick(trump)
	trick.Play(2, engine.Card{Suit: engine.Spades, Rank: engine.Ace})
	trick.Play(3, engine.Card{Suit: engine.Spades, Rank: engine.King})

	// Only off-suit cards
	hand := []engine.Card{
		{Suit: engine.Clubs, Rank: engine.Ace},   // Keep high cards
		{Suit: engine.Clubs, Rank: engine.Nine},  // Discard low
		{Suit: engine.Diamonds, Rank: engine.Ten},
	}

	play := strategy.SelectPlay(hand, trick, 0, trump)

	// Should discard lowest card
	if play.Rank != engine.Nine && play.Rank != engine.Ten {
		t.Errorf("Should discard low card, got %s", play)
	}
}
