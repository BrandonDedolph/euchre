package app

import (
	"fmt"

	"github.com/BrandonDedolph/euchre/internal/engine"
	"github.com/BrandonDedolph/euchre/internal/ui/theme"
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
	// Reserve a constant body height so the box doesn't grow or shrink between
	// tips of different lengths — otherwise the callout (and everything below it)
	// shifts each time the advice changes.
	bodyStyled := lipgloss.NewStyle().Width(innerW).Height(coachBoxBodyLines).
		Foreground(theme.ColText).Render(body)
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
		// If the human's play just completed the trick, lead with the move
		// feedback so it isn't swallowed by the trick result.
		body := g.trickNarration()
		if g.gradeMsg != "" {
			mark := "•"
			if g.gradeGood {
				mark = "✓"
			}
			body = mark + " " + g.gradeMsg + "  " + body
		}
		return "Trick", body, true
	case g.waitingForRoundAck:
		return "Hand over", g.message, true
	}

	// The human's own decision: give advice.
	if g.coach != nil && g.game.CurrentPlayer() == g.humanPlayer {
		if tip := g.coachTip(); tip != "" {
			return "Your turn", tip, true
		}
	}

	// Feedback on the move just made, shown while the AIs respond.
	if g.gradeMsg != "" {
		if g.gradeGood {
			return "Nice move ✓", g.gradeMsg, true
		}
		return "Heads up", g.gradeMsg, true
	}

	// Otherwise narrate what the player to act is doing.
	return g.opponentNarration()
}

// trickNarration describes who just won the trick and why, occasionally naming
// the specific winning card (a bower, a trump over the led suit) instead of the
// generic rule.
func (g *GamePlay) trickNarration() string {
	tr := g.completedTrick
	if tr == nil {
		return ""
	}
	who := g.tableView.PlayerNames[tr.Winner]
	if tr.Winner == g.humanPlayer {
		who = "You"
	}

	// Find the winning card so we can narrate the specific reason.
	var winCard engine.Card
	for _, pc := range tr.Cards {
		if pc.Player == tr.Winner {
			winCard = pc.Card
			break
		}
	}

	s := fmt.Sprintf("%s won the trick", who)
	switch {
	case winCard.IsRightBower(tr.Trump):
		return s + fmt.Sprintf(" — the right bower (%s) is the top trump, nothing beats it.", winCard)
	case winCard.IsLeftBower(tr.Trump):
		return s + fmt.Sprintf(" — the left bower (%s) plays as the second-highest trump.", winCard)
	case tr.WasTrumped:
		return s + fmt.Sprintf(" by trumping in with %s — any trump beats the led suit.", winCard)
	default:
		return s + ". The highest card of the led suit wins — unless someone trumps."
	}
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
		return g.tipDefendAlone()
	case engine.PhasePlay:
		return g.tipPlay()
	}
	return ""
}

// coachWould runs decide against a fresh game state and returns the coach's
// choice, or the zero Card when there's no coach (non-tutorial). Used to capture
// the recommendation just before the human commits a move, for grading.
func (g *GamePlay) coachWould(decide func(*engine.GameState) engine.Card) engine.Card {
	if g.coach == nil {
		return engine.Card{}
	}
	return decide(engine.NewGameState(g.game))
}

// gradeCard compares the human's played/discarded card to the coach's choice and
// records short feedback shown in the coach box until the human's next turn.
func (g *GamePlay) gradeCard(verb string, played, coachCard engine.Card) {
	if !g.tutorial || g.coach == nil {
		return
	}
	if played.Suit == coachCard.Suit && played.Rank == coachCard.Rank {
		g.gradeGood = true
		if verb == "discard" {
			g.gradeMsg = "Good pitch — Coach would discard the same card."
		} else {
			g.gradeMsg = "Nice — that's exactly the card Coach would play."
		}
		return
	}
	g.gradeGood = false
	if verb == "discard" {
		g.gradeMsg = fmt.Sprintf("Coach would have pitched %s instead.", coachCard)
	} else {
		g.gradeMsg = fmt.Sprintf("Coach would have played %s there.", coachCard)
	}
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

// --- Board-reading helpers (pure; unit-tested directly) ---

// trickStanding describes who currently holds a trick the human is following
// into: whether anyone is winning yet, whether that winner is the human's
// partner, and the card to beat.
type trickStanding struct {
	hasWinner       bool
	winnerSeat      int
	partnerWinning  bool
	winningCard     engine.Card
	winnerIsTrumped bool // current winner is an off-suit-trumping play
}

// readTrick inspects a trick from the perspective of `seat` (the human). It
// reports who's winning relative to the human's partner.
func readTrick(trick *engine.Trick, seat int, trump engine.Suit) trickStanding {
	var st trickStanding
	if trick == nil || trick.Size() == 0 {
		return st
	}
	win, ok := trick.WinningCard()
	if !ok {
		return st
	}
	st.hasWinner = true
	st.winnerSeat = trick.Winner()
	st.winningCard = win
	st.partnerWinning = engine.IsPartner(seat, st.winnerSeat)
	st.winnerIsTrumped = win.IsTrump(trump) && trick.LeadSuit() != trump
	return st
}

// seatPhrase names the seat from the human's point of view for tip text.
func seatPhrase(seat int, names []string) string {
	if seat >= 0 && seat < len(names) {
		return names[seat]
	}
	return "an opponent"
}

// positionClue returns a short positional-play clause based on how many cards
// have already been played when the human acts. Empty when nothing useful to
// say. canWin = the recommended card actually beats the current winner.
func positionClue(playedSoFar int, partnerWinning, canWin bool) string {
	switch playedSoFar {
	case 1:
		// 2nd hand: two opponents still to act behind you. The second-hand-low
		// rationale only applies when an opponent leads/is winning — if your own
		// partner is ahead, "duck under partner" is the right frame (said elsewhere).
		if !canWin && !partnerWinning {
			return "Second hand plays low — save high cards for when they'll matter."
		}
	case 2:
		// 3rd hand: only the last opponent acts after you.
		if !partnerWinning && canWin {
			return "Third hand high — win it now before the last player can."
		}
	case 3:
		// Last to act: full information.
		if canWin {
			return "You're last to act, so the minimum card that wins is enough."
		}
	}
	return ""
}

// trumpSeen counts how many trump cards have appeared across completed tricks
// plus the cards already on the table this trick.
func trumpSeen(history []engine.TrickResult, current []engine.PlayedCard, trump engine.Suit) int {
	n := 0
	for _, tr := range history {
		for _, pc := range tr.Cards {
			if pc.Card.IsTrump(trump) {
				n++
			}
		}
	}
	for _, pc := range current {
		if pc.Card.IsTrump(trump) {
			n++
		}
	}
	return n
}

// makerContext describes the human's stake in the round: whether their team
// made trump, and each team's trick count so far.
type makerContext struct {
	isMaker      bool
	myTricks     int
	theirTricks  int
	tricksPlayed int
}

func (g *GamePlay) makerContext() makerContext {
	var mc makerContext
	round := g.game.Round()
	if round == nil {
		return mc
	}
	myTeam := engine.Team(g.humanPlayer)
	mc.isMaker = round.MakerTeam() == myTeam
	mc.myTricks = round.TeamTricksWon(myTeam)
	mc.theirTricks = round.TeamTricksWon(1 - myTeam)
	mc.tricksPlayed = len(round.TrickHistory())
	return mc
}

// stakeClue folds the single most relevant maker/defender + trick-pressure
// insight into one short clause, or "" when nothing pressing applies.
func (mc makerContext) stakeClue() string {
	if mc.isMaker {
		switch {
		case mc.myTricks >= 3:
			return "You've made it — now press for the march."
		case mc.myTricks == 2:
			return "One more trick makes your bid — take it."
		case mc.theirTricks == 2:
			return "Two against you — you must take this or risk the euchre."
		}
		return ""
	}
	// Defending.
	switch {
	case mc.theirTricks == 2:
		return "You're defending — this trick stops the euchre, grab it."
	case mc.myTricks == 2:
		return "One more trick and they're euchred — fight for it."
	}
	return ""
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

	seat := g.seatNote(g.game.Dealer())
	body := strength + " Coach would " + rec + "."
	if dec.Alone {
		body += " Alone risks 2 to win 4 — worth it only with a near-sure hand."
	} else if seat != "" {
		body += " " + seat
	}
	return body
}

// seatNote returns a one-line caution about bidding from the human's seat
// relative to the dealer (left of dealer hands the dealer the turn-up;
// dealer's partner should be sure they want the dealer to take it).
func (g *GamePlay) seatNote(dealer int) string {
	switch (g.humanPlayer - dealer + 4) % 4 {
	case 1:
		return "Left of the dealer, ordering hands them the turn-up — bid only on real strength."
	case 2:
		return "You're the dealer's partner; order up only if you want them to take that card."
	default:
		return ""
	}
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
	body := fmt.Sprintf("Coach would %s — %s.", rec, reason)

	// Next theory: the same-colour suit as the turned-down card tends to be
	// strong in round 2 (its bowers were likely passed, so they're still live).
	if dec.CallSuit.SameColor(turnedDown) {
		body += " That's the next suit — same colour as the turn-down, often strong here."
	} else if dec.Alone {
		body += " Alone risks 2 to win 4 — worth it only with a near-sure hand."
	}
	return body
}

// tipDiscard advises which card to pitch after the dealer picks up trump. It
// only claims a void-creation rationale when the discard actually empties a
// suit; otherwise it states the real reason (weakest off-suit, keeping aces).
func (g *GamePlay) tipDiscard() string {
	hand := g.game.Hand(g.humanPlayer)
	trump := g.game.Trump()
	card := g.coach.DecideDiscard(engine.NewGameState(g.game), hand)

	return discardTipText(card, hand, trump)
}

// discardTipText explains the discard pick honestly: a void-creation rationale
// only when the chosen card actually empties a side suit, else the real reason.
// Pure for unit testing.
func discardTipText(card engine.Card, hand []engine.Card, trump engine.Suit) string {
	// Count how many cards share the discard's effective suit (the left bower
	// counts as trump, not its printed suit). If it's the only one, discarding
	// it creates a void in that side suit.
	suitCount := 0
	for _, c := range hand {
		if c.EffectiveSuit(trump) == card.EffectiveSuit(trump) {
			suitCount++
		}
	}

	if !card.IsTrump(trump) && suitCount == 1 {
		return fmt.Sprintf("Coach discards %s — pitching your last %s empties that suit so you can trump in there later.", card, card.Suit.String())
	}
	if card.Rank == engine.Ace {
		return fmt.Sprintf("Coach discards %s — even an ace goes when it's your only weak side card.", card)
	}
	return fmt.Sprintf("Coach discards %s — it's your weakest off-suit card; keep trump and off-aces.", card)
}

// tipDefendAlone advises whether to take a lone defense, derived from the
// actual hand. It mirrors the AI's shouldDefendAlone criteria: you need a
// realistic shot at 3 tricks single-handed (right bower, the left plus another
// trump, 3+ trump, or 2 trump with a bower and an off-ace).
func (g *GamePlay) tipDefendAlone() string {
	trump := g.game.Trump()
	return defendAloneTipText(shapeOf(g.game.Hand(g.humanPlayer), trump))
}

// defendAloneTipText mirrors the AI's shouldDefendAlone criteria from a hand
// shape. Pure for unit testing.
func defendAloneTipText(s handShape) string {
	hasBower := s.rightBower || s.leftBower
	strong := false
	switch {
	case s.rightBower && (s.trump >= 2 || s.offAces >= 1):
		strong = true
	case s.leftBower && s.trump >= 2:
		strong = true
	case s.trump >= 3:
		strong = true
	case s.trump >= 2 && hasBower && s.offAces >= 1:
		strong = true
	}

	if strong {
		desc := fmt.Sprintf("%d trump", s.trump)
		if bp := s.bowerPhrase(); bp != "" {
			desc += " with " + bp
		}
		return fmt.Sprintf("Worth defending alone — %s gives a real shot at 3 tricks for 4 points.", desc)
	}
	return "Decline — defend alone only with a very strong hand (a bower plus trump support), and this hand falls short."
}

// tipPlay advises which card to play to the current trick, tailoring the reason
// to whether you're leading, following suit, trumping in, or pitching. It reads
// the actual board: who's winning (partner vs opponent), seat position, trump
// already gone, and whether you're making or defending.
func (g *GamePlay) tipPlay() string {
	round := g.game.Round()
	if round == nil {
		return ""
	}
	state := engine.NewGameState(g.game)
	trump := state.Trump()
	card := g.coach.DecidePlay(state)
	trick := round.Trick()
	names := g.tableView.PlayerNames

	if trick == nil || trick.Size() == 0 {
		return g.tipLead(card, trump, round)
	}

	return playTipText(card, trump, trick, g.humanPlayer, names, g.makerContext())
}

// playTipText builds the follow/trump/pitch advice for a non-empty trick. Pure
// so it can be unit-tested against constructed tricks; tipPlay supplies the
// live game state.
func playTipText(card engine.Card, trump engine.Suit, trick *engine.Trick, seat int, names []string, mc makerContext) string {
	led := trick.LeadSuit()
	st := readTrick(trick, seat, trump)
	canWin := st.hasWinner && cardBeats(card, st.winningCard, trump)
	pos := positionClue(trick.Size(), st.partnerWinning, canWin)
	stake := mc.stakeClue()

	var msg string
	switch {
	case card.EffectiveSuit(trump) == led:
		// Following suit.
		if st.partnerWinning {
			msg = fmt.Sprintf("Coach plays %s — your partner is winning, so duck low and don't waste a high card.", card)
		} else if canWin {
			msg = fmt.Sprintf("Coach plays %s — %s is winning; beat it with the lowest card that does.", card, seatPhrase(st.winnerSeat, names))
		} else {
			msg = fmt.Sprintf("Coach plays %s — you must follow %s but can't beat %s, so throw your lowest.", card, suitLabel(led), seatPhrase(st.winnerSeat, names))
		}
	case card.IsTrump(trump):
		// Void in the lead suit, choosing to trump. When the partner was already
		// winning, trumping in actually takes the trick FROM the partner — it's a
		// forced overtrump, not a "partner keeps it" duck.
		if st.partnerWinning {
			msg = fmt.Sprintf("Coach plays %s — you're void and hold only trump, so you must trump in even though your partner was winning; play your lowest.", card)
		} else if canWin {
			msg = fmt.Sprintf("Coach plays %s — you're void in %s, so trump over %s to steal it.", card, suitLabel(led), seatPhrase(st.winnerSeat, names))
		} else {
			msg = fmt.Sprintf("Coach plays %s — forced to trump but it can't beat %s, so spend your lowest.", card, seatPhrase(st.winnerSeat, names))
		}
	default:
		// Void and discarding off-suit.
		if st.partnerWinning {
			msg = fmt.Sprintf("Coach plays %s — partner has the trick, so safely pitch your weakest loser.", card)
		} else {
			msg = fmt.Sprintf("Coach plays %s — you can't follow %s and can't win, so pitch your weakest loser.", card, suitLabel(led))
		}
	}

	// Append the single most relevant board-reading clue (stake pressure wins
	// over generic positional advice when both apply).
	if stake != "" {
		return msg + " " + stake
	}
	if pos != "" {
		return msg + " " + pos
	}
	return msg
}

// cardBeats reports whether card a beats card b under trump (mirrors the AI's
// comparison so tips agree with the highlighted pick).
func cardBeats(a, b engine.Card, trump engine.Suit) bool {
	aT, bT := a.IsTrump(trump), b.IsTrump(trump)
	switch {
	case aT && !bT:
		return true
	case !aT && bT:
		return false
	case aT && bT:
		return a.TrumpValue(trump) > b.TrumpValue(trump)
	default:
		// Both non-trump: only the lead suit can win; an off-suit card that
		// doesn't match the winner's suit can't beat it.
		if a.Suit != b.Suit {
			return false
		}
		return a.OffSuitValue() > b.OffSuitValue()
	}
}

// tipLead explains the rationale behind the coach's lead, matched to the card
// it actually picked: draw trump, cash an off-ace, or lead low to preserve.
func (g *GamePlay) tipLead(card engine.Card, trump engine.Suit, round *engine.Round) string {
	hand := g.game.Hand(g.humanPlayer)
	seen := trumpSeen(round.TrickHistory(), round.CurrentTrick(), trump)
	return leadTipText(card, trump, shapeOf(hand, trump), seen)
}

// leadTipText explains the lead rationale matched to the card actually chosen.
// Pure for unit testing.
func leadTipText(card engine.Card, trump engine.Suit, s handShape, trumpSeenCount int) string {
	// Trump-counting insight: if most trump are already gone, a remaining off-ace
	// is likely boss.
	var note string
	if trumpSeenCount >= 5 && card.Rank == engine.Ace && !card.IsTrump(trump) {
		note = " Most trump are gone, so your ace should be boss now."
	}

	switch {
	case card.IsTrump(trump) && s.trump >= 2:
		return fmt.Sprintf("Coach leads %s — with %d trump, lead high to draw out the opponents' trump.", card, s.trump) + note
	case card.Rank == engine.Ace && !card.IsTrump(trump):
		return fmt.Sprintf("Coach leads %s — cash a sure off-suit ace for a quick trick while it's safe.", card) + note
	case card.IsTrump(trump):
		return fmt.Sprintf("Coach leads %s — lead your lowest trump to preserve your bowers.", card) + note
	default:
		return fmt.Sprintf("Coach leads %s — lead low off-suit to keep your trump back for later.", card) + note
	}
}
