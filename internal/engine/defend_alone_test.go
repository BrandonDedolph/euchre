package engine

import "testing"

// setupAloneMakerToPlay sets up a round in PhasePlay where the maker (player 1)
// has gone alone with Hearts trump. Hands are assigned deterministically so the
// makers can be euchred. Returns the round ready for the first play.
func setupAloneMakerToPlay(t *testing.T, rules Rules) *Round {
	t.Helper()
	r := NewRoundWithRules(4, 0, rules)

	// Deterministic hands. Hearts is trump.
	// Player 1 (maker, alone) gets weak cards so the defenders can euchre.
	r.hands[0] = NewHandWith([]Card{
		{Spades, Nine}, {Spades, Ten}, {Spades, Queen}, {Spades, King}, {Spades, Ace},
	})
	r.hands[1] = NewHandWith([]Card{
		{Hearts, Nine}, {Hearts, Ten}, {Clubs, Nine}, {Clubs, Ten}, {Clubs, Queen},
	})
	r.hands[2] = NewHandWith([]Card{
		{Diamonds, Nine}, {Diamonds, Ten}, {Diamonds, Queen}, {Diamonds, King}, {Diamonds, Ace},
	})
	r.hands[3] = NewHandWith([]Card{
		{Hearts, Jack}, {Diamonds, Jack}, {Hearts, Ace}, {Hearts, King}, {Hearts, Queen},
	})

	r.trump = Hearts
	r.maker = 1
	r.makerTeam = Team(1)
	r.alone = true
	r.bidRound = 1
	r.phase = PhasePlay
	r.currentTrick = NewTrick(r.trump)
	return r
}

// setupAloneMakerToDefendAlone sets up the same alone-maker scenario but pauses
// in PhaseDefendAlone with the first eligible defender being polled.
func setupAloneMakerToDefendAlone(t *testing.T, rules Rules) *Round {
	t.Helper()
	r := setupAloneMakerToPlay(t, rules)
	r.phase = PhaseDefendAlone
	r.currentTrick = nil
	r.defendAlonePoll = r.firstDefenderToPoll()
	return r
}

// callTrumpAloneRound1 walks a fresh round through round-1 bidding so that
// player 1 orders up alone, then the dealer (0) discards. Returns after the
// dealer's discard so the round is poised at the post-trump transition.
func orderUpAloneAndDiscard(t *testing.T, rules Rules) *Round {
	t.Helper()
	r := NewRoundWithRules(4, 0, rules)
	deck := NewStandardDeck()
	deck.Seed(7)
	r.Deal(deck)

	// Player 1 (left of dealer) orders up alone.
	if r.CurrentPlayer() != 1 {
		t.Fatalf("expected player 1 to act first, got %d", r.CurrentPlayer())
	}
	if err := r.ApplyAction(OrderUpAction{PlayerIdx: 1, Alone: true}); err != nil {
		t.Fatalf("order up alone failed: %v", err)
	}
	if r.Phase() != PhaseDiscard {
		t.Fatalf("expected discard phase, got %s", r.Phase())
	}
	// Dealer discards one card.
	discard := r.Hand(0)[0]
	if err := r.ApplyAction(DiscardAction{PlayerIdx: 0, Card: discard}); err != nil {
		t.Fatalf("discard failed: %v", err)
	}
	return r
}

func TestDefendAlonePhaseEnteredAfterDiscard(t *testing.T) {
	r := orderUpAloneAndDiscard(t, Rules{AllowDefendAlone: true, AllowMisdeal: true})

	if r.Phase() != PhaseDefendAlone {
		t.Fatalf("with AllowDefendAlone and a lone maker, should enter PhaseDefendAlone after discard, got %s", r.Phase())
	}
	// The player being polled must be a defender (not on the maker team).
	cur := r.CurrentPlayer()
	if Team(cur) == r.MakerTeam() {
		t.Fatalf("defend-alone phase should poll a defender, got player %d on maker team", cur)
	}
	// First polled defender is left of dealer among defenders: maker is player 1,
	// so defenders are 0 and 2. Polling starts left of dealer (player 1) in seat
	// order, so the first defender encountered is player 2.
	if cur != 2 {
		t.Fatalf("expected first polled defender to be 2, got %d", cur)
	}
}

func TestDefendAloneDeclaredViaLegalActions(t *testing.T) {
	r := orderUpAloneAndDiscard(t, Rules{AllowDefendAlone: true, AllowMisdeal: true})

	// The polled defender must be offered both DefendAlone and Pass.
	cur := r.CurrentPlayer()
	var declare Action
	hasPass := false
	for _, a := range r.LegalActions() {
		switch a.Type() {
		case ActionDefendAlone:
			declare = a
		case ActionPass:
			hasPass = true
		}
	}
	if declare == nil {
		t.Fatal("polled defender should be offered a defend-alone action")
	}
	if !hasPass {
		t.Fatal("polled defender should be offered a pass action")
	}

	// Declaring via the offered action should set the lone defender, sit out the
	// partner, and transition to play.
	if err := r.ApplyAction(declare); err != nil {
		t.Fatalf("declaring defend-alone via LegalActions failed: %v", err)
	}
	if r.Phase() != PhasePlay {
		t.Fatalf("declaring should transition to play, got %s", r.Phase())
	}
	if !r.isSittingOut(Partner(cur)) {
		t.Errorf("lone defender's partner (%d) should sit out", Partner(cur))
	}
	if r.currentTrick == nil {
		t.Fatal("play trick should be started on transition to play")
	}
}

func TestDefendAloneAllPassProceedsToPlay(t *testing.T) {
	r := orderUpAloneAndDiscard(t, Rules{AllowDefendAlone: true, AllowMisdeal: true})

	// All defenders pass during the declaration window.
	guard := 0
	for r.Phase() == PhaseDefendAlone {
		p := r.CurrentPlayer()
		if err := r.ApplyAction(PassAction{PlayerIdx: p}); err != nil {
			t.Fatalf("defender pass failed: %v", err)
		}
		guard++
		if guard > 4 {
			t.Fatal("defend-alone polling did not terminate")
		}
	}
	if r.Phase() != PhasePlay {
		t.Fatalf("all defenders passing should proceed to play, got %s", r.Phase())
	}
	if r.aloneDefender != -1 {
		t.Errorf("no lone defender should be set when all pass, got %d", r.aloneDefender)
	}
	if r.currentTrick == nil {
		t.Fatal("play trick should be started after all defenders pass")
	}
}

func TestDefendAloneDefaultPathSkipsPhase(t *testing.T) {
	// Rule off: a lone maker should go straight from discard to play with no
	// PhaseDefendAlone, leaving the default app/AI path unchanged.
	r := orderUpAloneAndDiscard(t, Rules{AllowDefendAlone: false, AllowMisdeal: true})
	if r.Phase() != PhasePlay {
		t.Fatalf("with defend-alone off, should go straight to play, got %s", r.Phase())
	}
	if r.currentTrick == nil {
		t.Fatal("play trick should be started")
	}
}

func TestDefendAloneNotEnteredWhenMakerNotAlone(t *testing.T) {
	// Maker not alone: even with the rule on, no defend-alone phase.
	r := NewRoundWithRules(4, 0, Rules{AllowDefendAlone: true, AllowMisdeal: true})
	deck := NewStandardDeck()
	deck.Seed(7)
	r.Deal(deck)
	if err := r.ApplyAction(OrderUpAction{PlayerIdx: 1, Alone: false}); err != nil {
		t.Fatalf("order up failed: %v", err)
	}
	discard := r.Hand(0)[0]
	if err := r.ApplyAction(DiscardAction{PlayerIdx: 0, Card: discard}); err != nil {
		t.Fatalf("discard failed: %v", err)
	}
	if r.Phase() != PhasePlay {
		t.Fatalf("non-alone maker should go straight to play, got %s", r.Phase())
	}
}

func TestDefendAloneOfferedInLegalActions(t *testing.T) {
	r := setupAloneMakerToDefendAlone(t, Rules{AllowDefendAlone: true})

	// The polled defender (a non-maker-team player) must be offered defend-alone.
	polled := r.CurrentPlayer()
	if Team(polled) == r.makerTeam {
		t.Fatalf("defend-alone should poll a defender, got maker-team player %d", polled)
	}
	found := false
	for _, a := range r.LegalActions() {
		if a.Type() == ActionDefendAlone {
			found = true
		}
	}
	if !found {
		t.Fatal("polled defender should be offered defend-alone")
	}
}

func TestDefendAloneSetsLoneDefenderAndSitsOutPartner(t *testing.T) {
	r := setupAloneMakerToDefendAlone(t, Rules{AllowDefendAlone: true})

	// Declare defend-alone for the currently polled defender.
	defender := r.CurrentPlayer()
	if err := r.ApplyAction(DefendAloneAction{PlayerIdx: defender}); err != nil {
		t.Fatalf("defend-alone declaration should succeed: %v", err)
	}

	// Lone defender's partner should now be sitting out.
	if !r.isSittingOut(Partner(defender)) {
		t.Errorf("lone defender's partner (%d) should be sitting out", Partner(defender))
	}
	// Maker's partner (player 3) still sits out too.
	if !r.isSittingOut(Partner(1)) {
		t.Errorf("maker's partner (%d) should be sitting out", Partner(1))
	}

	// A second declaration must be rejected (window has closed; now in play).
	if err := r.ApplyAction(DefendAloneAction{PlayerIdx: 2}); err == nil {
		t.Error("a second defend-alone declaration should be rejected")
	}
}

func TestDefendAloneTrickCompletesWithTwoActivePlayers(t *testing.T) {
	r := setupAloneMakerToDefendAlone(t, Rules{AllowDefendAlone: true})

	// Polled defender defends alone. Now only that defender and the maker (1) are active.
	defender := r.CurrentPlayer()
	if err := r.ApplyAction(DefendAloneAction{PlayerIdx: defender}); err != nil {
		t.Fatalf("defend-alone failed: %v", err)
	}
	if r.sittingOutCount() != 2 {
		t.Fatalf("expected 2 players sitting out, got %d", r.sittingOutCount())
	}

	// Play one trick: leader plays, then the single other active player plays.
	first := r.CurrentPlayer()
	firstCard := LegalPlays(r.hands[first], r.currentTrick)[0]
	if err := r.ApplyAction(PlayCardAction{PlayerIdx: first, Card: firstCard}); err != nil {
		t.Fatalf("first play failed: %v", err)
	}
	second := r.CurrentPlayer()
	if second == first || r.isSittingOut(second) {
		t.Fatalf("second active player resolved incorrectly: %d", second)
	}
	secondCard := LegalPlays(r.hands[second], r.currentTrick)[0]
	if err := r.ApplyAction(PlayCardAction{PlayerIdx: second, Card: secondCard}); err != nil {
		t.Fatalf("second play failed: %v", err)
	}

	// Trick should now be complete (2 active players) -> one entry in history.
	if len(r.TrickHistory()) != 1 {
		t.Fatalf("trick should complete after 2 plays with 2 active players, history=%d", len(r.TrickHistory()))
	}
}

func TestDefendAloneEuchredScoresFourPoints(t *testing.T) {
	r := NewRoundWithRules(4, 0, Rules{AllowDefendAlone: true})

	// Maker (player 1) alone, Hearts trump, but euchred. We force the euchre by
	// directly setting trick counts and the lone-defender flag, then check Result.
	r.trump = Hearts
	r.maker = 1
	r.makerTeam = Team(1)
	r.alone = true
	r.aloneDefender = 0 // a defender declared defend-alone
	r.phase = PhaseRoundEnd

	// Makers (team 1) win only 1 trick -> euchred. Defenders win the other 4.
	r.tricksWon[1] = 1
	r.tricksWon[0] = 4

	result := r.Result()
	if !result.WasEuchred {
		t.Fatal("makers should be euchred")
	}
	if !result.WasDefendedAlone {
		t.Error("result should mark WasDefendedAlone")
	}
	if result.DefendPoints != 4 {
		t.Errorf("defend-alone euchre should be 4 points, got %d", result.DefendPoints)
	}
}

func TestDefendAloneNotOfferedAfterPlayStarts(t *testing.T) {
	r := setupAloneMakerToPlay(t, Rules{AllowDefendAlone: true})

	// Play one card to start the trick.
	p := r.CurrentPlayer()
	card := LegalPlays(r.hands[p], r.currentTrick)[0]
	if err := r.ApplyAction(PlayCardAction{PlayerIdx: p, Card: card}); err != nil {
		t.Fatalf("play failed: %v", err)
	}
	// Now defend-alone must not be offered.
	for _, a := range r.LegalActions() {
		if a.Type() == ActionDefendAlone {
			t.Fatal("defend-alone must not be offered once a card has been played")
		}
	}
}

func TestDefendAloneDisabledByRules(t *testing.T) {
	r := setupAloneMakerToPlay(t, Rules{AllowDefendAlone: false})
	for _, a := range r.LegalActions() {
		if a.Type() == ActionDefendAlone {
			t.Fatal("defend-alone must not be offered when rule is disabled")
		}
	}
	if err := r.ApplyAction(DefendAloneAction{PlayerIdx: 0}); err == nil {
		t.Error("defend-alone should be rejected when rule is disabled")
	}
}
