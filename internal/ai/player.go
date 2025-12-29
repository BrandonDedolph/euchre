package ai

import "github.com/bran/euchre/internal/engine"

// Player represents an AI-controlled player
type Player interface {
	// DecideBid decides what to do during bidding
	DecideBid(state *engine.GameState, round int) engine.BidDecision

	// DecidePlay chooses which card to play
	DecidePlay(state *engine.GameState) engine.Card

	// DecideDiscard chooses which card to discard (when dealer picks up)
	DecideDiscard(state *engine.GameState, hand []engine.Card) engine.Card

	// Name returns a display name for this AI
	Name() string
}

// Difficulty represents AI skill level
type Difficulty int

const (
	DifficultyEasy Difficulty = iota
	DifficultyMedium
	DifficultyHard
)

func (d Difficulty) String() string {
	switch d {
	case DifficultyEasy:
		return "Easy"
	case DifficultyMedium:
		return "Medium"
	case DifficultyHard:
		return "Hard"
	default:
		return "Unknown"
	}
}

// PlayerNames provides default names for AI players
var PlayerNames = []string{
	"Alice",
	"Bob",
	"Carol",
	"Dave",
}
