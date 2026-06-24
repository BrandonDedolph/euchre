package app

import (
	"fmt"

	"github.com/BrandonDedolph/euchre/internal/engine"
	"github.com/BrandonDedolph/euchre/internal/ui/components"
	"github.com/BrandonDedolph/euchre/internal/ui/theme"
	"github.com/charmbracelet/lipgloss"
)

// concept is a single teachable-moment popup: a title, an explanatory body, and
// an optional card to render in context (e.g. the actual left bower in hand).
type concept struct {
	id        string
	title     string
	body      string
	card      *engine.Card
	cardTrump engine.Suit
}

var goingAloneConcept = &concept{
	id:    "going_alone",
	title: "GOING ALONE",
	body:  "The maker can play the hand without their partner, who sits out. It's riskier with one fewer player — but sweeping all five tricks alone is worth 4 points instead of 2.",
}

var euchreConcept = &concept{
	id:    "euchre",
	title: "EUCHRED!",
	body:  "The makers failed to take 3 of the 5 tricks, so the defending team scores 2 points. Calling trump on a weak hand is the fastest way to get euchred — bid carefully.",
}

var marchConcept = &concept{
	id:    "march",
	title: "A MARCH",
	body:  "Taking all five tricks is a march — 2 points (4 if you went alone). It's the reward for a strong trump hand that keeps the lead.",
}

// leftBowerConcept builds the popup for the actual left bower in the human's
// hand, showing the card with its trump pip.
func leftBowerConcept(c engine.Card, trump engine.Suit) *concept {
	cc := c
	return &concept{
		id:    "left_bower",
		title: "THE LEFT BOWER",
		body: fmt.Sprintf("Your %s plays as trump right now — not as a %s. With %s as trump, the jack of the same color becomes the SECOND-highest trump, beaten only by the %s jack (the right bower).",
			c, singularSuit(c.Suit), trump.String(), trump.String()),
		card:      &cc,
		cardTrump: trump,
	}
}

// singularSuit returns the singular lowercase suit name (e.g. "club").
func singularSuit(s engine.Suit) string {
	switch s {
	case engine.Clubs:
		return "club"
	case engine.Spades:
		return "spade"
	case engine.Hearts:
		return "heart"
	case engine.Diamonds:
		return "diamond"
	}
	return "card"
}

// teachableConcept returns the highest-priority not-yet-shown concept whose
// trigger condition currently holds, or nil if there's nothing to teach.
func (g *GamePlay) teachableConcept() *concept {
	if g.shownConcepts == nil {
		return nil
	}

	trump := g.tableView.Trump
	if !g.shownConcepts["left_bower"] && trump != engine.NoSuit && !g.isDealing && !g.isShuffling {
		for _, c := range g.game.Hand(g.humanPlayer) {
			if c.IsLeftBower(trump) {
				return leftBowerConcept(c, trump)
			}
		}
	}

	if !g.shownConcepts["going_alone"] && g.tableView.MakerAlone {
		return goingAloneConcept
	}

	if g.waitingForRoundAck {
		if hist := g.game.RoundHistory(); len(hist) > 0 {
			last := hist[len(hist)-1]
			if !g.shownConcepts["euchre"] && last.WasEuchred {
				return euchreConcept
			}
			if !g.shownConcepts["march"] && last.MakerTricks == 5 {
				return marchConcept
			}
		}
	}

	return nil
}

// maybeShowTeachable queues a teachable popup if one is due and none is showing.
// It marks the concept shown immediately so it fires at most once per game.
func (g *GamePlay) maybeShowTeachable() {
	if !g.tutorial || g.pendingPopup != nil {
		return
	}
	if c := g.teachableConcept(); c != nil {
		g.shownConcepts[c.id] = true
		g.pendingPopup = c
	}
}

// dismissPopup closes the current popup and chains to the next one, if any.
func (g *GamePlay) dismissPopup() {
	g.pendingPopup = nil
	g.maybeShowTeachable()
}

// renderPopup renders the active teachable-moment modal centered on the screen.
func (g *GamePlay) renderPopup(width, height int) string {
	p := g.pendingPopup
	const textW = 50

	parts := []string{
		lipgloss.NewStyle().Foreground(theme.ColGold).Bold(true).Render("💡  " + p.title),
		"",
	}
	if p.card != nil {
		cv := components.NewCardView(*p.card)
		cv.Trump = p.cardTrump
		parts = append(parts, lipgloss.PlaceHorizontal(textW, lipgloss.Center, cv.Render()), "")
	}
	parts = append(parts,
		lipgloss.NewStyle().Width(textW).Align(lipgloss.Center).Foreground(theme.ColText).Render(p.body),
		"",
		theme.Current.Muted.Italic(true).Render("Press Enter to continue"),
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.ColGold).
		Padding(1, 3).
		Render(lipgloss.JoinVertical(lipgloss.Center, parts...))

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
