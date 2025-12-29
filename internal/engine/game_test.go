package engine

import "testing"

func TestNewGame(t *testing.T) {
	config := DefaultGameConfig()
	game := NewGame(config)

	if game.NumPlayers() != 4 {
		t.Errorf("Should have 4 players, got %d", game.NumPlayers())
	}
	if game.TargetScore() != 10 {
		t.Errorf("Target score should be 10, got %d", game.TargetScore())
	}
	if game.Score(0) != 0 || game.Score(1) != 0 {
		t.Error("Initial scores should be 0")
	}
	if game.IsOver() {
		t.Error("New game should not be over")
	}
}

func TestGameStartRound(t *testing.T) {
	game := NewGame(DefaultGameConfig())
	game.StartRound()

	round := game.Round()
	if round == nil {
		t.Error("Round should be started")
	}
	if game.Phase() != PhaseBidRound1 {
		t.Errorf("Should be in bid round 1, got %s", game.Phase())
	}

	// Players should have cards
	for i := 0; i < 4; i++ {
		hand := game.Hand(i)
		if len(hand) != 5 {
			t.Errorf("Player %d should have 5 cards, got %d", i, len(hand))
		}
	}
}

func TestGameApplyAction(t *testing.T) {
	game := NewGame(DefaultGameConfig())
	game.StartRound()

	// Apply a pass action
	currentPlayer := game.CurrentPlayer()
	err := game.ApplyAction(PassAction{PlayerIdx: currentPlayer})
	if err != nil {
		t.Errorf("Pass should succeed: %v", err)
	}
}

func TestGameScoring(t *testing.T) {
	game := NewGame(DefaultGameConfig())

	// Simulate a round where team 0 makes with 3 tricks
	game.StartRound()

	// Set up to play phase by having player 1 order up
	game.ApplyAction(OrderUpAction{PlayerIdx: game.CurrentPlayer(), Alone: false})
	game.ApplyAction(DiscardAction{PlayerIdx: game.Dealer(), Card: game.Hand(game.Dealer())[0]})

	// Play through the round
	for !game.NeedsNewRound() && !game.IsOver() {
		currentPlayer := game.CurrentPlayer()
		if currentPlayer < 0 {
			break
		}
		actions := game.LegalActions()
		if len(actions) > 0 {
			// Just play the first legal action
			game.ApplyAction(actions[0])
		}
	}

	// Check that scores updated
	scores := game.Scores()
	totalPoints := scores[0] + scores[1]
	if totalPoints == 0 {
		t.Error("Some team should have scored after a round")
	}
}

func TestGameNeedsNewRound(t *testing.T) {
	game := NewGame(DefaultGameConfig())

	// Before starting, needs new round
	if !game.NeedsNewRound() {
		t.Error("Should need new round before starting")
	}

	game.StartRound()

	// After starting, doesn't need new round
	if game.NeedsNewRound() {
		t.Error("Should not need new round while in progress")
	}
}

func TestGameDealerRotation(t *testing.T) {
	game := NewGame(DefaultGameConfig())

	initialDealer := game.Dealer()

	// Start and complete a round
	game.StartRound()

	// Play through quickly
	for !game.NeedsNewRound() && !game.IsOver() {
		currentPlayer := game.CurrentPlayer()
		if currentPlayer < 0 {
			break
		}
		actions := game.LegalActions()
		if len(actions) > 0 {
			game.ApplyAction(actions[0])
		}
	}

	// Dealer should rotate
	expectedDealer := (initialDealer + 1) % 4
	if game.Dealer() != expectedDealer {
		t.Errorf("Dealer should rotate to %d, got %d", expectedDealer, game.Dealer())
	}
}

func TestGameWinner(t *testing.T) {
	// Create game with low target score for testing
	config := GameConfig{
		NumPlayers:  4,
		TargetScore: 2, // Low for quick test
		DeckConfig:  StandardDeckConfig{},
	}
	game := NewGame(config)

	// Winner should be -1 at start
	if game.Winner() != -1 {
		t.Errorf("Winner should be -1 before game ends, got %d", game.Winner())
	}
}

func TestGameRoundHistory(t *testing.T) {
	game := NewGame(DefaultGameConfig())

	// Initially empty
	history := game.RoundHistory()
	if len(history) != 0 {
		t.Errorf("History should be empty initially, got %d entries", len(history))
	}

	// Start and complete a round
	game.StartRound()
	for !game.NeedsNewRound() && !game.IsOver() {
		currentPlayer := game.CurrentPlayer()
		if currentPlayer < 0 {
			break
		}
		actions := game.LegalActions()
		if len(actions) > 0 {
			game.ApplyAction(actions[0])
		}
	}

	history = game.RoundHistory()
	if len(history) != 1 {
		t.Errorf("History should have 1 entry after round, got %d", len(history))
	}
}

func TestGameState(t *testing.T) {
	game := NewGame(DefaultGameConfig())
	game.StartRound()

	state := NewGameState(game)

	// State should mirror game
	if state.Phase() != game.Phase() {
		t.Error("State phase should match game")
	}
	if state.CurrentPlayer() != game.CurrentPlayer() {
		t.Error("State current player should match game")
	}
	if state.Dealer() != game.Dealer() {
		t.Error("State dealer should match game")
	}
	if state.NumPlayers() != game.NumPlayers() {
		t.Error("State num players should match game")
	}
	if state.TargetScore() != game.TargetScore() {
		t.Error("State target score should match game")
	}
}

func TestDefaultGameConfig(t *testing.T) {
	config := DefaultGameConfig()

	if config.NumPlayers != 4 {
		t.Errorf("Default should have 4 players, got %d", config.NumPlayers)
	}
	if config.TargetScore != 10 {
		t.Errorf("Default should have target score 10, got %d", config.TargetScore)
	}
	if config.DeckConfig == nil {
		t.Error("Default should have deck config")
	}
}

func TestPhaseString(t *testing.T) {
	tests := []struct {
		phase    GamePhase
		expected string
	}{
		{PhaseDeal, "Deal"},
		{PhaseBidRound1, "Bid Round 1"},
		{PhaseBidRound2, "Bid Round 2"},
		{PhaseDiscard, "Discard"},
		{PhasePlay, "Play"},
		{PhaseTrickEnd, "Trick End"},
		{PhaseRoundEnd, "Round End"},
		{PhaseGameEnd, "Game End"},
	}

	for _, tt := range tests {
		if got := tt.phase.String(); got != tt.expected {
			t.Errorf("%d.String() = %s, want %s", tt.phase, got, tt.expected)
		}
	}
}

func TestActionTypeString(t *testing.T) {
	tests := []struct {
		action   ActionType
		expected string
	}{
		{ActionPass, "Pass"},
		{ActionOrderUp, "Order Up"},
		{ActionCallTrump, "Call Trump"},
		{ActionGoAlone, "Go Alone"},
		{ActionDiscard, "Discard"},
		{ActionPlayCard, "Play Card"},
	}

	for _, tt := range tests {
		if got := tt.action.String(); got != tt.expected {
			t.Errorf("%d.String() = %s, want %s", tt.action, got, tt.expected)
		}
	}
}
