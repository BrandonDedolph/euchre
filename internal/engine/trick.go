package engine

// Trick represents a single trick (round of card play)
type Trick struct {
	cards    []PlayedCard
	leadSuit Suit
	trump    Suit
}

// NewTrick creates a new empty trick
func NewTrick(trump Suit) *Trick {
	return &Trick{
		cards: make([]PlayedCard, 0, 4),
		trump: trump,
	}
}

// Play adds a card to the trick
func (t *Trick) Play(player int, card Card) {
	pc := PlayedCard{Player: player, Card: card}
	t.cards = append(t.cards, pc)

	// First card establishes the lead suit
	if len(t.cards) == 1 {
		t.leadSuit = card.EffectiveSuit(t.trump)
	}
}

// Cards returns the played cards in order
func (t *Trick) Cards() []PlayedCard {
	result := make([]PlayedCard, len(t.cards))
	copy(result, t.cards)
	return result
}

// Size returns the number of cards played
func (t *Trick) Size() int {
	return len(t.cards)
}

// IsComplete returns true if all expected players have played
func (t *Trick) IsComplete(playerCount int) bool {
	return len(t.cards) >= playerCount
}

// LeadSuit returns the effective suit of the first card played
func (t *Trick) LeadSuit() Suit {
	return t.leadSuit
}

// Trump returns the trump suit for this trick
func (t *Trick) Trump() Suit {
	return t.trump
}

// Leader returns the player who led this trick
func (t *Trick) Leader() int {
	if len(t.cards) == 0 {
		return -1
	}
	return t.cards[0].Player
}

// Winner determines which player won the trick
// Returns -1 if trick is empty
func (t *Trick) Winner() int {
	if len(t.cards) == 0 {
		return -1
	}

	winningCard := t.cards[0]
	winningValue := t.cardValue(winningCard.Card)

	for _, pc := range t.cards[1:] {
		value := t.cardValue(pc.Card)
		if value > winningValue {
			winningCard = pc
			winningValue = value
		}
	}

	return winningCard.Player
}

// WinningCard returns the card that is currently winning
func (t *Trick) WinningCard() (Card, bool) {
	if len(t.cards) == 0 {
		return Card{}, false
	}

	winner := t.cards[0].Card
	winningValue := t.cardValue(winner)

	for _, pc := range t.cards[1:] {
		value := t.cardValue(pc.Card)
		if value > winningValue {
			winner = pc.Card
			winningValue = value
		}
	}

	return winner, true
}

// cardValue calculates the trick-taking power of a card
// Trump cards have highest values, then lead suit, then others (value 0)
func (t *Trick) cardValue(card Card) int {
	// Trump cards are most valuable
	if card.IsTrump(t.trump) {
		return 1000 + card.TrumpValue(t.trump)
	}

	// Cards following lead suit
	if card.EffectiveSuit(t.trump) == t.leadSuit {
		return 100 + card.OffSuitValue()
	}

	// Off-suit cards can't win (unless they trump)
	return 0
}

// CanBeat returns true if the given card can beat the current winning card
func (t *Trick) CanBeat(card Card) bool {
	if len(t.cards) == 0 {
		return true // First card always "wins"
	}

	currentBest, _ := t.WinningCard()
	return t.cardValue(card) > t.cardValue(currentBest)
}

// MustFollowSuit returns the suit that must be followed, if any
// Returns the lead suit, or NoSuit if trick is empty
func (t *Trick) MustFollowSuit() Suit {
	if len(t.cards) == 0 {
		return NoSuit
	}
	return t.leadSuit
}

// Clear resets the trick for reuse
func (t *Trick) Clear() {
	t.cards = t.cards[:0]
	t.leadSuit = NoSuit
}

// TrickResult contains the outcome of a completed trick
type TrickResult struct {
	Winner    int
	Cards     []PlayedCard
	LeadSuit  Suit
	Trump     Suit
	WasTrumped bool
}

// Result returns the trick result after the trick is complete
func (t *Trick) Result() TrickResult {
	winner := t.Winner()
	wasTrumped := false

	// Check if the winning card was a trump when a non-trump was led
	if winner >= 0 && len(t.cards) > 0 {
		winnerCard := t.findWinningCard()
		if winnerCard.IsTrump(t.trump) && t.leadSuit != t.trump {
			wasTrumped = true
		}
	}

	return TrickResult{
		Winner:     winner,
		Cards:      t.Cards(),
		LeadSuit:   t.leadSuit,
		Trump:      t.trump,
		WasTrumped: wasTrumped,
	}
}

func (t *Trick) findWinningCard() Card {
	card, _ := t.WinningCard()
	return card
}

// ValidatePlay checks if a card play is legal given the player's hand
func ValidatePlay(hand *Hand, card Card, trick *Trick) error {
	// Must have the card in hand
	if !hand.Contains(card) {
		return ErrCardNotInHand
	}

	// If trick is empty, any card is valid
	if trick.Size() == 0 {
		return nil
	}

	leadSuit := trick.LeadSuit()
	trump := trick.Trump()

	// Must follow suit if able
	if hand.HasSuit(leadSuit, trump) {
		if card.EffectiveSuit(trump) != leadSuit {
			return ErrMustFollowSuit
		}
	}

	return nil
}

// LegalPlays returns all cards that can legally be played
func LegalPlays(hand *Hand, trick *Trick) []Card {
	cards := hand.Cards()
	if trick.Size() == 0 {
		return cards // Any card is legal when leading
	}

	leadSuit := trick.LeadSuit()
	trump := trick.Trump()

	// If we have cards of the lead suit, we must play one
	suitCards := hand.CardsOfSuit(leadSuit, trump)
	if len(suitCards) > 0 {
		return suitCards
	}

	// Otherwise, any card is legal
	return cards
}

// Errors for card play validation
type PlayError string

func (e PlayError) Error() string {
	return string(e)
}

const (
	ErrCardNotInHand  PlayError = "card not in hand"
	ErrMustFollowSuit PlayError = "must follow suit if able"
	ErrNotYourTurn    PlayError = "not your turn"
)
