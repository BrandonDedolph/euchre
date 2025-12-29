package engine

// Game represents a complete Euchre game
type Game struct {
	// Configuration
	numPlayers  int
	targetScore int
	deckConfig  DeckConfig

	// State
	scores       []int
	dealer       int
	currentRound *Round
	deck         *Deck

	// History
	roundHistory []RoundResult
}

// GameConfig contains configuration for a new game
type GameConfig struct {
	NumPlayers  int
	TargetScore int
	DeckConfig  DeckConfig
}

// DefaultGameConfig returns the standard 4-player Euchre configuration
func DefaultGameConfig() GameConfig {
	return GameConfig{
		NumPlayers:  4,
		TargetScore: 10,
		DeckConfig:  StandardDeckConfig{},
	}
}

// NewGame creates a new game with the given configuration
func NewGame(config GameConfig) *Game {
	if config.NumPlayers == 0 {
		config.NumPlayers = 4
	}
	if config.TargetScore == 0 {
		config.TargetScore = 10
	}
	if config.DeckConfig == nil {
		config.DeckConfig = StandardDeckConfig{}
	}

	numTeams := 2 // Standard Euchre has 2 teams

	return &Game{
		numPlayers:   config.NumPlayers,
		targetScore:  config.TargetScore,
		deckConfig:   config.DeckConfig,
		scores:       make([]int, numTeams),
		dealer:       0,
		deck:         config.DeckConfig.CreateDeck(),
		roundHistory: make([]RoundResult, 0),
	}
}

// StartRound begins a new round
func (g *Game) StartRound() {
	g.deck = g.deckConfig.CreateDeck()
	g.currentRound = NewRound(g.numPlayers, g.dealer)
	g.currentRound.Deal(g.deck)
}

// Round returns the current round
func (g *Game) Round() *Round {
	return g.currentRound
}

// Score returns the score for a team
func (g *Game) Score(team int) int {
	if team < 0 || team >= len(g.scores) {
		return 0
	}
	return g.scores[team]
}

// Scores returns all team scores
func (g *Game) Scores() []int {
	result := make([]int, len(g.scores))
	copy(result, g.scores)
	return result
}

// TargetScore returns the score needed to win
func (g *Game) TargetScore() int {
	return g.targetScore
}

// Dealer returns the current dealer
func (g *Game) Dealer() int {
	return g.dealer
}

// NumPlayers returns the number of players
func (g *Game) NumPlayers() int {
	return g.numPlayers
}

// Phase returns the current game phase
func (g *Game) Phase() GamePhase {
	if g.IsOver() {
		return PhaseGameEnd
	}
	if g.currentRound == nil {
		return PhaseDeal
	}
	return g.currentRound.Phase()
}

// CurrentPlayer returns whose turn it is
func (g *Game) CurrentPlayer() int {
	if g.currentRound == nil {
		return -1
	}
	return g.currentRound.CurrentPlayer()
}

// ApplyAction applies a player action
func (g *Game) ApplyAction(action Action) error {
	if g.currentRound == nil {
		return PlayError("no round in progress")
	}

	err := g.currentRound.ApplyAction(action)
	if err != nil {
		return err
	}

	// Check if round is complete
	if g.currentRound.IsComplete() {
		g.endRound()
	}

	return nil
}

func (g *Game) endRound() {
	result := g.currentRound.Result()
	g.roundHistory = append(g.roundHistory, result)

	// Update scores
	if result.MakerPoints > 0 {
		g.scores[result.Makers] += result.MakerPoints
	}
	if result.DefendPoints > 0 {
		defenderTeam := 1 - result.Makers
		g.scores[defenderTeam] += result.DefendPoints
	}

	// Advance dealer for next round
	g.dealer = NextPlayer(g.dealer, g.numPlayers)
}

// IsOver returns true if the game is finished
func (g *Game) IsOver() bool {
	for _, score := range g.scores {
		if score >= g.targetScore {
			return true
		}
	}
	return false
}

// Winner returns the winning team (-1 if game not over)
func (g *Game) Winner() int {
	for team, score := range g.scores {
		if score >= g.targetScore {
			return team
		}
	}
	return -1
}

// LegalActions returns all legal actions for the current player
func (g *Game) LegalActions() []Action {
	if g.currentRound == nil {
		return nil
	}
	return g.currentRound.LegalActions()
}

// Hand returns a player's current hand
func (g *Game) Hand(playerIdx int) []Card {
	if g.currentRound == nil {
		return nil
	}
	return g.currentRound.Hand(playerIdx)
}

// Trump returns the current trump suit
func (g *Game) Trump() Suit {
	if g.currentRound == nil {
		return NoSuit
	}
	return g.currentRound.Trump()
}

// TurnedCard returns the turned up card
func (g *Game) TurnedCard() Card {
	if g.currentRound == nil {
		return Card{}
	}
	return g.currentRound.TurnedCard()
}

// RoundHistory returns all completed round results
func (g *Game) RoundHistory() []RoundResult {
	result := make([]RoundResult, len(g.roundHistory))
	copy(result, g.roundHistory)
	return result
}

// NeedsNewRound returns true if we need to start a new round
func (g *Game) NeedsNewRound() bool {
	if g.IsOver() {
		return false
	}
	return g.currentRound == nil || g.currentRound.IsComplete()
}

// GameState provides a read-only view of the game state
type GameState struct {
	game *Game
}

// NewGameState creates a new game state view
func NewGameState(game *Game) *GameState {
	return &GameState{game: game}
}

// Phase returns the current phase
func (s *GameState) Phase() GamePhase {
	return s.game.Phase()
}

// CurrentPlayer returns whose turn it is
func (s *GameState) CurrentPlayer() int {
	return s.game.CurrentPlayer()
}

// Dealer returns the dealer
func (s *GameState) Dealer() int {
	return s.game.Dealer()
}

// Trump returns the trump suit
func (s *GameState) Trump() Suit {
	return s.game.Trump()
}

// Score returns a team's score
func (s *GameState) Score(team int) int {
	return s.game.Score(team)
}

// TargetScore returns the winning score
func (s *GameState) TargetScore() int {
	return s.game.TargetScore()
}

// Hand returns a player's hand
func (s *GameState) Hand(playerIdx int) []Card {
	return s.game.Hand(playerIdx)
}

// LegalActions returns legal actions for current player
func (s *GameState) LegalActions() []Action {
	return s.game.LegalActions()
}

// TurnedCard returns the turned up card
func (s *GameState) TurnedCard() Card {
	return s.game.TurnedCard()
}

// IsOver returns true if game is finished
func (s *GameState) IsOver() bool {
	return s.game.IsOver()
}

// NumPlayers returns player count
func (s *GameState) NumPlayers() int {
	return s.game.NumPlayers()
}

// Round returns the current round
func (s *GameState) Round() *Round {
	return s.game.Round()
}
