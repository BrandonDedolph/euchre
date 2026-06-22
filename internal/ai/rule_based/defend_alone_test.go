package rule_based

import (
	"testing"

	"github.com/bran/euchre/internal/engine"
)

func TestShouldDefendAlone_WeakHandDeclines(t *testing.T) {
	trump := engine.Hearts

	// No trump, no off-aces - clearly should decline.
	hand := []engine.Card{
		{Suit: engine.Spades, Rank: engine.Nine},
		{Suit: engine.Spades, Rank: engine.Ten},
		{Suit: engine.Clubs, Rank: engine.Nine},
		{Suit: engine.Clubs, Rank: engine.King},
		{Suit: engine.Diamonds, Rank: engine.Ten},
	}

	if shouldDefendAlone(hand, trump) {
		t.Error("should decline to defend alone with a weak hand")
	}
}

func TestShouldDefendAlone_SingleBareBowerDeclines(t *testing.T) {
	trump := engine.Hearts

	// Right bower but no other trump and no off-ace: too thin to take 3 tricks alone.
	hand := []engine.Card{
		{Suit: engine.Hearts, Rank: engine.Jack}, // Right bower
		{Suit: engine.Spades, Rank: engine.Nine},
		{Suit: engine.Spades, Rank: engine.Ten},
		{Suit: engine.Clubs, Rank: engine.Nine},
		{Suit: engine.Clubs, Rank: engine.King},
	}

	if shouldDefendAlone(hand, trump) {
		t.Error("should decline with only a bare right bower and no support")
	}
}

func TestShouldDefendAlone_StrongHandAccepts(t *testing.T) {
	trump := engine.Hearts

	// Right bower, left bower, ace of trump, plus an off-ace - very strong.
	hand := []engine.Card{
		{Suit: engine.Hearts, Rank: engine.Jack},   // Right bower
		{Suit: engine.Diamonds, Rank: engine.Jack}, // Left bower
		{Suit: engine.Hearts, Rank: engine.Ace},    // Ace of trump
		{Suit: engine.Spades, Rank: engine.Ace},    // Off-suit ace
		{Suit: engine.Clubs, Rank: engine.Nine},
	}

	if !shouldDefendAlone(hand, trump) {
		t.Error("should accept defending alone with a clearly strong hand")
	}
}

func TestShouldDefendAlone_RightBowerWithSupportAccepts(t *testing.T) {
	trump := engine.Hearts

	// Right bower + exactly one other trump (only 2 trump total, no off-ace):
	// exercises the bower-with-support branch, NOT the 3+ trump branch.
	hand := []engine.Card{
		{Suit: engine.Hearts, Rank: engine.Jack}, // Right bower
		{Suit: engine.Hearts, Rank: engine.King}, // Second trump
		{Suit: engine.Spades, Rank: engine.Nine},
		{Suit: engine.Spades, Rank: engine.Ten},
		{Suit: engine.Clubs, Rank: engine.Nine},
	}

	if !shouldDefendAlone(hand, trump) {
		t.Error("should accept with the right bower plus a second trump")
	}
}

func TestShouldDefendAlone_ThreeTrumpAccepts(t *testing.T) {
	trump := engine.Spades

	// Three trump (no bowers) is enough trump length to try.
	hand := []engine.Card{
		{Suit: engine.Spades, Rank: engine.Ace},
		{Suit: engine.Spades, Rank: engine.King},
		{Suit: engine.Spades, Rank: engine.Queen},
		{Suit: engine.Hearts, Rank: engine.Ten},
		{Suit: engine.Diamonds, Rank: engine.Nine},
	}

	if !shouldDefendAlone(hand, trump) {
		t.Error("should accept defending alone with three trump cards")
	}
}

func TestShouldDefendAlone_NoTrumpSuitDeclines(t *testing.T) {
	hand := []engine.Card{
		{Suit: engine.Hearts, Rank: engine.Jack},
		{Suit: engine.Hearts, Rank: engine.Ace},
		{Suit: engine.Spades, Rank: engine.Ace},
	}
	if shouldDefendAlone(hand, engine.NoSuit) {
		t.Error("should decline when trump is not set")
	}
}
