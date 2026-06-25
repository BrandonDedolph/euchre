package app

import (
	"strings"
	"testing"

	"github.com/BrandonDedolph/euchre/internal/engine"
	"github.com/BrandonDedolph/euchre/internal/ui/components"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// fiveCardHand is a fixed, legal-looking hand used to render the hand area in
// tests without dealing a real round.
func fiveCardHand() []engine.Card {
	return []engine.Card{
		engine.NewCard(engine.Hearts, engine.Nine),
		engine.NewCard(engine.Spades, engine.Ten),
		engine.NewCard(engine.Clubs, engine.Jack),
		engine.NewCard(engine.Diamonds, engine.Queen),
		engine.NewCard(engine.Hearts, engine.Ace),
	}
}

// TestHandAreaConstantHeight is the core invariant behind the diegetic legend:
// the hand block must be exactly handAreaHeight tall in every phase so the table
// above it never shifts as the verb tag, arrows, and chip row come and go.
func TestHandAreaConstantHeight(t *testing.T) {
	g := renderableGamePlay(t, false, fullLayoutWidth, 40)
	g.suitSelector = components.NewSuitSelector(engine.Hearts) // exercise the bid-2 sub-line
	hand := fiveCardHand()
	hc := func(sel int) string {
		return components.RenderHand(hand, sel, nil, engine.Hearts, -1)
	}

	cases := []struct {
		name     string
		phase    engine.GamePhase
		yourTurn bool
		selected int
	}{
		{"play, selecting", engine.PhasePlay, true, 2},
		{"play, not your turn", engine.PhasePlay, false, -1},
		{"discard, selecting first", engine.PhaseDiscard, true, 0},
		{"discard, selecting last", engine.PhaseDiscard, true, 4},
		{"bid round 1, your turn", engine.PhaseBidRound1, true, -1},
		{"bid round 1, waiting", engine.PhaseBidRound1, false, -1},
		{"bid round 2, your turn", engine.PhaseBidRound2, true, -1},
		{"defend alone", engine.PhaseDefendAlone, true, -1},
	}
	for _, tc := range cases {
		area := g.renderHandArea("You (0)", tc.phase, tc.yourTurn, tc.selected, len(hand), hc(tc.selected))
		if h := lipgloss.Height(area); h != handAreaHeight {
			t.Errorf("%s: hand area height = %d, want %d (table would shift)", tc.name, h, handAreaHeight)
		}
	}
}

// TestHandAreaVerbTag checks the action verb for the selected card during play
// and discard: it appears on the bottom row (below the cards), not floating up by
// the player name, and is indented rather than flush-left (i.e. centered).
func TestHandAreaVerbTag(t *testing.T) {
	g := renderableGamePlay(t, false, fullLayoutWidth, 40)
	hand := fiveCardHand()

	for _, tc := range []struct {
		phase engine.GamePhase
		verb  string
	}{
		{engine.PhasePlay, "Play"},
		{engine.PhaseDiscard, "Discard"},
	} {
		const sel = 2
		hc := components.RenderHand(hand, sel, nil, engine.Hearts, -1)
		area := g.renderHandArea("You (0)", tc.phase, true, sel, len(hand), hc)
		verbPos := posOf(area, tc.verb)
		if verbPos.row < 0 {
			t.Fatalf("%v: verb tag %q not rendered", tc.phase, tc.verb)
		}
		if lastRow := lipgloss.Height(area) - 1; verbPos.row != lastRow {
			t.Errorf("%v: verb tag on row %d, want bottom row %d (below the cards)", tc.phase, verbPos.row, lastRow)
		}
		if verbPos.col == 0 {
			t.Errorf("%v: verb tag is flush-left, want it centered under the hand", tc.phase)
		}
	}
}

// TestHandAreaArrows checks move arrows appear during selection (and dim, rather
// than vanish, at the ends) but are absent when there is no cursor.
func TestHandAreaArrows(t *testing.T) {
	g := renderableGamePlay(t, false, fullLayoutWidth, 40)
	hand := fiveCardHand()
	render := func(phase engine.GamePhase, yourTurn bool, sel int) string {
		return g.renderHandArea("You (0)", phase, yourTurn, sel, len(hand), components.RenderHand(hand, sel, nil, engine.Hearts, -1))
	}

	// Mid-hand selection: both arrows present.
	area := render(engine.PhasePlay, true, 2)
	if !strings.Contains(area, "◄") || !strings.Contains(area, "►") {
		t.Errorf("play+selection: expected both ◄ and ► arrows")
	}
	// Boundary arrows still render (just dimmed) so the row width stays constant.
	if a := render(engine.PhasePlay, true, 0); !strings.Contains(a, "◄") {
		t.Errorf("leftmost selection: ◄ should still render (dimmed)")
	}
	// No cursor (bidding): no arrows.
	if a := render(engine.PhaseBidRound1, true, -1); strings.Contains(a, "◄") || strings.Contains(a, "►") {
		t.Errorf("bidding: arrows should not render without a card cursor")
	}
}

// TestHandChipsPerPhase verifies the cursor-less choices render as chips for the
// phases that need them, only on the human's turn, and never during play.
func TestHandChipsPerPhase(t *testing.T) {
	g := renderableGamePlay(t, false, fullLayoutWidth, 40)

	mustContain := func(phase engine.GamePhase, yourTurn bool, want ...string) {
		chips := g.handChips(phase, yourTurn)
		for _, w := range want {
			if !strings.Contains(chips, w) {
				t.Errorf("phase %v (yourTurn=%v): chips %q missing %q", phase, yourTurn, chips, w)
			}
		}
	}
	mustContain(engine.PhaseBidRound1, true, "Order up", "Pass", "Alone")
	mustContain(engine.PhaseBidRound2, true, "Call", "Pass")
	mustContain(engine.PhaseDefendAlone, true, "Defend alone", "Decline")

	if chips := g.handChips(engine.PhaseBidRound1, false); chips != "" {
		t.Errorf("bidding while waiting on others: want no chips, got %q", chips)
	}
	if chips := g.handChips(engine.PhasePlay, true); chips != "" {
		t.Errorf("play phase: controls live on the hand, want no chip row, got %q", chips)
	}
}

// TestHelpOverlayToggle checks "?" opens the in-place help sheet (preserving the
// game), any key closes it, and the minimal corner hint is always present.
func TestHelpOverlayToggle(t *testing.T) {
	g := renderableGamePlay(t, false, fullLayoutWidth, 40)

	if !strings.Contains(g.View(), "esc quit · ? help") {
		t.Errorf("corner control hint should always be present")
	}

	key := func(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

	if _, _ = g.handleKeyPress(key("?")); !g.showHelp {
		t.Fatalf("\"?\" should open the help overlay")
	}
	view := g.View()
	if !strings.Contains(view, "Controls") || !strings.Contains(view, "Press any key to close") {
		t.Errorf("open overlay should render the help sheet")
	}

	// Any key dismisses it, and the game underneath is still intact.
	if _, _ = g.handleKeyPress(key("x")); g.showHelp {
		t.Errorf("any key should close the help overlay")
	}
	if g.game == nil {
		t.Errorf("game state must survive opening/closing help")
	}
}
