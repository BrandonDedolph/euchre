package engine

import (
	"math/rand"
)

// Deck represents a deck of cards
type Deck struct {
	cards []Card
}

// NewStandardDeck creates a standard 24-card Euchre deck (9, 10, J, Q, K, A of each suit)
func NewStandardDeck() *Deck {
	cards := make([]Card, 0, 24)
	suits := []Suit{Clubs, Diamonds, Hearts, Spades}
	ranks := []Rank{Nine, Ten, Jack, Queen, King, Ace}

	for _, suit := range suits {
		for _, rank := range ranks {
			cards = append(cards, NewCard(suit, rank))
		}
	}

	return &Deck{cards: cards}
}

// NewDeck32 creates a 32-card Euchre deck (7, 8, 9, 10, J, Q, K, A of each suit)
// Used in some regional variants like New Zealand
func NewDeck32() *Deck {
	cards := make([]Card, 0, 32)
	suits := []Suit{Clubs, Diamonds, Hearts, Spades}
	// Note: We'd need to add Seven and Eight ranks for this
	// For now, using Nine as minimum
	ranks := []Rank{Nine, Ten, Jack, Queen, King, Ace}

	for _, suit := range suits {
		for _, rank := range ranks {
			cards = append(cards, NewCard(suit, rank))
		}
	}

	return &Deck{cards: cards}
}

// NewBritishDeck creates a 25-card deck with a Joker (Benny)
func NewBritishDeck() *Deck {
	deck := NewStandardDeck()
	deck.cards = append(deck.cards, NewCard(NoSuit, Joker))
	return deck
}

// Cards returns a copy of the cards in the deck
func (d *Deck) Cards() []Card {
	result := make([]Card, len(d.cards))
	copy(result, d.cards)
	return result
}

// Size returns the number of cards in the deck
func (d *Deck) Size() int {
	return len(d.cards)
}

// Shuffle randomizes the order of cards in the deck
func (d *Deck) Shuffle() {
	rand.Shuffle(len(d.cards), func(i, j int) {
		d.cards[i], d.cards[j] = d.cards[j], d.cards[i]
	})
}

// Draw removes and returns the top card from the deck
// Returns an empty card and false if deck is empty
func (d *Deck) Draw() (Card, bool) {
	if len(d.cards) == 0 {
		return Card{}, false
	}
	card := d.cards[len(d.cards)-1]
	d.cards = d.cards[:len(d.cards)-1]
	return card, true
}

// DrawN removes and returns n cards from the top of the deck
// Returns all available cards if n exceeds deck size
func (d *Deck) DrawN(n int) []Card {
	if n > len(d.cards) {
		n = len(d.cards)
	}
	if n <= 0 {
		return nil
	}

	start := len(d.cards) - n
	cards := make([]Card, n)
	copy(cards, d.cards[start:])
	d.cards = d.cards[:start]
	return cards
}

// Reset returns all dealt cards back to the deck and reshuffles
func (d *Deck) Reset(config DeckConfig) {
	d.cards = config.CreateDeck().cards
	d.Shuffle()
}

// DeckConfig defines how to create a deck for a variant
type DeckConfig interface {
	CreateDeck() *Deck
}

// StandardDeckConfig creates standard 24-card decks
type StandardDeckConfig struct{}

func (c StandardDeckConfig) CreateDeck() *Deck {
	return NewStandardDeck()
}

// BritishDeckConfig creates 25-card decks with Joker
type BritishDeckConfig struct{}

func (c BritishDeckConfig) CreateDeck() *Deck {
	return NewBritishDeck()
}

// Hand represents a player's hand of cards
type Hand struct {
	cards []Card
}

// NewHand creates a new empty hand
func NewHand() *Hand {
	return &Hand{cards: make([]Card, 0, 6)}
}

// NewHandWith creates a hand with the given cards
func NewHandWith(cards []Card) *Hand {
	h := &Hand{cards: make([]Card, len(cards))}
	copy(h.cards, cards)
	return h
}

// Cards returns a copy of the cards in the hand
func (h *Hand) Cards() []Card {
	result := make([]Card, len(h.cards))
	copy(result, h.cards)
	return result
}

// Size returns the number of cards in the hand
func (h *Hand) Size() int {
	return len(h.cards)
}

// Add adds a card to the hand
func (h *Hand) Add(card Card) {
	h.cards = append(h.cards, card)
}

// AddAll adds multiple cards to the hand
func (h *Hand) AddAll(cards []Card) {
	h.cards = append(h.cards, cards...)
}

// Remove removes a specific card from the hand
// Returns true if the card was found and removed
func (h *Hand) Remove(card Card) bool {
	for i, c := range h.cards {
		if c.Suit == card.Suit && c.Rank == card.Rank {
			h.cards = append(h.cards[:i], h.cards[i+1:]...)
			return true
		}
	}
	return false
}

// Contains returns true if the hand contains the specified card
func (h *Hand) Contains(card Card) bool {
	for _, c := range h.cards {
		if c.Suit == card.Suit && c.Rank == card.Rank {
			return true
		}
	}
	return false
}

// HasSuit returns true if the hand contains any card of the given suit
// Accounts for the Left Bower belonging to trump suit
func (h *Hand) HasSuit(suit Suit, trump Suit) bool {
	for _, c := range h.cards {
		if c.EffectiveSuit(trump) == suit {
			return true
		}
	}
	return false
}

// CardsOfSuit returns all cards that effectively belong to the given suit
func (h *Hand) CardsOfSuit(suit Suit, trump Suit) []Card {
	result := make([]Card, 0)
	for _, c := range h.cards {
		if c.EffectiveSuit(trump) == suit {
			result = append(result, c)
		}
	}
	return result
}

// Trumps returns all trump cards in the hand
func (h *Hand) Trumps(trump Suit) []Card {
	return h.CardsOfSuit(trump, trump)
}

// CountTrumps returns the number of trump cards
func (h *Hand) CountTrumps(trump Suit) int {
	return len(h.Trumps(trump))
}

// HighestTrump returns the highest trump card in the hand
// Returns an empty card and false if no trumps are held
func (h *Hand) HighestTrump(trump Suit) (Card, bool) {
	trumps := h.Trumps(trump)
	if len(trumps) == 0 {
		return Card{}, false
	}

	highest := trumps[0]
	highestValue := highest.TrumpValue(trump)

	for _, c := range trumps[1:] {
		if v := c.TrumpValue(trump); v > highestValue {
			highest = c
			highestValue = v
		}
	}

	return highest, true
}

// SortByTrump sorts the hand with trump cards first, then by suit and rank
func (h *Hand) SortByTrump(trump Suit) {
	// Simple bubble sort for small hand sizes
	for i := 0; i < len(h.cards)-1; i++ {
		for j := 0; j < len(h.cards)-i-1; j++ {
			if h.compareCards(h.cards[j], h.cards[j+1], trump) > 0 {
				h.cards[j], h.cards[j+1] = h.cards[j+1], h.cards[j]
			}
		}
	}
}

// compareCards compares two cards for sorting purposes
// Returns negative if a < b, positive if a > b, zero if equal
func (h *Hand) compareCards(a, b Card, trump Suit) int {
	// Trump cards come first
	aTrump := a.IsTrump(trump)
	bTrump := b.IsTrump(trump)

	if aTrump && !bTrump {
		return -1
	}
	if !aTrump && bTrump {
		return 1
	}

	// Both trump or both non-trump
	if aTrump {
		// Sort trumps by value (higher first)
		return b.TrumpValue(trump) - a.TrumpValue(trump)
	}

	// Non-trumps: sort by suit, then rank
	if a.Suit != b.Suit {
		return int(a.Suit) - int(b.Suit)
	}
	return int(b.Rank) - int(a.Rank)
}

// Clear removes all cards from the hand
func (h *Hand) Clear() {
	h.cards = h.cards[:0]
}
