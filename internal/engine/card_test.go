package engine

import "testing"

func TestSuitSameColor(t *testing.T) {
	tests := []struct {
		a, b     Suit
		expected bool
	}{
		{Hearts, Diamonds, true},
		{Diamonds, Hearts, true},
		{Spades, Clubs, true},
		{Clubs, Spades, true},
		{Hearts, Spades, false},
		{Hearts, Clubs, false},
		{Diamonds, Spades, false},
		{Diamonds, Clubs, false},
		{NoSuit, Hearts, false},
		{Hearts, NoSuit, false},
	}

	for _, tt := range tests {
		result := tt.a.SameColor(tt.b)
		if result != tt.expected {
			t.Errorf("%s.SameColor(%s) = %v, want %v", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestSuitString(t *testing.T) {
	tests := []struct {
		suit     Suit
		expected string
	}{
		{Clubs, "Clubs"},
		{Diamonds, "Diamonds"},
		{Hearts, "Hearts"},
		{Spades, "Spades"},
		{NoSuit, "NoSuit"},
	}

	for _, tt := range tests {
		if got := tt.suit.String(); got != tt.expected {
			t.Errorf("%d.String() = %s, want %s", tt.suit, got, tt.expected)
		}
	}
}

func TestSuitSymbol(t *testing.T) {
	tests := []struct {
		suit     Suit
		expected string
	}{
		{Clubs, "♣"},
		{Diamonds, "♦"},
		{Hearts, "♥"},
		{Spades, "♠"},
	}

	for _, tt := range tests {
		if got := tt.suit.Symbol(); got != tt.expected {
			t.Errorf("%s.Symbol() = %s, want %s", tt.suit, got, tt.expected)
		}
	}
}

func TestRankString(t *testing.T) {
	tests := []struct {
		rank     Rank
		expected string
	}{
		{Nine, "9"},
		{Ten, "10"},
		{Jack, "J"},
		{Queen, "Q"},
		{King, "K"},
		{Ace, "A"},
		{Joker, "Joker"},
	}

	for _, tt := range tests {
		if got := tt.rank.String(); got != tt.expected {
			t.Errorf("%d.String() = %s, want %s", tt.rank, got, tt.expected)
		}
	}
}

func TestCardString(t *testing.T) {
	tests := []struct {
		card     Card
		expected string
	}{
		{Card{Hearts, Ace}, "A♥"},
		{Card{Spades, Jack}, "J♠"},
		{Card{Diamonds, Nine}, "9♦"},
		{Card{Clubs, King}, "K♣"},
		{Card{NoSuit, Joker}, "Joker"},
	}

	for _, tt := range tests {
		if got := tt.card.String(); got != tt.expected {
			t.Errorf("Card{%s, %s}.String() = %s, want %s", tt.card.Suit, tt.card.Rank, got, tt.expected)
		}
	}
}

func TestIsRightBower(t *testing.T) {
	tests := []struct {
		card     Card
		trump    Suit
		expected bool
	}{
		{Card{Hearts, Jack}, Hearts, true},
		{Card{Spades, Jack}, Spades, true},
		{Card{Diamonds, Jack}, Hearts, false}, // Left bower, not right
		{Card{Hearts, Ace}, Hearts, false},
		{Card{Hearts, Jack}, Spades, false},
	}

	for _, tt := range tests {
		if got := tt.card.IsRightBower(tt.trump); got != tt.expected {
			t.Errorf("%s.IsRightBower(%s) = %v, want %v", tt.card, tt.trump, got, tt.expected)
		}
	}
}

func TestIsLeftBower(t *testing.T) {
	tests := []struct {
		card     Card
		trump    Suit
		expected bool
	}{
		{Card{Diamonds, Jack}, Hearts, true},   // Red jack when hearts trump
		{Card{Hearts, Jack}, Diamonds, true},   // Red jack when diamonds trump
		{Card{Clubs, Jack}, Spades, true},      // Black jack when spades trump
		{Card{Spades, Jack}, Clubs, true},      // Black jack when clubs trump
		{Card{Hearts, Jack}, Hearts, false},    // Right bower, not left
		{Card{Spades, Jack}, Hearts, false},    // Wrong color
		{Card{Diamonds, Ace}, Hearts, false},   // Not a jack
	}

	for _, tt := range tests {
		if got := tt.card.IsLeftBower(tt.trump); got != tt.expected {
			t.Errorf("%s.IsLeftBower(%s) = %v, want %v", tt.card, tt.trump, got, tt.expected)
		}
	}
}

func TestEffectiveSuit(t *testing.T) {
	tests := []struct {
		card     Card
		trump    Suit
		expected Suit
	}{
		{Card{Hearts, Ace}, Hearts, Hearts},
		{Card{Hearts, Ace}, Spades, Hearts},
		{Card{Diamonds, Jack}, Hearts, Hearts},  // Left bower belongs to trump
		{Card{Clubs, Jack}, Spades, Spades},     // Left bower belongs to trump
		{Card{Hearts, Jack}, Hearts, Hearts},    // Right bower
		{Card{Spades, Nine}, Clubs, Spades},
	}

	for _, tt := range tests {
		if got := tt.card.EffectiveSuit(tt.trump); got != tt.expected {
			t.Errorf("%s.EffectiveSuit(%s) = %s, want %s", tt.card, tt.trump, got, tt.expected)
		}
	}
}

func TestIsTrump(t *testing.T) {
	tests := []struct {
		card     Card
		trump    Suit
		expected bool
	}{
		{Card{Hearts, Ace}, Hearts, true},
		{Card{Hearts, Nine}, Hearts, true},
		{Card{Diamonds, Jack}, Hearts, true},  // Left bower is trump
		{Card{Spades, Ace}, Hearts, false},
		{Card{Clubs, Jack}, Hearts, false},    // Wrong color jack
		{Card{NoSuit, Joker}, Hearts, true},   // Joker is always trump
	}

	for _, tt := range tests {
		if got := tt.card.IsTrump(tt.trump); got != tt.expected {
			t.Errorf("%s.IsTrump(%s) = %v, want %v", tt.card, tt.trump, got, tt.expected)
		}
	}
}

func TestTrumpValue(t *testing.T) {
	trump := Hearts

	// Test that trump cards have correct relative ordering
	rightBower := Card{Hearts, Jack}
	leftBower := Card{Diamonds, Jack}
	aceOfTrump := Card{Hearts, Ace}
	kingOfTrump := Card{Hearts, King}
	nineOfTrump := Card{Hearts, Nine}
	offSuitAce := Card{Spades, Ace}

	if rightBower.TrumpValue(trump) <= leftBower.TrumpValue(trump) {
		t.Error("Right bower should beat left bower")
	}
	if leftBower.TrumpValue(trump) <= aceOfTrump.TrumpValue(trump) {
		t.Error("Left bower should beat ace of trump")
	}
	if aceOfTrump.TrumpValue(trump) <= kingOfTrump.TrumpValue(trump) {
		t.Error("Ace of trump should beat king of trump")
	}
	if kingOfTrump.TrumpValue(trump) <= nineOfTrump.TrumpValue(trump) {
		t.Error("King of trump should beat nine of trump")
	}
	if offSuitAce.TrumpValue(trump) != 0 {
		t.Error("Off-suit card should have trump value of 0")
	}
}

func TestTeamAndPartner(t *testing.T) {
	// Players 0 and 2 are team 0
	// Players 1 and 3 are team 1
	if Team(0) != 0 || Team(2) != 0 {
		t.Error("Players 0 and 2 should be team 0")
	}
	if Team(1) != 1 || Team(3) != 1 {
		t.Error("Players 1 and 3 should be team 1")
	}

	// Partner relationships
	if Partner(0) != 2 || Partner(2) != 0 {
		t.Error("Players 0 and 2 should be partners")
	}
	if Partner(1) != 3 || Partner(3) != 1 {
		t.Error("Players 1 and 3 should be partners")
	}

	// IsPartner
	if !IsPartner(0, 2) || !IsPartner(1, 3) {
		t.Error("IsPartner should return true for partners")
	}
	if IsPartner(0, 1) || IsPartner(0, 3) {
		t.Error("IsPartner should return false for opponents")
	}
}

func TestNextPlayer(t *testing.T) {
	tests := []struct {
		current, numPlayers, expected int
	}{
		{0, 4, 1},
		{1, 4, 2},
		{2, 4, 3},
		{3, 4, 0}, // Wraps around
		{0, 3, 1},
		{2, 3, 0}, // Wraps around with 3 players
	}

	for _, tt := range tests {
		if got := NextPlayer(tt.current, tt.numPlayers); got != tt.expected {
			t.Errorf("NextPlayer(%d, %d) = %d, want %d", tt.current, tt.numPlayers, got, tt.expected)
		}
	}
}
