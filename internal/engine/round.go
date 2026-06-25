package engine

// Round represents a single round of Euchre (one deal until scoring)
type Round struct {
	// Configuration
	numPlayers int
	dealer     int
	rules      Rules

	// State
	phase         GamePhase
	misdeal       bool // true if round 2 all-pass throw-in occurred
	trump         Suit
	turnedCard    Card
	maker         int  // Player who called trump
	makerTeam     int  // Team that called trump
	alone         bool // Whether maker is going alone
	aloneDefender int  // -1 if no lone defender, else player idx

	// defendAlonePoll is the defender currently being polled during
	// PhaseDefendAlone, or -1 when not in that phase.
	defendAlonePoll int

	// Bidding state
	bidRound      int // 1 or 2
	currentBidder int

	// Cards
	hands        []*Hand
	currentTrick *Trick
	tricksWon    []int // Tricks won by each player

	// History
	trickHistory []TrickResult
}

// NewRound creates a new round with the given dealer using the default rules.
func NewRound(numPlayers, dealer int) *Round {
	return NewRoundWithRules(numPlayers, dealer, DefaultRules())
}

// NewRoundWithRules creates a new round with the given dealer and rule configuration.
func NewRoundWithRules(numPlayers, dealer int, rules Rules) *Round {
	r := &Round{
		numPlayers:      numPlayers,
		dealer:          dealer,
		rules:           rules,
		phase:           PhaseDeal,
		trump:           NoSuit,
		maker:           -1,
		makerTeam:       -1,
		aloneDefender:   -1,
		defendAlonePoll: -1,
		hands:           make([]*Hand, numPlayers),
		tricksWon:       make([]int, numPlayers),
		trickHistory:    make([]TrickResult, 0, 5),
	}

	for i := 0; i < numPlayers; i++ {
		r.hands[i] = NewHand()
	}

	return r
}

// IsMisdeal returns true if the round ended as a throw-in (all passed in round 2
// with stick-the-dealer off). A misdeal is not scored and does not rotate the dealer.
func (r *Round) IsMisdeal() bool {
	return r.misdeal
}

// dealPasses returns the two dealing passes (cards per deal position, left of
// dealer first ... dealer last) for a deal of numPlayers.
//
// The authentic 4-player Euchre deal goes out in two passes with alternating
// packet sizes ({2,3,2,3} then the complement {3,2,3,2}) that only sum to 5 per
// player when there are exactly 4 players. For any other player count we fall
// back to a simple uniform deal (3 then 2) so each player still receives 5.
func dealPasses(numPlayers int) (first, second []int) {
	first = make([]int, numPlayers)
	second = make([]int, numPlayers)
	if numPlayers == 4 {
		copy(first, []int{2, 3, 2, 3})
		copy(second, []int{3, 2, 3, 2})
		return first, second
	}
	for i := 0; i < numPlayers; i++ {
		first[i] = 3
		second[i] = 2
	}
	return first, second
}

// Deal deals cards from the deck to all players
func (r *Round) Deal(deck *Deck) {
	deck.Shuffle()

	// Deal ORDER is preserved: left of dealer first ... dealer last.
	firstPass, secondPass := dealPasses(r.numPlayers)

	for i := 0; i < r.numPlayers; i++ {
		playerIdx := NextPlayer(r.dealer+i, r.numPlayers)
		r.hands[playerIdx].AddAll(deck.DrawN(firstPass[i]))
	}
	for i := 0; i < r.numPlayers; i++ {
		playerIdx := NextPlayer(r.dealer+i, r.numPlayers)
		r.hands[playerIdx].AddAll(deck.DrawN(secondPass[i]))
	}

	// Turn up the next card
	turnedCard, ok := deck.Draw()
	if ok {
		r.turnedCard = turnedCard
	}

	r.phase = PhaseBidRound1
	r.bidRound = 1
	r.currentBidder = NextPlayer(r.dealer, r.numPlayers)
}

// Phase returns the current game phase
func (r *Round) Phase() GamePhase {
	return r.phase
}

// Dealer returns the dealer's player index
func (r *Round) Dealer() int {
	return r.dealer
}

// Trump returns the trump suit (NoSuit if not yet determined)
func (r *Round) Trump() Suit {
	return r.trump
}

// TurnedCard returns the card turned up for trump selection
func (r *Round) TurnedCard() Card {
	return r.turnedCard
}

// Maker returns the player who called trump (-1 if none)
func (r *Round) Maker() int {
	return r.maker
}

// MakerTeam returns the team that called trump (-1 if none)
func (r *Round) MakerTeam() int {
	return r.makerTeam
}

// IsAlone returns true if the maker is playing alone
func (r *Round) IsAlone() bool {
	return r.alone
}

// CurrentPlayer returns whose turn it is
func (r *Round) CurrentPlayer() int {
	switch r.phase {
	case PhaseBidRound1, PhaseBidRound2:
		return r.currentBidder
	case PhaseDiscard:
		return r.dealer
	case PhaseDefendAlone:
		return r.defendAlonePoll
	case PhasePlay:
		if r.currentTrick == nil || r.currentTrick.Size() == 0 {
			// First trick: left of dealer leads
			// Subsequent tricks: winner of previous trick leads
			if len(r.trickHistory) == 0 {
				return r.findFirstLeader()
			}
			return r.trickHistory[len(r.trickHistory)-1].Winner
		}
		return r.nextToPlay()
	default:
		return -1
	}
}

// findFirstLeader finds who leads the first trick
// Normally left of dealer, but skips sitting-out partners in alone hands
func (r *Round) findFirstLeader() int {
	leader := NextPlayer(r.dealer, r.numPlayers)
	for i := 0; i < r.numPlayers; i++ {
		if !r.isSittingOut(leader) {
			return leader
		}
		leader = NextPlayer(leader, r.numPlayers)
	}
	return NextPlayer(r.dealer, r.numPlayers)
}

// nextToPlay returns the next player to play a card in the current trick
func (r *Round) nextToPlay() int {
	if r.currentTrick == nil {
		return -1
	}

	// Start from the leader and find the next player who hasn't played
	leader := r.currentTrick.Leader()
	if leader < 0 {
		return r.findFirstLeader()
	}

	playedPlayers := make(map[int]bool)
	for _, pc := range r.currentTrick.Cards() {
		playedPlayers[pc.Player] = true
	}

	current := leader
	for i := 0; i < r.numPlayers; i++ {
		if !playedPlayers[current] && !r.isSittingOut(current) {
			return current
		}
		current = NextPlayer(current, r.numPlayers)
	}

	return -1
}

// isSittingOut returns true if the player is sitting out: the lone maker's
// partner, and (if a defender declared defend-alone) the lone defender's partner.
func (r *Round) isSittingOut(p int) bool {
	if r.alone && p == Partner(r.maker) {
		return true
	}
	if r.aloneDefender >= 0 && p == Partner(r.aloneDefender) {
		return true
	}
	return false
}

// sittingOutCount returns how many players are sitting out this round.
func (r *Round) sittingOutCount() int {
	count := 0
	for p := 0; p < r.numPlayers; p++ {
		if r.isSittingOut(p) {
			count++
		}
	}
	return count
}

// Hand returns a copy of the specified player's hand
func (r *Round) Hand(playerIdx int) []Card {
	if playerIdx < 0 || playerIdx >= len(r.hands) {
		return nil
	}
	return r.hands[playerIdx].Cards()
}

// TricksWon returns how many tricks a player has won
func (r *Round) TricksWon(playerIdx int) int {
	if playerIdx < 0 || playerIdx >= len(r.tricksWon) {
		return 0
	}
	return r.tricksWon[playerIdx]
}

// TeamTricksWon returns how many tricks a team has won
func (r *Round) TeamTricksWon(team int) int {
	total := 0
	for i := 0; i < r.numPlayers; i++ {
		if Team(i) == team {
			total += r.tricksWon[i]
		}
	}
	return total
}

// CurrentTrick returns the cards played in the current trick
func (r *Round) CurrentTrick() []PlayedCard {
	if r.currentTrick == nil {
		return nil
	}
	return r.currentTrick.Cards()
}

// Trick returns the current trick pointer for legal play calculation
func (r *Round) Trick() *Trick {
	return r.currentTrick
}

// ApplyAction applies a player action to the round
func (r *Round) ApplyAction(action Action) error {
	switch a := action.(type) {
	case PassAction:
		return r.handlePass(a)
	case OrderUpAction:
		return r.handleOrderUp(a)
	case CallTrumpAction:
		return r.handleCallTrump(a)
	case DefendAloneAction:
		return r.handleDefendAlone(a)
	case DiscardAction:
		return r.handleDiscard(a)
	case PlayCardAction:
		return r.handlePlayCard(a)
	default:
		return PlayError("unknown action type")
	}
}

func (r *Round) handlePass(action PassAction) error {
	// During the defend-alone declaration window a passing defender declines and
	// we advance to the next defender (or start play if none remain).
	if r.phase == PhaseDefendAlone {
		if action.PlayerIdx != r.defendAlonePoll {
			return ErrNotYourTurn
		}
		next := r.nextDefenderToPoll(r.defendAlonePoll)
		if next < 0 {
			// All defenders passed: no lone defender, proceed to play.
			r.startPlay()
		} else {
			r.defendAlonePoll = next
		}
		return nil
	}

	if r.phase != PhaseBidRound1 && r.phase != PhaseBidRound2 {
		return PlayError("cannot pass in this phase")
	}
	if action.PlayerIdx != r.currentBidder {
		return ErrNotYourTurn
	}

	// Stick-the-dealer: the dealer may not pass in round 2 and must name trump.
	if r.phase == PhaseBidRound2 && r.rules.StickTheDealer && r.currentBidder == r.dealer {
		return PlayError("stick-the-dealer: dealer must call trump and cannot pass")
	}

	// Move to next bidder
	r.currentBidder = NextPlayer(r.currentBidder, r.numPlayers)

	// Check if round of bidding is complete
	if r.currentBidder == NextPlayer(r.dealer, r.numPlayers) {
		if r.bidRound == 1 {
			// Move to round 2
			r.bidRound = 2
			r.phase = PhaseBidRound2
		} else {
			// All passed in round 2. Stick-the-dealer is handled earlier (the
			// dealer cannot reach this all-pass branch because they may not pass),
			// so here we resolve via the misdeal rule. AllowMisdeal gates a classic
			// throw-in: re-deal with the same dealer, no score.
			//
			// Defensive fallback: if AllowMisdeal is somehow false here while
			// stick-the-dealer is also off (a misconfiguration), bidding would
			// otherwise dead-end with no way to resolve the round. We still fall
			// back to a misdeal so the round can end. For a valid game exactly one
			// of StickTheDealer / AllowMisdeal resolves an all-pass round 2.
			if r.rules.AllowMisdeal || !r.rules.StickTheDealer {
				r.misdeal = true
			}
			r.phase = PhaseRoundEnd
		}
	}

	return nil
}

func (r *Round) handleOrderUp(action OrderUpAction) error {
	if r.phase != PhaseBidRound1 {
		return PlayError("can only order up in round 1")
	}
	if action.PlayerIdx != r.currentBidder {
		return ErrNotYourTurn
	}

	// Set trump to the turned card's suit
	r.trump = r.turnedCard.Suit
	r.maker = action.PlayerIdx
	r.makerTeam = Team(action.PlayerIdx)
	r.alone = action.Alone

	// Dealer picks up the turned card
	r.hands[r.dealer].Add(r.turnedCard)

	// Move to discard phase (dealer must discard)
	r.phase = PhaseDiscard

	return nil
}

func (r *Round) handleCallTrump(action CallTrumpAction) error {
	if r.phase != PhaseBidRound2 {
		return PlayError("can only call trump in round 2")
	}
	if action.PlayerIdx != r.currentBidder {
		return ErrNotYourTurn
	}
	if action.Suit == r.turnedCard.Suit {
		return PlayError("cannot call the turned suit in round 2")
	}

	r.trump = action.Suit
	r.maker = action.PlayerIdx
	r.makerTeam = Team(action.PlayerIdx)
	r.alone = action.Alone

	// Sanity check: everyone should have exactly 5 cards before play starts
	for i := 0; i < r.numPlayers; i++ {
		if r.hands[i].Size() != 5 {
			return PlayError("all players must have 5 cards before play phase")
		}
	}

	// No discard phase in round 2. Either open the defend-alone declaration
	// window (lone maker + rule on) or go straight to play.
	r.beginPostTrump()

	return nil
}

// beginPostTrump runs after trump is finalized (round-2 call, or round-1 order-up
// once the dealer has discarded). If the maker is going alone and the
// defend-alone rule is on, it opens the defend-alone declaration window
// (PhaseDefendAlone). Otherwise it starts play immediately.
//
// NOTE: AI and TUI support for PhaseDefendAlone is a follow-up; the standard
// variant defaults AllowDefendAlone to false, so the default game path never
// enters this phase and the app/AI are unaffected.
func (r *Round) beginPostTrump() {
	if r.rules.AllowDefendAlone && r.alone {
		r.phase = PhaseDefendAlone
		r.defendAlonePoll = r.firstDefenderToPoll()
		// If somehow no eligible defender exists, fall through to play.
		if r.defendAlonePoll < 0 {
			r.startPlay()
		}
		return
	}
	r.startPlay()
}

// pollOrder returns the seat order in which defenders are polled: left of the
// dealer first ... dealer last (offsets 1, 2, ..., numPlayers-1, 0).
func (r *Round) pollOrder() []int {
	order := make([]int, r.numPlayers)
	for i := 0; i < r.numPlayers; i++ {
		order[i] = (r.dealer + 1 + i) % r.numPlayers
	}
	return order
}

// firstDefenderToPoll returns the first eligible defender (not on the maker
// team, not already sitting out) in poll order, or -1 if none.
func (r *Round) firstDefenderToPoll() int {
	for _, p := range r.pollOrder() {
		if Team(p) != r.makerTeam && !r.isSittingOut(p) {
			return p
		}
	}
	return -1
}

// nextDefenderToPoll returns the next eligible defender to poll after `from`, or
// -1 if there are no more. Each seat is visited exactly once per round, so
// polling terminates rather than wrapping around forever.
func (r *Round) nextDefenderToPoll(from int) int {
	order := r.pollOrder()
	seen := false
	for _, p := range order {
		if seen && Team(p) != r.makerTeam && !r.isSittingOut(p) {
			return p
		}
		if p == from {
			seen = true
		}
	}
	return -1
}

// startPlay transitions into the play phase and starts the opening trick. This
// is the single place the play trick is created on the bidding->play transition.
func (r *Round) startPlay() {
	r.defendAlonePoll = -1
	r.phase = PhasePlay
	r.currentTrick = NewTrick(r.trump)
}

// handleDefendAlone lets a defender declare they will defend alone for 4 points.
// Only legal during the pre-lead defend-alone declaration window, when the maker
// is going alone, the declaring player is the defender currently being polled
// (on the defending team), the rule is enabled, and no defender has already
// declared. Declaring sits out the defender's partner and starts play.
func (r *Round) handleDefendAlone(action DefendAloneAction) error {
	if !r.rules.AllowDefendAlone {
		return PlayError("defend-alone is not allowed by the current rules")
	}
	if r.phase != PhaseDefendAlone {
		return PlayError("defend-alone can only be declared during the defend-alone window")
	}
	if !r.alone {
		return PlayError("defend-alone is only allowed when the maker is going alone")
	}
	if r.aloneDefender >= 0 {
		return PlayError("a defender has already declared defend-alone")
	}
	if Team(action.PlayerIdx) == r.makerTeam {
		return PlayError("only a defender may declare defend-alone")
	}
	if action.PlayerIdx != r.defendAlonePoll {
		return ErrNotYourTurn
	}

	r.aloneDefender = action.PlayerIdx
	// One defender declaring ends the window; proceed to play.
	r.startPlay()
	return nil
}

func (r *Round) handleDiscard(action DiscardAction) error {
	if r.phase != PhaseDiscard {
		return PlayError("not in discard phase")
	}
	if action.PlayerIdx != r.dealer {
		return PlayError("only dealer can discard")
	}
	if !r.hands[r.dealer].Contains(action.Card) {
		return ErrCardNotInHand
	}

	// Verify the removal actually works
	beforeSize := r.hands[r.dealer].Size()
	removed := r.hands[r.dealer].Remove(action.Card)
	afterSize := r.hands[r.dealer].Size()

	if !removed {
		return PlayError("failed to remove card from hand")
	}
	if afterSize != beforeSize-1 {
		return PlayError("hand size did not decrease after discard")
	}
	if afterSize != 5 {
		return PlayError("dealer should have exactly 5 cards after discard")
	}

	// Sanity check: ALL players should have exactly 5 cards before play starts
	for i := 0; i < r.numPlayers; i++ {
		if r.hands[i].Size() != 5 {
			return PlayError("all players must have 5 cards before play phase")
		}
	}

	r.beginPostTrump()

	return nil
}

func (r *Round) handlePlayCard(action PlayCardAction) error {
	if r.phase != PhasePlay {
		return PlayError("not in play phase")
	}

	current := r.CurrentPlayer()
	if action.PlayerIdx != current {
		return ErrNotYourTurn
	}

	hand := r.hands[action.PlayerIdx]

	// Sanity check: no player should have more than 5 cards during play
	if hand.Size() > 5 {
		return PlayError("player has more than 5 cards - this is a bug")
	}

	if err := ValidatePlay(hand, action.Card, r.currentTrick); err != nil {
		return err
	}

	// Play the card
	hand.Remove(action.Card)
	r.currentTrick.Play(action.PlayerIdx, action.Card)

	// Check if trick is complete. Count active players (those not sitting out):
	// the lone maker's partner and any lone defender's partner sit out.
	playersInTrick := r.numPlayers - r.sittingOutCount()

	if r.currentTrick.Size() >= playersInTrick {
		r.completeTrick()
	}

	return nil
}

func (r *Round) completeTrick() {
	result := r.currentTrick.Result()
	if result.Winner >= 0 && result.Winner < len(r.tricksWon) {
		r.tricksWon[result.Winner]++
	}
	r.trickHistory = append(r.trickHistory, result)

	// Check if round is over (all 5 tricks played)
	if len(r.trickHistory) >= 5 {
		r.phase = PhaseRoundEnd
		return
	}

	// Start new trick with winner leading
	r.currentTrick = NewTrick(r.trump)
}

// Result returns the round result (only valid in PhaseRoundEnd)
func (r *Round) Result() RoundResult {
	makerTricks := r.TeamTricksWon(r.makerTeam)

	result := RoundResult{
		Makers:      r.makerTeam,
		MakerTricks: makerTricks,
		WasAlone:    r.alone,
		WasEuchred:  makerTricks < 3,
	}

	if result.WasEuchred {
		// Defenders score 2 points for a euchre, or 4 if a defender went alone.
		result.DefendPoints = 2
		if r.aloneDefender >= 0 {
			result.WasDefendedAlone = true
			result.DefendPoints = 4
		}
	} else if makerTricks == 5 {
		// March (all 5 tricks)
		if r.alone {
			result.MakerPoints = 4
		} else {
			result.MakerPoints = 2
		}
	} else {
		// Made it (3 or 4 tricks)
		result.MakerPoints = 1
	}

	// Handle case where no trump was called (misdeal)
	if r.maker < 0 {
		result.MakerPoints = 0
		result.DefendPoints = 0
	}

	return result
}

// IsComplete returns true if the round is over
func (r *Round) IsComplete() bool {
	return r.phase == PhaseRoundEnd || r.phase == PhaseGameEnd
}

// LegalActions returns all legal actions for the current player
func (r *Round) LegalActions() []Action {
	player := r.CurrentPlayer()
	if player < 0 {
		return nil
	}

	var actions []Action

	switch r.phase {
	case PhaseBidRound1:
		actions = append(actions, PassAction{PlayerIdx: player})
		actions = append(actions, OrderUpAction{PlayerIdx: player, Alone: false})
		actions = append(actions, OrderUpAction{PlayerIdx: player, Alone: true})

	case PhaseBidRound2:
		// Under stick-the-dealer the dealer cannot pass; they must name a suit.
		if !r.rules.StickTheDealer || player != r.dealer {
			actions = append(actions, PassAction{PlayerIdx: player})
		}
		// Can call any suit except the turned card's suit
		for _, suit := range []Suit{Clubs, Diamonds, Hearts, Spades} {
			if suit != r.turnedCard.Suit {
				actions = append(actions, CallTrumpAction{PlayerIdx: player, Suit: suit, Alone: false})
				actions = append(actions, CallTrumpAction{PlayerIdx: player, Suit: suit, Alone: true})
			}
		}

	case PhaseDiscard:
		// Can discard any card in hand
		for _, card := range r.hands[r.dealer].Cards() {
			actions = append(actions, DiscardAction{PlayerIdx: player, Card: card})
		}

	case PhaseDefendAlone:
		// The polled defender may declare defend-alone or pass.
		actions = append(actions, DefendAloneAction{PlayerIdx: player})
		actions = append(actions, PassAction{PlayerIdx: player})

	case PhasePlay:
		// Can play any legal card
		for _, card := range LegalPlays(r.hands[player], r.currentTrick) {
			actions = append(actions, PlayCardAction{PlayerIdx: player, Card: card})
		}
	}

	return actions
}

// BidRound returns which bidding round we're in (1 or 2)
func (r *Round) BidRound() int {
	return r.bidRound
}

// TrickHistory returns all completed tricks
func (r *Round) TrickHistory() []TrickResult {
	result := make([]TrickResult, len(r.trickHistory))
	copy(result, r.trickHistory)
	return result
}
