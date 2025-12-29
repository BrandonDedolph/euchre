package standard

import (
	"github.com/bran/euchre/internal/engine"
	"github.com/bran/euchre/internal/variants"
)

// Standard implements the standard 4-player Euchre variant
type Standard struct {
	variants.BaseVariant
}

// New creates a new standard Euchre variant
func New() *Standard {
	s := &Standard{
		BaseVariant: variants.NewBaseVariant(),
	}

	// Set default options
	s.SetOption("stick_the_dealer", false)
	s.SetOption("farmers_hand", false)
	s.SetOption("defend_alone", false)

	return s
}

// Name returns the variant name
func (s *Standard) Name() string {
	return "Standard"
}

// Description returns a description of the variant
func (s *Standard) Description() string {
	return "Standard 4-player Euchre with 24-card deck. Partners sit across from each other. First team to 10 points wins."
}

// PlayerCount returns the number of players
func (s *Standard) PlayerCount() int {
	return 4
}

// TeamCount returns the number of teams
func (s *Standard) TeamCount() int {
	return 2
}

// CardsPerHand returns cards dealt to each player
func (s *Standard) CardsPerHand() int {
	return 5
}

// TargetScore returns the score needed to win
func (s *Standard) TargetScore() int {
	return 10
}

// CreateDeck creates a standard 24-card Euchre deck
func (s *Standard) CreateDeck() *engine.Deck {
	return engine.NewStandardDeck()
}

// HasJoker returns whether this variant uses a joker
func (s *Standard) HasJoker() bool {
	return false
}

// TrumpHierarchy returns the trump cards in order from highest to lowest
func (s *Standard) TrumpHierarchy(trump engine.Suit) []engine.Card {
	// Find the suit of the same color for Left Bower
	var leftBowerSuit engine.Suit
	switch trump {
	case engine.Hearts:
		leftBowerSuit = engine.Diamonds
	case engine.Diamonds:
		leftBowerSuit = engine.Hearts
	case engine.Spades:
		leftBowerSuit = engine.Clubs
	case engine.Clubs:
		leftBowerSuit = engine.Spades
	}

	return []engine.Card{
		{Suit: trump, Rank: engine.Jack},          // Right Bower
		{Suit: leftBowerSuit, Rank: engine.Jack},  // Left Bower
		{Suit: trump, Rank: engine.Ace},
		{Suit: trump, Rank: engine.King},
		{Suit: trump, Rank: engine.Queen},
		{Suit: trump, Rank: engine.Ten},
		{Suit: trump, Rank: engine.Nine},
	}
}

// IsLeftBower checks if a card is the Left Bower
func (s *Standard) IsLeftBower(card engine.Card, trump engine.Suit) bool {
	return card.IsLeftBower(trump)
}

// BiddingRounds returns the number of bidding rounds
func (s *Standard) BiddingRounds() int {
	return 2
}

// CanGoAlone returns whether players can go alone
func (s *Standard) CanGoAlone() bool {
	return true
}

// HasStickTheDealer returns whether dealer must call if all pass
func (s *Standard) HasStickTheDealer() bool {
	return s.GetBoolOption("stick_the_dealer", false)
}

// ScoreRound calculates the score for a completed round
func (s *Standard) ScoreRound(result engine.RoundResult) engine.ScoreUpdate {
	update := engine.ScoreUpdate{}

	if result.WasEuchred {
		// Defending team scores 2 points
		if result.Makers == 0 {
			update.Team1Delta = 2
		} else {
			update.Team0Delta = 2
		}
		return update
	}

	// Making team scored
	points := result.MakerPoints
	if result.Makers == 0 {
		update.Team0Delta = points
	} else {
		update.Team1Delta = points
	}

	return update
}

// HasFarmersHand returns whether the "no ace, no face, no trump" rule is used
func (s *Standard) HasFarmersHand() bool {
	return s.GetBoolOption("farmers_hand", false)
}

// AllowMisdeal returns whether a misdeal is allowed on stuck hands
func (s *Standard) AllowMisdeal() bool {
	return !s.HasStickTheDealer()
}

// Options returns all configurable options
func (s *Standard) Options() []variants.RuleOption {
	return []variants.RuleOption{
		{
			Key:         "stick_the_dealer",
			Name:        "Stick the Dealer",
			Description: "Dealer must call trump if everyone passes",
			Type:        variants.OptionBool,
			Default:     false,
		},
		{
			Key:         "farmers_hand",
			Name:        "Farmer's Hand",
			Description: "Allow redeal with no ace, no face, no trump",
			Type:        variants.OptionBool,
			Default:     false,
		},
		{
			Key:         "defend_alone",
			Name:        "Defend Alone",
			Description: "Allow defenders to go alone for 4 points on euchre",
			Type:        variants.OptionBool,
			Default:     false,
		},
	}
}

func init() {
	variants.Register(New())
}
