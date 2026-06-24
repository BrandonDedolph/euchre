package rule_based

import (
	"testing"

	"github.com/BrandonDedolph/euchre/internal/ai"
	"github.com/BrandonDedolph/euchre/internal/engine"
)

// TestDecideBid_StickTheDealer_AIDealerForcedToCall reproduces the original
// crash bug: with stick-the-dealer ON, an AI sitting as dealer in bidding round
// 2 with a weak hand used to PASS (because DecideBid hardcoded
// stickTheDealer=false). The engine rejects a dealer pass under stick-the-dealer,
// which the app turned into a fatal error. This test drives a real game to round
// 2 with an AI dealer and asserts DecideBid returns a legal trump CALL that the
// engine ACCEPTS (not a pass, and not the turned-down suit).
func TestDecideBid_StickTheDealer_AIDealerForcedToCall(t *testing.T) {
	config := engine.DefaultGameConfig()
	config.Rules = engine.Rules{StickTheDealer: true, AllowMisdeal: false}
	game := engine.NewGame(config)
	game.StartRound()

	state := engine.NewGameState(game)
	dealer := game.Dealer()

	// Round 1: everyone passes, starting with the player left of the dealer.
	first := engine.NextPlayer(dealer, game.NumPlayers())
	for player := first; ; {
		if err := game.ApplyAction(engine.PassAction{PlayerIdx: player}); err != nil {
			t.Fatalf("round 1 pass for player %d failed: %v", player, err)
		}
		player = engine.NextPlayer(player, game.NumPlayers())
		if player == first {
			break
		}
	}

	if game.Phase() != engine.PhaseBidRound2 {
		t.Fatalf("expected PhaseBidRound2 after all round-1 passes, got %v", game.Phase())
	}

	// Round 2: every non-dealer passes, leaving the dealer stuck.
	for player := first; player != dealer; player = engine.NextPlayer(player, game.NumPlayers()) {
		if err := game.ApplyAction(engine.PassAction{PlayerIdx: player}); err != nil {
			t.Fatalf("round 2 pass for player %d failed: %v", player, err)
		}
	}

	if game.CurrentPlayer() != dealer {
		t.Fatalf("expected dealer %d to be on the clock in round 2, got %d", dealer, game.CurrentPlayer())
	}

	// The AI dealer must now make a decision. Under the bug it would pass.
	aiDealer := New("Dealer", dealer, ai.DifficultyMedium)
	decision := aiDealer.DecideBid(state, 2)

	if decision.Pass {
		t.Fatal("AI dealer passed in round 2 under stick-the-dealer; engine would reject this and crash the game")
	}
	turnedSuit := state.TurnedCard().Suit
	if decision.CallSuit == turnedSuit {
		t.Errorf("AI dealer called the turned-down suit %v, which is illegal in round 2", turnedSuit)
	}
	if decision.CallSuit == engine.NoSuit {
		t.Fatal("AI dealer returned NoSuit as its call")
	}

	// The decision must be an action the engine ACCEPTS.
	action := engine.CallTrumpAction{
		PlayerIdx: dealer,
		Suit:      decision.CallSuit,
		Alone:     decision.Alone,
	}
	if err := game.ApplyAction(action); err != nil {
		t.Fatalf("engine rejected the AI dealer's forced call %v: %v", decision.CallSuit, err)
	}
}

// TestEvaluateRound2_StickTheDealer_AlwaysLegalCall checks the lower-level
// guarantee directly: a stuck dealer always gets a legal, non-excluded suit,
// even for a maximally weak hand.
func TestEvaluateRound2_StickTheDealer_AlwaysLegalCall(t *testing.T) {
	evaluator := NewBiddingEvaluator(55)

	hand := []engine.Card{
		{Suit: engine.Spades, Rank: engine.Nine},
		{Suit: engine.Spades, Rank: engine.Ten},
		{Suit: engine.Hearts, Rank: engine.Nine},
		{Suit: engine.Clubs, Rank: engine.Nine},
		{Suit: engine.Diamonds, Rank: engine.Nine},
	}

	excluded := engine.Hearts
	shouldBid, suit, _ := evaluator.EvaluateRound2(hand, excluded, true, true)
	if !shouldBid {
		t.Fatal("stuck dealer must bid under stick-the-dealer")
	}
	if suit == engine.NoSuit {
		t.Fatal("stuck dealer must name a real suit, got NoSuit")
	}
	if suit == excluded {
		t.Errorf("stuck dealer named the turned-down suit %v", excluded)
	}
}
