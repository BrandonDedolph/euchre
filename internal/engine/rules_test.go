package engine

import "testing"

// passToDealerInRound2 advances bidding so that the dealer is the player to act
// in round 2. Players 1, 2, 3 pass round 1, then 1, 2, 3 pass round 2, leaving
// the dealer (0) to act.
func passToDealerInRound2(t *testing.T, round *Round) {
	t.Helper()
	// Round 1: everyone passes (4 passes -> round 2)
	for i := 0; i < 4; i++ {
		p := round.CurrentPlayer()
		if err := round.ApplyAction(PassAction{PlayerIdx: p}); err != nil {
			t.Fatalf("round 1 pass by %d failed: %v", p, err)
		}
	}
	if round.Phase() != PhaseBidRound2 {
		t.Fatalf("expected round 2, got %s", round.Phase())
	}
	// Round 2: players 1, 2, 3 pass, leaving dealer (0) to act
	for i := 0; i < 3; i++ {
		p := round.CurrentPlayer()
		if err := round.ApplyAction(PassAction{PlayerIdx: p}); err != nil {
			t.Fatalf("round 2 pass by %d failed: %v", p, err)
		}
	}
	if round.CurrentPlayer() != round.Dealer() {
		t.Fatalf("expected dealer (%d) to act, got %d", round.Dealer(), round.CurrentPlayer())
	}
}

func TestStickTheDealerNoPassAction(t *testing.T) {
	round := NewRoundWithRules(4, 0, Rules{StickTheDealer: true})
	deck := NewStandardDeck()
	deck.Seed(1)
	round.Deal(deck)

	passToDealerInRound2(t, round)

	// Dealer's legal actions must NOT include a pass
	for _, a := range round.LegalActions() {
		if a.Type() == ActionPass {
			t.Fatal("stick-the-dealer: dealer should not have a pass action in round 2")
		}
	}

	// Dealer attempting to pass should error
	if err := round.ApplyAction(PassAction{PlayerIdx: round.Dealer()}); err == nil {
		t.Fatal("stick-the-dealer: dealer pass should return an error")
	}
}

func TestNonDealerCanStillPassUnderStickTheDealer(t *testing.T) {
	round := NewRoundWithRules(4, 0, Rules{StickTheDealer: true})
	deck := NewStandardDeck()
	deck.Seed(1)
	round.Deal(deck)

	// Round 1 all pass
	for i := 0; i < 4; i++ {
		round.ApplyAction(PassAction{PlayerIdx: round.CurrentPlayer()})
	}
	// Player 1 (non-dealer) should be able to pass in round 2
	hasPass := false
	for _, a := range round.LegalActions() {
		if a.Type() == ActionPass {
			hasPass = true
		}
	}
	if !hasPass {
		t.Fatal("non-dealer should still be able to pass under stick-the-dealer")
	}
}

func TestAllPassRound2IsMisdeal(t *testing.T) {
	round := NewRoundWithRules(4, 0, Rules{AllowMisdeal: true})
	deck := NewStandardDeck()
	deck.Seed(1)
	round.Deal(deck)

	passToDealerInRound2(t, round)
	// Dealer passes too -> all passed round 2
	if err := round.ApplyAction(PassAction{PlayerIdx: round.Dealer()}); err != nil {
		t.Fatalf("dealer pass should succeed when stick-the-dealer is off: %v", err)
	}

	if !round.IsMisdeal() {
		t.Fatal("all-pass in round 2 should be a misdeal")
	}
	if round.Phase() != PhaseRoundEnd {
		t.Fatalf("misdeal should end the round, got %s", round.Phase())
	}
}

// allPassRound2 makes every player pass both bidding rounds, driving an all-pass
// round 2. Used to exercise the misdeal/stick-the-dealer resolution branch.
func allPassRound2(t *testing.T, round *Round) {
	t.Helper()
	for i := 0; i < 8; i++ {
		p := round.CurrentPlayer()
		if p < 0 {
			return
		}
		if err := round.ApplyAction(PassAction{PlayerIdx: p}); err != nil {
			t.Fatalf("pass by %d failed: %v", p, err)
		}
	}
}

func TestMisdealBranchReadsAllowMisdeal(t *testing.T) {
	// With AllowMisdeal enabled (and stick-the-dealer off), an all-pass round 2
	// resolves to a misdeal that ends the round.
	round := NewRoundWithRules(4, 0, Rules{AllowMisdeal: true})
	deck := NewStandardDeck()
	deck.Seed(1)
	round.Deal(deck)

	allPassRound2(t, round)

	if !round.IsMisdeal() {
		t.Fatal("with AllowMisdeal true, all-pass round 2 should set the misdeal flag")
	}
	if round.Phase() != PhaseRoundEnd {
		t.Fatalf("misdeal should end the round, got %s", round.Phase())
	}
}

func TestMisdealFallbackWhenMisconfigured(t *testing.T) {
	// Defensive fallback: if BOTH StickTheDealer and AllowMisdeal are false (a
	// misconfiguration), an all-pass round 2 would otherwise dead-end bidding.
	// The engine must still fall back to a misdeal so the round can end.
	round := NewRoundWithRules(4, 0, Rules{StickTheDealer: false, AllowMisdeal: false})
	deck := NewStandardDeck()
	deck.Seed(1)
	round.Deal(deck)

	allPassRound2(t, round)

	if !round.IsMisdeal() {
		t.Fatal("misconfigured rules (both false) must fall back to a misdeal so bidding does not dead-end")
	}
	if round.Phase() != PhaseRoundEnd {
		t.Fatalf("fallback misdeal should end the round, got %s", round.Phase())
	}
}

func TestEndRoundMisdealDoesNotRotateOrScore(t *testing.T) {
	game := NewGame(GameConfig{
		NumPlayers:  4,
		TargetScore: 10,
		DeckConfig:  StandardDeckConfig{},
		Rules:       Rules{AllowMisdeal: true},
	})
	game.StartRound()
	round := game.Round()

	dealerBefore := game.Dealer()
	scoresBefore := game.Scores()

	// All players pass both rounds -> misdeal
	for i := 0; i < 4; i++ {
		game.ApplyAction(PassAction{PlayerIdx: game.CurrentPlayer()})
	}
	for i := 0; i < 4; i++ {
		game.ApplyAction(PassAction{PlayerIdx: game.CurrentPlayer()})
	}

	if !round.IsMisdeal() {
		t.Fatal("expected misdeal")
	}
	if game.Dealer() != dealerBefore {
		t.Errorf("misdeal must not rotate dealer: before %d, after %d", dealerBefore, game.Dealer())
	}
	scoresAfter := game.Scores()
	for i := range scoresBefore {
		if scoresBefore[i] != scoresAfter[i] {
			t.Errorf("misdeal must not change scores: team %d before %d after %d", i, scoresBefore[i], scoresAfter[i])
		}
	}
	if len(game.RoundHistory()) != 0 {
		t.Errorf("misdeal must not be appended to round history, got %d entries", len(game.RoundHistory()))
	}
}
