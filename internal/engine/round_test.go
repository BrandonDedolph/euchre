package engine

import "testing"

func TestNewRound(t *testing.T) {
	round := NewRound(4, 0)

	if round.Phase() != PhaseDeal {
		t.Errorf("New round should be in deal phase, got %s", round.Phase())
	}
	if round.Dealer() != 0 {
		t.Errorf("Dealer should be 0, got %d", round.Dealer())
	}
	if round.Trump() != NoSuit {
		t.Errorf("Trump should be NoSuit initially, got %s", round.Trump())
	}
	if round.Maker() != -1 {
		t.Errorf("Maker should be -1 initially, got %d", round.Maker())
	}
}

func TestRoundDeal(t *testing.T) {
	round := NewRound(4, 0)
	deck := NewStandardDeck()
	deck.Seed(1) // Deterministic shuffle for a stable assertion

	round.Deal(deck)

	// Each player should have 5 cards
	for i := 0; i < 4; i++ {
		hand := round.Hand(i)
		if len(hand) != 5 {
			t.Errorf("Player %d should have 5 cards, got %d", i, len(hand))
		}
	}

	// Should be in bid round 1
	if round.Phase() != PhaseBidRound1 {
		t.Errorf("Should be in bid round 1, got %s", round.Phase())
	}

	// Turned card should genuinely be set aside: it must not be in anyone's hand
	turnedCard := round.TurnedCard()
	for i := 0; i < 4; i++ {
		for _, c := range round.Hand(i) {
			if c == turnedCard {
				t.Errorf("Turned card %s should not be in player %d's hand", turnedCard, i)
			}
		}
	}

	// Current player should be left of dealer
	if round.CurrentPlayer() != 1 {
		t.Errorf("Current player should be 1 (left of dealer 0), got %d", round.CurrentPlayer())
	}
}

func TestDealPassesFourPlayerAlternates(t *testing.T) {
	// 4-player Euchre uses authentic alternating packets: {2,3,2,3} then the
	// complement {3,2,3,2}, summing to 5 per player.
	first, second := dealPasses(4)
	wantFirst := []int{2, 3, 2, 3}
	wantSecond := []int{3, 2, 3, 2}

	if len(first) != 4 || len(second) != 4 {
		t.Fatalf("4-player passes should have length 4, got %d and %d", len(first), len(second))
	}
	for i := 0; i < 4; i++ {
		if first[i] != wantFirst[i] || second[i] != wantSecond[i] {
			t.Fatalf("4-player packet mismatch at %d: got (%d,%d) want (%d,%d)",
				i, first[i], second[i], wantFirst[i], wantSecond[i])
		}
		if first[i]+second[i] != 5 {
			t.Errorf("player at deal position %d should receive 5 cards, got %d", i, first[i]+second[i])
		}
	}
}

func TestDealPassesNonFourPlayerFallsBackToUniform(t *testing.T) {
	// For non-4-player counts the alternating pattern does not sum to 5, so the
	// deal falls back to a simple uniform 3-then-2 split.
	for _, n := range []int{2, 3, 5, 6} {
		first, second := dealPasses(n)
		if len(first) != n || len(second) != n {
			t.Fatalf("n=%d: passes should have length %d, got %d and %d", n, n, len(first), len(second))
		}
		for i := 0; i < n; i++ {
			if first[i] != 3 || second[i] != 2 {
				t.Errorf("n=%d: uniform deal expected (3,2) at position %d, got (%d,%d)", n, i, first[i], second[i])
			}
			if first[i]+second[i] != 5 {
				t.Errorf("n=%d: player at position %d should receive 5 cards, got %d", n, i, first[i]+second[i])
			}
		}
	}
}

func TestRoundDealFourPlayerEachGetsFive(t *testing.T) {
	round := NewRound(4, 0)
	deck := NewStandardDeck()
	deck.Seed(7)
	round.Deal(deck)
	for i := 0; i < 4; i++ {
		if len(round.Hand(i)) != 5 {
			t.Errorf("player %d should have 5 cards, got %d", i, len(round.Hand(i)))
		}
	}
}

func TestRoundDealDistinctCards(t *testing.T) {
	round := NewRound(4, 0)
	deck := NewStandardDeck()
	deck.Seed(7)
	round.Deal(deck)

	seen := make(map[Card]bool)
	total := 0
	for i := 0; i < 4; i++ {
		hand := round.Hand(i)
		if len(hand) != 5 {
			t.Errorf("player %d should have exactly 5 cards, got %d", i, len(hand))
		}
		for _, c := range hand {
			if seen[c] {
				t.Errorf("duplicate card dealt: %s", c)
			}
			seen[c] = true
			total++
		}
	}
	// Add the turned card; all 21 must be distinct.
	tc := round.TurnedCard()
	if seen[tc] {
		t.Errorf("turned card %s duplicates a dealt card", tc)
	}
	seen[tc] = true

	if total != 20 {
		t.Errorf("expected 20 dealt cards, got %d", total)
	}
	if len(seen) != 21 {
		t.Errorf("expected 21 distinct cards (20 dealt + turned), got %d", len(seen))
	}
}

func TestRoundBidRound1Pass(t *testing.T) {
	round := NewRound(4, 0)
	deck := NewStandardDeck()
	round.Deal(deck)

	// All players pass in round 1
	for i := 0; i < 4; i++ {
		currentPlayer := round.CurrentPlayer()
		err := round.ApplyAction(PassAction{PlayerIdx: currentPlayer})
		if err != nil {
			t.Errorf("Player %d pass should succeed: %v", currentPlayer, err)
		}
	}

	// Should now be in round 2
	if round.Phase() != PhaseBidRound2 {
		t.Errorf("Should be in bid round 2, got %s", round.Phase())
	}
	if round.BidRound() != 2 {
		t.Errorf("Bid round should be 2, got %d", round.BidRound())
	}
}

func TestRoundOrderUp(t *testing.T) {
	round := NewRound(4, 0)
	deck := NewStandardDeck()
	round.Deal(deck)

	turnedCard := round.TurnedCard()
	currentPlayer := round.CurrentPlayer()

	err := round.ApplyAction(OrderUpAction{PlayerIdx: currentPlayer, Alone: false})
	if err != nil {
		t.Errorf("Order up should succeed: %v", err)
	}

	// Trump should be set to turned card's suit
	if round.Trump() != turnedCard.Suit {
		t.Errorf("Trump should be %s, got %s", turnedCard.Suit, round.Trump())
	}

	// Maker should be set
	if round.Maker() != currentPlayer {
		t.Errorf("Maker should be %d, got %d", currentPlayer, round.Maker())
	}

	// Should be in discard phase
	if round.Phase() != PhaseDiscard {
		t.Errorf("Should be in discard phase, got %s", round.Phase())
	}

	// Dealer should have 6 cards (picked up turned card)
	dealerHand := round.Hand(0)
	if len(dealerHand) != 6 {
		t.Errorf("Dealer should have 6 cards, got %d", len(dealerHand))
	}
}

func TestRoundDiscard(t *testing.T) {
	round := NewRound(4, 0)
	deck := NewStandardDeck()
	round.Deal(deck)

	// Order up
	round.ApplyAction(OrderUpAction{PlayerIdx: 1, Alone: false})

	// Dealer discards
	dealerHand := round.Hand(0)
	cardToDiscard := dealerHand[0]

	err := round.ApplyAction(DiscardAction{PlayerIdx: 0, Card: cardToDiscard})
	if err != nil {
		t.Errorf("Discard should succeed: %v", err)
	}

	// Dealer should have 5 cards again
	if len(round.Hand(0)) != 5 {
		t.Errorf("Dealer should have 5 cards after discard, got %d", len(round.Hand(0)))
	}

	// Should be in play phase
	if round.Phase() != PhasePlay {
		t.Errorf("Should be in play phase, got %s", round.Phase())
	}
}

func TestRoundCallTrump(t *testing.T) {
	round := NewRound(4, 0)
	deck := NewStandardDeck()
	round.Deal(deck)

	turnedSuit := round.TurnedCard().Suit

	// All pass round 1
	for i := 0; i < 4; i++ {
		round.ApplyAction(PassAction{PlayerIdx: round.CurrentPlayer()})
	}

	// Find a different suit to call
	callSuit := Hearts
	if callSuit == turnedSuit {
		callSuit = Spades
	}

	currentPlayer := round.CurrentPlayer()
	err := round.ApplyAction(CallTrumpAction{PlayerIdx: currentPlayer, Suit: callSuit, Alone: false})
	if err != nil {
		t.Errorf("Call trump should succeed: %v", err)
	}

	if round.Trump() != callSuit {
		t.Errorf("Trump should be %s, got %s", callSuit, round.Trump())
	}

	// Should be in play phase (no discard in round 2)
	if round.Phase() != PhasePlay {
		t.Errorf("Should be in play phase, got %s", round.Phase())
	}
}

func TestRoundCallTurnedSuitInvalid(t *testing.T) {
	round := NewRound(4, 0)
	deck := NewStandardDeck()
	round.Deal(deck)

	turnedSuit := round.TurnedCard().Suit

	// All pass round 1
	for i := 0; i < 4; i++ {
		round.ApplyAction(PassAction{PlayerIdx: round.CurrentPlayer()})
	}

	// Try to call the turned suit (should fail)
	currentPlayer := round.CurrentPlayer()
	err := round.ApplyAction(CallTrumpAction{PlayerIdx: currentPlayer, Suit: turnedSuit, Alone: false})
	if err == nil {
		t.Error("Calling turned suit in round 2 should fail")
	}
}

func TestRoundPlayCard(t *testing.T) {
	round := NewRound(4, 0)
	deck := NewStandardDeck()
	round.Deal(deck)

	// Set up game to play phase
	round.ApplyAction(OrderUpAction{PlayerIdx: 1, Alone: false})
	dealerHand := round.Hand(0)
	round.ApplyAction(DiscardAction{PlayerIdx: 0, Card: dealerHand[0]})

	// Now in play phase, player 1 leads (left of dealer)
	if round.CurrentPlayer() != 1 {
		t.Errorf("Player 1 should lead, got player %d", round.CurrentPlayer())
	}

	player1Hand := round.Hand(1)
	cardToPlay := player1Hand[0]

	err := round.ApplyAction(PlayCardAction{PlayerIdx: 1, Card: cardToPlay})
	if err != nil {
		t.Errorf("Play should succeed: %v", err)
	}

	// Card should be removed from hand
	if len(round.Hand(1)) != 4 {
		t.Errorf("Player 1 should have 4 cards, got %d", len(round.Hand(1)))
	}

	// Card should be in current trick
	trick := round.CurrentTrick()
	if len(trick) != 1 {
		t.Errorf("Trick should have 1 card, got %d", len(trick))
	}
}

func TestRoundGoAlone(t *testing.T) {
	round := NewRound(4, 0)
	deck := NewStandardDeck()
	round.Deal(deck)

	// Player 1 goes alone
	err := round.ApplyAction(OrderUpAction{PlayerIdx: 1, Alone: true})
	if err != nil {
		t.Errorf("Going alone should succeed: %v", err)
	}

	if !round.IsAlone() {
		t.Error("Round should be marked as alone")
	}
}

func TestRoundWrongPlayerTurn(t *testing.T) {
	round := NewRound(4, 0)
	deck := NewStandardDeck()
	round.Deal(deck)

	// Player 0 tries to pass, but it's player 1's turn
	err := round.ApplyAction(PassAction{PlayerIdx: 0})
	if err != ErrNotYourTurn {
		t.Errorf("Wrong player should get ErrNotYourTurn, got %v", err)
	}
}

func TestRoundLegalActions(t *testing.T) {
	round := NewRound(4, 0)
	deck := NewStandardDeck()
	round.Deal(deck)

	// In round 1, should have pass and order up options
	actions := round.LegalActions()
	if len(actions) < 2 {
		t.Errorf("Should have at least 2 actions (pass, order up), got %d", len(actions))
	}

	hasPass := false
	hasOrderUp := false
	for _, a := range actions {
		if a.Type() == ActionPass {
			hasPass = true
		}
		if a.Type() == ActionOrderUp {
			hasOrderUp = true
		}
	}

	if !hasPass || !hasOrderUp {
		t.Error("Should have both pass and order up as legal actions")
	}
}

func TestRoundResult(t *testing.T) {
	round := NewRound(4, 0)
	deck := NewStandardDeck()
	round.Deal(deck)

	// Quick setup to play phase
	round.ApplyAction(OrderUpAction{PlayerIdx: 1, Alone: false})
	round.ApplyAction(DiscardAction{PlayerIdx: 0, Card: round.Hand(0)[0]})

	// Play 5 tricks (simplified - just play first legal card)
	for trick := 0; trick < 5; trick++ {
		for card := 0; card < 4; card++ {
			currentPlayer := round.CurrentPlayer()
			if currentPlayer < 0 {
				break
			}
			hand := round.Hand(currentPlayer)
			if len(hand) > 0 {
				// Find a legal play
				actions := round.LegalActions()
				for _, a := range actions {
					if a.Type() == ActionPlayCard {
						round.ApplyAction(a)
						break
					}
				}
			}
		}
	}

	// Round should be complete
	if !round.IsComplete() {
		t.Error("Round should be complete after 5 tricks")
	}

	result := round.Result()
	if result.MakerTricks+round.TeamTricksWon(1-round.MakerTeam()) != 5 {
		t.Error("Total tricks should be 5")
	}
}
