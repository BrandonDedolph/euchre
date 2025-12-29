package engine

// GamePhase represents the current phase of a round
type GamePhase int

const (
	PhaseDeal GamePhase = iota
	PhaseBidRound1  // Order up or pass
	PhaseBidRound2  // Name trump or pass (if all passed round 1)
	PhaseDiscard    // Dealer discards if trump was ordered up
	PhasePlay       // Card play
	PhaseTrickEnd   // Brief pause after trick ends
	PhaseRoundEnd   // Round scoring
	PhaseGameEnd    // Game is over
)

func (p GamePhase) String() string {
	switch p {
	case PhaseDeal:
		return "Deal"
	case PhaseBidRound1:
		return "Bid Round 1"
	case PhaseBidRound2:
		return "Bid Round 2"
	case PhaseDiscard:
		return "Discard"
	case PhasePlay:
		return "Play"
	case PhaseTrickEnd:
		return "Trick End"
	case PhaseRoundEnd:
		return "Round End"
	case PhaseGameEnd:
		return "Game End"
	default:
		return "Unknown"
	}
}

// ActionType represents the type of action a player can take
type ActionType int

const (
	ActionPass ActionType = iota
	ActionOrderUp     // Accept turned card as trump
	ActionCallTrump   // Name a trump suit
	ActionGoAlone     // Play without partner
	ActionDefendAlone // Defend without partner (optional rule)
	ActionDiscard     // Dealer discards a card
	ActionPlayCard    // Play a card to the trick
)

func (a ActionType) String() string {
	switch a {
	case ActionPass:
		return "Pass"
	case ActionOrderUp:
		return "Order Up"
	case ActionCallTrump:
		return "Call Trump"
	case ActionGoAlone:
		return "Go Alone"
	case ActionDefendAlone:
		return "Defend Alone"
	case ActionDiscard:
		return "Discard"
	case ActionPlayCard:
		return "Play Card"
	default:
		return "Unknown"
	}
}

// Action represents an action a player can take
type Action interface {
	Type() ActionType
	Player() int
}

// PassAction represents passing during bidding
type PassAction struct {
	PlayerIdx int
}

func (a PassAction) Type() ActionType { return ActionPass }
func (a PassAction) Player() int      { return a.PlayerIdx }

// OrderUpAction represents ordering up the turned card
type OrderUpAction struct {
	PlayerIdx int
	Alone     bool // Go alone flag
}

func (a OrderUpAction) Type() ActionType { return ActionOrderUp }
func (a OrderUpAction) Player() int      { return a.PlayerIdx }

// CallTrumpAction represents naming a trump suit in round 2
type CallTrumpAction struct {
	PlayerIdx int
	Suit      Suit
	Alone     bool
}

func (a CallTrumpAction) Type() ActionType { return ActionCallTrump }
func (a CallTrumpAction) Player() int      { return a.PlayerIdx }

// DiscardAction represents the dealer discarding a card
type DiscardAction struct {
	PlayerIdx int
	Card      Card
}

func (a DiscardAction) Type() ActionType { return ActionDiscard }
func (a DiscardAction) Player() int      { return a.PlayerIdx }

// PlayCardAction represents playing a card to the trick
type PlayCardAction struct {
	PlayerIdx int
	Card      Card
}

func (a PlayCardAction) Type() ActionType { return ActionPlayCard }
func (a PlayCardAction) Player() int      { return a.PlayerIdx }

// BidDecision represents an AI's bidding decision
type BidDecision struct {
	Pass     bool
	OrderUp  bool // Round 1: order up the turned card
	CallSuit Suit // Round 2: name a suit
	Alone    bool
}

// RoundResult contains the outcome of a completed round
type RoundResult struct {
	Makers       int  // Team that called trump (0 or 1)
	MakerTricks  int  // Tricks won by making team
	WasAlone     bool // Whether it was a loner attempt
	WasEuchred   bool // Whether makers were euchred
	MakerPoints  int  // Points scored by makers
	DefendPoints int  // Points scored by defenders (if euchred)
}

// ScoreUpdate represents point changes after a round
type ScoreUpdate struct {
	Team0Delta int
	Team1Delta int
}

// Team returns which team a player is on (0 or 1)
// In 4-player Euchre: players 0,2 are team 0; players 1,3 are team 1
func Team(playerIdx int) int {
	return playerIdx % 2
}

// Partner returns the partner's player index
// In 4-player Euchre: 0<->2, 1<->3
func Partner(playerIdx int) int {
	return (playerIdx + 2) % 4
}

// IsPartner returns true if the two players are partners
func IsPartner(a, b int) bool {
	return Team(a) == Team(b)
}

// NextPlayer returns the next player in clockwise order
func NextPlayer(current, numPlayers int) int {
	return (current + 1) % numPlayers
}

// PlayerPosition returns a descriptive position name
func PlayerPosition(playerIdx, dealerIdx, numPlayers int) string {
	offset := (playerIdx - dealerIdx + numPlayers) % numPlayers
	switch offset {
	case 0:
		return "Dealer"
	case 1:
		return "Left of Dealer"
	case 2:
		return "Across from Dealer"
	case 3:
		return "Right of Dealer"
	default:
		return "Unknown"
	}
}
