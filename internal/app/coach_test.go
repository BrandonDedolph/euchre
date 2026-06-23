package app

import (
	"testing"

	"github.com/bran/euchre/internal/engine"
)

func card(s engine.Suit, r engine.Rank) engine.Card { return engine.Card{Suit: s, Rank: r} }

func TestShapeOf(t *testing.T) {
	trump := engine.Spades
	// Right bower (J♠), left bower (J♣), one more trump (A♠), and two off aces.
	hand := []engine.Card{
		card(engine.Spades, engine.Jack),  // right bower
		card(engine.Clubs, engine.Jack),   // left bower (same color as spades)
		card(engine.Spades, engine.Ace),   // trump
		card(engine.Hearts, engine.Ace),   // off ace
		card(engine.Diamonds, engine.Ace), // off ace
	}
	s := shapeOf(hand, trump)
	if s.trump != 3 {
		t.Errorf("trump count = %d, want 3", s.trump)
	}
	if !s.rightBower || !s.leftBower {
		t.Errorf("bowers: right=%v left=%v, want both true", s.rightBower, s.leftBower)
	}
	if s.offAces != 2 {
		t.Errorf("offAces = %d, want 2", s.offAces)
	}
	if got := s.bowerPhrase(); got != "both bowers" {
		t.Errorf("bowerPhrase = %q, want %q", got, "both bowers")
	}
}

func TestShapeLeftBowerCountedAsTrump(t *testing.T) {
	// The left bower's printed suit is NOT trump, but it must count as trump.
	trump := engine.Hearts
	hand := []engine.Card{
		card(engine.Diamonds, engine.Jack), // left bower of hearts (same color)
		card(engine.Spades, engine.Ace),    // off ace, NOT trump
	}
	s := shapeOf(hand, trump)
	if s.trump != 1 || !s.leftBower || s.rightBower {
		t.Errorf("got trump=%d left=%v right=%v, want trump=1 left=true right=false", s.trump, s.leftBower, s.rightBower)
	}
	if s.offAces != 1 {
		t.Errorf("offAces = %d, want 1 (the off-suit A♠)", s.offAces)
	}
	if got := s.bowerPhrase(); got != "the left bower" {
		t.Errorf("bowerPhrase = %q, want %q", got, "the left bower")
	}
}

func TestBowerPhraseNone(t *testing.T) {
	s := shapeOf([]engine.Card{card(engine.Clubs, engine.Nine)}, engine.Spades)
	if got := s.bowerPhrase(); got != "" {
		t.Errorf("bowerPhrase = %q, want empty", got)
	}
}
