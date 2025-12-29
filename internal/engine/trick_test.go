package engine

import "testing"

func TestNewTrick(t *testing.T) {
	trick := NewTrick(Hearts)

	if trick.Size() != 0 {
		t.Errorf("New trick should be empty, got %d cards", trick.Size())
	}
	if trick.Trump() != Hearts {
		t.Errorf("Trump should be Hearts, got %s", trick.Trump())
	}
}

func TestTrickPlay(t *testing.T) {
	trick := NewTrick(Hearts)

	trick.Play(0, Card{Spades, Ace})
	if trick.Size() != 1 {
		t.Errorf("Trick should have 1 card, got %d", trick.Size())
	}
	if trick.LeadSuit() != Spades {
		t.Errorf("Lead suit should be Spades, got %s", trick.LeadSuit())
	}
	if trick.Leader() != 0 {
		t.Errorf("Leader should be player 0, got %d", trick.Leader())
	}

	trick.Play(1, Card{Spades, King})
	trick.Play(2, Card{Spades, Queen})
	trick.Play(3, Card{Spades, Nine})

	if trick.Size() != 4 {
		t.Errorf("Trick should have 4 cards, got %d", trick.Size())
	}
}

func TestTrickWinner_HighestOfLeadSuit(t *testing.T) {
	trick := NewTrick(Hearts)

	// All follow suit, highest wins
	trick.Play(0, Card{Spades, Nine})
	trick.Play(1, Card{Spades, Ace})  // Highest
	trick.Play(2, Card{Spades, King})
	trick.Play(3, Card{Spades, Ten})

	winner := trick.Winner()
	if winner != 1 {
		t.Errorf("Player 1 (A♠) should win, got player %d", winner)
	}
}

func TestTrickWinner_TrumpBeatsOffSuit(t *testing.T) {
	trick := NewTrick(Hearts)

	trick.Play(0, Card{Spades, Ace})  // Lead with A♠
	trick.Play(1, Card{Spades, King})
	trick.Play(2, Card{Hearts, Nine}) // Trump with lowly 9♥
	trick.Play(3, Card{Spades, Queen})

	winner := trick.Winner()
	if winner != 2 {
		t.Errorf("Player 2 (9♥ trump) should win, got player %d", winner)
	}
}

func TestTrickWinner_HighestTrumpWins(t *testing.T) {
	trick := NewTrick(Hearts)

	trick.Play(0, Card{Spades, Ace})
	trick.Play(1, Card{Hearts, Nine})  // Low trump
	trick.Play(2, Card{Hearts, Ace})   // High trump
	trick.Play(3, Card{Hearts, Ten})

	winner := trick.Winner()
	if winner != 2 {
		t.Errorf("Player 2 (A♥) should win, got player %d", winner)
	}
}

func TestTrickWinner_RightBowerBeatsAll(t *testing.T) {
	trick := NewTrick(Hearts)

	trick.Play(0, Card{Hearts, Ace})
	trick.Play(1, Card{Hearts, Jack})    // Right bower
	trick.Play(2, Card{Diamonds, Jack})  // Left bower
	trick.Play(3, Card{Hearts, King})

	winner := trick.Winner()
	if winner != 1 {
		t.Errorf("Player 1 (Right Bower J♥) should win, got player %d", winner)
	}
}

func TestTrickWinner_LeftBowerBeatsAce(t *testing.T) {
	trick := NewTrick(Hearts)

	trick.Play(0, Card{Hearts, Ace})
	trick.Play(1, Card{Hearts, King})
	trick.Play(2, Card{Diamonds, Jack}) // Left bower
	trick.Play(3, Card{Hearts, Queen})

	winner := trick.Winner()
	if winner != 2 {
		t.Errorf("Player 2 (Left Bower J♦) should win, got player %d", winner)
	}
}

func TestTrickWinner_OffSuitCantWin(t *testing.T) {
	trick := NewTrick(Hearts)

	trick.Play(0, Card{Spades, Nine})   // Lead
	trick.Play(1, Card{Clubs, Ace})     // Can't follow, discards high
	trick.Play(2, Card{Spades, Ten})    // Follows
	trick.Play(3, Card{Diamonds, Ace})  // Can't follow, discards high

	winner := trick.Winner()
	if winner != 2 {
		t.Errorf("Player 2 (10♠) should win, got player %d", winner)
	}
}

func TestTrickWinner_LeftBowerAsLead(t *testing.T) {
	trick := NewTrick(Hearts)

	// Left bower led - others must follow with hearts (not diamonds!)
	trick.Play(0, Card{Diamonds, Jack}) // Left bower, effectively hearts
	trick.Play(1, Card{Hearts, Ace})
	trick.Play(2, Card{Hearts, King})
	trick.Play(3, Card{Hearts, Queen})

	// Left bower is second highest trump
	winner := trick.Winner()
	if winner != 0 {
		t.Errorf("Player 0 (Left Bower) should win, got player %d", winner)
	}

	// Verify lead suit is hearts (trump)
	if trick.LeadSuit() != Hearts {
		t.Errorf("Lead suit should be Hearts (left bower), got %s", trick.LeadSuit())
	}
}

func TestTrickIsComplete(t *testing.T) {
	trick := NewTrick(Hearts)

	if trick.IsComplete(4) {
		t.Error("Empty trick should not be complete")
	}

	trick.Play(0, Card{Spades, Ace})
	trick.Play(1, Card{Spades, King})
	trick.Play(2, Card{Spades, Queen})

	if trick.IsComplete(4) {
		t.Error("Trick with 3 cards should not be complete for 4 players")
	}

	trick.Play(3, Card{Spades, Nine})

	if !trick.IsComplete(4) {
		t.Error("Trick with 4 cards should be complete for 4 players")
	}

	// Test with 3 players
	trick2 := NewTrick(Hearts)
	trick2.Play(0, Card{Spades, Ace})
	trick2.Play(1, Card{Spades, King})
	trick2.Play(2, Card{Spades, Queen})

	if !trick2.IsComplete(3) {
		t.Error("Trick with 3 cards should be complete for 3 players")
	}
}

func TestTrickCanBeat(t *testing.T) {
	trick := NewTrick(Hearts)
	trick.Play(0, Card{Spades, King})

	// Higher card of same suit can beat
	if !trick.CanBeat(Card{Spades, Ace}) {
		t.Error("A♠ should beat K♠")
	}

	// Lower card of same suit cannot beat
	if trick.CanBeat(Card{Spades, Nine}) {
		t.Error("9♠ should not beat K♠")
	}

	// Trump can beat
	if !trick.CanBeat(Card{Hearts, Nine}) {
		t.Error("9♥ (trump) should beat K♠")
	}

	// Off-suit non-trump cannot beat
	if trick.CanBeat(Card{Clubs, Ace}) {
		t.Error("A♣ (off-suit) should not beat K♠")
	}
}

func TestTrickClear(t *testing.T) {
	trick := NewTrick(Hearts)
	trick.Play(0, Card{Spades, Ace})
	trick.Play(1, Card{Spades, King})

	trick.Clear()

	if trick.Size() != 0 {
		t.Errorf("Cleared trick should be empty, got %d cards", trick.Size())
	}
	if trick.LeadSuit() != NoSuit {
		t.Errorf("Cleared trick should have no lead suit, got %s", trick.LeadSuit())
	}
}

func TestValidatePlay(t *testing.T) {
	trump := Hearts
	trick := NewTrick(trump)
	trick.Play(0, Card{Spades, Ace}) // Spades led

	// Hand with spades - must follow
	handWithSpades := NewHandWith([]Card{
		{Spades, King},
		{Hearts, Ace},
		{Clubs, Queen},
	})

	// Playing spade should be valid
	err := ValidatePlay(handWithSpades, Card{Spades, King}, trick)
	if err != nil {
		t.Errorf("Playing K♠ should be valid: %v", err)
	}

	// Playing non-spade when holding spades should be invalid
	err = ValidatePlay(handWithSpades, Card{Hearts, Ace}, trick)
	if err != ErrMustFollowSuit {
		t.Errorf("Playing A♥ when holding spades should fail with ErrMustFollowSuit, got %v", err)
	}

	// Hand without spades - can play anything
	handWithoutSpades := NewHandWith([]Card{
		{Hearts, Ace},
		{Clubs, Queen},
		{Diamonds, Jack},
	})

	err = ValidatePlay(handWithoutSpades, Card{Hearts, Ace}, trick)
	if err != nil {
		t.Errorf("Playing A♥ when void in spades should be valid: %v", err)
	}

	// Playing card not in hand should be invalid
	err = ValidatePlay(handWithSpades, Card{Diamonds, Nine}, trick)
	if err != ErrCardNotInHand {
		t.Errorf("Playing card not in hand should fail with ErrCardNotInHand, got %v", err)
	}
}

func TestValidatePlay_LeftBower(t *testing.T) {
	trump := Hearts
	trick := NewTrick(trump)
	trick.Play(0, Card{Hearts, Ace}) // Hearts led

	// Hand with left bower only (no other hearts)
	hand := NewHandWith([]Card{
		{Diamonds, Jack}, // Left bower - is a heart!
		{Spades, Ace},
		{Clubs, King},
	})

	// Must play left bower when hearts led
	err := ValidatePlay(hand, Card{Spades, Ace}, trick)
	if err != ErrMustFollowSuit {
		t.Errorf("Must play left bower when hearts led and it's the only heart, got %v", err)
	}

	err = ValidatePlay(hand, Card{Diamonds, Jack}, trick)
	if err != nil {
		t.Errorf("Playing left bower should be valid: %v", err)
	}
}

func TestLegalPlays(t *testing.T) {
	trump := Hearts

	// Empty trick - all cards are legal
	emptyTrick := NewTrick(trump)
	hand := NewHandWith([]Card{
		{Spades, Ace},
		{Hearts, King},
		{Clubs, Queen},
	})

	legal := LegalPlays(hand, emptyTrick)
	if len(legal) != 3 {
		t.Errorf("All 3 cards should be legal when leading, got %d", len(legal))
	}

	// Spades led, have spades
	trick := NewTrick(trump)
	trick.Play(0, Card{Spades, King})

	handWithSpades := NewHandWith([]Card{
		{Spades, Ace},
		{Spades, Nine},
		{Hearts, King},
		{Clubs, Queen},
	})

	legal = LegalPlays(handWithSpades, trick)
	if len(legal) != 2 {
		t.Errorf("Only 2 spades should be legal, got %d", len(legal))
	}

	// Void in lead suit - all cards legal
	handVoid := NewHandWith([]Card{
		{Hearts, Ace},
		{Clubs, Queen},
		{Diamonds, Jack},
	})

	legal = LegalPlays(handVoid, trick)
	if len(legal) != 3 {
		t.Errorf("All 3 cards should be legal when void, got %d", len(legal))
	}
}

func TestTrickResult(t *testing.T) {
	trick := NewTrick(Hearts)

	trick.Play(0, Card{Spades, Ace})
	trick.Play(1, Card{Spades, King})
	trick.Play(2, Card{Hearts, Nine}) // Trump
	trick.Play(3, Card{Spades, Queen})

	result := trick.Result()

	if result.Winner != 2 {
		t.Errorf("Winner should be player 2, got %d", result.Winner)
	}
	if result.LeadSuit != Spades {
		t.Errorf("Lead suit should be Spades, got %s", result.LeadSuit)
	}
	if result.Trump != Hearts {
		t.Errorf("Trump should be Hearts, got %s", result.Trump)
	}
	if !result.WasTrumped {
		t.Error("Trick should be marked as trumped")
	}
	if len(result.Cards) != 4 {
		t.Errorf("Result should have 4 cards, got %d", len(result.Cards))
	}
}

func TestTrickWinner_UserScenario(t *testing.T) {
	// Trump is Hearts, lead is Spades
	// East (3) plays A♠, Partner (2) plays Q♠, West (1) plays 10♠, You (0) plays J♠
	// East should win with the Ace
	trick := NewTrick(Hearts)

	trick.Play(3, Card{Spades, Ace})  // East leads with A♠
	trick.Play(2, Card{Spades, Queen}) // Partner plays Q♠
	trick.Play(1, Card{Spades, Ten})   // West plays 10♠
	trick.Play(0, Card{Spades, Jack})  // You play J♠

	winner := trick.Winner()
	if winner != 3 {
		t.Errorf("Player 3 (East, A♠) should win, got player %d", winner)
	}

	// Verify card values for debugging
	t.Logf("A♠ value: %d", trick.cardValue(Card{Spades, Ace}))
	t.Logf("Q♠ value: %d", trick.cardValue(Card{Spades, Queen}))
	t.Logf("10♠ value: %d", trick.cardValue(Card{Spades, Ten}))
	t.Logf("J♠ value: %d", trick.cardValue(Card{Spades, Jack}))
}
