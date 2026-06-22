package rule_based

import (
	"github.com/bran/euchre/internal/ai"
	"github.com/bran/euchre/internal/engine"
)

// AI implements a rule-based Euchre AI player
type AI struct {
	name       string
	playerIdx  int
	difficulty ai.Difficulty
	bidder     *BiddingEvaluator
	player     *PlayStrategy
}

// New creates a new rule-based AI
func New(name string, playerIdx int, difficulty ai.Difficulty) *AI {
	// Set bidding threshold based on difficulty
	threshold := 55 // Medium default
	switch difficulty {
	case ai.DifficultyEasy:
		threshold = 65 // More conservative
	case ai.DifficultyMedium:
		threshold = 55
	case ai.DifficultyHard:
		threshold = 45 // More aggressive
	}

	return &AI{
		name:       name,
		playerIdx:  playerIdx,
		difficulty: difficulty,
		bidder:     NewBiddingEvaluator(threshold),
		player:     NewPlayStrategy(),
	}
}

// Name returns the AI's display name
func (a *AI) Name() string {
	return a.name
}

// DecideBid decides what to do during bidding
func (a *AI) DecideBid(state *engine.GameState, bidRound int) engine.BidDecision {
	hand := state.Hand(a.playerIdx)
	turnedCard := state.TurnedCard()
	dealer := state.Dealer()

	// Calculate position relative to dealer
	position := (a.playerIdx - dealer + 4) % 4
	isDealer := a.playerIdx == dealer

	decision := engine.BidDecision{Pass: true}

	if bidRound == 1 {
		shouldBid, goAlone := a.bidder.EvaluateRound1(hand, turnedCard, position, isDealer)
		if shouldBid {
			decision.Pass = false
			decision.OrderUp = true
			decision.Alone = goAlone
		}
	} else {
		// Round 2 - check if stick the dealer applies
		stickTheDealer := false // Could get this from variant config

		shouldBid, suit, goAlone := a.bidder.EvaluateRound2(hand, turnedCard.Suit, isDealer, stickTheDealer)
		if shouldBid {
			decision.Pass = false
			decision.CallSuit = suit
			decision.Alone = goAlone
		}
	}

	return decision
}

// DecidePlay chooses which card to play
func (a *AI) DecidePlay(state *engine.GameState) engine.Card {
	hand := state.Hand(a.playerIdx)
	trump := state.Trump()

	round := state.Round()
	if round == nil {
		return engine.Card{}
	}

	trick := engine.NewTrick(trump)
	for _, pc := range round.CurrentTrick() {
		trick.Play(pc.Player, pc.Card)
	}

	return a.player.SelectPlay(hand, trick, a.playerIdx, trump)
}

// DecideDiscard chooses which card to discard when dealer picks up
func (a *AI) DecideDiscard(state *engine.GameState, hand []engine.Card) engine.Card {
	turnedCard := state.TurnedCard()
	return a.bidder.SelectDiscard(hand, turnedCard)
}

// DecideDefendAlone decides whether to declare a lone defense against a lone
// maker. Defending alone is high risk: you must take 3+ of 5 tricks single-
// handed to euchre the maker (for 4 points), so the AI is conservative and
// declines unless the hand is genuinely strong against the known trump.
func (a *AI) DecideDefendAlone(state *engine.GameState) bool {
	hand := state.Hand(a.playerIdx)
	trump := state.Trump()
	return shouldDefendAlone(hand, trump)
}

// shouldDefendAlone is a pure helper that decides whether a hand is strong
// enough to defend alone against a lone maker for the given trump suit.
//
// It is deliberately conservative (declining is the default). It only accepts
// when the hand offers a realistic shot at 3 tricks single-handed:
//   - holds the right bower, OR
//   - holds the left bower plus at least one other trump, OR
//   - holds 3 or more trump cards, OR
//   - holds 2 trump including a bower plus an off-suit ace.
func shouldDefendAlone(hand []engine.Card, trump engine.Suit) bool {
	if trump == engine.NoSuit {
		return false
	}

	trumpCount := 0
	hasRightBower := false
	hasLeftBower := false
	offAces := 0
	for _, card := range hand {
		if card.IsTrump(trump) {
			trumpCount++
			if card.IsRightBower(trump) {
				hasRightBower = true
			}
			if card.IsLeftBower(trump) {
				hasLeftBower = true
			}
		} else if card.Rank == engine.Ace {
			offAces++
		}
	}
	hasBower := hasRightBower || hasLeftBower

	// Right bower is the single strongest card; combined with any other trump
	// or off-ace it gives a strong lone defense. Require it not be totally bare.
	if hasRightBower && (trumpCount >= 2 || offAces >= 1) {
		return true
	}

	// Left bower backed by another trump.
	if hasLeftBower && trumpCount >= 2 {
		return true
	}

	// Lots of trump length.
	if trumpCount >= 3 {
		return true
	}

	// Two trump including a bower, plus an off-suit ace for an extra trick.
	if trumpCount >= 2 && hasBower && offAces >= 1 {
		return true
	}

	return false
}

// CreateAIPlayers creates AI players for a game
func CreateAIPlayers(humanPlayer int, difficulty ai.Difficulty) []ai.Player {
	players := make([]ai.Player, 4)

	for i := 0; i < 4; i++ {
		if i == humanPlayer {
			players[i] = nil // Human player slot
		} else {
			name := ai.PlayerNames[i]
			players[i] = New(name, i, difficulty)
		}
	}

	return players
}
