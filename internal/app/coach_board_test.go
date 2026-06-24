package app

import (
	"strings"
	"testing"

	"github.com/BrandonDedolph/euchre/internal/engine"
)

// names mirrors the default seating: human at 0, partner at 2, opponents 1 & 3.
var testNames = []string{"You", "West", "Partner", "East"}

func newTrick(trump engine.Suit, plays ...engine.PlayedCard) *engine.Trick {
	tr := engine.NewTrick(trump)
	for _, p := range plays {
		tr.Play(p.Player, p.Card)
	}
	return tr
}

// --- H1/H2: partner-winning vs opponent-winning while following suit ---

func TestPlayTip_PartnerWinning_DucksNotSteal(t *testing.T) {
	trump := engine.Spades
	// Partner (seat 2) leads and is winning with A♥; human follows with a low ♥.
	trick := newTrick(trump, engine.PlayedCard{Player: 2, Card: card(engine.Hearts, engine.Ace)})
	pick := card(engine.Hearts, engine.Nine)

	tip := playTipText(pick, trump, trick, 0, testNames, makerContext{})
	if !strings.Contains(tip, "duck") {
		t.Errorf("partner winning should advise ducking: %q", tip)
	}
	if strings.Contains(strings.ToLower(tip), "steal") {
		t.Errorf("must not frame as stealing when partner is winning: %q", tip)
	}
	if !strings.Contains(tip, "partner is winning") {
		t.Errorf("should name partner as winner: %q", tip)
	}
}

func TestPlayTip_OpponentWinning_BeatIt(t *testing.T) {
	trump := engine.Spades
	// Opponent West (seat 1) leads K♥; human can win with A♥.
	trick := newTrick(trump, engine.PlayedCard{Player: 1, Card: card(engine.Hearts, engine.King)})
	pick := card(engine.Hearts, engine.Ace)

	tip := playTipText(pick, trump, trick, 0, testNames, makerContext{})
	if !strings.Contains(tip, "beat it") {
		t.Errorf("opponent winning + can beat should say beat it: %q", tip)
	}
	if !strings.Contains(tip, "West") {
		t.Errorf("should name the opponent (West): %q", tip)
	}
}

func TestPlayTip_OpponentWinning_CantBeat_ThrowLow(t *testing.T) {
	trump := engine.Spades
	trick := newTrick(trump, engine.PlayedCard{Player: 1, Card: card(engine.Hearts, engine.Ace)})
	pick := card(engine.Hearts, engine.Nine) // cannot beat the ace
	tip := playTipText(pick, trump, trick, 0, testNames, makerContext{})
	if !strings.Contains(tip, "can't beat") || !strings.Contains(tip, "lowest") {
		t.Errorf("can't-beat case should advise throwing lowest: %q", tip)
	}
}

// --- H2 bug: void + only trump while partner is winning is NOT a steal ---

func TestPlayTip_VoidOnlyTrump_PartnerWinning_NotSteal(t *testing.T) {
	trump := engine.Spades
	// Partner (seat 2) is winning with A♥ (led). Human is void in ♥ and trumps.
	trick := newTrick(trump, engine.PlayedCard{Player: 2, Card: card(engine.Hearts, engine.Ace)})
	pick := card(engine.Spades, engine.Nine) // forced trump

	tip := playTipText(pick, trump, trick, 0, testNames, makerContext{})
	if strings.Contains(strings.ToLower(tip), "steal") {
		t.Errorf("forced trump while partner wins must not be framed as stealing: %q", tip)
	}
	// Trumping in here takes the trick FROM the partner, so the tip must NOT
	// claim the partner keeps/has the trick.
	if strings.Contains(strings.ToLower(tip), "partner has the trick") {
		t.Errorf("must not claim partner keeps the trick when forced to overtrump: %q", tip)
	}
	if !strings.Contains(tip, "trump in") || !strings.Contains(strings.ToLower(tip), "partner was winning") {
		t.Errorf("should frame it as a forced overtrump even though the partner was ahead: %q", tip)
	}
}

func TestPlayTip_VoidTrump_OpponentWinning_Steal(t *testing.T) {
	trump := engine.Spades
	// Opponent East (seat 3) winning with A♥; human void and trumps over it.
	trick := newTrick(trump, engine.PlayedCard{Player: 3, Card: card(engine.Hearts, engine.Ace)})
	pick := card(engine.Spades, engine.Ten)
	tip := playTipText(pick, trump, trick, 0, testNames, makerContext{})
	if !strings.Contains(tip, "trump over") || !strings.Contains(tip, "steal it") {
		t.Errorf("trumping over an opponent should be framed as stealing: %q", tip)
	}
}

// --- H3: positional play clues ---

func TestPositionClue_ThirdHandHigh(t *testing.T) {
	got := positionClue(2, false, true)
	if !strings.Contains(got, "Third hand high") {
		t.Errorf("3rd hand, opp winning, can win → third hand high: %q", got)
	}
}

func TestPositionClue_SecondHandLow(t *testing.T) {
	got := positionClue(1, false, false)
	if !strings.Contains(got, "Second hand") {
		t.Errorf("2nd hand, can't win → second hand low: %q", got)
	}
}

func TestPositionClue_NoneWhenPartnerWinningThird(t *testing.T) {
	if got := positionClue(2, true, false); got != "" {
		t.Errorf("partner winning should suppress third-hand-high: %q", got)
	}
}

func TestPositionClue_SecondHandLow_SuppressedWhenPartnerWinning(t *testing.T) {
	// 2nd hand, can't win, but the partner is the one ahead: the second-hand-low
	// frame is wrong here (it's really "duck under partner", said elsewhere).
	if got := positionClue(1, true, false); got != "" {
		t.Errorf("partner winning should suppress second-hand-low: %q", got)
	}
}

// --- M1: trump counting helper ---

func TestTrumpSeen(t *testing.T) {
	trump := engine.Spades
	hist := []engine.TrickResult{
		{Cards: []engine.PlayedCard{
			{Player: 0, Card: card(engine.Spades, engine.Jack)}, // right bower (trump)
			{Player: 1, Card: card(engine.Clubs, engine.Jack)},  // left bower (trump)
			{Player: 2, Card: card(engine.Hearts, engine.Ace)},  // not trump
			{Player: 3, Card: card(engine.Spades, engine.Nine)}, // trump
		}},
	}
	cur := []engine.PlayedCard{{Player: 0, Card: card(engine.Spades, engine.Ace)}} // trump
	if n := trumpSeen(hist, cur, trump); n != 4 {
		t.Errorf("trumpSeen = %d, want 4", n)
	}
}

// --- M2: maker vs defender stake framing ---

func TestStakeClue_DefenderStopsEuchre(t *testing.T) {
	mc := makerContext{isMaker: false, theirTricks: 2}
	if got := mc.stakeClue(); !strings.Contains(got, "stops the euchre") {
		t.Errorf("defender with opp at 2 should warn about euchre: %q", got)
	}
}

func TestStakeClue_MakerMadeIt(t *testing.T) {
	mc := makerContext{isMaker: true, myTricks: 3}
	if got := mc.stakeClue(); !strings.Contains(got, "march") {
		t.Errorf("maker with 3 tricks should mention march: %q", got)
	}
}

func TestStakeClue_AppendedToPlayTip(t *testing.T) {
	trump := engine.Spades
	trick := newTrick(trump, engine.PlayedCard{Player: 1, Card: card(engine.Hearts, engine.King)})
	pick := card(engine.Hearts, engine.Ace)
	mc := makerContext{isMaker: false, theirTricks: 2}
	tip := playTipText(pick, trump, trick, 0, testNames, mc)
	if !strings.Contains(tip, "defending") {
		t.Errorf("defensive stake should be folded into the play tip: %q", tip)
	}
}

// --- cardBeats sanity ---

func TestCardBeats(t *testing.T) {
	trump := engine.Spades
	if !cardBeats(card(engine.Spades, engine.Nine), card(engine.Hearts, engine.Ace), trump) {
		t.Error("any trump should beat an off-suit ace")
	}
	if cardBeats(card(engine.Hearts, engine.Nine), card(engine.Hearts, engine.Ace), trump) {
		t.Error("9 should not beat A in the same suit")
	}
	if cardBeats(card(engine.Diamonds, engine.Ace), card(engine.Hearts, engine.King), trump) {
		t.Error("off-suit ace of a different suit can't beat the led-suit king")
	}
}

// --- L1: richer trick narration ---

func TestTrickNarration_RightBower(t *testing.T) {
	g := NewGamePlayWithSettings(GameSettings{Variant: "Standard", Tutorial: true})
	g.completedTrick = &engine.TrickResult{
		Winner: 2,
		Trump:  engine.Spades,
		Cards: []engine.PlayedCard{
			{Player: 2, Card: card(engine.Spades, engine.Jack)}, // right bower
		},
	}
	got := g.trickNarration()
	if !strings.Contains(got, "right bower") {
		t.Errorf("should narrate the right bower: %q", got)
	}
}

func TestTrickNarration_Trumped(t *testing.T) {
	g := NewGamePlayWithSettings(GameSettings{Variant: "Standard", Tutorial: true})
	g.completedTrick = &engine.TrickResult{
		Winner:     1,
		Trump:      engine.Spades,
		LeadSuit:   engine.Hearts,
		WasTrumped: true,
		Cards: []engine.PlayedCard{
			{Player: 0, Card: card(engine.Hearts, engine.Ace)},
			{Player: 1, Card: card(engine.Spades, engine.Nine)},
		},
	}
	got := g.trickNarration()
	if !strings.Contains(got, "trumping in with") {
		t.Errorf("should narrate the trumping card: %q", got)
	}
}

// --- H4: lead rationale matches the chosen card ---

func TestLeadTip_DrawTrump(t *testing.T) {
	trump := engine.Spades
	s := handShape{trump: 3, rightBower: true}
	got := leadTipText(card(engine.Spades, engine.Jack), trump, s, 0)
	if !strings.Contains(got, "draw out") {
		t.Errorf("leading high trump with 2+ should mention drawing trump: %q", got)
	}
}

func TestLeadTip_OffAce(t *testing.T) {
	trump := engine.Spades
	s := handShape{trump: 1, offAces: 1}
	got := leadTipText(card(engine.Hearts, engine.Ace), trump, s, 0)
	if !strings.Contains(got, "sure off-suit ace") {
		t.Errorf("leading an off-ace should mention cashing a sure trick: %q", got)
	}
}

func TestLeadTip_BossAceWhenTrumpGone(t *testing.T) {
	trump := engine.Spades
	s := handShape{offAces: 1}
	got := leadTipText(card(engine.Hearts, engine.Ace), trump, s, 5)
	if !strings.Contains(got, "Most trump are gone") {
		t.Errorf("ace with most trump gone should be flagged as boss: %q", got)
	}
}

func TestLeadTip_BossAceThreshold(t *testing.T) {
	trump := engine.Spades
	s := handShape{offAces: 1}
	// 7 trump exist total; with only 4 seen, up to 3 are still out, so the "boss"
	// claim is not yet reliable.
	if got := leadTipText(card(engine.Hearts, engine.Ace), trump, s, 4); strings.Contains(got, "Most trump are gone") {
		t.Errorf("4 trump seen should NOT claim the ace is boss: %q", got)
	}
	// At 5 seen, at most 2 remain — the claim is reliable.
	if got := leadTipText(card(engine.Hearts, engine.Ace), trump, s, 5); !strings.Contains(got, "Most trump are gone") {
		t.Errorf("5 trump seen should claim the ace is boss: %q", got)
	}
}

func TestLeadTip_LowTrumpPreservesBowers(t *testing.T) {
	trump := engine.Spades
	s := handShape{trump: 1}
	got := leadTipText(card(engine.Spades, engine.Nine), trump, s, 0)
	if !strings.Contains(got, "preserve your bowers") {
		t.Errorf("single low trump lead should mention preserving bowers: %q", got)
	}
}

// --- M4: honest discard rationale ---

func TestDiscardTip_CreatesVoid(t *testing.T) {
	trump := engine.Spades
	hand := []engine.Card{
		card(engine.Spades, engine.Ace),    // trump
		card(engine.Spades, engine.King),   // trump
		card(engine.Hearts, engine.King),   // two hearts...
		card(engine.Hearts, engine.Queen),  // ...
		card(engine.Diamonds, engine.Nine), // lone diamond
	}
	got := discardTipText(card(engine.Diamonds, engine.Nine), hand, trump)
	if !strings.Contains(got, "empties that suit") {
		t.Errorf("discarding a lone side card should claim a void: %q", got)
	}
}

func TestDiscardTip_NoVoidClaimed(t *testing.T) {
	trump := engine.Spades
	hand := []engine.Card{
		card(engine.Spades, engine.Ace),
		card(engine.Hearts, engine.King),
		card(engine.Hearts, engine.Queen),
		card(engine.Hearts, engine.Nine), // three hearts: pitching one leaves the suit
		card(engine.Diamonds, engine.Ten),
	}
	got := discardTipText(card(engine.Hearts, engine.Nine), hand, trump)
	if strings.Contains(got, "empties that suit") {
		t.Errorf("discard that does not empty a suit must not claim a void: %q", got)
	}
	if !strings.Contains(got, "weakest off-suit") {
		t.Errorf("should state the real reason: %q", got)
	}
}

// --- L2: per-hand defend-alone advice ---

func TestDefendAloneTip_StrongHand(t *testing.T) {
	got := defendAloneTipText(handShape{trump: 3, rightBower: true})
	if !strings.Contains(got, "Worth defending alone") {
		t.Errorf("right bower + 3 trump should encourage defending: %q", got)
	}
}

func TestDefendAloneTip_WeakHand(t *testing.T) {
	got := defendAloneTipText(handShape{trump: 1})
	if !strings.Contains(got, "Decline") {
		t.Errorf("one trump, no bower should discourage defending: %q", got)
	}
}

// --- M3: bidding seat note ---

func TestSeatNote_LeftOfDealer(t *testing.T) {
	g := NewGamePlayWithSettings(GameSettings{Variant: "Standard", Tutorial: true})
	// Human is seat 0; dealer at seat 3 → human is left of the dealer (offset 1).
	if got := g.seatNote(3); !strings.Contains(got, "hands them the turn-up") {
		t.Errorf("left-of-dealer note: %q", got)
	}
	// Dealer is seat 2 → human is dealer's partner (offset 2).
	if got := g.seatNote(2); !strings.Contains(got, "dealer's partner") {
		t.Errorf("dealer's-partner note: %q", got)
	}
	// Human is the dealer → no extra seat caution.
	if got := g.seatNote(0); got != "" {
		t.Errorf("dealer seat should give no caution: %q", got)
	}
}
