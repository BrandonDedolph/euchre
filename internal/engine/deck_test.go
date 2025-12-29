package engine

import "testing"

func TestNewStandardDeck(t *testing.T) {
	deck := NewStandardDeck()

	if deck.Size() != 24 {
		t.Errorf("Standard deck should have 24 cards, got %d", deck.Size())
	}

	// Verify all expected cards are present
	cards := deck.Cards()
	cardSet := make(map[string]bool)
	for _, c := range cards {
		cardSet[c.String()] = true
	}

	suits := []Suit{Clubs, Diamonds, Hearts, Spades}
	ranks := []Rank{Nine, Ten, Jack, Queen, King, Ace}

	for _, suit := range suits {
		for _, rank := range ranks {
			card := NewCard(suit, rank)
			if !cardSet[card.String()] {
				t.Errorf("Missing card: %s", card)
			}
		}
	}
}

func TestNewBritishDeck(t *testing.T) {
	deck := NewBritishDeck()

	if deck.Size() != 25 {
		t.Errorf("British deck should have 25 cards, got %d", deck.Size())
	}

	// Check for joker
	hasJoker := false
	for _, c := range deck.Cards() {
		if c.IsJoker() {
			hasJoker = true
			break
		}
	}

	if !hasJoker {
		t.Error("British deck should contain a joker")
	}
}

func TestDeckDraw(t *testing.T) {
	deck := NewStandardDeck()
	initialSize := deck.Size()

	card, ok := deck.Draw()
	if !ok {
		t.Error("Draw should succeed on non-empty deck")
	}
	if card.Rank == 0 && card.Suit == 0 {
		t.Error("Drawn card should not be empty")
	}
	if deck.Size() != initialSize-1 {
		t.Errorf("Deck size should decrease after draw, got %d", deck.Size())
	}
}

func TestDeckDrawN(t *testing.T) {
	deck := NewStandardDeck()

	cards := deck.DrawN(5)
	if len(cards) != 5 {
		t.Errorf("DrawN(5) should return 5 cards, got %d", len(cards))
	}
	if deck.Size() != 19 {
		t.Errorf("Deck should have 19 cards remaining, got %d", deck.Size())
	}

	// Drawing more than available
	cards = deck.DrawN(100)
	if len(cards) != 19 {
		t.Errorf("DrawN(100) should return all remaining 19 cards, got %d", len(cards))
	}
	if deck.Size() != 0 {
		t.Errorf("Deck should be empty, got %d cards", deck.Size())
	}
}

func TestDeckDrawEmpty(t *testing.T) {
	deck := NewStandardDeck()

	// Empty the deck
	deck.DrawN(24)

	card, ok := deck.Draw()
	if ok {
		t.Error("Draw should fail on empty deck")
	}
	if card.Rank != 0 || card.Suit != 0 {
		t.Error("Draw on empty deck should return empty card")
	}
}

func TestDeckShuffle(t *testing.T) {
	deck1 := NewStandardDeck()
	deck2 := NewStandardDeck()

	deck1.Shuffle()

	// After shuffle, cards should (very likely) be in different order
	cards1 := deck1.Cards()
	cards2 := deck2.Cards()

	same := true
	for i := range cards1 {
		if cards1[i] != cards2[i] {
			same = false
			break
		}
	}

	// This could theoretically fail with astronomically low probability
	if same {
		t.Error("Shuffled deck should (almost certainly) have different order")
	}
}

func TestNewHand(t *testing.T) {
	hand := NewHand()
	if hand.Size() != 0 {
		t.Errorf("New hand should be empty, got %d cards", hand.Size())
	}
}

func TestNewHandWith(t *testing.T) {
	cards := []Card{
		{Hearts, Ace},
		{Spades, King},
		{Diamonds, Jack},
	}

	hand := NewHandWith(cards)
	if hand.Size() != 3 {
		t.Errorf("Hand should have 3 cards, got %d", hand.Size())
	}

	// Verify independence (modifying original shouldn't affect hand)
	cards[0] = Card{Clubs, Nine}
	if hand.Cards()[0].Suit == Clubs {
		t.Error("Hand should be independent of original slice")
	}
}

func TestHandAddRemove(t *testing.T) {
	hand := NewHand()

	card1 := Card{Hearts, Ace}
	card2 := Card{Spades, King}

	hand.Add(card1)
	hand.Add(card2)

	if hand.Size() != 2 {
		t.Errorf("Hand should have 2 cards, got %d", hand.Size())
	}

	if !hand.Contains(card1) {
		t.Error("Hand should contain added card")
	}

	removed := hand.Remove(card1)
	if !removed {
		t.Error("Remove should return true for existing card")
	}
	if hand.Size() != 1 {
		t.Errorf("Hand should have 1 card after removal, got %d", hand.Size())
	}
	if hand.Contains(card1) {
		t.Error("Hand should not contain removed card")
	}

	removed = hand.Remove(card1)
	if removed {
		t.Error("Remove should return false for non-existing card")
	}
}

func TestHandHasSuit(t *testing.T) {
	hand := NewHandWith([]Card{
		{Hearts, Ace},
		{Hearts, King},
		{Spades, Queen},
		{Diamonds, Jack}, // Left bower if hearts is trump
	})

	trump := Hearts

	if !hand.HasSuit(Hearts, trump) {
		t.Error("Hand should have hearts")
	}
	if !hand.HasSuit(Spades, trump) {
		t.Error("Hand should have spades")
	}
	if hand.HasSuit(Clubs, trump) {
		t.Error("Hand should not have clubs")
	}
	if hand.HasSuit(Diamonds, trump) {
		t.Error("Hand should not have diamonds (J♦ is trump when hearts is trump)")
	}
}

func TestHandCardsOfSuit(t *testing.T) {
	hand := NewHandWith([]Card{
		{Hearts, Ace},
		{Hearts, King},
		{Diamonds, Jack}, // Left bower
		{Spades, Queen},
	})

	trump := Hearts

	hearts := hand.CardsOfSuit(Hearts, trump)
	if len(hearts) != 3 { // A♥, K♥, and J♦ (left bower)
		t.Errorf("Should have 3 hearts (including left bower), got %d", len(hearts))
	}

	spades := hand.CardsOfSuit(Spades, trump)
	if len(spades) != 1 {
		t.Errorf("Should have 1 spade, got %d", len(spades))
	}
}

func TestHandTrumps(t *testing.T) {
	hand := NewHandWith([]Card{
		{Hearts, Ace},
		{Hearts, Nine},
		{Diamonds, Jack}, // Left bower
		{Spades, King},
	})

	trumps := hand.Trumps(Hearts)
	if len(trumps) != 3 {
		t.Errorf("Should have 3 trumps, got %d", len(trumps))
	}
}

func TestHandHighestTrump(t *testing.T) {
	hand := NewHandWith([]Card{
		{Hearts, Ace},
		{Hearts, Jack}, // Right bower
		{Diamonds, Jack}, // Left bower
		{Spades, King},
	})

	highest, ok := hand.HighestTrump(Hearts)
	if !ok {
		t.Error("Should find highest trump")
	}
	if !highest.IsRightBower(Hearts) {
		t.Errorf("Highest trump should be right bower, got %s", highest)
	}

	// Test with no trumps
	noTrumpHand := NewHandWith([]Card{
		{Spades, Ace},
		{Clubs, King},
	})
	_, ok = noTrumpHand.HighestTrump(Hearts)
	if ok {
		t.Error("Should not find trump in hand with no trumps")
	}
}

func TestHandClear(t *testing.T) {
	hand := NewHandWith([]Card{
		{Hearts, Ace},
		{Spades, King},
	})

	hand.Clear()
	if hand.Size() != 0 {
		t.Errorf("Cleared hand should be empty, got %d cards", hand.Size())
	}
}
