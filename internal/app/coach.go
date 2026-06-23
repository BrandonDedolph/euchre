package app

import (
	"fmt"

	"github.com/bran/euchre/internal/engine"
)

// coachTip returns a short, contextual coaching tip for the human's current
// decision in interactive-tutorial mode, or "" when there's nothing to advise
// (not tutorial mode, not the human's turn, or an animation/ack is pending).
//
// The advice comes from a strong rule-based AI sitting in the human's seat, so
// it always reflects the actual randomly-dealt hand rather than a script.
func (g *GamePlay) coachTip() string {
	if !g.tutorial || g.coach == nil {
		return ""
	}
	if g.isShuffling || g.isDealing || g.waitingForTrickAck || g.waitingForRoundAck {
		return ""
	}
	if g.game.CurrentPlayer() != g.humanPlayer {
		return ""
	}

	switch g.game.Phase() {
	case engine.PhaseBidRound1:
		return g.tipBidRound1()
	case engine.PhaseBidRound2:
		return g.tipBidRound2()
	case engine.PhaseDiscard:
		return g.tipDiscard()
	case engine.PhaseDefendAlone:
		return "Defending alone is a long shot — only take it with the right bower or three-plus trump."
	case engine.PhasePlay:
		return g.tipPlay()
	}
	return ""
}

// handShape summarizes a hand relative to a candidate trump suit: how much
// trump it holds, which bowers, and how many off-suit aces.
type handShape struct {
	trump      int
	rightBower bool
	leftBower  bool
	offAces    int
}

func shapeOf(hand []engine.Card, trump engine.Suit) handShape {
	var s handShape
	for _, c := range hand {
		switch {
		case c.IsRightBower(trump):
			s.trump++
			s.rightBower = true
		case c.IsLeftBower(trump):
			s.trump++
			s.leftBower = true
		case c.IsTrump(trump):
			s.trump++
		case c.Rank == engine.Ace:
			s.offAces++
		}
	}
	return s
}

// bowerPhrase describes the bowers held, e.g. "both bowers" / "the right bower".
func (s handShape) bowerPhrase() string {
	switch {
	case s.rightBower && s.leftBower:
		return "both bowers"
	case s.rightBower:
		return "the right bower"
	case s.leftBower:
		return "the left bower"
	}
	return ""
}

func suitLabel(s engine.Suit) string { return s.Symbol() + " " + s.String() }

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// tipBidRound1 advises on ordering up the turned card.
func (g *GamePlay) tipBidRound1() string {
	trump := g.game.TurnedCard().Suit
	s := shapeOf(g.game.Hand(g.humanPlayer), trump)

	strength := fmt.Sprintf("With %s as trump you'd hold %d trump", suitLabel(trump), s.trump)
	if bp := s.bowerPhrase(); bp != "" {
		strength += " (" + bp + ")"
	}
	if s.offAces > 0 {
		strength += fmt.Sprintf(" plus %d off-suit ace%s", s.offAces, plural(s.offAces))
	}
	strength += "."
	if g.game.Dealer() == g.humanPlayer {
		strength += fmt.Sprintf(" As dealer you'd also pick up the %s.", g.game.TurnedCard())
	}

	dec := g.coach.DecideBid(engine.NewGameState(g.game), 1)
	rec := "pass"
	if !dec.Pass {
		rec = "order it up"
		if dec.Alone {
			rec = "order it up and go alone"
		}
	}
	return strength + " Coach would " + rec + "."
}

// tipBidRound2 advises on naming a suit after the turned card is rejected.
func (g *GamePlay) tipBidRound2() string {
	turnedDown := g.game.TurnedCard().Suit
	dec := g.coach.DecideBid(engine.NewGameState(g.game), 2)
	if dec.Pass {
		return fmt.Sprintf("No suit gives you enough strength, so Coach would pass. (You can't name %s — it was turned down.)", suitLabel(turnedDown))
	}

	s := shapeOf(g.game.Hand(g.humanPlayer), dec.CallSuit)
	rec := "call " + suitLabel(dec.CallSuit)
	if dec.Alone {
		rec += " and go alone"
	}
	reason := fmt.Sprintf("it's your strongest suit (%d trump", s.trump)
	if bp := s.bowerPhrase(); bp != "" {
		reason += ", " + bp
	}
	reason += ")"
	return fmt.Sprintf("Coach would %s — %s.", rec, reason)
}

// tipDiscard advises which card to pitch after the dealer picks up trump.
func (g *GamePlay) tipDiscard() string {
	hand := g.game.Hand(g.humanPlayer)
	card := g.coach.DecideDiscard(engine.NewGameState(g.game), hand)
	return fmt.Sprintf("Coach would discard %s — pitch your weakest card, ideally leaving a side suit short so you can trump in later.", card)
}

// tipPlay advises which card to play to the current trick, tailoring the reason
// to whether you're leading, following suit, trumping in, or pitching.
func (g *GamePlay) tipPlay() string {
	round := g.game.Round()
	if round == nil {
		return ""
	}
	state := engine.NewGameState(g.game)
	trump := state.Trump()
	card := g.coach.DecidePlay(state)
	trick := round.Trick()

	if trick == nil || trick.Size() == 0 {
		return fmt.Sprintf("Coach would lead %s — on lead, a high trump or an off-suit ace puts your opponents under pressure.", card)
	}

	led := trick.LeadSuit()
	switch {
	case card.EffectiveSuit(trump) == led:
		return fmt.Sprintf("Coach would play %s — you must follow %s; go high to win the trick unless your partner is already taking it.", card, suitLabel(led))
	case card.IsTrump(trump):
		return fmt.Sprintf("Coach would play %s — you're void in %s, so trump in to steal the trick.", card, suitLabel(led))
	default:
		return fmt.Sprintf("Coach would play %s — you can't follow %s and can't win, so pitch your weakest loser.", card, suitLabel(led))
	}
}
