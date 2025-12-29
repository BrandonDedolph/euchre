package rule_based

import "github.com/bran/euchre/internal/engine"

// PlayStrategy handles card play decisions
type PlayStrategy struct{}

// NewPlayStrategy creates a new play strategy
func NewPlayStrategy() *PlayStrategy {
	return &PlayStrategy{}
}

// SelectPlay chooses the best card to play from the legal options
func (s *PlayStrategy) SelectPlay(hand []engine.Card, trick *engine.Trick, playerIdx int, trump engine.Suit) engine.Card {
	// Get legal plays
	legalPlays := engine.LegalPlays(engine.NewHandWith(hand), trick)
	if len(legalPlays) == 0 {
		return engine.Card{} // Shouldn't happen
	}
	if len(legalPlays) == 1 {
		return legalPlays[0] // Only one choice
	}

	// If leading, use lead strategy
	if trick.Size() == 0 {
		return s.selectLead(legalPlays, trump, playerIdx)
	}

	// Otherwise, use follow strategy
	return s.selectFollow(legalPlays, trick, playerIdx, trump)
}

// selectLead chooses the best card to lead
func (s *PlayStrategy) selectLead(options []engine.Card, trump engine.Suit, playerIdx int) engine.Card {
	// If we have trump, consider leading it
	var trumps []engine.Card
	var offSuit []engine.Card

	for _, card := range options {
		if card.IsTrump(trump) {
			trumps = append(trumps, card)
		} else {
			offSuit = append(offSuit, card)
		}
	}

	// If we have multiple trumps, lead our highest trump to draw out opponents' trumps
	if len(trumps) >= 2 {
		return s.highestCard(trumps, trump)
	}

	// If we have off-suit aces, lead them
	for _, card := range offSuit {
		if card.Rank == engine.Ace {
			return card
		}
	}

	// Lead lowest off-suit to preserve trumps
	if len(offSuit) > 0 {
		return s.lowestCard(offSuit)
	}

	// Only have trumps, lead the lowest to preserve bowers
	return s.lowestCard(trumps)
}

// selectFollow chooses the best card when following
func (s *PlayStrategy) selectFollow(options []engine.Card, trick *engine.Trick, playerIdx int, trump engine.Suit) engine.Card {
	leadSuit := trick.LeadSuit()
	winningCard, _ := trick.WinningCard()
	winningPlayer := trick.Winner()
	isPartnerWinning := engine.IsPartner(playerIdx, winningPlayer)

	// Separate options by type
	var followSuit []engine.Card
	var trumps []engine.Card
	var offSuit []engine.Card

	for _, card := range options {
		effectiveSuit := card.EffectiveSuit(trump)
		if effectiveSuit == leadSuit {
			followSuit = append(followSuit, card)
		} else if card.IsTrump(trump) {
			trumps = append(trumps, card)
		} else {
			offSuit = append(offSuit, card)
		}
	}

	// Must follow suit
	if len(followSuit) > 0 {
		return s.playFollowSuit(followSuit, winningCard, isPartnerWinning, trump)
	}

	// Can't follow suit - decide whether to trump or discard
	if len(trumps) > 0 {
		return s.playTrump(trumps, trick, isPartnerWinning, trump)
	}

	// Can only discard
	return s.selectDiscard(offSuit)
}

// playFollowSuit chooses the best card when following suit
func (s *PlayStrategy) playFollowSuit(options []engine.Card, winningCard engine.Card, isPartnerWinning bool, trump engine.Suit) engine.Card {
	if isPartnerWinning {
		// Partner is winning - play our lowest card
		return s.lowestCard(options)
	}

	// Try to beat the winning card with minimum effort
	var beaters []engine.Card
	for _, card := range options {
		if s.beats(card, winningCard, trump) {
			beaters = append(beaters, card)
		}
	}

	if len(beaters) > 0 {
		// Play the lowest card that beats the current winner
		return s.lowestCard(beaters)
	}

	// Can't beat it - play lowest
	return s.lowestCard(options)
}

// playTrump decides whether and how to trump
func (s *PlayStrategy) playTrump(trumps []engine.Card, trick *engine.Trick, isPartnerWinning bool, trump engine.Suit) engine.Card {
	if isPartnerWinning {
		// Partner is winning - don't waste a trump, but we must play one
		// This situation means we're void in the lead suit and only have trumps
		return s.lowestCard(trumps)
	}

	// Opponent is winning - trump with lowest trump that wins
	winningCard, _ := trick.WinningCard()

	var beaters []engine.Card
	for _, card := range trumps {
		if s.beats(card, winningCard, trump) {
			beaters = append(beaters, card)
		}
	}

	if len(beaters) > 0 {
		return s.lowestCard(beaters)
	}

	// All our trumps lose to the current winner (e.g., they already trumped high)
	// Play our lowest trump
	return s.lowestCard(trumps)
}

// selectDiscard chooses which card to throw away
func (s *PlayStrategy) selectDiscard(options []engine.Card) engine.Card {
	if len(options) == 0 {
		return engine.Card{}
	}
	// Discard our lowest card
	return s.lowestCard(options)
}

// beats returns true if card a beats card b
func (s *PlayStrategy) beats(a, b engine.Card, trump engine.Suit) bool {
	// Compare based on trump status
	aTrump := a.IsTrump(trump)
	bTrump := b.IsTrump(trump)

	if aTrump && !bTrump {
		return true
	}
	if !aTrump && bTrump {
		return false
	}
	if aTrump && bTrump {
		return a.TrumpValue(trump) > b.TrumpValue(trump)
	}

	// Both off-suit - compare by rank
	return a.OffSuitValue() > b.OffSuitValue()
}

// highestCard finds the highest value card
func (s *PlayStrategy) highestCard(cards []engine.Card, trump engine.Suit) engine.Card {
	if len(cards) == 0 {
		return engine.Card{}
	}

	best := cards[0]
	bestValue := s.cardValue(best, trump)

	for _, card := range cards[1:] {
		value := s.cardValue(card, trump)
		if value > bestValue {
			best = card
			bestValue = value
		}
	}

	return best
}

// lowestCard finds the lowest value card
func (s *PlayStrategy) lowestCard(cards []engine.Card) engine.Card {
	if len(cards) == 0 {
		return engine.Card{}
	}

	worst := cards[0]
	worstValue := worst.OffSuitValue()

	for _, card := range cards[1:] {
		value := card.OffSuitValue()
		if value < worstValue {
			worst = card
			worstValue = value
		}
	}

	return worst
}

// cardValue returns a comparable value for a card
func (s *PlayStrategy) cardValue(card engine.Card, trump engine.Suit) int {
	if card.IsTrump(trump) {
		return 100 + card.TrumpValue(trump)
	}
	return card.OffSuitValue()
}
