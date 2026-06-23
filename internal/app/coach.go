package app

import (
	"fmt"

	"github.com/bran/euchre/internal/engine"
	"github.com/bran/euchre/internal/ui/theme"
	"github.com/charmbracelet/lipgloss"
)

// renderCoachBox renders the always-on tutorial callout as a bordered gold box,
// wrapping the body to fit maxWidth. Returns "" when there's nothing to show.
func (g *GamePlay) renderCoachBox(maxWidth int) string {
	title, body, ok := g.tutorBox()
	if !ok {
		return ""
	}
	innerW := maxWidth - 4 // account for border + horizontal padding
	if innerW > 66 {
		innerW = 66
	}
	if innerW < 12 {
		innerW = 12
	}

	header := lipgloss.NewStyle().Bold(true).Foreground(theme.ColGold).
		Render("💡 COACH · " + title)
	bodyStyled := lipgloss.NewStyle().Width(innerW).Foreground(theme.ColText).Render(body)
	content := lipgloss.JoinVertical(lipgloss.Left, header, bodyStyled)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColGold).
		Padding(0, 1).
		Render(content)
}

// tutorBox returns the content for the always-on coach callout in tutorial
// mode: a short title, a body that narrates whatever is happening (the deal,
// other players' turns, trick results) or advises on the human's own decision,
// and ok=false when there's nothing to show or tutorial mode is off.
func (g *GamePlay) tutorBox() (title, body string, ok bool) {
	if !g.tutorial {
		return "", "", false
	}

	switch {
	case g.isShuffling:
		return "Dealing", "Shuffling the 24-card Euchre deck — only 9, 10, Jack, Queen, King, Ace in each suit.", true
	case g.isDealing:
		return "Dealing", "Euchre deals in packets — 2s and 3s — until everyone has 5 cards. The next card is turned face-up to start the bidding.", true
	case g.waitingForTrickAck && g.completedTrick != nil:
		return "Trick", g.trickNarration(), true
	case g.waitingForRoundAck:
		return "Hand over", g.message, true
	}

	// The human's own decision: give advice.
	if g.coach != nil && g.game.CurrentPlayer() == g.humanPlayer {
		if tip := g.coachTip(); tip != "" {
			return "Your turn", tip, true
		}
	}

	// Otherwise narrate what the player to act is doing.
	return g.opponentNarration()
}

// trickNarration describes who just won the trick and why.
func (g *GamePlay) trickNarration() string {
	tr := g.completedTrick
	if tr == nil {
		return ""
	}
	who := g.tableView.PlayerNames[tr.Winner]
	if tr.Winner == g.humanPlayer {
		who = "You"
	}
	s := fmt.Sprintf("%s won the trick", who)
	if tr.WasTrumped {
		s += " by trumping in"
	}
	s += ". The highest card of the led suit wins — unless someone trumps."
	return s
}

// opponentNarration describes what the player currently to act (an AI) is doing.
func (g *GamePlay) opponentNarration() (title, body string, ok bool) {
	cur := g.game.CurrentPlayer()
	if cur < 0 || cur == g.humanPlayer {
		return "", "", false
	}
	name := g.tableView.PlayerNames[cur]

	switch g.game.Phase() {
	case engine.PhaseBidRound1:
		return "Bidding", fmt.Sprintf("%s is deciding whether to order up the %s and make it trump.", name, g.game.TurnedCard()), true
	case engine.PhaseBidRound2:
		return "Bidding", fmt.Sprintf("The turn-up was passed. %s may now name a different trump suit — or pass.", name), true
	case engine.PhaseDiscard:
		return "Discard", fmt.Sprintf("%s took the turn card into hand and is pitching one card back.", name), true
	case engine.PhaseDefendAlone:
		return "Defend alone?", fmt.Sprintf("%s is deciding whether to defend alone against the lone hand.", name), true
	case engine.PhasePlay:
		return "Playing", fmt.Sprintf("%s is choosing a card to play.", name), true
	}
	return "", "", false
}

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

// coachPickIndex returns the index in the human's hand of the card the coach
// would play or discard right now, or -1 when there's no pick to spotlight
// (not tutorial mode, not the human's play/discard turn, or mid-animation).
func (g *GamePlay) coachPickIndex() int {
	if !g.tutorial || g.coach == nil {
		return -1
	}
	if g.isShuffling || g.isDealing || g.waitingForTrickAck || g.waitingForRoundAck {
		return -1
	}
	if g.game.CurrentPlayer() != g.humanPlayer {
		return -1
	}

	hand := g.game.Hand(g.humanPlayer)
	var pick engine.Card
	switch g.game.Phase() {
	case engine.PhasePlay:
		pick = g.coach.DecidePlay(engine.NewGameState(g.game))
	case engine.PhaseDiscard:
		pick = g.coach.DecideDiscard(engine.NewGameState(g.game), hand)
	default:
		return -1
	}

	for i, c := range hand {
		if c.Suit == pick.Suit && c.Rank == pick.Rank {
			return i
		}
	}
	return -1
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
