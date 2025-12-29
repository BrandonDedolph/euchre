package engine

import "fmt"

// Suit represents a card suit
type Suit int

const (
	Clubs Suit = iota
	Diamonds
	Hearts
	Spades
	NoSuit // For jokers or no trump
)

// SameSuitColor returns true if the suits are the same color
func (s Suit) SameColor(other Suit) bool {
	if s == NoSuit || other == NoSuit {
		return false
	}
	// Clubs & Spades are black, Diamonds & Hearts are red
	return (s == Clubs || s == Spades) == (other == Clubs || other == Spades)
}

// Color returns "black" or "red" for the suit
func (s Suit) Color() string {
	switch s {
	case Clubs, Spades:
		return "black"
	case Diamonds, Hearts:
		return "red"
	default:
		return ""
	}
}

// String returns the suit name
func (s Suit) String() string {
	switch s {
	case Clubs:
		return "Clubs"
	case Diamonds:
		return "Diamonds"
	case Hearts:
		return "Hearts"
	case Spades:
		return "Spades"
	case NoSuit:
		return "NoSuit"
	default:
		return "Unknown"
	}
}

// Symbol returns the Unicode symbol for the suit
func (s Suit) Symbol() string {
	switch s {
	case Clubs:
		return "♣"
	case Diamonds:
		return "♦"
	case Hearts:
		return "♥"
	case Spades:
		return "♠"
	default:
		return "?"
	}
}

// Rank represents a card rank
type Rank int

const (
	Nine Rank = iota
	Ten
	Jack
	Queen
	King
	Ace
	Joker // For British Euchre with Benny
)

// String returns the rank name
func (r Rank) String() string {
	switch r {
	case Nine:
		return "9"
	case Ten:
		return "10"
	case Jack:
		return "J"
	case Queen:
		return "Q"
	case King:
		return "K"
	case Ace:
		return "A"
	case Joker:
		return "Joker"
	default:
		return "?"
	}
}

// Card represents a playing card
type Card struct {
	Suit Suit
	Rank Rank
}

// NewCard creates a new card
func NewCard(suit Suit, rank Rank) Card {
	return Card{Suit: suit, Rank: rank}
}

// String returns a human-readable representation of the card
func (c Card) String() string {
	if c.Rank == Joker {
		return "Joker"
	}
	return fmt.Sprintf("%s%s", c.Rank.String(), c.Suit.Symbol())
}

// ShortString returns an abbreviated representation (e.g., "9♣", "J♠")
func (c Card) ShortString() string {
	return c.String()
}

// IsJoker returns true if this card is a joker
func (c Card) IsJoker() bool {
	return c.Rank == Joker
}

// IsRightBower returns true if this card is the Right Bower (Jack of trump)
func (c Card) IsRightBower(trump Suit) bool {
	return c.Rank == Jack && c.Suit == trump
}

// IsLeftBower returns true if this card is the Left Bower (Jack of same color as trump)
func (c Card) IsLeftBower(trump Suit) bool {
	if c.Rank != Jack {
		return false
	}
	return c.Suit != trump && c.Suit.SameColor(trump)
}

// IsBower returns true if this card is either bower
func (c Card) IsBower(trump Suit) bool {
	return c.IsRightBower(trump) || c.IsLeftBower(trump)
}

// EffectiveSuit returns the suit of the card, accounting for the Left Bower
// which is considered part of the trump suit
func (c Card) EffectiveSuit(trump Suit) Suit {
	if c.IsLeftBower(trump) {
		return trump
	}
	return c.Suit
}

// IsTrump returns true if this card is a trump card
func (c Card) IsTrump(trump Suit) bool {
	if c.IsJoker() {
		return true // Joker is always trump if in play
	}
	return c.EffectiveSuit(trump) == trump
}

// TrumpValue returns the card's trick-taking power within the trump suit
// Higher values beat lower values
// Returns 0 for non-trump cards
func (c Card) TrumpValue(trump Suit) int {
	if !c.IsTrump(trump) {
		return 0
	}

	// Joker (Benny) is highest if present
	if c.IsJoker() {
		return 100
	}

	// Right Bower (Jack of trump) is highest standard card
	if c.IsRightBower(trump) {
		return 90
	}

	// Left Bower (Jack of same color) is second highest
	if c.IsLeftBower(trump) {
		return 80
	}

	// Regular trump cards: A, K, Q, 10, 9
	switch c.Rank {
	case Ace:
		return 70
	case King:
		return 60
	case Queen:
		return 50
	case Ten:
		return 40
	case Nine:
		return 30
	default:
		return 0
	}
}

// OffSuitValue returns the card's trick-taking power in a non-trump suit
func (c Card) OffSuitValue() int {
	switch c.Rank {
	case Ace:
		return 60
	case King:
		return 50
	case Queen:
		return 40
	case Jack:
		return 30
	case Ten:
		return 20
	case Nine:
		return 10
	default:
		return 0
	}
}

// PlayedCard represents a card played by a specific player
type PlayedCard struct {
	Player int
	Card   Card
}
